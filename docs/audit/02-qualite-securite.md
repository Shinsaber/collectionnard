# Livrable 2 — Qualité et sécurité du code

> Audit réalisé le 2026-10-11 sur le commit `279efc4`. Revue **défensive** : constat, impact,
> correctif. Aucun exploit n'est décrit ni testé.
> **Constaté** = vérifié dans le code ou par une commande ; **supposé** = à confirmer.
> Gravités : **critique** (compromission ou perte de données probable en prod) · **haute**
> (exploitable par un utilisateur authentifié ou impact fort) · **moyenne** · **basse**.

## Commandes exécutées (via `nix develop path:$PWD -c …`, le flake n'étant pas suivi par git)

| Commande | Résultat |
|----------|----------|
| `go vet ./...` | **3 avertissements** : `collection_test.go:130`, `relation_test.go:82`, `relation_test.go:100` (« using resp before checking for errors »). |
| `go test ./...` | **OK** : `handler` (0,34 s) et `middleware` passent ; `store`, `validator`, `model`, `config`, `main` : « no test files ». |
| `gofmt -l .` | **7 fichiers non formatés** : 5 `*_test.go`, `model/model.go`, `testutil/testutil.go`. |
| `govulncheck ./...` (v1.8.0, base du 2026-10-08) | **14 vulnérabilités atteintes** par le code : `golang.org/x/text` ×2, `go-chi/chi` ×2, `jackc/pgx` ×1, `golang-jwt/jwt` ×1, bibliothèque standard ×8 (toolchain 1.26.8 → corrigées en 1.26.9). Détail en S3. |
| `go list -m -u all` | Mises à jour disponibles pour tous les modules directs sauf `google/uuid` et `jsonschema/v5` (v6 est un nouveau module). |
| `npm install` | OK, 163 paquets. Avertissement : `@primevue/themes@4.5.4` **déprécié**, migrer vers `@primeuix/themes`. |
| `npm run type-check` (`vue-tsc --build --force`) | **OK**, aucune erreur. |
| `npm audit` | **17 vulnérabilités** (15 hautes, 2 modérées). Runtime : `axios 1.15.0` (nombreuses CVE, fix 1.20.0), `vue ≤3.5.41` (`@vue/server-renderer`, SSR non utilisé). Le reste est outillage de build (vite, postcss, tailwind 3 → braces/micromatch, nanoid, browserslist…). |
| `npm outdated` | Majeures disponibles : PrimeVue 5.0.2, Tailwind 4.3.3, Vite 8.3.4, Pinia 4.0.3, vue-router 5.4.0, TypeScript 7.0.2, vue-tsc 3.3.13, @vitejs/plugin-vue 6.0.10, primeicons 8.0.2. |

---

## A. Sécurité

### S1 — JWT sans expiration, sans révocation, rôle figé dans le token — **critique**

