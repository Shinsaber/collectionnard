# Collectionnard

Application de gestion de collections générique et dynamique.
Séries TV, DVD, livres, cartes à collectionner, vinyles… tout type de collection est supporté sans modification de code.

> **Document de conception** : voir [Collectionnard.md](Collectionnard.md) pour la vision, le modèle conceptuel et les décisions de design.

---

## Stack technique

| Composant           | Technologie             | Version    |
| ------------------- | ----------------------- | ---------- |
| **Backend**         | Go (chi, pgx, JWT)      | 1.22       |
| **Base de données** | PostgreSQL              | 16         |
| **Frontend**        | Vue 3 + TypeScript      | 3.5 / 5.6  |
| **UI**              | PrimeVue + Tailwind CSS | 4.3 / 3.4  |
| **Graphes**         | Cytoscape.js            | 3.30       |
| **Build**           | Vite                    | 6.0        |
| **Déploiement**     | Docker Compose          | 3 services |

---

## Démarrage rapide

### Prérequis

- Docker & Docker Compose

### Lancer le projet

```bash
docker compose up --build
```

| Service        | URL                          | Description                        |
| -------------- | ---------------------------- | ---------------------------------- |
| **Frontend**   | http://localhost:3000        | SPA Vue 3                          |
| **API**        | http://localhost:8080/api/v1 | API REST                           |
| **PostgreSQL** | localhost:5432               | BDD (user: `colec`, pass: `colec`) |

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
export JWT_SECRET="dev-secret"
go run .
```

**Frontend** :

```bash
cd web
npm install
npm run dev
```

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
│   ├── schema.sql          # Schéma PostgreSQL (tables, index, triggers)
│   ├── seed.sql            # Données de test (admin, types, items, relations)
│   └── queries_test.sql    # Requêtes SQL de test
├── server/                 # Backend Go
│   ├── main.go
│   ├── Dockerfile
│   ├── config/             # Configuration (env vars)
│   └── internal/
│       ├── handler/        # Handlers HTTP (1 fichier par domaine)
│       ├── middleware/      # Auth JWT, RequireAdmin
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
| Auth            | `POST /auth/login`, `GET /auth/me`              | Public / JWT                      |
| ItemTypes       | CRUD `/item-types`                              | Admin                             |
| RelationTypes   | CRUD `/relation-types`                          | Admin                             |
| CollectionTypes | CRUD `/collection-types`                        | Admin                             |
| Items           | CRUD `/items` + recherche, filtres, tri         | Admin (écriture)                  |
| Relations       | CRUD `/relations` + `GET /items/{id}/relations` | Admin (écriture)                  |
| Graph           | `GET /items/{id}/graph`                         | JWT                               |
| Collections     | CRUD `/collections` + gestion items             | JWT (owner)                       |
| Tags            | `GET /tags`                                     | JWT                               |
| Stats           | `GET /stats`                                    | JWT                               |
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
