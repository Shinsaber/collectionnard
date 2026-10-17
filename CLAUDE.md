# Collectionnard — Contexte pour Claude

Application **générique** de gestion de collections (séries TV, DVD, BD/manga, cartes…).
Principe central : **aucun type de collection n'est codé en dur**. Tout passe par des
méta-types qui portent un **JSON Schema**, et des instances qui portent des valeurs JSONB
validées contre ce schéma.

Langue du projet : **français** (docs, commits, échanges). Le socle de données est en **anglais**
snake_case ASCII : identifiants, clés JSON des attributs, noms de RelationTypes, valeurs d'enum ;
l'affichage traduit via `x-labels`, `x-title`, `x-enum-labels` (fr/en, sélecteur dans l'en-tête). L'interface passe par
vue-i18n (`web/src/i18n/`, registre des langues `langs.ts`) ; l'API renvoie des codes d'erreur traduits par le front et
des messages techniques en anglais. Ajouter une langue : `Collectionnard.md` § 3 « Langues ». Les types suivent cette convention
(`title`, `start_year`, `broadcast_on`, `Ownership`…) ; un changement de clé = nouvelle version du paquet de domaine.

## Documents de référence (à lire selon le besoin, ne pas tout charger d'office)

| Fichier                                | Contenu                                                          |
| -------------------------------------- | ---------------------------------------------------------------- |
| [Collectionnard.md](Collectionnard.md) | Vision, modèle conceptuel, phases de développement (roadmap)     |
| [les.collection.md](les.collection.md) | Modèle de données livres : BD, manga, romans                     |
| [musique.md](musique.md)               | Modèle de données musique : albums, éditions (CD, vinyles), pistes |
| [video.md](video.md)                   | Modèle de données vidéo : films, éditions vidéo (DVD, Blu-ray, VHS), partagées avec les séries |
| [jeux-video.md](jeux-video.md)         | Modèle de données jeux vidéo : jeux, éditions, plateformes rattachées à leur constructeur |
| [api/openapi.yaml](api/openapi.yaml)   | Spécification OpenAPI 3.1 — **source de vérité de l'API**        |
| [db/migrations/](db/migrations/)       | Migrations SQL (tern) ; `001_init.sql` = schéma complet commenté |
| [server/internal/domains/packages/](server/internal/domains/packages/) | **Paquets de domaine** (types : `collections`, `tv`, `people`, `video`, `books`, `music`, `games` ; `requires` : paquets installés avant) — installés par la page « Domaines » ou `install-domain`, mis à jour par fusion |
| [db/seed.sql](db/seed.sql)             | Données de démo « Breaking Bad » (multi-chemin Diffusion + DVD) ; les types viennent des paquets |
| [db/seed_bd_manga.sql](db/seed_bd_manga.sql) | Données de démo BD / Manga (lot 13), chargé après `seed.sql` |
| [db/seed_music.sql](db/seed_music.sql) | Données de démo Musique (P3-4, Discovery), chargé après `seed_bd_manga.sql` |
| [db/seed_video.sql](db/seed_video.sql) | Données de démo Vidéo (P3-5, Amélie et son Blu-ray), chargé après `seed_music.sql` |
| [db/seed_games.sql](db/seed_games.sql) | Données de démo Jeux vidéo (P3-7, Mario Kart 8 Deluxe), chargé après `seed_video.sql` |
| [server/CLAUDE.md](server/CLAUDE.md)   | Conventions backend Go                                           |
| [web/CLAUDE.md](web/CLAUDE.md)         | Conventions frontend Vue                                         |
| [docs/audit/](docs/audit/BRIEF.md)     | Brief et résultats de l'audit approfondi (`/audit`)              |
| [docs/sources-externes.md](docs/sources-externes.md) | Bases ouvertes BD / Manga : licences, accès (Wikidata, BnF retenues), constats du connecteur |

## Modèle en une table

| Méta-type        | Instance         | Le JSON Schema décrit…                        |
| ---------------- | ---------------- | --------------------------------------------- |
| `ItemType`       | `Item`           | les attributs de l'item (titre, année, ISBN…) |
| `RelationType`   | `ItemRelation`   | les attributs du lien (disque, piste, rôle…)  |
| `CollectionType` | `CollectionItem` | les attributs user↔item (état, prix, note…)   |

- Les items forment un **graphe orienté libre** : un même nœud peut être atteint par plusieurs
  chemins (ex. Épisode via Saison _et_ via DVD). Ne jamais modéliser en arbre strict.