**Constaté.** `GenerateToken` n'émet que `sub` et `role` : ni `exp`, ni `iat`, ni `nbf`
([server/internal/middleware/auth.go:95-102](../../server/internal/middleware/auth.go#L95-L102)).
Le parseur n'impose ni `jwt.WithValidMethods` ni `jwt.WithExpirationRequired`
([auth.go:51-57](../../server/internal/middleware/auth.go#L51-L57)) ; il accepte toute la famille
HMAC (HS256/384/512) signée avec le secret. Le rôle est lu dans le token et jamais relu en base
([auth.go:75](../../server/internal/middleware/auth.go#L75)). Le « logout » du front ne fait que
supprimer le token du `localStorage` ([web/src/stores/auth.ts:29-33](../../web/src/stores/auth.ts#L29-L33)).

**Impact.** Un token émis une fois est valide pour toujours. Un utilisateur rétrogradé d'admin à
user, ou supprimé, conserve ses droits tant qu'il garde son token. Un token volé (cf. S10) ne peut
pas être invalidé sans changer le secret pour tout le monde. La spec promet pourtant « 24 h »
([api/openapi.yaml:1124](../../api/openapi.yaml#L1124)).

**Correctif.**

```go
// config.go : JWTTTL time.Duration (env JWT_TTL, défaut 24h)

// middleware/auth.go
type Claims struct {
    Role string `json:"role"`
    jwt.RegisteredClaims
}

func (a *Auth) GenerateToken(userID uuid.UUID, role string) (string, error) {
    now := time.Now()
    claims := Claims{
        Role: role,
        RegisteredClaims: jwt.RegisteredClaims{
            Subject:   userID.String(),
            IssuedAt:  jwt.NewNumericDate(now),
            ExpiresAt: jwt.NewNumericDate(now.Add(a.TTL)),
            Issuer:    "collectionnard",
        },
    }
    return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(a.Secret)
}

// Parse
token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, keyFunc,
    jwt.WithValidMethods([]string{"HS256"}),
    jwt.WithExpirationRequired(),
    jwt.WithIssuer("collectionnard"),
)
```

Pour la révocation et le rôle : soit relire l'utilisateur en base dans `Authenticate` (une requête
indexée par PK par appel, acceptable à cette échelle), soit un TTL court (1 h) + un endpoint
`POST /auth/refresh`. Côté front, lire `exp` pour déconnecter proactivement.

### S2 — Secret JWT par défaut codé en dur — **critique**

**Constaté.** Si `JWT_SECRET` est vide, le serveur démarre avec
`"dev-secret-change-in-production"` ([server/config/config.go:32-35](../../server/config/config.go#L32-L35)).
`docker-compose.yml` fournit `"change-me-in-production"` en clair ([docker-compose.yml:28](../../docker-compose.yml#L28)) ;
les mots de passe PostgreSQL `colec/colec` sont aussi en clair ([docker-compose.yml:6-8](../../docker-compose.yml#L6-L8), [config.go:29](../../server/config/config.go#L29)) et le compte `admin/admin` du seed est documenté dans le README
([README.md:46-49](../../README.md#L46-L49)).

**Impact.** Un déploiement « tel quel » (le README dit : `docker compose up --build`) permet à
quiconque connaît le dépôt de forger un token admin, sans expiration (S1). Avec S1, c'est une
compromission complète du catalogue.

**Correctif.**

```go
jwtSecret := os.Getenv("JWT_SECRET")
if len(jwtSecret) < 32 {
    return nil, errors.New("JWT_SECRET manquant ou trop court (32 octets minimum)")
}
```

Dans `docker-compose.yml` : `JWT_SECRET: ${JWT_SECRET:?JWT_SECRET requis}` et
`POSTGRES_PASSWORD: ${POSTGRES_PASSWORD:?}`, valeurs dans un `.env` ignoré par git (déjà interdit
en lecture par `.claude/settings.json`). Garder un défaut uniquement dans `flake.nix` (dev) et le
dire dans le README. Forcer le changement du mot de passe `admin` au premier démarrage, ou
générer un mot de passe aléatoire affiché dans les logs.

### S3 — Dépendances avec vulnérabilités connues — **haute**

**Constaté** par `govulncheck` (symboles réellement atteints) :

| ID | Module | Trouvé | Corrigé | Chemin d'appel dans le projet | Pertinence |
|----|--------|--------|---------|-------------------------------|------------|
| GO-2026-5777, GO-2026-5775 | `go-chi/chi/v5` | 5.0.12 | 5.3.0 | `middleware.RealIP` ([handler.go:36](../../server/internal/handler/handler.go#L36)) : usurpation d'IP via `X-Forwarded-For` | Réelle : le port 8080 est publié directement (S6), donc l'en-tête peut venir du client. Aujourd'hui l'IP ne sert qu'aux logs. |
| GO-2026-5004 | `jackc/pgx/v5` | 5.5.5 | 5.9.2 | `pgxpool.Pool.Query` → `sanitize.SanitizeSQL` | Faible : nécessite le protocole simple (non utilisé). À corriger par principe. |
| GO-2025-3553 | `golang-jwt/jwt/v5` | 5.2.1 | 5.2.2 | `jwt.Parse` ([auth.go:51](../../server/internal/middleware/auth.go#L51)) : allocation excessive au parsing d'en-tête | Réelle : déni de service sur toute route authentifiée. |
| GO-2026-6629, GO-2026-5970 | `golang.org/x/text` | 0.14.0 | 0.41.0 | via `pgxpool.New` (SASL/precis) | Faible (entrée = mot de passe PG). |
| GO-2026-6603…6617 (8) | stdlib `net/http`, `crypto/tls`, `net/textproto` | go1.26.8 | go1.26.9 | `ListenAndServe`, `ParseMultipartForm`, `ServeFile` | **La prod compile avec `golang:1.22-alpine`** ([server/Dockerfile:1](../../server/Dockerfile#L1)), bien plus ancien : le nombre réel de CVE stdlib en prod est supérieur. |

**Constaté** par `npm audit` : `axios 1.15.0` (prototype pollution, header injection, ReDoS…,
corrigé en 1.20.0) est la seule dépendance **runtime navigateur** réellement concernée. `vue
3.5.32` : l'avis porte sur `@vue/server-renderer` (SSR, non utilisé). Les 14 autres sont des
dépendances de build (`vite`, `postcss`, `tailwindcss 3` via `braces`/`micromatch`/`chokidar`,
`nanoid`, `browserslist`, `form-data`…) : exposition limitée au poste de dev et au build Docker.

**Correctif.** Lot « sécurité » du livrable 3 : `go get -u` ciblé (chi 5.3.2, pgx 5.11.0 →
impose `go 1.25+` dans `go.mod`, jwt 5.3.1, x/crypto 0.58.0, x/text 0.43.0), `golang:1.26-alpine`
dans le Dockerfile ; `npm audit fix` (sans `--force`) pour axios/vue/postcss/vite 6.x. Ajouter
`govulncheck` et `npm audit --audit-level=high` à la CI.

### S4 — Traversée de graphe sans borne de résultat — **haute**

**Constaté.** La CTE récursive explore jusqu'à 10 niveaux dans chaque direction
([server/internal/handler/graph.go:18-24](../../server/internal/handler/graph.go#L18-L24)) avec
`UNION ALL` et un anti-cycle par chemin (`NOT i2.id = ANY(g.path)`,
[store/graph.go:26-61](../../server/internal/store/graph.go#L26-L61)). Cette forme énumère **tous
les chemins** distincts vers chaque nœud avant le `DISTINCT ON (id)` final ; il n'y a aucun
`LIMIT` sur le nombre de lignes ni de nœuds. `relation_type_ids` accepte une liste non bornée
([graph.go:27-33](../../server/internal/handler/graph.go#L27-L33)). Accessible à tout utilisateur
authentifié.

**Impact.** Sur le seed (12 items) c'est instantané. Sur un catalogue dense (ex. BD/Manga :
Personne reliée à des dizaines d'œuvres, chaque œuvre à plusieurs éditions, chaque édition à 40
tomes, plus les relations inverses), le nombre de chemins croît de façon combinatoire : une seule
requête `max_depth=10` peut occuper PostgreSQL plusieurs minutes et retourner des mégaoctets.
Déni de service sans intention malveillante (un simple clic sur « profondeur 10 »).

**Correctif.**
1. Remplacer la stratégie « chemins » par une stratégie « nœuds visités » : `UNION` (dédoublonne par
   ligne) sur `(id, depth)` sans colonne `path`, ou mieux une boucle côté Go niveau par niveau avec
   `WHERE id <> ALL($visited)`.
2. Ajouter `LIMIT $3` (ex. 500 nœuds) à la requête des nœuds et renvoyer un indicateur
   `truncated: true` dans `GraphResponse` (spec + types TS à mettre à jour).
3. Baisser le maximum à 5 par défaut, 10 réservé aux admins (ou configurable).
4. Borner `relation_type_ids` (ex. 20) et renvoyer 400 au-delà.
5. `context.WithTimeout` (ex. 5 s) sur la requête graphe et un `statement_timeout` sur le pool.

### S5 — Absence de limites sur les entrées HTTP et de limitation de débit — **moyenne**

**Constaté.**
- `decodeJSON` lit le corps sans `http.MaxBytesReader` ni `DisallowUnknownFields`
  ([server/internal/handler/handler.go:147-150](../../server/internal/handler/handler.go#L147-L150)).
  Seul l'upload est borné ([upload.go:26](../../server/internal/handler/upload.go#L26)). Un
  `attributes` de 500 Mo est accepté puis stocké en JSONB.
- `http.Server` a `ReadTimeout` 10 s / `WriteTimeout` 30 s mais ni `ReadHeaderTimeout` ni
  `MaxHeaderBytes` ([server/main.go:42-48](../../server/main.go#L42-L48)).
- Aucune limitation de débit sur `POST /auth/login` ; `bcrypt` coût 10 à chaque tentative
  ([handler/auth.go:34](../../server/internal/handler/auth.go#L34)). Force brute et épuisement CPU
  possibles ; la liste des utilisateurs du seed est publique.
- `search` et `tag` ne sont pas bornés en longueur ([handler/item.go:18-19](../../server/internal/handler/item.go#L18-L19)).

**Correctif.** `r.Body = http.MaxBytesReader(w, r.Body, 1<<20)` dans `decodeJSON` (1 Mo suffit
largement pour des attributs) ; `ReadHeaderTimeout: 5 * time.Second` ; limiteur par IP sur
`/auth/login` (`golang.org/x/time/rate` ou `chi/httprate`, 10 tentatives/min) — en se souvenant
que `RealIP` n'est fiable que derrière nginx (S6) ; `maxLength` 200 sur les paramètres de
recherche.

### S6 — Configuration de déploiement : exposition réseau, conteneurs root, nginx nu — **moyenne**

**Constaté.**
- PostgreSQL publié sur l'hôte `5432:5432` avec `colec/colec` ([docker-compose.yml:9-10](../../docker-compose.yml#L9-L10)) ; l'API publiée `8080:8080` ([docker-compose.yml:31-32](../../docker-compose.yml#L31-L32)) alors que nginx la proxifie déjà ([web/nginx.conf:12-18](../../web/nginx.conf#L12-L18)). L'exposition directe de l'API rend `X-Forwarded-For` falsifiable (S3, chi).
- Aucun `USER` dans `server/Dockerfile` ([server/Dockerfile:9-14](../../server/Dockerfile#L9-L14)) : le binaire tourne en root avec un volume `uploads` en écriture. `web/Dockerfile` idem (`nginx:alpine` tourne en root par défaut).
- `nginx.conf` : aucun en-tête de sécurité (`X-Content-Type-Options`, `X-Frame-Options`/`frame-ancestors`, `Referrer-Policy`, `Content-Security-Policy`), pas de `client_max_body_size` (voir S7), pas de TLS. La spec OpenAPI suppose « transmis via HTTPS » ([api/openapi.yaml:191](../../api/openapi.yaml#L191)) mais rien ne termine TLS.
- Pas de `.dockerignore` dans le dépôt (`git ls-files`) : `COPY . .` embarque `node_modules/`, `.git/`, `.pgdata/` s'ils sont présents localement ([web/Dockerfile:6](../../web/Dockerfile#L6), [server/Dockerfile:6](../../server/Dockerfile#L6)).
- Pas de healthcheck sur le service `server` ; `web` démarre dès que `server` est lancé, pas quand il écoute ([docker-compose.yml:45-46](../../docker-compose.yml#L45-L46)).

**Correctif.** Retirer `ports` de `postgres` et `server` (réseau interne compose) ; `USER nonroot`
(`adduser -D -u 10001 app` + `chown` du dossier uploads) ; `.dockerignore` dans `server/` et
`web/` ; dans nginx :

```nginx
client_max_body_size 10m;
add_header X-Content-Type-Options nosniff always;
add_header X-Frame-Options DENY always;
add_header Referrer-Policy strict-origin-when-cross-origin always;
add_header Content-Security-Policy "default-src 'self'; img-src 'self' data:; style-src 'self' 'unsafe-inline'; script-src 'self'" always;
```

Documenter le reverse-proxy TLS attendu (Caddy/Traefik) et ajouter `GET /healthz` côté Go
(utilisé par `healthcheck:` compose).

### S7 — Upload : incohérence de taille et détails de robustesse — **moyenne**

**Constaté.** Les fondamentaux sont bons : `MaxBytesReader` ([upload.go:26](../../server/internal/handler/upload.go#L26)),
type réel via `http.DetectContentType` sur 512 octets ([upload.go:42-48](../../server/internal/handler/upload.go#L42-L48)),
liste blanche JPEG/PNG/GIF/WebP (pas de SVG → pas de XSS par image), extension **dérivée du MIME**
et nom `uuid` ([upload.go:50-71](../../server/internal/handler/upload.go#L50-L71)), service des
fichiers avec rejet de `..` et `/` et liste blanche d'extensions ([handler.go:44-61](../../server/internal/handler/handler.go#L44-L61)),
test de traversée présent ([upload_test.go:150-166](../../server/internal/handler/upload_test.go#L150-L166)).

Écarts :
- nginx limite les corps à **1 Mo** par défaut (pas de `client_max_body_size`) alors que le serveur
  accepte 10 Mo : via le front, tout upload > 1 Mo échoue avec un 413 nginx que le front
  n'interprète pas (erreur avalée, cf. Q5). **Bug fonctionnel probable en prod.**
- `size` renvoyé = `header.Size` fourni par le client ([upload.go:91](../../server/internal/handler/upload.go#L91)), pas le nombre d'octets écrits par `io.Copy`.
- `ParseMultipartForm(maxSize)` : toute erreur est renvoyée comme « Fichier trop volumineux » ([upload.go:28-31](../../server/internal/handler/upload.go#L28-L31)).
- Aucun quota global ni nettoyage des fichiers orphelins (une image retirée d'une galerie reste sur disque) ; aucun lien BDD ↔ fichier.
- `http.ServeFile` sans `X-Content-Type-Options: nosniff` ni `Cache-Control` (les noms sont des UUID immuables : `Cache-Control: public, max-age=31536000, immutable` serait idéal).

**Correctif.** `client_max_body_size 10m;` dans nginx (ou lire `MAX_UPLOAD_SIZE` des deux
côtés) ; `n, err := io.Copy(...)` et renvoyer `n` ; distinguer `*http.MaxBytesError` ; en-têtes
de cache sur `/uploads/*` ; prévoir une table `uploads` (ou un job de garbage collection) en Phase 2.

### S8 — Les JSON Schemas des méta-types ne sont jamais validés à l'écriture — **moyenne**

**Constaté.** `CreateItemType` / `UpdateItemType` (et les équivalents relation/collection) stockent
`attributes_schema` tel quel ; le seul traitement est « vide → `{}` »
([handler/item_type.go:42-44](../../server/internal/handler/item_type.go#L42-L44)). Le schéma n'est
compilé qu'à la première écriture d'un item ([validator/jsonschema.go:18-25](../../server/internal/validator/jsonschema.go#L18-L25)),
et recompilé **à chaque requête**.

**Impact.** Un admin qui enregistre un schéma invalide (faute de frappe dans `type`, `$ref`
cassé) rend impossible toute création/modification d'items de ce type : 422 « invalid schema »
jusqu'à correction, sans message à l'enregistrement. **Supposé** : jsonschema v5 résout les `$ref`
par URL avec un chargeur `file://` par défaut ; un `$ref` externe dans un schéma pourrait
provoquer une lecture de fichier côté serveur (admin uniquement, à confirmer sur la version).

**Correctif.** Dans les trois handlers de méta-types, compiler le schéma avant insertion
(`validator.CompileSchema(raw) error`) et renvoyer 422 détaillé ; exiger `"type": "object"` à la
racine (promis par la spec [api/openapi.yaml:267](../../api/openapi.yaml#L267)) ; désactiver le
chargement d'URL (`compiler.LoadURL = func(string) (io.ReadCloser, error) { return nil, errors.New("refs externes interdites") }`) ;
mettre en cache le schéma compilé par `(type_id, updated_at)` (`sync.Map`).

### S9 — Énumération d'utilisateurs par temps de réponse au login — **basse**

**Constaté.** Utilisateur inconnu → 401 immédiat ([handler/auth.go:24-29](../../server/internal/handler/auth.go#L24-L29)) ;
utilisateur connu → `bcrypt.CompareHashAndPassword` (~70 ms à coût 10) puis 401. Les messages
sont identiques (bien), pas les temps.

**Correctif.** Comparer contre un hash factice constant quand l'utilisateur n'existe pas, ou
toujours exécuter bcrypt.

### S10 — Token JWT dans `localStorage` — **basse** (acceptée, à documenter)

**Constaté.** [web/src/stores/auth.ts:8,17,32](../../web/src/stores/auth.ts#L8) et
[web/src/api/client.ts:43](../../web/src/api/client.ts#L43). Aucun `v-html`, `innerHTML` ni
`eval` dans `web/src` (grep), les seules données utilisateur liées à une URL sont des `:src`
d'images ([ItemDetailView.vue:57,123](../../web/src/views/ItemDetailView.vue#L57), [DynamicForm.vue:136](../../web/src/components/common/DynamicForm.vue#L136), [GraphView.vue:76](../../web/src/views/GraphView.vue#L76)) : surface XSS faible.

**Impact.** Si une XSS apparaît (dépendance, future fonctionnalité), le token est volé et, avec
S1, utilisable sans limite.

**Correctif.** Avec S1 (TTL court) et S6 (CSP), le risque devient acceptable pour un usage entre
amis. Alternative plus robuste pour la Phase 3 (profils publics) : cookie `HttpOnly; Secure;
SameSite=Strict` + protection CSRF.

### S11 — Recherche `ILIKE` sur `attributes::text` : jokers non échappés et faux positifs — **basse**

**Constaté.** `"%"+f.Search+"%"` ([store/item.go:25-26](../../server/internal/store/item.go#L25-L26)) :
`%` et `_` saisis par l'utilisateur sont interprétés ; la recherche porte sur le texte JSON brut
(clés comprises : chercher `type` renvoie tous les items ayant une clé `type`). Pas d'injection
SQL (paramètre `$n`), mais résultats surprenants et balayage complet de la table.

**Correctif.** Échapper `%`, `_`, `\` ; chercher sur les valeurs (`jsonb_each_text`) ou mieux un
`tsvector` généré (voir Q3).

### Points vérifiés sans écart (pour mémoire)

- **Injection SQL** : toutes les valeurs HTTP passent par `$n`. Les `fmt.Sprintf` interpolent
  uniquement des numéros de placeholder, des fragments constants, la clause de tri issue de la
  liste blanche `buildSortClause` ([store/helpers.go:31-49](../../server/internal/store/helpers.go#L31-L49))
  et un nom de table constant ([store/relation_type.go:270](../../server/internal/store/relation_type.go#L270)). **Constaté.**
- **Autorisation collections** : contrôle `col.UserID != GetUserID` présent sur les 8 handlers
  concernés ([handler/collection.go:37,76,104,129,158,212,259](../../server/internal/handler/collection.go#L37)) et filtre
  `c.user_id = $2` dans `ListItemCollections` ([store/collection.go:278](../../server/internal/store/collection.go#L278)).
  Réponse 404 (pas 403) pour ne pas révéler l'existence. **Constaté.** Le store, lui, ne filtre pas
  par `user_id` : la sécurité repose sur le handler (voir Q4 pour la factorisation).
- **Validation JSON Schema** : appelée sur les 6 écritures d'attributs. **Constaté.**
- **Admin** : les 16 routes d'écriture du catalogue et l'upload sont sous `RequireAdmin`. **Constaté.**
- **Mots de passe** : bcrypt, hash jamais exposé (`GetUserByID` ne le sélectionne pas). **Constaté.**

---

## B. Qualité

### Q1 — Couverture de tests réelle insuffisante — **haute**

**Constaté.**
- **19 handlers sur 42 sans aucun test** : `UpdateItemType`, les 5 routes `RelationType`, les
  5 routes `CollectionType`, `UpdateItem`, `DeleteItem`, `ListItemRelations`,
  `ListItemCollections`, `GetRelation`, `UpdateRelation`, `DeleteRelation`, `ListTags`,
  `UpdateCollection`, `ListCollectionItems`, `UpdateCollectionItem`, `RemoveItemFromCollection`.
- Les tests existants ne couvrent que le chemin nominal + un cas d'erreur ; aucun test des
  **contrôles de propriété** sur `UpdateCollection`, `ListCollectionItems`,
  `UpdateCollectionItem`, `RemoveItemFromCollection` (seuls `GetCollection` et `DeleteCollection`
  sont testés, [collection_test.go:87-165](../../server/internal/handler/collection_test.go#L87-L165)).
- **Zéro test du store** (tout le SQL, y compris la CTE du graphe, `buildSortClause`, les
  transactions, les cascades manuelles de `DeleteItem`) : `go test` affiche `[no test files]`.
- **Zéro test du validateur** (`FormatValidationError`, schémas vides/`null`).
- **Zéro test front** : pas de vitest, alors que `SchemaBuilder` (parse ↔ build de schéma, 908 l.)
  et `DynamicForm` sont de la logique pure testable.
- `go vet` : 3 avertissements ; `gofmt` : 7 fichiers.

**Correctif.**
1. Tests handlers manquants avec le `MockStore` existant (pattern déjà en place, 1 session).
2. Tests store **d'intégration** : paquet `store` avec `//go:build integration`, connexion via
   `DATABASE_URL`, base jetable créée par `pg-start` (ou `testcontainers-go`), rejouer
   `schema.sql` + `seed.sql`, cas : CTE graphe multi-chemin (Pilot atteint par 2 chemins → 1 nœud,
   2 arêtes), tri, filtres, cascade `DeleteItem`, 409 sur doublons.
3. `vitest` + `@vue/test-utils` : `SchemaBuilder.parseSchema(build(x)) ≡ x` sur les 13 schémas
   du seed, `DynamicForm` (rendu des widgets, validation `required`).
4. Corriger les 3 `vet` (`if err != nil { t.Fatal }` avant `resp.Body`) et `gofmt -w`.

### Q2 — Bug : `PUT /relations/{id}` et `PUT /collections/{c}/items/{i}` sans `attributes` — **moyenne**

**Constaté.** Si le corps JSON omet `attributes` (autorisé par la spec, voir livrable 1 § 3.1),
`in.Attributes` est `nil` : avec un schéma non vide, `json.Unmarshal(nil)` échoue → 422
« invalid JSON data » ([validator/jsonschema.go:27-30](../../server/internal/validator/jsonschema.go#L27-L30)) ;
avec un schéma `{}` la validation est sautée et `UPDATE … SET attributes = $2` reçoit `NULL` →
violation `NOT NULL` → **500** ([store/relation.go:123-126](../../server/internal/store/relation.go#L123-L126),
[store/collection.go:239-243](../../server/internal/store/collection.go#L239-L243), [db/schema.sql:108,137](../../db/schema.sql#L108)).

**Correctif.** Dans les deux handlers :

```go
if len(in.Attributes) == 0 {
    respondValidationError(w, "attributes est obligatoire", nil)
    return
}
```

et `required: [attributes]` dans la spec. Même garde-fou dans `CreateItem` : `item_type_id`
nul est testé par comparaison de chaîne ([handler/item.go:50](../../server/internal/handler/item.go#L50)) ;
utiliser `in.ItemTypeID == uuid.Nil`.

### Q3 — Performance : N+1, recherche sans index, recompilation de schéma — **moyenne**

**Constaté.**
- `ListItems` charge les tags par **une requête par item** ([store/item.go:88-95](../../server/internal/store/item.go#L88-L95)) : 21 requêtes pour une page de 20.
- `ListRelationTypes` charge les jonctions par **deux requêtes par type** ([store/relation_type.go:69-74](../../server/internal/store/relation_type.go#L69-L74)).
- `ILIKE` sur `attributes::text` ([store/item.go:25](../../server/internal/store/item.go#L25)) : l'index GIN `idx_items_attributes` ([db/schema.sql:166](../../db/schema.sql#L166)) ne sert pas ; balayage séquentiel + conversion texte de chaque JSONB.
- Le filtre tag compare `lower(t.name) = lower($n)` ([store/item.go:31](../../server/internal/store/item.go#L31)) alors que les tags sont déjà normalisés en minuscules à l'insertion ([store/item.go:270](../../server/internal/store/item.go#L270)) : l'index `UNIQUE(name)` n'est pas utilisable.
- Schéma JSON recompilé à chaque validation (S8).
- Les index présents couvrent les jointures du graphe (`source`, `target`, `type`) et les collections : adaptés. Aucun index sur `items.created_at` / `updated_at` (seuls tris autorisés) : acceptable tant que la table reste petite.

**Correctif.** `LEFT JOIN LATERAL (SELECT array_agg(json_build_object(...)) …)` ou une seconde
requête `WHERE item_id = ANY($1)` regroupée en Go ; idem pour les jonctions ; `lower()` retiré du
filtre tag ; pour la recherche : colonne générée
`search_tsv tsvector GENERATED ALWAYS AS (jsonb_to_tsvector('simple', attributes, '["string"]')) STORED`
+ index GIN, ou à minima `jsonb_path_exists`. Cache des schémas compilés.

### Q4 — Architecture : duplication et responsabilités mal placées — **moyenne**

**Constaté.**
- Le bloc « charger la collection + vérifier le propriétaire + 404 » est copié **7 fois** dans
  `handler/collection.go` (l. 31-40, 71-79, 99-107, 124-132, 153-161, 207-215, 254-262).
- La construction SQL dynamique (`where`, `argN`, `fmt.Sprintf`) est réécrite à la main dans
  `item.go`, `collection.go`, `relation_type.go`, `relation.go`, `tag.go`, `graph.go`.
- `extractLabel` (logique métier d'affichage, clés en dur) vit dans le store ([store/graph.go:142-153](../../server/internal/store/graph.go#L142-L153)) ; voir livrable 1 § 4.4.
- `UploadFile` renvoie une `map[string]interface{}` au lieu d'un `model.UploadResponse` ([handler/upload.go:88-92](../../server/internal/handler/upload.go#L88-L92)).
- `handleStoreError` compare avec `switch err` ([handler.go:195](../../server/internal/handler/handler.go#L195)) et les stores avec `err == pgx.ErrNoRows` : fonctionne aujourd'hui car les erreurs ne sont jamais enveloppées, mais fragile ; `errors.Is` partout.
- `interface{}` au lieu de `any` (Go ≥ 1.18) dans tout le store et les handlers (style).
- Les séquences « lire, vérifier, écrire » des handlers ne sont pas transactionnelles (relation :
  type + 2 items puis insert, [handler/relation.go:55-89](../../server/internal/handler/relation.go#L55-L89)). Les FK et contraintes UNIQUE rattrapent les cas de course : acceptable, à documenter.

**Correctif.** `func (h *Handler) ownedCollection(w, r, param) (*model.Collection, bool)` ;
un petit builder `where.Add(cond, arg)` dans `store/helpers.go` (ou `squirrel`) ; déplacer
l'extraction de label dans `model`/`label` avec le schéma en entrée ; struct `UploadResponse` ;
`errors.Is` ; `gofmt -r 'interface{} -> any'`.

### Q5 — Front : composant monolithique, duplication, erreurs avalées, accessibilité — **moyenne**

**Constaté.**
- `ItemDetailView.vue` : **1 032 lignes**, 3 dialogues (ajout à une collection, création et édition de relation), 25 `ref`/`computed`. `SchemaBuilder.vue` : 908 lignes.
- Fonction « label d'item » dupliquée 6 fois (livrable 1 § 4.4) ; `attrLabel`/`ctxLabel`/`ctxAttrLabel` (lecture `x-labels`) dupliquée 5 fois.
- Erreurs réseau avalées sans retour utilisateur : `DynamicForm.handleFileUpload` (`catch {}` [l. 402-404](../../web/src/components/common/DynamicForm.vue#L402-L404), c'est là que le 413 nginx de S7 disparaît), `GraphView.loadGraph` ([l. 279-281](../../web/src/views/GraphView.vue#L279-L281)), `ItemDetailView.openAddToCollection` ([l. 712-714](../../web/src/views/ItemDetailView.vue#L712-L714)), `submitAddToCollection` sans `catch` ([l. 719-731](../../web/src/views/ItemDetailView.vue#L719-L731)), `ItemsView.confirmDelete` sans `catch` ([l. 201-205](../../web/src/views/ItemsView.vue#L201-L205)), `CollectionsView.save` toast sans détail ([l. 206-208](../../web/src/views/CollectionsView.vue#L206-L208)).
- Accessibilité de base : les `<label>` de `DynamicForm` ne sont pas reliés aux champs (pas de `for`/`id`, [l. 4-7](../../web/src/components/common/DynamicForm.vue#L4-L7)) ; boutons icône sans libellé accessible (ex. `Button icon="pi pi-trash" text` dans toutes les listes) ; `<img>` sans `alt` ([DynamicForm.vue:135-146](../../web/src/components/common/DynamicForm.vue#L135-L146), [ItemDetailView.vue:122-133](../../web/src/views/ItemDetailView.vue#L122-L133)) ; navigation par `@row-click` sur `DataTable` ([ItemsView.vue:50](../../web/src/views/ItemsView.vue#L50)) non atteignable au clavier ; thème sombre et contraste gérés (point positif).
- Typage : **aucun `any`**, `noUncheckedIndexedAccess` activé ([web/tsconfig.json:18](../../web/tsconfig.json#L18)) : point très positif.
- `tsconfig.json` utilise `baseUrl` ([l. 19](../../web/tsconfig.json#L19)) : option **dépréciée en TypeScript 6 et supprimée en 7** (voir livrable 3) ; `paths` fonctionne sans `baseUrl` avec des chemins relatifs.
- Pas de gestion d'état hors `auth` : acceptable à cette taille ; les listes de types (`itemTypeApi.list({per_page:100})`) sont rechargées par chaque vue.

**Correctif.** Extraire `AddToCollectionDialog.vue`, `RelationCreateDialog.vue`,
`RelationEditDialog.vue` ; composables `useItemLabel()` et `useSchemaLabels()` ; helper
`notifyError(e)` appelé dans tous les `catch` ; `id`/`for` générés dans `DynamicForm` ;
`aria-label` sur les boutons icône ; `<img :alt>` ; un store Pinia `useMetaTypes()` avec cache
des ItemTypes/RelationTypes/CollectionTypes ; retirer `baseUrl`.

### Q6 — Robustesse d'exécution et observabilité — **moyenne**

**Constaté.**
- `config.Load` fournit des défauts silencieux pour `DATABASE_URL` et `JWT_SECRET` ([config.go:27-35](../../server/config/config.go#L27-L35)) : un oubli de variable en prod ne se voit pas au démarrage (cf. S2).
- Journalisation : `chimiddleware.Logger` seulement (texte, par requête) ; les erreurs 500 de `handleStoreError` **ne sont pas loguées** ([handler.go:200-202](../../server/internal/handler/handler.go#L200-L202)) : impossible de diagnostiquer un 500 en prod.
- `pgxpool.New` sans réglage (`MaxConns` défaut = 4 × CPU, pas de `statement_timeout`) ([store/postgres.go:16](../../server/internal/store/postgres.go#L16)).
- Pas d'endpoint de santé (`/healthz`), pas de métriques.
- Arrêt propre et `Recoverer` présents (point positif).

**Correctif.** `log/slog` (JSON) avec `slog.Error("store", "err", err)` dans `handleStoreError`
et un `request_id` (`chimiddleware.RequestID`) ; `pgxpool.ParseConfig` + `MaxConns`,
`ConnConfig.RuntimeParams["statement_timeout"] = "10000"` ; `GET /healthz` qui `Ping` la base.

### Q7 — Outillage manquant — **moyenne**

**Constaté.** Ni CI, ni lint, ni formateur configuré, ni migrations (livrable 1 § 6.4 et 6.3).
`npm run type-check` est la seule vérification front ; `gofmt`/`go vet` ne sont lancés que
manuellement (et échouent aujourd'hui).

**Correctif.** Un workflow GitHub Actions (`.github/workflows/ci.yml`) avec deux jobs :
- Go : `gofmt -l`, `go vet`, `golangci-lint` (config minimale : `errcheck`, `staticcheck`,
  `gosec`, `bodyclose`), `go test -race ./...`, `govulncheck` ; job d'intégration avec service
  `postgres:16` et `go test -tags integration`.
- Web : `npm ci`, `eslint` (`eslint-plugin-vue` + `typescript-eslint`), `prettier --check`,
  `vue-tsc`, `vitest`, `npm audit --audit-level=high`, `npm run build`.
Migrations : `tern` (même auteur que pgx, SQL pur) ou `goose` ; `schema.sql` devient
`db/migrations/0001_init.sql`, et `pg-start`/compose appliquent les migrations.

### Q8 — Dette mineure diverse — **basse**

**Constaté.**
- `README.md:46-49` et `CLAUDE.md:97` documentent `admin/admin` mais pas le mot de passe de `shinsaber`.
- `web/Dockerfile` : `COPY . .` après `npm ci` invalide le cache à chaque changement de source (normal) mais copie aussi `node_modules` local sans `.dockerignore`.
- `nginx:alpine`, `postgres:16-alpine` et `golang:1.22-alpine` : tags flottants ou obsolètes (livrable 1 § 6).
- `GraphView.vue` : palette de couleurs par type réinitialisée à chaque montage mais `typeColors`/`colorIdx` sont des variables de module ([l. 141-152](../../web/src/views/GraphView.vue#L141-L152)) : les couleurs dérivent entre deux visites (cosmétique).
- `RelationTree.vue` recharge les sous-relations via l'API à chaque dépliage sans limite de profondeur côté serveur (OK, `maxDepth` 5 côté front).

---

## C. Tableau de synthèse (trié par gravité)

| Réf. | Gravité | Constat | Fichiers principaux | Effort |
|------|---------|---------|---------------------|--------|
| S1 | **Critique** | JWT sans `exp`, rôle figé, aucune révocation | `middleware/auth.go`, `config.go`, `stores/auth.ts` | S |
| S2 | **Critique** | Secret JWT et identifiants par défaut en dur | `config/config.go`, `docker-compose.yml`, `README.md` | S |
| S3 | **Haute** | 14 vulnérabilités Go atteintes (chi, pgx, jwt, x/text, stdlib) + axios ; toolchain Go 1.22 EOL en prod | `go.mod`, `server/Dockerfile`, `package.json` | M |
| S4 | **Haute** | Graphe sans borne : CTE `UNION ALL` par chemins, profondeur 10, pas de `LIMIT` | `store/graph.go`, `handler/graph.go` | M |
| Q1 | **Haute** | 19 handlers, tout le store et tout le front sans test ; `vet`/`gofmt` en échec | `internal/handler/*_test.go`, `internal/store`, `web/` | L |
| S5 | Moyenne | Corps JSON non borné, pas de `ReadHeaderTimeout`, pas de rate limit login | `handler/handler.go`, `main.go` | S |
| S6 | Moyenne | PG et API exposés sur l'hôte, conteneurs root, nginx sans en-têtes, pas de `.dockerignore`, pas de healthcheck | `docker-compose.yml`, `*/Dockerfile`, `web/nginx.conf` | S |
| S7 | Moyenne | nginx 1 Mo vs serveur 10 Mo ; `size` client ; pas de cache ni GC des uploads | `web/nginx.conf`, `handler/upload.go` | S |
| S8 | Moyenne | Schémas JSON jamais validés à l'écriture, recompilés à chaque requête | `handler/*_type.go`, `validator/jsonschema.go` | S |
| Q2 | Moyenne | `PUT` relation / collection-item sans `attributes` → 500 ou 422 trompeur | `handler/relation.go`, `handler/collection.go` | S |
| Q3 | Moyenne | N+1 tags et jonctions, `ILIKE` plein texte sans index | `store/item.go`, `store/relation_type.go` | M |
| Q4 | Moyenne | Duplication (contrôle propriétaire ×7, SQL dynamique), label métier dans le store | `handler/collection.go`, `store/*.go` | M |
| Q5 | Moyenne | `ItemDetailView` 1 032 l., label dupliqué ×6, erreurs avalées, a11y de base, `baseUrl` déprécié | `web/src/views/*`, `DynamicForm.vue`, `tsconfig.json` | L |
| Q6 | Moyenne | 500 non logués, pas de `/healthz`, pool non configuré | `handler/handler.go`, `main.go`, `store/postgres.go` | S |
| Q7 | Moyenne | Ni CI, ni lint, ni migrations | `.github/`, `.golangci.yml`, `eslint.config.js`, `db/migrations/` | M |
| S9 | Basse | Énumération d'utilisateurs par timing au login | `handler/auth.go` | S |
| S10 | Basse | Token en `localStorage` (acceptable avec S1 + CSP) | `stores/auth.ts` | — |
| S11 | Basse | Jokers `%`/`_` non échappés, recherche sur les clés JSON | `store/item.go` | S |
| Q8 | Basse | Divers (doc comptes, tags d'images flottants, couleurs du graphe) | `README.md`, `Dockerfile`, `GraphView.vue` | S |

Effort : S < ½ journée · M = 1 session Claude Code · L = plusieurs sessions.

---

## Questions au mainteneur

- **QS1 — Modèle de menace.** L'application est « auto-hébergée, entre amis ». Est-elle exposée
  sur Internet (reverse-proxy, nom de domaine) ou seulement sur un réseau local/VPN ? La réponse
  change la priorité de S5/S6 (rate limit, CSP, TLS) mais pas celle de S1/S2.
- **QS2 — Révocation.** Préférence entre « relire l'utilisateur en base à chaque requête »
  (simple, une requête PK) et « TTL court + refresh » (plus de code, front à adapter) ?
- **QS3 — Fichiers uploadés publics.** Le choix « `/uploads/*` public car `<img>` ne peut pas
  envoyer d'en-tête » est documenté. Est-il acceptable que les posters soient accessibles sans
  compte si l'URL est connue (UUID, non devinable) ? Alternative : cookie de session (S10) qui
  rendrait les images protégeables.
- **QS4 — Migrations.** Accord pour introduire un outil de migration (tern/goose) dès le lot
  outillage, sachant que cela change `pg-start` et `docker-compose` ?
