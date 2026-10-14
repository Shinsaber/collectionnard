# Brief d'audit approfondi — Collectionnard

> Destinataire : Claude Fable (analyse approfondie).
> Commanditaire : le mainteneur du projet (seul développeur).
> Objectif : repartir sur des bases saines avant de reprendre le développement (Phase 2).

---

## 1. Contexte

Collectionnard est un gestionnaire de collections **générique** (séries TV, DVD, BD/manga, cartes…)
auto-hébergé, en usage personnel et entre amis. Principe : aucun type n'est codé en dur,
tout repose sur des méta-types portant un **JSON Schema** et des instances JSONB validées
contre ce schéma, avec un **graphe libre** de relations entre items.

Le MVP (Phase 1) est déclaré terminé. Le projet est repris après plusieurs mois de pause.

Lis d'abord, dans cet ordre :

1. `CLAUDE.md`, `server/CLAUDE.md`, `web/CLAUDE.md` — conventions et commandes (environnement **Nix**)
2. `Collectionnard.md` — vision, modèle conceptuel, roadmap (§8)
3. `les.collection.md` — modèle BD/Manga, prochain domaine à intégrer
4. `api/openapi.yaml` — contrat d'API (source de vérité déclarée)
5. `db/schema.sql`, `db/seed.sql` — base de données
6. Le code : `server/` (Go) puis `web/` (Vue 3 + TS)

## 2. Mission

Trois livrables, chacun dans son fichier sous `docs/audit/`. Produis-les dans l'ordre 1 → 2 → 3 :
la feuille de route s'appuie sur les constats des deux premiers.

### Livrable 1 — Cohérence du projet → `docs/audit/01-coherence.md`

Vérifier que les différentes représentations du projet disent la même chose :

- **Doc ↔ code** : `Collectionnard.md` (ex. `AttributeDefinition` vs JSON Schema réel), README,
  `les.collection.md` (TODO déjà traités ?), cases cochées de la roadmap réellement livrées ?
- **OpenAPI ↔ handlers Go** : chaque route de `Router()` est-elle spécifiée, et inversement ?
  Corps, codes de réponse, droits (public / JWT / admin), pagination.
- **Modèles Go ↔ `web/src/api/types.ts` ↔ schéma SQL** : champs, nullabilité, noms.
- **Front ↔ API** : appels de `client.ts` vs endpoints existants ; usages de `x-order`, `x-labels`,
  `x-widget`, `x-image-categories` cohérents entre `DynamicForm`, `SchemaBuilder`, seed et README.
- **Modèle conceptuel** : le schéma permet-il le modèle BD/Manga de `les.collection.md` sans changement ?
- **Outillage** : versions déclarées (README, `go.mod`, `package.json`, Dockerfiles, `flake.nix`) alignées ?

Format : tableau des écarts (zone, constat, fichier:ligne, gravité, correction proposée).

### Livrable 2 — Qualité et sécurité du code → `docs/audit/02-qualite-securite.md`

Revue **défensive** de notre propre code (pas d'écriture d'exploit : constat, impact, correctif).

**Sécurité** (au minimum) :

- Authentification JWT : algorithme, expiration, secret par défaut, révocation, stockage côté front (`localStorage`).
- Autorisation : contrôles admin et propriétaire (collections, collection items) sur **toutes** les routes ; IDOR.
- Injections SQL : tout SQL construit avec `fmt.Sprintf` (`store/collection.go`, `store/graph.go`, `buildSortClause`…).
- Upload : taille, type MIME réel vs extension, nommage, service des fichiers, traversée de chemin, SVG/XSS.
- Validation des entrées : JSON Schema sur toutes les écritures d'attributs, bornes de pagination,
  profondeur de `GET /items/{id}/graph` (déni de service par traversée).
- Configuration de déploiement : identifiants en dur (docker-compose, seed `admin/admin`),
  en-têtes HTTP de sécurité, nginx, CORS, conteneurs (utilisateur root ?), absence de limitation
  de débit sur `/auth/login`.
- Dépendances : versions Go / npm obsolètes ou avec vulnérabilités connues (utilise `govulncheck`,
  `npm audit` si disponibles via `nix develop -c` ; sinon signale-le).

**Qualité** :

