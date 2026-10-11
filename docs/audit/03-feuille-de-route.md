# Livrable 3 — Feuille de route de remise à niveau

> Établie le 2026-10-11 à partir des livrables [01-coherence.md](01-coherence.md) et
> [02-qualite-securite.md](02-qualite-securite.md). Les références `S1`, `Q3`, `1.12`, `4.4`… renvoient
> aux constats numérotés de ces deux documents.
> Chaque lot est dimensionné pour **une session Claude Code** (S < ½ j, M ≈ 1 session, L = 2 sessions
> à découper au moment venu). Les lots sont à exécuter **dans l'ordre** ; les dépendances sont indiquées.

## 1. État de l'art des versions (vérifié en ligne le 2026-10-11)

Sources : endoflife.date (API JSON), registre npm (`dist-tags`), GitHub Releases (API), proxy.golang.org
(`@latest`), guides de migration officiels (Tailwind, Vite 7, Vue Router 5, Pinia 3, discussion #172
de jsonschema), `npm outdated` et `go list -m -u all` exécutés dans le dépôt.

### 1.1 Plateforme

| Composant | Projet | Actuel (2026-10-11) | Statut du projet | Cible proposée |
|-----------|--------|---------------------|------------------|----------------|
| Go | `go 1.22` (`go.mod`), `golang:1.22-alpine` (Dockerfile), 1.26.8 (flake) | **1.27.2** (2026-10-08) et **1.26.9** (2026-10-08) supportés ; 1.25 EOL 2026-08-19 | **1.22 hors support** depuis Go 1.24 (février 2025) | `go 1.26` dans `go.mod` (minimum imposé par pgx ≥ 5.9 : Go 1.25+), image `golang:1.26-alpine`, flake sur 1.26 ou 1.27. |
| PostgreSQL | 16 (compose, flake) | **18.6** (18 sorti 2025-09-25), 17.11, 16.15 ; 16 supporté jusqu'au 2028-11-09 | Supporté | Rester en 16 pour ce cycle ; 17/18 en Phase 2 (pas de rupture attendue pour ce schéma : JSONB, CTE, triggers standards). |
| Node | `node:20-alpine` (Dockerfile), 22.23.3 (flake), `@types/node ^25` | 20 **EOL 2026-04-30** ; 22 LTS (22.23.3) jusqu'au 2027-04-30 ; **24 LTS actif** (24.21.0) jusqu'au 2028-04-30 ; 26 LTS à partir du 2026-10-28 | **Node 20 en fin de vie** | `node:22-alpine` (aligné sur le flake) ou 24 ; `@types/node` aligné (`^22` ou `^24`). |
| Alpine | `alpine:3.20` | 3.24.2 ; **3.20 EOL 2026-04-01** | **EOL** | `alpine:3.22` (support jusqu'en 2027-05) ou image `distroless/static`. |
| nginx | `nginx:alpine` (flottant) | — | Non reproductible | Épingler `nginx:1.28-alpine` (ou la stable du moment). |

### 1.2 Backend Go (`go list -m -u all`, proxy.golang.org)

| Module | Actuel | Dernière | Vulnérabilités corrigées (govulncheck) | Ruptures attendues |
|--------|--------|----------|----------------------------------------|--------------------|
| `go-chi/chi/v5` | 5.0.12 | **5.3.2** (2026-08-20) | GO-2026-5775/5777 (RealIP) | Aucune API cassée en v5 ; `middleware.RealIP` reste à utiliser derrière un proxy de confiance uniquement. |
| `jackc/pgx/v5` | 5.5.5 | **5.11.0** (2026-09-07) | GO-2026-5004 | **Go 1.25+ requis** (depuis 5.9.0). Parseur d'URI réécrit en 5.11 (le `+` devient littéral, IPv6 entre crochets) : la DSN `postgres://colec:colec@localhost:5432/colec?sslmode=disable` n'est pas concernée. `Rows` gagne `TypeMap()` : les mocks de `Rows` (aucun ici) doivent l'implémenter. `timestamptz` texte renvoyé en `time.Local`. |
| `golang-jwt/jwt/v5` | 5.2.1 | **5.3.1** (2026-01-28) | GO-2025-3553 | Aucune rupture ; `WithExpirationRequired` disponible. |
| `santhosh-tekuri/jsonschema` | v5.3.1 | **v6.0.3** (2026-06-28) | — | **Nouveau module, API cassée** : `AddResource(url, parsedJSON)` (plus de `io.Reader`), `httploader` supprimé (bien : interdit les refs distantes par défaut), `ValidationError.Message` remplacé par `ErrorKind.LocalizedString` / `DetailedOutput()`, `InstanceLocation` devient un tableau. Impact : réécriture de `validator/jsonschema.go` (56 l.) et de `FormatValidationError`. **Optionnel** : v5.3.1 n'a pas de vulnérabilité connue ; migrer seulement pour la localisation des messages ou les vocabulaires personnalisés. |
| `google/uuid` | 1.6.0 | 1.6.0 | — | — |
| `golang.org/x/crypto` | 0.22.0 | **0.58.0** | — | Aucune (bcrypt stable). |
| `golang.org/x/text` (indirect) | 0.14.0 | **0.43.0** | GO-2026-6629, GO-2026-5970 | Aucune. |

### 1.3 Frontend (`npm outdated`, npm, GitHub Releases)

| Paquet | Actuel (lock) | Dernière | Saut | Ruptures attendues |
|--------|---------------|----------|------|--------------------|
| `vue` | 3.5.32 | **3.5.43** (3.6 en RC) | mineur | Aucune. Corrige l'avis `@vue/server-renderer` (3.5.42+). |
| `primevue` | 4.5.5 | **5.0.2** (5.0.0 le 2026-07-15) | **majeur** | Guide officiel : « pas de rupture d'API publique ». Mais : **base de taille 16 px** (au lieu de 14 px) → tous les composants grossissent ; presets `*Compat` fournis jusqu'en juin 2027. `@primevue/themes` **déprécié** → `@primeuix/themes` (3.0.1) : `import Aura from "@primeuix/themes/aura"` dans `main.ts`. Dépréciés (retrait en v6) : `MultiSelect` (→ `Select multiple`, utilisé dans `RelationTypesView.vue:81-100`), `Image`, `Galleria`. `InputMask` devient une directive (non utilisé). |
| `@primevue/themes` | 4.5.4 | déprécié | — | Remplacer par `@primeuix/themes` (obligatoire avec PrimeVue 5, possible dès 4.5). |
| `primeicons` | 7.0.0 | 8.0.2 | majeur | À vérifier : renommages d'icônes possibles ; le projet utilise ~20 icônes `pi-*`. |
| `tailwindcss` | 3.4.19 | **4.3.3** (2026-07-16) | **majeur** | Configuration en CSS (`@import "tailwindcss"`, `@theme`), `tailwind.config.js` à charger explicitement via `@config` ou à supprimer, plugin `@tailwindcss/vite` (remplace PostCSS + autoprefixer), `darkMode: "selector"` → `@custom-variant dark (&:where(.dark, .dark *))`, renommages (`shadow-sm`→`shadow-xs`, `ring`→`ring-3`, `!bg-x`→`bg-x!`), couleur de bordure par défaut `currentColor`. Navigateurs : Safari 16.4+, Chrome 111+, Firefox 128+. Outil `npx @tailwindcss/upgrade` (Node 20+). Corrige aussi la chaîne `braces`/`micromatch`/`chokidar` de `npm audit`. **Attention** : cohabitation avec PrimeVue (classes `!important` dans `AppLayout.vue:19,40`, `SchemaBuilder.vue:20,65,75`). |
| `vite` | 6.4.2 | **8.3.4** (2026-10-08) | **2 majeurs** | 6→7 : Node 20.19+/22.12+, cible navigateur `baseline-widely-available` (Chrome 107, Safari 16), suppression `splitVendorChunkPlugin`. 7→8 : **Rolldown** remplace Rollup/esbuild, minification Oxc/Lightning CSS, `build.rollupOptions` → `rolldownOptions`. Le `vite.config.ts` du projet (alias + proxy) n'utilise rien de cassé. |
| `@vitejs/plugin-vue` | 5.2.4 | 6.0.10 | majeur | Suit Vite 7+. |
| `typescript` | 5.6.3 | **7.0.2** (2026-08-20, compilateur natif Go) ; 6.0.x | **2 majeurs** | TS 6 déprécie `baseUrl`, `moduleResolution: node`, `target: es5`… ; TS 7 **les supprime**. `tsconfig.json:19` utilise `baseUrl`. **`vue-tsc` ne supporte pas TS 7** (pas d'API JS stable) : rester sur **TS 6.x** tant que `vue-tsc`/`@vue/language-tools` n'annoncent pas le support (discussion vuejs/language-tools #6121). |
| `vue-tsc` | 2.2.12 | 3.3.13 | majeur | Suit TS 6. |
| `pinia` | 2.3.1 | **4.0.3** (2026-08-12) | **2 majeurs** | 2→3 : Vue 3 seul, TS ≥ 5, `PiniaStorePlugin` supprimé, `defineStore({id})` déprécié. 3→4 : aucune rupture listée dans les releases consultées (**à confirmer** avec le guide officiel au moment du lot). Le store `auth.ts` (setup store, 36 l.) n'est pas concerné. |
| `vue-router` | 4.6.4 | **5.4.0** (2026-10-07) | majeur | Guide officiel : **aucune rupture** sans `unplugin-vue-router`. Ruptures ultérieures limitées à l'API expérimentale (`vue-router/experimental`). v6 sera ESM-only. |
| `cytoscape` | 3.33.2 | 3.34.3 | mineur | Aucune. |
| `axios` | 1.15.0 | **1.20.0** | mineur | Aucune ; corrige la totalité des avis `npm audit`. |
| `@types/node` | 25.6.0 | 26.6.5 | — | Aligner sur la version de Node réellement utilisée (22 ou 24), pas « la dernière ». |

---

## 2. Lots ordonnés

Vue d'ensemble :

```
Lot 0  Sécurité critique (S1, S2, S3-min, S5, Q2)              S   ← premier à attaquer
Lot 1  Outillage : flake suivi, CI, lint, formatage, Nix       M
Lot 2  Migrations BDD + healthcheck + conteneurs durcis         M
Lot 3  Mises à jour backend (Go 1.26, modules, Dockerfile)      M
Lot 4  Mises à jour front mineures (axios, vue, Node 22)        S
Lot 5  Robustesse API : graphe borné, limites, schémas validés  M
Lot 6  Tests : handlers manquants, store d'intégration, vitest  L (2 sessions)
Lot 7  Cohérence contrat : OpenAPI ↔ code, Collectionnard.md    M
Lot 8  Extension x-label + refactor labels + perf N+1           M
Lot 9  Front : découpage ItemDetailView, erreurs, a11y          M
Lot 10 Mises à jour front majeures (PrimeVue 5, Vite 8, TS 6…)  L (2 sessions)
Lot 11 Tailwind 4                                                M (optionnel, séparé)
Lot 12 Phase 2 — import/export JSON/CSV                         M
Lot 13 Phase 2 — domaine BD/Manga (seed + CollectionType)       M
Lot 14 Phase 2 — recherche avancée (attributes[], tri, index)   M
```

### Lot 0 — Correctifs de sécurité critiques

- **Objectif** : rendre un déploiement « tel quel » non compromettable et fermer les CVE atteintes
  sans changer de version majeure.
- **Constats** : S1, S2, S3 (partie dépendances mineures), S5, S9, Q2.
- **Tâches**
  1. `server/config/config.go` : refuser de démarrer si `JWT_SECRET` < 32 octets ; ajouter `JWT_TTL`
     (défaut `24h`) ; retirer le défaut `DATABASE_URL` ou le réserver à un flag `--dev`.
  2. `server/internal/middleware/auth.go` : `RegisteredClaims` avec `exp`/`iat`/`iss`,
     `ParseWithClaims` + `WithValidMethods([]string{"HS256"})` + `WithExpirationRequired()`.
     Relire l'utilisateur en base dans `Authenticate` (rôle + existence) : nouvelle méthode
     `GetUserByID` déjà disponible dans `Store`.
  3. `server/internal/handler/handler.go` : `MaxBytesReader(1 Mo)` dans `decodeJSON` ;
     `server/main.go` : `ReadHeaderTimeout`.
  4. `server/internal/handler/auth.go` : bcrypt factice si utilisateur inconnu ; limiteur
     `httprate.LimitByIP(10, time.Minute)` sur `/auth/login` (`github.com/go-chi/httprate`).
  5. `handler/relation.go`, `handler/collection.go` : 422 si `attributes` absent (Q2).
  6. `go get github.com/golang-jwt/jwt/v5@v5.3.1 golang.org/x/crypto@latest golang.org/x/text@latest`
     (sans toucher à pgx/chi qui imposent Go 1.25 : lot 3) ; `go mod tidy`.
  7. `web/src/stores/auth.ts` : décoder `exp` et déconnecter proactivement ; `web/package.json` :
     `axios ^1.20.0`.
  8. `docker-compose.yml` : `${JWT_SECRET:?}`, `${POSTGRES_PASSWORD:?}`, fichier `.env.example`.
  9. `api/openapi.yaml:1124` : décrire `JWT_TTL`. Tests : `middleware/auth_test.go` (token expiré →
     401, alg `none`/HS512 → 401), `auth_test.go` (rate limit), `relation_test.go` (422 sans attributes).
- **Dépendances** : aucune. **Risque** : faible ; les tokens existants deviennent invalides (reconnexion).
- **Effort** : S (une demi-session). **Terminé quand** : `go test ./...` vert avec les nouveaux tests ;
  démarrage sans `JWT_SECRET` → erreur explicite ; `govulncheck` ne liste plus jwt ni x/text.

### Lot 1 — Outillage de base : dépôt, CI, lint, formatage

- **Objectif** : que « terminé » soit vérifié par une machine, et que l'environnement Nix fonctionne
  depuis un clone.
- **Constats** : 1.13, 1.16, 6.1, 6.2, 6.4, 6.5, Q1 (vet/gofmt), Q7.
- **Tâches**
  1. `git add flake.nix .envrc CLAUDE.md server/CLAUDE.md web/CLAUDE.md docs/ .claude/ .gitignore` ;
     générer et commiter `flake.lock` (`nix flake lock`) ; épingler `nixpkgs` sur une release
     (`nixos-25.11`) ou garder unstable avec le lock. Mettre à jour `CLAUDE.md` § Nix (le flake doit
     être suivi ; `path:` en secours).
  2. `gofmt -w` sur les 7 fichiers ; corriger les 3 `go vet` ; `interface{}` → `any`.
  3. `.golangci.yml` minimal (`errcheck`, `staticcheck`, `gosec`, `bodyclose`, `gofmt`) ; ajouter
     `golangci-lint` au `flake.nix`.
  4. `web/` : `eslint.config.js` (flat config, `eslint-plugin-vue`, `typescript-eslint`),
     `.prettierrc`, scripts `lint`, `format`, `test` ; `vitest` + `@vue/test-utils` installés (tests
     écrits au lot 6).
  5. `.github/workflows/ci.yml` : jobs `go` (fmt, vet, lint, test, govulncheck) et `web` (ci, lint,
     type-check, build, `npm audit --audit-level=high`), déclenchés sur push/PR. Variante Nix :
     `nix develop -c` avec `cachix/install-nix-action`.
  6. `.claude/settings.json` : autoriser `npm run lint`, `golangci-lint run`, `nix develop path:*`.
  7. Mettre à jour `.claude/commands/verif.md` (lint inclus).
- **Dépendances** : lot 0 (pour que la CI soit verte dès le premier run). **Risque** : nul.
- **Effort** : M. **Terminé quand** : CI verte sur `main` ; `git clone` + `direnv allow` + `nix develop`
  fonctionne sans manipulation.

### Lot 2 — Migrations BDD, santé et durcissement des conteneurs

- **Objectif** : pouvoir modifier le schéma sans `pg-reset`, et déployer des conteneurs sains.
- **Constats** : 6.3, S6, S7 (nginx), Q6, QS4.
- **Tâches**
  1. Outil : `tern` (`github.com/jackc/tern/v2`, SQL pur, même écosystème que pgx). `db/schema.sql` →
     `db/migrations/001_init.sql` ; `db/seed.sql` reste un seed de dev appliqué après migration.
     Commande `migrate` dans le binaire (`colec-server migrate`) ou job compose `migrate`.
  2. `flake.nix` : `pg-start` applique les migrations ; `docker-compose.yml` : service `migrate`
     (`depends_on: postgres`, `server` dépend de `migrate: service_completed_successfully`).
  3. `GET /healthz` (ping BDD) ; `healthcheck:` sur `server` ; `web` dépend de `server: service_healthy`.
  4. `server/Dockerfile` : `golang:1.26-alpine` → `alpine:3.22` (ou distroless), `USER 10001`,
     `chown` de `/data/uploads`. `web/Dockerfile` : `node:22-alpine`, `nginx:1.28-alpine`.
     `.dockerignore` dans `server/` et `web/`.
  5. `web/nginx.conf` : `client_max_body_size 10m`, en-têtes de sécurité (CSP compatible
     PrimeVue : `style-src 'self' 'unsafe-inline'`), `Cache-Control` sur `/api/v1/uploads/`.
  6. `docker-compose.yml` : supprimer `ports` de `postgres` et `server` ; README : section
     « Mise en production » (reverse-proxy TLS, variables requises).
  7. `handler.go` : `slog` + `RequestID` ; loguer les 500 dans `handleStoreError`.
- **Dépendances** : lot 1 (CI). **Risque** : moyen (changement du cycle de vie de la base ; prévenir
  que `pg-reset` / `down -v` est nécessaire une dernière fois). **Effort** : M.
- **Terminé quand** : `docker compose up --build` démarre les 4 services avec healthchecks verts ;
  un upload de 5 Mo passe ; PG n'est plus joignable depuis l'hôte.

### Lot 3 — Mises à jour backend

- **Objectif** : toolchain et modules supportés, zéro vulnérabilité atteinte.
- **Constats** : S3, 1.10 (versions), tableau § 1.2.
- **Tâches**
  1. `go.mod` : `go 1.26` ; `go get github.com/jackc/pgx/v5@v5.11.0 github.com/go-chi/chi/v5@v5.3.2` ;
     `go get -u ./...` pour les indirects ; `go mod tidy`.
  2. Vérifier les points pgx 5.11 : DSN, `timestamptz` (`created_at` scanné en `time.Time` : le fuseau
     passe en `time.Local` ; sérialisation JSON RFC 3339 inchangée mais avec offset local → forcer
     `UTC()` dans les scans ou `TZ=UTC` dans le conteneur).
  3. `middleware.RealIP` : ne l'activer que si `TRUSTED_PROXY=true` (nginx) ; sinon retirer.
  4. Décision jsonschema v6 : **reporter** (pas de CVE ; coût de réécriture) et noter dans
     `Collectionnard.md` § dette.
  5. `govulncheck ./...` → 0 atteinte ; mettre à jour `README.md` et `Collectionnard.md` § 9.
- **Dépendances** : lot 1. **Risque** : faible à moyen (pgx). **Effort** : M.
- **Terminé quand** : `govulncheck` vide ; tests verts ; image Docker construite sur 1.26.

### Lot 4 — Mises à jour front mineures

- **Objectif** : fermer `npm audit` runtime et aligner Node sans changer de majeure.
- **Tâches** : `npm update` (vue 3.5.43, axios 1.20.0, cytoscape 3.34.3, vite 6.4.4, postcss,
  autoprefixer) ; `@types/node ^22` ; `npm audit fix` (sans `--force`) ; vérifier
  `npm run type-check && npm run build` ; `@primevue/themes` → `@primeuix/themes` (compatible
  PrimeVue 4.5, prépare le lot 10) : `main.ts:4`.
- **Dépendances** : lot 1. **Risque** : faible. **Effort** : S.
- **Terminé quand** : `npm audit --audit-level=high` ne signale plus que des dépendances de build
  (tailwind 3) ; build OK ; vérification visuelle du thème.

### Lot 5 — Robustesse de l'API

- **Objectif** : aucune requête légitime ne doit pouvoir mettre PostgreSQL à genoux ; aucun schéma
  invalide ne doit entrer en base.
- **Constats** : S4, S8, S11, 2.12, 2.13.
- **Tâches**
  1. `store/graph.go` : réécriture de la CTE en « nœuds visités » (`UNION`, sans `path`) ou boucle
     niveau par niveau en Go ; `LIMIT` nœuds (config `GRAPH_MAX_NODES`, défaut 500) ; `truncated`
     dans `GraphResponse` (spec + `model.go` + `types.ts` + bandeau dans `GraphView.vue`).
  2. `handler/graph.go` : `max_depth` max 5 (10 pour admin), `relation_type_ids` ≤ 20 → 400 ;
     `context.WithTimeout` 5 s.
  3. `validator` : `CompileSchema` + cache `sync.Map` par `(id, updated_at)` ; interdiction des `$ref`
     externes ; appel dans les 6 handlers de méta-types (422 détaillé) ; exiger `type: object`.
  4. `handler.go` : `per_page` borné à 100 ; UUID de filtre invalide → 400.
  5. `store/item.go` : échappement des jokers ; `store/postgres.go` : `statement_timeout`.
  6. Tests : graphe tronqué (mock), schéma invalide → 422, `per_page=1000` → 100.
- **Dépendances** : lot 3 (pgx à jour pour `ParseConfig`). **Risque** : moyen (algorithme du graphe ;
  à valider sur le seed : Pilot = 1 nœud, 2 arêtes). **Effort** : M.
- **Terminé quand** : requête `max_depth=10` sur le seed < 50 ms ; test d'intégration (lot 6) sur un
  graphe synthétique dense (1 000 relations) < 2 s ; schéma `{"type":"objet"}` refusé à la création.

### Lot 6 — Tests (2 sessions)

- **Objectif** : couvrir chaque handler, le SQL réel et la logique front.
- **Constats** : Q1, 1.5.
- **Session 6a — Go**
  1. Tests `MockStore` pour les 19 handlers manquants, dont les 4 contrôles de propriété absents.
  2. Paquet `store` : `//go:build integration`, `TestMain` qui crée une base `colec_test` depuis
     `DATABASE_URL`, applique les migrations + seed ; cas : graphe multi-chemin, `ListItems` filtres/tri,
     `DeleteItem` cascade, 409 doublons, `UpdateRelationType` jonctions. Job CI avec service postgres.
  3. Tests `validator` (schéma vide, `null`, erreurs formatées).
- **Session 6b — Front**
  1. `vitest` : `SchemaBuilder` aller-retour sur les 13 schémas du seed (extraits dans
     `web/tests/fixtures/`), `DynamicForm` (widgets, `required`, `x-order`), `extractApiError`,
     `usePagination`.
  2. Smoke test de rendu `LoginView`, `ItemsView` avec `msw` ou mock de `client.ts`.
- **Dépendances** : lots 1, 2 (migrations pour la base de test). **Risque** : nul. **Effort** : L.
- **Terminé quand** : `go test -tags integration ./...` et `npm test` verts en CI ; couverture
  handlers 100 % des routes ; `Collectionnard.md` § 8 « Tests des requêtes BDD » devient vrai.

### Lot 7 — Cohérence du contrat et de la documentation

- **Objectif** : `api/openapi.yaml` redevient la source de vérité ; les docs décrivent le code réel.
- **Constats** : 1.1–1.12, 1.14–1.15, 2.1–2.15, 3.1–3.7, 4.8, Q8.
- **Tâches**
  1. Spec : retirer (ou marquer `x-status: planned`) `attributes[champ]` et `sort=attributes.x`
     (décision Q5) ; `required: [attributes]` sur les deux `*Update` ; réponses `BadRequest`,
     `InternalError`, 401 global ; 422 sur `POST /collection-types` et `/collections` ; 404 sur
     `POST /collections` ; retirer 404 de `GET /items/{id}/collections` ; nommer `GraphEdge`, `Stats` ;
     corriger le tag Uploads et le texte « source_item_type_id » ; description de `ItemUpdate`
     (remplacement, décision Q4).
  2. Code : `model.UploadResponse` ; `respondError` partagé pour le middleware (JSON) ; `ApiError`
     renommé côté Go ou TS.
  3. Validation automatique de la spec en CI (`redocly lint` ou `spectral`) ; test Go qui compare
     les routes de `Router()` aux `paths` de la spec (lecture YAML) pour empêcher toute dérive future.
  4. `Collectionnard.md` : §2 schéma sans `AttributeDefinition`, §3 réécrit « JSON Schema +
     extensions », §4 noms ASCII, §6 vues matérialisées en piste, §8 corrections (voir § 3 ci-dessous),
     §9 versions.
  5. `les.collection.md` : supprimer le TODO et la note (1.8) ; fixer le sens de `created_by` (Q1) ;
     ajouter la décision Q3 (rôles multiples).
  6. `README.md` : versions majeures seules, section Nix, arborescence complète, règle réelle du
     slider, mot de passe `shinsaber` (1.15) ; `CLAUDE.md` : idem slider, convention de nommage (Q2).
- **Dépendances** : lots 0 et 5 (pour documenter les comportements finaux). **Risque** : nul.
  **Effort** : M.
- **Terminé quand** : `redocly lint` sans erreur ; test routes ↔ spec vert ; relecture des trois docs.

### Lot 8 — Extension `x-label`, refactor des labels, performances

- **Objectif** : supprimer les champs métier en dur et préparer BD/Manga (`Personne`, `Tome`).
- **Constats** : 4.4, 4.5, 4.2, 4.3, Q3, Q4.
- **Tâches**
  1. Spécifier `x-label` (racine du schéma) : chaîne gabarit `"{title}"`, `"{last_name} {first_name}"`,
     `"Tome {number} — {title}"` ; résolution i18n des sous-objets `{fr,en}` ; fallback actuel si absent.
     Documenter dans README + CLAUDE.md ; champ « Libellé » dans `SchemaBuilder.vue`.
  2. Go : paquet `internal/label` (`Render(schema, attrs, lang)`) ; utilisé par `graph.go`.
  3. Front : `composables/useItemLabel.ts` + `useSchemaLabels.ts` (langue paramétrable) ; remplacer
     les 6 copies et les 5 lecteurs `x-labels` ; `GraphView` lit `x-widget` au lieu de `/image/i` ;
     `RelationTree`/`GraphView` affichent les attributs de relation via `x-labels` (plus de
     `order`/`ordre` en dur).
  4. Seed : `x-order` + `x-label` sur les 13 schémas (`"{titre}"`, `"{nom}"`, `"S{numero}"`…).
  5. Perf : tags et jonctions chargés en une requête groupée ; `lower()` retiré du filtre tag ;
     cache des schémas compilés (si non fait au lot 5).
  6. `SchemaBuilder` : préserver les clés inconnues (`description`, futurs `x-*`) à la reconstruction
     (4.7) ; `description: true` par défaut.
- **Dépendances** : lot 7 (spec), lot 6b (tests SchemaBuilder protègent le refactor). **Risque** :
  moyen (refactor transversal front). **Effort** : M.
- **Terminé quand** : `grep -r '"titre"' web/src server/internal` ne renvoie plus que le seed ; un
  item `Personne {last_name:"Toriyama"}` s'affiche « Toriyama » partout.

### Lot 9 — Découpage du front, erreurs, accessibilité

- **Constats** : Q5.
- **Tâches** : extraire `AddToCollectionDialog.vue`, `RelationCreateDialog.vue`,
  `RelationEditDialog.vue` d'`ItemDetailView.vue` ; `notifyError(e)` partagé dans tous les `catch`
  (dont l'upload : afficher le 413) ; `id`/`for` dans `DynamicForm`, `aria-label` sur les boutons
  icône, `alt` sur les images, lien clavier sur les lignes d'items ; store `useMetaTypes()` avec
  cache ; retirer `baseUrl` de `tsconfig.json` (préparation TS 6).
- **Dépendances** : lot 8. **Risque** : faible. **Effort** : M.
- **Terminé quand** : aucun fichier `.vue` > 400 lignes ; audit Lighthouse a11y ≥ 90 sur
  `/items/:id` ; `vue-tsc` vert sans `baseUrl`.

### Lot 10 — Mises à jour front majeures (2 sessions)

- **Objectif** : PrimeVue 5, Vite 8, TypeScript 6, Pinia 4, vue-router 5, vue-tsc 3.
- **Session 10a — Build** : `vite ^8`, `@vitejs/plugin-vue ^6`, `typescript ~6.0`, `vue-tsc ^3`,
  `pinia ^4`, `vue-router ^5`, `@types/node ^22`. Vérifier `vite.config.ts` (rien de cassé attendu),
  `tsconfig` (TS 6 : `baseUrl` retiré au lot 9, `moduleResolution: bundler` conservé), `main.ts`.
  **Ne pas** passer à TypeScript 7 tant que `vue-tsc` ne le supporte pas (vérifier
  vuejs/language-tools #6121 le jour J).
- **Session 10b — PrimeVue 5** : `primevue ^5`, `@primeuix/themes ^3`, `primeicons ^8` ; démarrer avec
  le preset `AuraCompat` (14 px) puis décider du passage à 16 px ; remplacer `MultiSelect` par
  `Select multiple` (`RelationTypesView.vue`) ; vérifier les 20 icônes `pi-*` ; revue visuelle des
  `DataTable`, `Dialog`, `Tabs`, `Slider`, `Password`.
- **Dépendances** : lots 4, 6b, 9. **Risque** : moyen (visuel). **Effort** : L.
- **Terminé quand** : `npm run build`, `npm test`, `npm audit` (hors Tailwind 3) propres ; parcours
  manuel login → catalogue → fiche → graphe → collection → admin sans régression visuelle.

### Lot 11 — Tailwind 4 (optionnel, séparé)

- **Objectif** : fermer la dernière source d'avis `npm audit` (chaîne `braces`) et moderniser le CSS.
- **Tâches** : `npx @tailwindcss/upgrade` sur une branche ; `@import "tailwindcss"` dans `main.css` ;
  `@custom-variant dark (&:where(.dark, .dark *))` (le thème PrimeVue utilise déjà `.dark`) ;
  `@tailwindcss/vite` et suppression de `postcss.config.js`/`autoprefixer` ; relire les renommages
  (`shadow-sm`, `ring`, `!`) ; vérifier les surcharges `!important` sur PrimeVue.
- **Dépendances** : lot 10. **Risque** : moyen (revue visuelle complète) ; bénéfice modeste pour un
  projet qui utilise peu Tailwind. Peut être différé indéfiniment : Tailwind 3.4 reste fonctionnel.
  **Effort** : M.

### Lot 12 — Phase 2 : import/export JSON/CSV

- **Objectif** : première fonctionnalité Phase 2, utile aussi comme sauvegarde.
- **Tâches** : `GET /export` (admin) → JSON complet (types, items, relations, tags ; pas les
  collections d'autrui) et `GET /collections/{id}/export?format=csv|json` (owner) ; `POST /import`
  (admin, JSON, transactionnel, validation JSON Schema de chaque item, rapport d'erreurs par
  ligne, mode `dry_run`) ; spec → model → store → handler → tests → `client.ts` → boutons dans
  `ItemsView`/`CollectionDetailView`. Les UUID du seed sont conservés à l'import pour permettre
  une ré-importation idempotente.
- **Dépendances** : lots 5, 6, 7. **Effort** : M.
- **Terminé quand** : export puis `pg-reset` puis import reproduit la base (test d'intégration).

### Lot 13 — Phase 2 : domaine BD/Manga

- **Objectif** : premier domaine hors TV, validation du caractère générique.
- **Tâches** : trancher Q1/Q2/Q3 ; `.claude/commands/modele-vers-seed.md` sur `les.collection.md`
  → `db/seed_bd_manga.sql` (plages `b…010`+, `c…010`+, `e…100`+ ; `x-order`, `x-labels` fr/en,
  `x-label`, `x-image-categories`) ; CollectionType « Lecture » (`current_volume`, `progress`,
  `rereads`) ; brancher dans `pg-start` et compose ; vérifier le graphe Dragon Ball (profondeur 4,
  multi-chemin Saga/Univers) et la borne du lot 5.
- **Dépendances** : lot 8 (`x-label`), lot 2 (migrations/seed). **Effort** : M.
- **Terminé quand** : la fiche « Dragon Ball » montre auteur, éditions, tomes ; « Toriyama »
  relié à deux œuvres avec deux rôles ; les tests d'intégration couvrent le nouveau seed.

### Lot 14 — Phase 2 : recherche avancée

- **Objectif** : implémenter ce que la spec promettait (2.1, 2.2) proprement.
- **Tâches** : `attributes[champ]=valeur` → `attributes @> $n::jsonb` (index GIN existant) ;
  `sort=attributes.<clé>` avec liste blanche dérivée du schéma de l'`item_type_id` demandé ;
  colonne générée `search_tsv` + index GIN ; filtres par relation (`related_to=<uuid>`) ; front :
  panneau de filtres piloté par le JSON Schema du type sélectionné (réutilise `DynamicForm`).
- **Dépendances** : lots 2 (migration), 7 (spec), 8. **Effort** : M.

---

## 3. Mises à jour proposées de `Collectionnard.md` § 8 (à appliquer au lot 7)

Remplacer la section « Phase 1 » et « Phase 2 » par :

```markdown
### Phase 1 : MVP — Valider le dynamisme (terminée)

- [x] Schéma de BDD avec attributs dynamiques (PostgreSQL 16, JSONB, index GIN)
- [x] Requêtes SQL de validation manuelles (`db/queries_test.sql`) — tests automatisés : voir Phase 1.5
- [x] API REST (CRUD sur tous les objets) — spécification OpenAPI 3.1 dans `api/openapi.yaml`
- [x] Authentification utilisateur (admin/user) — JWT Bearer
- [x] Jeu de données de test : Séries TV (multi-chemin Série→Diffusion→Saison→Épisode + Série→DVD→Épisode)
- [x] Interface web (SPA Vue 3 + PrimeVue)
- [x] Upload d'images (galerie, catégories avec ratios, onglets)
- [x] Visualisation de graphe (Cytoscape.js, traversée récursive bidirectionnelle, profondeur configurable)
- [x] Gestion des collections utilisateur (CRUD, ajout/retrait d'items, attributs contextuels)
- [x] Gestion des relations (CRUD, arbre groupé par type, direction source/target)
- [x] Formulaires dynamiques générés depuis JSON Schema (DynamicForm)
- [x] Éditeur de JSON Schema visuel (SchemaBuilder) avec `x-order`, `x-labels`, `x-widget`, `x-image-categories`
- [x] Recherche simple (type, texte libre, tag) et tri par date
- [x] Statistiques (dashboard avec compteurs par type)
- [x] Tags libres sur les items
- [x] Déploiement Docker Compose (3 services : postgres, server, web)
- [x] Environnement de développement Nix (`flake.nix`, `pg-start`)

### Phase 1.5 : Remise à niveau (audit du 2026-10-11, `docs/audit/`)

- [ ] Sécurité : JWT avec expiration et révocation, secrets obligatoires, dépendances à jour (lot 0, 3, 4)
- [ ] Outillage : flake suivi, CI GitHub Actions, golangci-lint, ESLint/Prettier, vitest (lot 1)
- [ ] Migrations BDD (tern), `/healthz`, conteneurs non-root, nginx durci (lot 2)
- [ ] API robuste : graphe borné, limites d'entrée, schémas JSON validés à l'écriture (lot 5)
- [ ] Tests : 100 % des handlers, store d'intégration, front (lot 6)
- [ ] Contrat OpenAPI aligné sur le code et vérifié en CI (lot 7)
- [ ] Extension `x-label` ; plus aucun champ métier codé en dur (lot 8)
- [ ] Front découpé et accessible (lot 9) ; PrimeVue 5 / Vite 8 / TS 6 (lot 10) ; Tailwind 4 (lot 11, optionnel)

### Phase 2 : Enrichissement

- [ ] Import/Export JSON/CSV (lot 12)
- [ ] Domaine BD / Manga (`les.collection.md`, lot 13)
- [ ] Recherche avancée : filtres sur attributs, tri par attribut, recherche plein texte indexée (lot 14)
- [ ] Statistiques et visualisations graphiques
- [ ] Soumission d'items par les users (validation admin)
- [ ] Choix de la langue d'affichage (`x-labels` fr/en)
```

Et dans § 9 « Stack technique » :

```markdown
| Backend         | Go (chi, pgx, JWT, JSON Schema) | 1.26      |
| Frontend        | Vue 3 + TypeScript              | 3.5 / 6.x (après lot 10) |
| UI              | PrimeVue + Tailwind CSS         | 5 / 3.4 (4 après lot 11) |
| Graphes         | Cytoscape.js                    | 3.34      |
| Build           | Vite                            | 8         |
| Packaging       | Nix flake (`nix develop`)       | ✔         |
```

---

## 4. Décisions à prendre avant de démarrer

| Avant le lot | Question (livrables 1 et 2) |
|--------------|-----------------------------|
| 0 | QS1 (exposition Internet ?), QS2 (révocation : relecture BDD ou refresh ?) |
| 2 | QS4 (outil de migration), QS3 (uploads publics) |
| 7 | Q4 (`PUT` remplacement ou fusion), Q5 (`attributes[]` retiré ou « planned »), Q6 (langue) |
| 8 / 13 | Q1 (sens de `created_by`), Q2 (convention de nommage fr/en), Q3 (rôles multiples) |
