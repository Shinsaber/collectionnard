# Collectionnard

Application de gestion de collections générique et dynamique.
Séries TV, DVD, livres, cartes à collectionner, vinyles… tout type de collection est supporté sans modification de code.

> **Document de conception** : voir [Collectionnard.md](Collectionnard.md) pour la vision, le modèle conceptuel et les décisions de design.

---

## Stack technique

| Composant           | Technologie             | Version    |
| ------------------- | ----------------------- | ---------- |
| **Backend**         | Go (chi 5.3, pgx 5.11, JWT 5.3) | 1.27       |
| **Base de données** | PostgreSQL              | 16         |
| **Frontend**        | Vue 3 + TypeScript      | 3.5 / 5.6  |
| **UI**              | PrimeVue (@primeuix/themes) + Tailwind CSS | 4.5 / 3.4  |
| **Graphes**         | Cytoscape.js            | 3.34       |
| **Build**           | Vite                    | 6.4        |
| **Déploiement**     | Docker Compose          | 4 services |

---

## Démarrage rapide

### Prérequis

- Docker & Docker Compose

### Lancer le projet

```bash
cp .env.example .env      # puis renseigner JWT_SECRET (openssl rand -base64 48) et POSTGRES_PASSWORD
docker compose up --build
```

Sans `.env`, `docker compose` refuse de démarrer : aucun secret n'a de valeur par défaut.

| Service        | URL                          | Description                        |
| -------------- | ---------------------------- | ---------------------------------- |
| **Frontend**   | http://localhost:3000        | SPA Vue 3                          |
| **API**        | http://localhost:8080/api/v1 | API REST                           |
| **PostgreSQL** | localhost:5432               | BDD (user: `colec`, pass: `POSTGRES_PASSWORD`) |

### Utilisateur par défaut

Le fichier `db/seed.sql` crée un utilisateur admin pour le développement :

```
username: admin
password: admin
```

### Développement local (sans Docker)

**Backend** :

```bash
cd server
export DATABASE_URL="postgres://colec:colec@localhost:5432/colec?sslmode=disable"
export JWT_SECRET="dev-secret-dev-secret-dev-secret-0000"   # 32 caractères minimum
go run .
```

(Le shell Nix du dépôt exporte déjà ces variables.)

### Variables d'environnement du serveur

| Variable            | Défaut      | Rôle                                                                 |
| ------------------- | ----------- | -------------------------------------------------------------------- |
| `DATABASE_URL`      | **requis**  | DSN PostgreSQL                                                       |
| `JWT_SECRET`        | **requis**  | Clé HS256 des access tokens, 32 caractères minimum                   |
| `ACCESS_TOKEN_TTL`  | `15m`       | Durée de vie des access tokens JWT (format Go)                        |
| `REFRESH_TOKEN_TTL` | `720h`      | Durée de vie des refresh tokens (rotation à chaque utilisation)       |
| `LOGIN_RATE_LIMIT`  | `10`        | Appels par minute et par IP sur `/auth/login`, `/auth/refresh`, `/auth/logout` |
| `TRUSTED_PROXY`     | `false`     | `true` derrière un reverse-proxy qui pose `X-Real-IP` (nginx du compose) ; sinon l'IP serait usurpable |
| `PORT`              | `8080`      | Port HTTP                                                            |
| `UPLOAD_DIR`        | `uploads`   | Dossier des images uploadées                                         |
| `MAX_UPLOAD_SIZE`   | `10485760`  | Taille max d'un upload (octets) ; les corps JSON sont limités à 1 Mio |

**Frontend** :

```bash
cd web
npm install
npm run dev
```

### Base de données et migrations