- Architecture et découpage (handler / store / model), duplication, gestion d'erreurs, transactions.
- Tests : couverture réelle des handlers, absence de tests store (SQL) et front, pistes de tests d'intégration.
- Performance : index vs requêtes réelles, N+1, requête récursive du graphe, pagination.
- Front : typage, composants trop gros, gestion d'état, accessibilité de base.
- Outillage manquant : CI, lint (`golangci-lint`, ESLint/Prettier), migrations BDD, formatage.

Format : une section par constat avec **gravité** (critique / haute / moyenne / basse),
fichier:ligne, explication, correctif proposé (extrait de code si utile). Termine par un
tableau de synthèse trié par gravité.

### Livrable 3 — Feuille de route de remise à niveau → `docs/audit/03-feuille-de-route.md`

À partir des livrables 1 et 2 et de l'état de l'art actuel :

- **Mises à jour** : versions actuelles de Go, PostgreSQL, Node, Vue, PrimeVue, Tailwind, Vite,
  Cytoscape et des dépendances Go. Vérifie les versions en ligne plutôt que de mémoire.
  Pour chaque saut majeur, indique les ruptures attendues (ex. Tailwind 3 → 4, PrimeVue 4 → suivante).
- **Lots ordonnés** : d'abord les correctifs de sécurité critiques, puis l'outillage (CI, lint,
  migrations, Nix), puis les mises à jour, puis la dette, et enfin le démarrage de la Phase 2
  (import/export, domaine BD/Manga).
- Pour chaque lot : objectif, tâches concrètes (fichiers touchés), dépendances, risque, effort
  (S / M / L), critère de « terminé ».
- Lots dimensionnés pour être réalisés un par un avec Claude Code (une session ≈ un lot).
- Propose les mises à jour de `Collectionnard.md` (section Phases) à faire en conséquence.

## 3. Règles du jeu

- **Lecture seule sur le code** : tu ne modifies que des fichiers sous `docs/audit/`.
  Aucun correctif appliqué, aucun commit, aucun push.
- Tu peux lancer des commandes non destructives : `nix develop -c go vet ./...`,
  `nix develop -c go test ./...`, `nix develop -c npm run type-check`, `govulncheck`, `npm audit`.
  Pas de `docker compose down -v`, pas de `pg-reset`.
- Toujours citer `fichier:ligne`. Distingue **constaté** (vérifié dans le code ou par une commande)
  et **supposé** (à confirmer).
- Rédige en français. Sois factuel et priorisé : mieux vaut 15 constats utiles que 60 triviaux.
- Si une question bloque l'analyse (intention de conception ambiguë), note-la dans une section
  « Questions au mainteneur » à la fin du livrable concerné plutôt que de deviner.

## 4. Pistes déjà repérées (à vérifier, pas à recopier telles quelles)

Relevées lors d'un premier survol, sans analyse approfondie :

- Aucune CI (`.github/` absent), aucun lint configuré, aucun test front, pas de migrations BDD
  (`schema.sql` rejoué uniquement à la création du volume).
- Go 1.22 dans `go.mod` et `server/Dockerfile`, Node 20 dans `web/Dockerfile` alors que le
  `flake.nix` fournit Node 22 (Node 20 a été retiré de nixpkgs).
- `JWT_SECRET` a une valeur par défaut dans `config/config.go` et `docker-compose.yml`.
- Token JWT stocké dans `localStorage` (`web/src/stores/auth.ts`).
- SQL construit par `fmt.Sprintf` dans `store/collection.go` et `store/graph.go` : les valeurs
  semblent passer en paramètres et le tri par une liste blanche (`buildSortClause`), à confirmer.
- Upload : `MaxBytesReader`, `DetectContentType` et nom en UUID présents ; fichiers servis
  publiquement sous `/api/v1/uploads/`.
- Contrôle propriétaire visible dans `handler/collection.go` (`col.UserID != GetUserID`), à vérifier
  sur toutes les routes de collection items.
- Conteneur serveur sur `alpine` sans utilisateur non-root explicite.
- `les.collection.md` mentionne un TODO « arrays dans relation_types » qui semble déjà traité
  (tables `relation_type_sources` / `relation_type_targets`).

## 5. Fin de mission

Termine par un court message récapitulatif : les 5 constats les plus importants, le premier
lot à attaquer, et la liste des questions au mainteneur.
