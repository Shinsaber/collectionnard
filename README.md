# Collectionnard

Application de gestion de collections générique et dynamique.
Séries TV, DVD, livres, cartes à collectionner, vinyles… tout type de collection est supporté sans modification de code.

> **Documentation** : le document de conception, les modèles de données par domaine et les notes de phase sont
> rassemblés dans [docs/](docs/) ; voir la section [Documentation](#documentation) ci-dessous.

---

## Stack technique

| Composant           | Technologie                                                                                            | Version    |
| ------------------- | ------------------------------------------------------------------------------------------------------ | ---------- |
| **Backend**         | Go (chi 5.3, pgx 5.11, JWT 5.3)                                                                        | 1.27       |
| **Base de données** | PostgreSQL                                                                                             | 16         |
| **Frontend**        | Vue 3 + TypeScript                                                                                     | 3.5 / 6.0  |
| **UI**              | Element Plus (MIT) + Tailwind CSS (plugin Vite) ; identité « Étiquette », responsive (mobile → bureau) | 2.14 / 4.3 |
| **Graphes**         | Cytoscape.js                                                                                           | 3.34       |
| **Statistiques**    | Apache ECharts (Apache 2.0, chargé à la demande)                                                       | 6.1        |
| **Build**           | Vite (Rolldown) + vue-tsc 3, Pinia 4, vue-router 5                                                     | 8.3        |
| **Déploiement**     | Docker Compose                                                                                         | 4 services |

---

## Démarrage rapide

### Prérequis

- Docker & Docker Compose pour la stack complète ;
- ou **Nix** (avec `direnv` conseillé) pour le développement : `flake.nix` fournit Go, Node 22, PostgreSQL 16,
  golangci-lint et les commandes `pg-start` / `pg-stop` / `pg-reset`. Sans direnv : `nix develop`, ou préfixer
  chaque commande par `nix develop -c`. Les variables de dev (`DATABASE_URL`, `JWT_SECRET`, `UPLOAD_DIR`,
  `LOG_FORMAT=text`) sont exportées par le shell. Toutes les commandes de ce fichier sont données avec la variante
  `nix develop -c` quand elle s'applique.

### Lancer le projet

```bash
cp .env.example .env      # puis renseigner JWT_SECRET (openssl rand -base64 48) et POSTGRES_PASSWORD
docker compose up --build
```

Sans `.env`, `docker compose` refuse de démarrer : aucun secret n'a de valeur par défaut.

| Service        | URL                          | Description                                    |
| -------------- | ---------------------------- | ---------------------------------------------- |
| **Frontend**   | http://localhost:3000        | SPA Vue 3                                      |
| **API**        | http://localhost:8080/api/v1 | API REST                                       |
| **PostgreSQL** | localhost:5432               | BDD (user: `colec`, pass: `POSTGRES_PASSWORD`) |

### Utilisateur par défaut

Le fichier `db/seed.sql` crée deux comptes de développement (appliqué si `SEED_ON_EMPTY=true`) :

| Utilisateur | Mot de passe | Rôle  |
| ----------- | ------------ | ----- |
| `admin`     | `admin`      | admin |
| `shinsaber` | `shinsaber`  | user  |

Pour créer un compte : `colec-server create-user <nom> <email> [--admin|--contributor|--moderator]` (mot de passe demandé
sans écho, ou lu sur l'entrée standard ; 8 caractères minimum). Avec Docker :
`docker compose run --rm server colec-server create-user <nom> <email> --admin`. Les autres comptes se créent
ensuite depuis l'application, sur invitation (page « Utilisateurs », ou « Mon profil » si `MEMBER_INVITES=true`).

### Développement local (sans Docker)

Deux façons d'avoir les outils : entrer dans le shell Nix (`nix develop`, automatique avec direnv), ou préfixer chaque
commande par `nix develop -c`. Dans les deux cas `DATABASE_URL`, `JWT_SECRET` et `UPLOAD_DIR` sont déjà exportés avec des
valeurs de développement. Sans Nix, exporter ces variables à la main (voir le tableau ci-dessous).

**Avec Nix** (depuis la racine) :

```bash
nix develop -c pg-start                      # PostgreSQL local dans ./.pgdata, migrations + seed à la création (pg-stop, pg-reset)
nix develop -c sh -c 'cd server && go run .'  # API sur :8080
nix develop -c sh -c 'cd web && npm install && npm run dev'   # Vite sur :5173, proxy vers l'API
```

Dans le shell (`nix develop` une fois, puis les commandes nues) :

```bash
pg-start
(cd server && go run .)
(cd web && npm install && npm run dev)
```

**Sans Nix, backend** (Go 1.27 et PostgreSQL 16 installés par ailleurs) :

```bash
cd server
export DATABASE_URL="postgres://colec:colec@localhost:5432/colec?sslmode=disable"
export JWT_SECRET="dev-secret-dev-secret-dev-secret-0000"   # 32 caractères minimum
go run .
```

**Sans Nix, frontend** (Node 22) :

```bash
cd web
npm install
npm run dev
```

### Variables d'environnement du serveur

| Variable                | Défaut                               | Rôle                                                                                                                                                                                                            |
| ----------------------- | ------------------------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `DATABASE_URL`          | **requis**                           | DSN PostgreSQL                                                                                                                                                                                                  |
| `JWT_SECRET`            | **requis**                           | Clé HS256 des access tokens, 32 caractères minimum                                                                                                                                                              |
| `ACCESS_TOKEN_TTL`      | `15m`                                | Durée de vie des access tokens JWT (format Go)                                                                                                                                                                  |
| `REFRESH_TOKEN_TTL`     | `720h`                               | Durée de vie des refresh tokens (rotation à chaque utilisation)                                                                                                                                                 |
| `LOGIN_RATE_LIMIT`      | `10`                                 | Appels par minute et par IP sur `/auth/*`, en deux compteurs : identifiants et jetons à usage unique (`login`, `register`, `reset-password`, `invitation`) d'une part, session (`refresh`, `logout`) de l'autre |
| `TRUSTED_PROXY`         | `false`                              | `true` derrière un reverse-proxy qui pose `X-Real-IP` (nginx du compose) ; sinon l'IP serait usurpable                                                                                                          |
| `GRAPH_MAX_NODES`       | `500`                                | Nombre max de nœuds renvoyés par `GET /items/{id}/graph` (au-delà : `truncated: true`)                                                                                                                          |
| `PORT`                  | `8080`                               | Port HTTP                                                                                                                                                                                                       |
| `UPLOAD_DIR`            | `uploads`                            | Dossier des images uploadées (vignettes générées dans `.thumbs/`)                                                                                                                                               |
| `MAX_UPLOAD_SIZE`       | `10485760`                           | Taille max d'un upload (octets) ; les corps JSON sont limités à 1 Mio ; une image de plus de 25 mégapixels est refusée (`image_too_large`)                                                                      |
| `UPLOAD_RATE_LIMIT`     | `60`                                 | Fichiers qu'un compte (hors admin) peut envoyer par heure (`429 upload_quota` au-delà)                                                                                                                          |
| `UPLOAD_RATE_BYTES`     | `314572800`                          | Octets qu'un compte (hors admin) peut envoyer par heure (300 Mio)                                                                                                                                               |
| `UPLOAD_PURGE_INTERVAL` | `24h`                                | Période de la purge automatique des images que rien ne référence (délai de grâce 24 h) ; `0` : seulement le bouton Administration → Images                                                                      |
| `MAX_IMPORT_SIZE`       | `20971520`                           | Taille max d'un fichier d'import du catalogue ou d'une collection (octets)                                                                                                                                      |
| `EXTERNAL_USER_AGENT`   | `Collectionnard/1.0 (+URL du dépôt)` | User-Agent envoyé à Wikidata, à la BnF, à MusicBrainz et à TVmaze (leurs règles demandent un nom et un contact)                                                                                                 |
| `PROPOSAL_QUOTA`        | `5`                                  | Propositions en attente par membre (+1 par proposition acceptée, plafond 50) ; contributeurs et admins sans quota                                                                                               |
| `MEMBER_INVITES`        | `false`                              | `true` : les membres peuvent aussi créer des invitations (sinon les admins seulement)                                                                                                                           |
| `LOG_FORMAT`            | `json`                               | `text` pour des logs lisibles en développement                                                                                                                                                                  |
| `LOG_LEVEL`             | `info`                               | `debug`, `info`, `warn` ou `error`                                                                                                                                                                              |

Sous-commandes `migrate` et `seed` : `MIGRATIONS_DIR` (dossier des migrations ; défaut : le premier `db/migrations`
trouvé en remontant depuis le dossier courant), `SEED_ON_EMPTY` (`true` : `migrate` charge les seeds si la table `users`
est vide ; exigé aussi par `seed`, qui refuse sinon) et `SEED_FILE` (liste de fichiers séparés par des virgules, sinon les
seeds par défaut, voir ci-dessous).

### Vérifier avant un commit

Les mêmes étapes que la CI (`.github/workflows/ci.yml`), ici en variante `nix develop -c` ; dans le shell Nix, retirer le
préfixe.

```bash
# Backend (depuis server/)
nix develop -c gofmt -l .                 # ne doit rien afficher
nix develop -c go vet ./...
nix develop -c golangci-lint run ./...
nix develop -c go test ./...
nix develop -c go test -tags integration ./internal/store/   # SQL réel : PostgreSQL requis (pg-start)
nix develop -c govulncheck ./...

# Frontend (depuis web/)
nix develop -c npm run lint
nix develop -c npm run lint:openapi       # dès que api/openapi.yaml change
nix develop -c npm run format:check       # `npm run format` pour réécrire
nix develop -c npm run type-check
nix develop -c npm test
nix develop -c npm run build
nix develop -c npm audit --audit-level=high
```

### Base de données et migrations

Le schéma est géré par des migrations SQL au format [tern](https://github.com/jackc/tern)
dans `db/migrations/` (`001_init.sql` = schéma complet). Elles sont appliquées par le binaire
lui-même :

```bash
cd server                             # préfixer par `nix develop -c` hors du shell Nix
go run . migrate                      # applique les migrations en attente (table schema_version)
SEED_ON_EMPTY=true go run . migrate   # idem, puis les seeds si la table users est vide
SEED_ON_EMPTY=true go run . seed      # applique les seeds sans condition (dev ; refusé sans la variable : admin/admin)
# Seeds : db/seed.sql, db/seed_bd_manga.sql, db/seed_music.sql, db/seed_video.sql puis db/seed_games.sql ; SEED_FILE=a.sql,b.sql pour une autre liste
go run . install-domain               # état des paquets de domaine ; --all ou <domaine> pour installer / mettre à jour
nix develop -c sh -c 'cd server && go run . migrate'          # la même chose, depuis la racine, sans entrer dans le shell
```

`pg-start` (shell Nix) et le service `migrate` de docker-compose les lancent automatiquement.
Pour changer le schéma : ajouter `db/migrations/NNN_description.sql`, ne jamais modifier une
migration déjà appliquée. Les types (séries TV, livres, musique…) ne sont pas dans les seeds : ce sont des
**paquets de domaine** embarqués dans le serveur, installés avant les seeds (voir « Mise en production »).

### Mise en production

La CI publie les images sur GitHub Container Registry (publiques, `linux/amd64` et `linux/arm64`),
après les vérifications : `ghcr.io/shinsaber/collectionnard-server` et `ghcr.io/shinsaber/collectionnard-web`.

| Déclencheur         | Étiquettes               |
| ------------------- | ------------------------ |
| tag `v1.2.0` poussé | `1.2.0`, `1.2`, `latest` |
| fusion dans `main`  | `main` (version de test) |
| tout push publié    | `sha-<commit>`           |

Sur le serveur, `docker-compose.prod.yml` utilise ces images (aucun build, aucun jeu de démonstration) :

```bash
cp .env.example .env              # JWT_SECRET, POSTGRES_PASSWORD ; COLEC_VERSION (latest), WEB_PORT (3000)
docker compose -f docker-compose.prod.yml pull
docker compose -f docker-compose.prod.yml up -d
# premier compte (la base démarre vide) :
docker compose -f docker-compose.prod.yml run --rm server colec-server create-user <nom> <email> --admin
# types de collection (séries TV, livres…) : page Administration → Domaines, ou :
docker compose -f docker-compose.prod.yml run --rm server colec-server install-domain --all
```

Publier une version : `git tag v1.0.0 && git push origin v1.0.0`. Mettre à jour le serveur : `pull` puis `up -d`
(les migrations s'appliquent au démarrage). Après une mise à jour, la page « Domaines » signale les paquets
à mettre à jour ; la mise à jour **fusionne** : les types modifiés sur l'instance gardent leurs modifications, seuls
les ajouts du paquet (champs, valeurs, libellés) sont appliqués.

- Seul le service `web` (nginx, non-root, port 3000) est publié ; `server` et `postgres` restent
  sur le réseau interne de compose. Placer devant un reverse-proxy qui termine TLS (Caddy,
  Traefik, nginx) : l'API suppose HTTPS pour les mots de passe et les tokens, et l'application
  installable (mode hors ligne) l'exige, comme l'accès à la caméra (photo de couverture, scanner de
  codes-barres ; `Permissions-Policy: camera=(self)` et CSP `'wasm-unsafe-eval'` dans `web/nginx-security-headers.conf`).
- `REAL_IP_FROM` (`.env`) : adresse ou réseau CIDR du reverse-proxy (ex. `172.18.0.0/16`). Sans elle, nginx
  ne voit que l'adresse du proxy : le limiteur de débit par IP de `/auth/*` deviendrait global à l'instance
  et les journaux inutilisables. Avec elle, nginx relit l'adresse du client dans `X-Forwarded-For` (dernier
  saut non fiable) et la transmet au serveur en `X-Real-IP` (`web/nginx-real-ip.conf.template`).
- `.env` : `JWT_SECRET` aléatoire (≥ 32 caractères), `POSTGRES_PASSWORD` aléatoire (le modèle
  `.env.example` ne fournit plus de valeur), `SEED_ON_EMPTY=false` avec `docker-compose.yml` (sinon le compte
  `admin/admin` est créé sur une base vide) ; `docker-compose.prod.yml` l'impose, et `colec-server seed` refuse de
  s'exécuter sans `SEED_ON_EMPTY=true`. `.gitignore` exclut tout `.env*` sauf `.env.example`.
- Les conteneurs tournent sans privilèges (uid 10001 pour l'API, 101 pour nginx) ; les images de base
  (`golang:1.27-alpine`, `alpine:3.22`, `node:22-alpine`, `nginxinc/nginx-unprivileged:1.28-alpine`,
  `postgres:16-alpine`) et les actions de la CI sont épinglées par digest / SHA de commit, suivis par Dependabot
  (`.github/dependabot.yml`).
- Santé : `GET /api/v1/healthz` (utilisé par le healthcheck compose) ; logs JSON sur stdout
  (`LOG_FORMAT=text` pour du lisible, `LOG_LEVEL=debug|info|warn|error`). Le journal d'accès nginx
  n'écrit pas la query string (les jetons de partage et d'invitation y passent).
- Pour accéder à PostgreSQL depuis l'hôte : `docker compose exec postgres psql -U colec colec`
  (ou publier le port dans un `docker-compose.override.yml` local, jamais commité).
- Si le build de `web` échoue sur « Cannot find native binding » (rolldown / Tailwind), une dépendance
  optionnelle npm n'a pas été téléchargée dans une couche mise en cache : `docker compose build --no-cache web`.

---

## Documentation

Tout est dans [docs/](docs/), sauf ce README et les fichiers `CLAUDE.md` (conventions pour Claude Code, lus à leur
emplacement).

| Document                                                                                     | Contenu                                                                                                                 |
| -------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------- |
| [docs/conception.md](docs/conception.md)                                                     | Vision, modèle conceptuel, attributs dynamiques et extensions `x-*`, langues, permissions, **roadmap** (§ 8)            |
| [docs/domaines/livres.md](docs/domaines/livres.md)                                           | Livres : BD, manga, romans (paquet `books`), sources Wikidata / BnF, connecteurs `bd-manga` et `novels`                 |
| [docs/domaines/musique.md](docs/domaines/musique.md)                                         | Musique : albums, éditions (CD, vinyles), pistes, artistes, labels (paquet `music`, MusicBrainz)                        |
| [docs/domaines/video.md](docs/domaines/video.md)                                             | Vidéo : films, éditions vidéo (DVD, Blu-ray) partagées avec les séries, séries en vidéo (paquets `video`, `tv`)         |
| [docs/domaines/jeux-video.md](docs/domaines/jeux-video.md)                                   | Jeux vidéo : jeux, éditions, plateformes rattachées à leur constructeur (paquet `games`)                                |
| [docs/sources-externes.md](docs/sources-externes.md)                                         | Bases ouvertes : licences, accès, constats faits en codant chaque connecteur, sources écartées                          |
| [docs/phases/evolutions-socle.md](docs/phases/evolutions-socle.md)                           | Lots E1–E5 : identifiants insensibles à la casse, ordre des types, types d'item permis, images, hors ligne              |
| [docs/phases/social.md](docs/phases/social.md)                                               | Phase 4 : comptes et invitations, statistiques personnelles, visibilité et partage, contributions                       |
| [docs/phases/mobile.md](docs/phases/mobile.md)                                               | Phase 5 : photo de couverture, redressement et recadrage, scanner de codes-barres, page « Scanner », retours v0.4.0     |
| [docs/phases/interactions.md](docs/phases/interactions.md)                                   | Phase 6 (livrée en test) : avis à critères, modération et réactions, commentaires, articles, recommandations et blocage |
| [docs/phases/exemplaires.md](docs/phases/exemplaires.md)                                     | Phase 7 (en cours, P7-1 livré) : exemplaires multiples (lots avec quantité), amis, prêts, cadeaux réservés              |
| [docs/idées.md](docs/idées.md)                                                               | Pistes notées mais non retenues : prêts, cadeaux, comparaison de collections, suivi, suppression du compte…             |
| [docs/audits/suivi.md](docs/audits/suivi.md)                                                 | Audits récurrents (`/audit`, lecture seule) : suivi des constats à identifiant stable et rapports datés `AAAA-MM-DD.md` |
| [api/openapi.yaml](api/openapi.yaml)                                                         | Spécification OpenAPI 3.1, **source de vérité de l'API**                                                                |
| [db/migrations/001_init.sql](db/migrations/001_init.sql)                                     | Schéma de base de données commenté (puis `002` à `012`)                                                                 |
| [CLAUDE.md](CLAUDE.md), [server/CLAUDE.md](server/CLAUDE.md), [web/CLAUDE.md](web/CLAUDE.md) | Règles de travail, checklists et organisation du code (backend Go, frontend Vue)                                        |

---

## Structure du projet

```
collectionnard/
├── README.md               # Ce fichier
├── LICENSE                 # MIT
├── CLAUDE.md               # Contexte et règles de travail pour Claude Code (+ server/CLAUDE.md, web/CLAUDE.md)
├── docs/                   # Documentation (voir ci-dessus)
│   ├── conception.md       # Vision, modèle conceptuel, extensions x-*, roadmap (§ 8)
│   ├── sources-externes.md # Bases ouvertes : licences, accès, constats des connecteurs
│   ├── domaines/           # Modèles de données : livres, musique, video, jeux-video
│   ├── audits/             # Audits récurrents : suivi.md (constats) et rapports datés
│   └── phases/             # Notes de conception : evolutions-socle, social, mobile
├── docker-compose.yml      # Développement : 4 services (postgres, migrate, server, web)
├── docker-compose.prod.yml # Production : images ghcr.io, sans seed
├── .env.example            # Variables de déploiement (copier en .env)
├── flake.nix, flake.lock   # Environnement de développement Nix (+ .envrc pour direnv)
├── .github/workflows/      # CI : Go (fmt, vet, lint, tests, govulncheck, intégration PG), web (lint, OpenAPI, tests, build, audit), images Docker
├── .claude/                # Claude Code : commandes (/verif, /endpoint, /roadmap, /modele-vers-seed), skill /audit, agent « auditeur », permissions
├── api/
│   └── openapi.yaml        # Spécification OpenAPI 3.1 (référence API)
├── db/
│   ├── migrations/         # Migrations SQL tern (001_init.sql = schéma complet, puis 002 à 012)
│   ├── seed.sql            # Démo : comptes de dev, Breaking Bad (séries TV, multi-chemin Diffusion + DVD)
│   ├── seed_bd_manga.sql   # Démo BD / Manga / Romans (Dragon Ball, Astérix, L'Étranger ; Wikidata / BnF)
│   ├── seed_music.sql      # Démo Musique (Discovery de Daft Punk ; MusicBrainz)
│   ├── seed_video.sql      # Démo Vidéo (Amélie, son Blu-ray ; Wikidata / BnF)
│   ├── seed_games.sql      # Démo Jeux vidéo (Mario Kart 8 Deluxe ; Wikidata / BnF)
│   └── queries_test.sql    # Requêtes SQL de test
├── server/                 # Backend Go
│   ├── main.go, cmd.go     # Serveur HTTP et sous-commandes (migrate, seed, create-user, install-domain)
│   ├── Dockerfile
│   ├── config/             # Configuration (variables d'environnement)
│   └── internal/
│       ├── handler/        # Handlers HTTP, un fichier par domaine ; toutes les routes dans handler.go
│       ├── middleware/     # Auth JWT, rôles, rate limiter, proxy de confiance, journal des requêtes
│       ├── model/          # Structs Go (tags JSON)
│       ├── store/          # Accès BDD (pgx), import / export, historique, propositions
│       ├── validator/      # Validation JSON Schema
│       ├── label/, identifier/, privacy/, collstats/, itemquery/, csvexport/
│       │                   # Extensions x-* (libellés, identifiants, champs privés, complétude), recherche, CSV
│       ├── domains/        # Paquets de domaine embarqués (packages/*.json) et fusion à trois versions
│       ├── external/       # Sources ouvertes : clients Wikidata, BnF (SRU), MusicBrainz, TVmaze ; connecteurs
│       ├── dbmigrate/      # Migrations (tern) + seeds
│       └── testutil/       # MockStore et serveur de test
└── web/                    # Frontend Vue 3
    ├── Dockerfile, nginx.conf, nginx-security-headers.conf
    ├── scripts/            # seed-fixtures.mjs : schémas des paquets de domaine pour les tests
    ├── tests/              # Configuration Vitest et fixtures
    └── src/
        ├── api/            # client.ts (axios, seul point d'appel HTTP), types.ts (miroir des models Go)
        ├── stores/         # Pinia : auth, metaTypes, notifications, offline
        ├── router/         # Routes Vue Router
        ├── i18n/           # vue-i18n : registre des langues, catalogues fr / en
        ├── lib/            # Logique pure testée : schema, itemSearch, csv, barcode, perspective, images, lightbox…
        ├── composables/    # useLang, useTheme, useNotify, useCamera, usePagination…
        ├── views/          # Pages (catalogue, fiche, graphe, collections, statistiques, scanner, propositions, profil), admin/, public/
        └── components/     # common/ (DynamicForm, SchemaBuilder, scanner, recadrage), item/, collection/, external/, sharing/, stats/, proposals/, layout/…
```

---

## Concepts clés

Le système repose sur un **pattern unique à 3 niveaux** : des méta-types définissent des JSON Schemas, et les instances portent les valeurs validées contre ces schémas.

| Méta-type          | Instances      | Exemple                                                       |
| ------------------ | -------------- | ------------------------------------------------------------- |
| **ItemType**       | Item           | "Series" → `{title, synopsis, start_year…}`                   |
| **RelationType**   | ItemRelation   | "includes_episode" → `{disc, track}`                          |
| **CollectionType** | CollectionItem | "Ownership" → `{condition, purchase_price, acquisition_date}` |
| **ReviewType**     | Review         | "ComicReview" → `{story, art}` : critères notés de 0 à 10     |

Les items forment un **graphe orienté libre** : un même épisode peut être atteint via une diffusion TV et un DVD. Voir l'endpoint `GET /items/{id}/graph`.

Les méta-types sont livrés par **paquets de domaine** (`collections`, `tv`, `people`, `books`, `music`, `video`, `games`),
installés depuis la page « Domaines » et mis à jour par fusion ; les connecteurs de la page « Sources externes » remplissent le
catalogue depuis des bases ouvertes (Wikidata, BnF, MusicBrainz, TVmaze).

---

## API

La spécification complète est dans [api/openapi.yaml](api/openapi.yaml). Trois rôles : **admin**, **contributeur**
(écrit dans le catalogue, modère, ne supprime pas d'item) et **membre** ; les pages publiques et les liens de partage
sont lisibles sans compte.

### Endpoints principaux

| Domaine              | Endpoints                                                                                                                                                                                                                                                                                                           | Auth                                                     |
| -------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------- |
| Santé                | `GET /healthz`                                                                                                                                                                                                                                                                                                      | Public                                                   |
| Auth                 | `POST /auth/login`, `/auth/refresh`, `/auth/logout`, `GET /auth/invitation`, `POST /auth/register`, `POST /auth/reset-password` ; `GET /auth/me`                                                                                                                                                                    | Public (rate-limited) / JWT                              |
| Mon compte           | `PUT /me/profile`, `PUT /me/password`, `GET /me/stats`, `GET /me/offline` (copie hors ligne), `GET /me/notifications`, `POST /me/notifications/read`                                                                                                                                                                | JWT                                                      |
| Invitations, membres | `GET` / `POST /invitations`, `DELETE /invitations/{id}` ; `GET /members`                                                                                                                                                                                                                                            | JWT (invitations : admin, ou membre si `MEMBER_INVITES`) |
| Utilisateurs         | `GET /users`, `PUT /users/{id}`, `POST /users/{id}/reset-link`                                                                                                                                                                                                                                                      | Admin                                                    |
| Méta-types           | `GET /item-types`, `/relation-types`, `/collection-types` (+ `/{id}`) ; `POST` / `PUT` / `DELETE`                                                                                                                                                                                                                   | JWT (lecture) / Admin (écriture)                         |
| Items                | `GET /items` (plein texte, filtres et tri par attribut, `related_to`), `GET /items/{id}` (+ `/relations`, `/relations/collections`, `/graph`, `/collections`, `/history`), `GET /items/export` (CSV), `POST /items/duplicates` ; `POST /items`, `PUT /items/{id}`, `POST /items/{id}/revert` ; `DELETE /items/{id}` | JWT / Admin ou contributeur / Admin                      |
| Relations            | `GET /relations/{id}` ; `POST /relations`, `PUT` / `DELETE /relations/{id}`                                                                                                                                                                                                                                         | JWT / Admin ou contributeur                              |
| Collections          | CRUD `/collections`, `/collections/{id}/items`, `/export` (JSON, CSV), `POST /collections/import`, `/stats`, `/share-links`                                                                                                                                                                                         | JWT (propriétaire)                                       |
| Pages publiques      | `GET /public/users/{username}` (+ `/collections`), `GET /public/collections/{id}` (+ `/items`, `/stats`), `GET /share?token=…` (+ `/items`, `/stats`)                                                                                                                                                               | Anonyme ou JWT (selon la visibilité)                     |
| Propositions         | `GET` / `POST /proposals`, `/proposals/{id}` (`submit`, `withdraw`, `conflicts`, `comments`), `/proposals/quota`, `/proposals/review-count` ; `accept`, `reject`, `request-changes`                                                                                                                                 | JWT / modération : admin ou contributeur                 |
| Sources externes     | `GET /external/connectors`, `GET /external/{connector}/works`, `/editions`, `POST /preview`, `POST /propose` ; `POST /external/{connector}/import`                                                                                                                                                                  | JWT / Admin ou contributeur                              |
| Import / export      | `GET /export`, `POST /import` (catalogue JSON, simulation `dry_run`)                                                                                                                                                                                                                                                | Admin                                                    |
| Domaines             | `GET /domains`, `POST /domains/{id}/install`                                                                                                                                                                                                                                                                        | Admin                                                    |
| Tags, statistiques   | `GET /tags`, `GET /stats`                                                                                                                                                                                                                                                                                           | JWT                                                      |
| Uploads              | `POST /uploads` ; `GET /uploads/{filename}` (`?w=240` : vignette) ; `POST /uploads/purge`                                                                                                                                                                                                                           | JWT / Public / Admin                                     |

---

## Extensions JSON Schema

Le projet utilise des extensions custom dans les JSON Schemas pour piloter le frontend :

| Extension            | Rôle                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
| -------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `x-label`            | Gabarit du libellé des instances (`"{title}"`, `"{number}. {title}"`) ; repli `title` / `name` / `label`                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| `x-order`            | Ordre d'affichage des champs (array de clés)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  |
| `x-labels`           | Libellés des champs par langue (`{fr: {champ: "Libellé"}, en: {…}}`) ; langue choisie dans l'en-tête                                                                                                                                                                                                                                                                                                                                                                                                                                          |
| `x-title`            | Nom affiché du type par langue (`{fr: "Série", en: "Series"}`) ; repli sur `name`                                                                                                                                                                                                                                                                                                                                                                                                                                                             |
| `x-description`      | Description du type par langue ; repli sur `description`                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| `x-enum-labels`      | (propriété) Libellés des valeurs d'enum par langue (`{fr: {tv: "Télévision"}, en: {tv: "TV"}}`)                                                                                                                                                                                                                                                                                                                                                                                                                                               |
| `x-widget`           | Widget spécifique : `image-gallery`, `i18n-text` (une chaîne par langue), `links` (liste de liens libellé + URL)                                                                                                                                                                                                                                                                                                                                                                                                                              |
| `x-private`          | (propriété, CollectionType) Champ caché aux autres lecteurs (prix payé) ; retiré par le serveur des pages publiques et liens de partage                                                                                                                                                                                                                                                                                                                                                                                                       |
| `x-identifier`       | (propriété, ItemType) Identifiant (ISBN, EAN, QID…) : doublons signalés avant l'enregistrement ; bouton « scanner un code-barres » (caméra) à côté du champ                                                                                                                                                                                                                                                                                                                                                                                   |
| `x-completeness`     | (racine, RelationType) Relation de composition : « 30 tomes sur 42 » dans les statistiques d'une collection                                                                                                                                                                                                                                                                                                                                                                                                                                   |
| `x-lendable`         | (racine, CollectionType) Les lots des collections de ce type se prêtent, à un ami ou à un simple nom (Possession, P7-3)                                                                                                                                                                                                                                                                                                                                                                                                                       |
| `x-reservable`       | (racine, CollectionType) Liste de souhaits : les amis du propriétaire y réservent des cadeaux, sans qu'il le sache (Liste d'envies, P7-4)                                                                                                                                                                                                                                                                                                                                                                                                     |
| `x-image-categories` | Catégories d'images avec ratios (`{poster: {ratio: "2/3"}}`) et contrôles facultatifs à l'envoi : `tolerance` (écart de ratio en %), `min_width` / `min_height` (px), `strict` (recadrage proposé plutôt qu'avertissement). Sur écran tactile, une photo peut être prise dans l'application (ou avec l'appareil du téléphone), redressée (bords détectés, coins ajustables avec loupe, rotation) puis recadrée au ratio ; les photos sont réduites à 2000 px. Les images s'ouvrent en grand (zoom, rotation) depuis la fiche et les vignettes |

Sans extension, le widget découle du schéma : `enum` → liste déroulante, `integer`/`number` avec `minimum`
**et** `maximum` → curseur, `format: date` / `uri` → champ dédié. Référence complète :
[docs/conception.md](docs/conception.md) § 3.

---

## Licence

[MIT](LICENSE)