Le schéma est géré par des migrations SQL au format [tern](https://github.com/jackc/tern)
dans `db/migrations/` (`001_init.sql` = schéma complet). Elles sont appliquées par le binaire
lui-même :

```bash
cd server
go run . migrate                      # applique les migrations en attente (table schema_version)
SEED_ON_EMPTY=true go run . migrate   # idem, puis db/seed.sql si la table users est vide
go run . seed                         # applique db/seed.sql sans condition (dev)
```

`pg-start` (shell Nix) et le service `migrate` de docker-compose les lancent automatiquement.
Pour changer le schéma : ajouter `db/migrations/NNN_description.sql`, ne jamais modifier une
migration déjà appliquée.

### Mise en production

- Seul le service `web` (nginx, non-root, port 3000) est publié ; `server` et `postgres` restent
  sur le réseau interne de compose. Placer devant un reverse-proxy qui termine TLS (Caddy,
  Traefik, nginx) : l'API suppose HTTPS pour les mots de passe et les tokens.
- `.env` : `JWT_SECRET` aléatoire (≥ 32 caractères), `POSTGRES_PASSWORD` fort,
  `SEED_ON_EMPTY=false` (sinon le compte `admin/admin` est créé sur une base vide).
- Les conteneurs tournent sans privilèges (uid 10001 pour l'API, 101 pour nginx) ; les images
  sont épinglées (`golang:1.27-alpine`, `alpine:3.22`, `node:22-alpine`,
  `nginxinc/nginx-unprivileged:1.28-alpine`, `postgres:16-alpine`).
- Santé : `GET /api/v1/healthz` (utilisé par le healthcheck compose) ; logs JSON sur stdout
  (`LOG_FORMAT=text` pour du lisible, `LOG_LEVEL=debug|info|warn|error`).
- Pour accéder à PostgreSQL depuis l'hôte : `docker compose exec postgres psql -U colec colec`
  (ou publier le port dans un `docker-compose.override.yml` local, jamais commité).

---

## Structure du projet

```
colec/
├── Collectionnard.md       # Document de conception (vision, modèle, phases)
├── README.md               # Ce fichier
├── docker-compose.yml      # Orchestration des 3 services
├── api/
│   └── openapi.yaml        # Spécification OpenAPI 3.1 (référence API)
├── db/
│   ├── migrations/         # Migrations SQL tern (001_init.sql = schéma complet)
│   ├── seed.sql            # Données de test (admin, types, items, relations)
│   └── queries_test.sql    # Requêtes SQL de test
├── server/                 # Backend Go
│   ├── main.go
│   ├── Dockerfile
│   ├── config/             # Configuration (env vars)
│   └── internal/
│       ├── handler/        # Handlers HTTP (1 fichier par domaine)
│       ├── middleware/     # Auth JWT, RequireAdmin, RateLimiter, RequestLogger
│       ├── dbmigrate/      # Migrations (tern) + seed
│       ├── model/          # Structs Go (JSON tags)
│       ├── store/          # Couche d'accès BDD (pgx)
│       ├── validator/      # Validation JSON Schema
│       └── testutil/       # Mocks pour les tests
└── web/                    # Frontend Vue 3
    ├── Dockerfile
    ├── nginx.conf
    └── src/
        ├── api/
        │   ├── client.ts   # Client Axios (tous les appels API)
        │   └── types.ts    # Types TypeScript (miroir des models Go)
        ├── stores/
        │   └── auth.ts     # Store Pinia (auth + JWT)
        ├── router/
        │   └── index.ts    # Routes Vue Router
        ├── views/
        │   ├── DashboardView.vue
        │   ├── ItemsView.vue
        │   ├── ItemDetailView.vue
        │   ├── GraphView.vue
        │   ├── CollectionsView.vue
        │   ├── CollectionDetailView.vue
        │   ├── LoginView.vue
        │   └── admin/
        │       ├── ItemTypesView.vue
        │       ├── RelationTypesView.vue
        │       └── CollectionTypesView.vue
        └── components/
            ├── layout/
            │   └── AppLayout.vue
            ├── common/
            │   ├── DynamicForm.vue      # Formulaire généré depuis JSON Schema
            │   └── SchemaBuilder.vue    # Éditeur de JSON Schema (admin)
            └── item/
                ├── ItemEditDialog.vue
                └── RelationTree.vue     # Arbre de relations groupé par type
```

---

## Concepts clés

Le système repose sur un **pattern unique à 3 niveaux** : des méta-types définissent des JSON Schemas, et les instances portent les valeurs validées contre ces schémas.

| Méta-type          | Instances      | Exemple                                     |
| ------------------ | -------------- | ------------------------------------------- |
| **ItemType**       | Item           | "Série" → `{titre, synopsis, année_début…}` |
| **RelationType**   | ItemRelation   | "inclut_épisode" → `{disque, piste}`        |
| **CollectionType** | CollectionItem | "Possession" → `{état, prix, date}`         |

Les items forment un **graphe orienté libre** : un même épisode peut être atteint via une diffusion TV et un DVD. Voir l'endpoint `GET /items/{id}/graph`.

---

## API

La spécification complète est dans [api/openapi.yaml](api/openapi.yaml).

### Endpoints principaux

| Domaine         | Endpoints                                       | Auth                              |
| --------------- | ----------------------------------------------- | --------------------------------- |
| Auth            | `POST /auth/login`, `POST /auth/refresh`, `POST /auth/logout`, `GET /auth/me` | Public (rate-limited) / JWT |
| ItemTypes       | CRUD `/item-types`                              | Admin                             |
| RelationTypes   | CRUD `/relation-types`                          | Admin                             |
| CollectionTypes | CRUD `/collection-types`                        | Admin                             |
| Items           | CRUD `/items` + recherche, filtres, tri         | Admin (écriture)                  |
| Relations       | CRUD `/relations` + `GET /items/{id}/relations` | Admin (écriture)                  |
| Graph           | `GET /items/{id}/graph`                         | JWT                               |
| Collections     | CRUD `/collections` + gestion items             | JWT (owner)                       |
| Tags            | `GET /tags`                                     | JWT                               |
| Stats           | `GET /stats`                                    | JWT                               |
| Santé           | `GET /healthz`                                  | Public                            |
| Uploads         | `POST /uploads`, `GET /uploads/{filename}`      | Admin (upload) / Public (lecture) |

---

## Extensions JSON Schema

Le projet utilise des extensions custom dans les JSON Schemas pour piloter le frontend :

| Extension            | Rôle                                                         |
| -------------------- | ------------------------------------------------------------ |
| `x-order`            | Ordre d'affichage des champs (array de clés)                 |
| `x-labels`           | Labels i18n par langue (`{fr: {champ: "Label"}, en: {…}}`)   |
| `x-widget`           | Widget spécifique (`image-gallery`, `slider`, `i18n-text`)   |
| `x-image-categories` | Catégories d'images avec ratios (`{poster: {ratio: "2/3"}}`) |

---

## Licence

MIT