- Un `CollectionType` peut limiter les types d'item de ses collections (`item_type_ids`, table
  `collection_type_item_types` ; vide = tous). Seuls les nouveaux ajouts sont contrôlés.
- Un `RelationType` a **plusieurs** types source et cible possibles (tables de jonction
  `relation_type_sources` / `relation_type_targets`, champs `source_item_type_ids` /
  `target_item_type_ids`). Le TODO « arrays » de `les.collection.md` est donc déjà traité.
- Les données propres à un utilisateur (état, prix payé, note) vont sur `CollectionItem`,
  **jamais** sur `Item`.
- `Collectionnard.md` parle d'`AttributeDefinition` : c'est le concept historique.
  **L'implémentation réelle est le JSON Schema** (`attributes_schema` JSONB).

### Extensions JSON Schema maison (pilotent le front)

`x-label` (gabarit du libellé d'une instance : `"{number}. {title}"` — Go `internal/label`, TS
`web/src/lib/schema.ts`, à garder alignés), `x-order` (ordre des champs), `x-labels` (`{fr:{…}, en:{…}}`),
`x-widget` (`image-gallery`, `i18n-text`), `x-image-categories` (`{poster:{ratio:"2/3", tolerance?, min_width?, min_height?, strict?}}`, contrôles à l'envoi dans `ImageGalleryField` via `checkImage`), `x-title` / `x-description`
(nom et description affichés du méta-type par langue, repli sur `name` / `description`), `x-enum-labels` (propriété :
`{fr:{tv:"Télévision"}}`). Pas de widget
`slider` : le curseur est automatique quand `minimum` **et** `maximum` sont définis.
Toute nouvelle extension doit être gérée dans `DynamicForm.vue` **et** `SchemaBuilder.vue` (qui conserve
les mots-clés qu'il ne connaît pas), lue via `lib/schema.ts`, et documentée dans le README et
`Collectionnard.md` § 3.

## Stack

Go 1.27 (chi 5.3, pgx 5.11, golang-jwt 5.3, santhosh-tekuri/jsonschema v5) · PostgreSQL 16 ·
Vue 3.5 + TypeScript 6 · Element Plus 2 (MIT) + Tailwind 4 (plugin Vite, configuration dans `main.css`) · Cytoscape.js · Vite 8 · Docker Compose.

## Environnement : Nix

Le développeur travaille sous **Nix**. Les outils (Go, Node 22, PostgreSQL 16, gopls) viennent
du `flake.nix` ; ils ne sont **pas** installés globalement.

- `flake.nix` et `flake.lock` **doivent rester suivis par git** : les flakes ignorent les fichiers non
  suivis (`error: Path … is not tracked by Git`). Après avoir ajouté un fichier référencé par le
  flake, le `git add` avant `nix develop` ; en secours, `nix develop path:$PWD -c …`.
- Si `direnv` est actif (`.envrc` → `use flake`), `go`, `node` et `psql` sont déjà dans le PATH.
- Sinon, ou si `go`/`npm` est introuvable : **préfixer chaque commande par `nix develop -c`**.
  Ne jamais proposer `apt`, `brew`, `go install` global ou `npm -g` pour installer un outil :
  l'ajouter à `packages` dans `flake.nix`.
- Le shell exporte `DATABASE_URL`, `JWT_SECRET` (≥ 32 caractères, exigé par le serveur), `UPLOAD_DIR`
  (valeurs de dev).

## Commandes

```bash
nix develop                     # entrer dans le shell (inutile avec direnv)
nix flake update                # mettre à jour nixpkgs (flake.lock)

# Base de données — au choix :
pg-start | pg-stop | pg-reset   # PostgreSQL local (./.pgdata) ; pg-start applique les migrations (+ seed à la création)
docker compose up postgres      # ou PostgreSQL via Docker (même port 5432 : n'en lancer qu'un)

# Backend (depuis server/)
nix develop -c go build ./...
nix develop -c gofmt -l .            # doit ne rien afficher
nix develop -c go vet ./...
nix develop -c golangci-lint run ./...
nix develop -c go test ./...
nix develop -c go test -tags integration ./internal/store/   # SQL réel : PostgreSQL requis (pg-start)
nix develop -c govulncheck ./...
nix develop -c go run .

# Frontend (depuis web/)
nix develop -c npm install
nix develop -c npm run dev          # Vite
nix develop -c npm run type-check   # vue-tsc — à lancer après toute modif TS/Vue
nix develop -c npm run lint         # eslint (flat config) ; `lint:fix` pour corriger
nix develop -c npm run format:check # prettier ; `format` pour réécrire
nix develop -c npm test             # vitest (src/**/*.test.ts)
nix develop -c npm run build

# Stack complète conteneurisée (front :3000, API :8080/api/v1, PG :5432)
cp .env.example .env                 # une fois : JWT_SECRET (≥ 32 car.) et POSTGRES_PASSWORD obligatoires
docker compose up --build
docker compose down -v && docker compose up --build   # repart d'une base vide (migrations + seed rejoués)
(cd server && go run . migrate)     # appliquer les migrations à la main (SEED_ON_EMPTY=true pour le seed)
(cd server && go run . install-domain)   # état des paquets de domaine ; `--all` ou `<domaine>` pour installer / mettre à jour
```

Comptes de dev (seed) : `admin` / `admin` (rôle admin), `shinsaber` / `shinsaber` (rôle user).
Nouveau compte : `go run . create-user <nom> <email> [--admin]` (mot de passe demandé).
Production : images publiées par la CI sur ghcr.io (job `images`, tags `v*` et `main`), `docker-compose.prod.yml`
(sans seed) — voir README « Mise en production ».

## Règles de travail

1. **Contrat d'API d'abord** : toute modif d'endpoint met à jour, dans cet ordre,
   `api/openapi.yaml` → `server/internal/model` → store → handler (+ test) →
   `web/src/api/types.ts` + `client.ts` → vues.
2. **Schéma BDD** : ne jamais modifier une migration déjà appliquée ; ajouter
   `db/migrations/NNN_description.sql` (numéro suivant, format tern : SQL « up », puis optionnellement
   `---- create above / drop below ----` et le SQL « down »). Les migrations sont appliquées par
   `colec-server migrate` (`pg-start` et le service compose `migrate` le font automatiquement ;
   à la main : `cd server && go run . migrate`). Signaler toute modif cassante ; `pg-reset` /
   `docker compose down -v` ne servent plus qu'à repartir d'une base vide.
3. **Types et seed** : les méta-types vivent dans les **paquets de domaine** (`server/internal/domains/packages/*.json`,
   format de l'import, UUID fixes) ; modifier un type = incrémenter `package_version` (la mise à jour fusionne,
   les modifications des instances sont gardées). Les seeds ne contiennent que des données de démonstration,
   chargées après l'installation des paquets. UUID lisibles par préfixe — `a…` users, `b…` item_types, `c…` relation_types,
   `d…` collection_types, `e…` items. Garder ce schéma pour les nouveaux jeux de données
   (prendre des plages libres, ne pas renuméroter l'existant).
4. Avant de dire « terminé » : `gofmt -l .` + `go vet` + `golangci-lint run` + `go test ./...` côté
   server, `npm run lint` + `npm run format:check` + `npm run type-check` + `npm test` côté web,
   selon ce qui a été touché (via `nix develop -c` si besoin). La CI (`.github/workflows/ci.yml`)
   rejoue exactement ces étapes.
5. Mettre à jour les cases à cocher de la section « Phases » de `Collectionnard.md`
   quand une fonctionnalité de la roadmap est livrée.
6. Ne jamais committer de secrets : `JWT_SECRET` et `POSTGRES_PASSWORD` viennent de `.env`
   (ignoré par git, modèle dans `.env.example`) ; le serveur refuse un `JWT_SECRET` < 32 caractères.

## Où en est le projet

Phase 1 (MVP) : terminée. Phase 1.5 (remise à niveau issue de l'audit, branche `remise-a-niveau`,
lots 0 à 11 livrés, voir `Collectionnard.md` §8 et `docs/audit/03-feuille-de-route.md`) : terminée, fusionnée dans `main`.
Phase 2 livrée : multilingue, recherche avancée (lot 14), import/export (lot 12), refonte graphique responsive
(identité « Étiquette », lots R0–R8, branche `refonte-graphique`), domaine BD / Manga (lot 13,
`les.collection.md`, sources Wikidata / BnF). Phase 3 (récupération automatique externe) en cours, branche `phase3-sources` : domaines installables (P3-0), connecteurs
BD / Manga, Romans, Musique, Films, Séries et Jeux vidéo (`server/internal/external/connectors`, profils `bd-manga` / `novels` / `music` / `films` / `series` / `games`, clients `wikidata`, `bnf`, `musicbrainz`, `tvmaze`, page admin « Sources externes ») ;
domaines P3-0 à P3-7 (jeux vidéo, V1 française) livrés, non poussés. Phase 4 : social.
