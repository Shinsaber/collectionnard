# Collectionnard

## Le document

Ce document est la **base documentaire de référence** du projet Collectionnard.
Il compile la vision, les décisions de conception et le modèle de données.
Il sera maintenu à jour au fil des échanges et itérations.

> **Voir aussi** :
>
> - [README.md](README.md) — Démarrage rapide, stack, structure du projet
> - [api/openapi.yaml](api/openapi.yaml) — Spécification complète de l'API REST
> - [db/migrations/001_init.sql](db/migrations/001_init.sql) — Schéma de base de données commenté (migrations tern)

---

## 1. Vue d'ensemble

### Description

Collectionnard est une application de gestion de collections générique. Elle a pour but de rassembler et faciliter la gestion de collections de tous types (livres, cartes à collectionner, figurines, vinyles, DVD, séries TV, etc.).

Le **concept clé** est le dynamisme : le système doit permettre de créer des types de collections, d'items et de relations de manière entièrement dynamique, sans modification de code.

### Architecture

- **API-first** : backend exposant une API REST, les interfaces (web, mobile, CLI) s'y connectent
- **Multi-utilisateur** dès le MVP avec authentification
- **Self-hosted** (pour usage personnel et entre amis) V0
- **Stack envisagée** : Go (backend), PostgreSQL (BDD), Docker (déploiement test), frontend web à définir

### Objectifs Principaux

- **Flexibilité** : supporter différents types de collections avec leurs attributs spécifiques
- **Extensibilité** : permettre l'ajout de nouveaux types de collections sans modification du code
- **Dynamisme** : les types, attributs et relations sont tous configurables par les administrateurs
- **Graphe de relations** : les items forment un graphe libre (pas d'arborescence stricte), un même item peut être atteint par plusieurs chemins
- **Partage** : faciliter le partage et la découverte de collections entre utilisateurs (post-MVP)
- **Recherche** : offrir des capacités de recherche et de filtrage avancées
- **Visualisation** : présenter les collections et leurs graphes de manière attractive

---

## 2. Modèle conceptuel

### Principe fondamental : les attributs dynamiques

Le système repose sur **un pattern unique appliqué à 3 niveaux**. Chaque niveau a un "méta-type" qui définit un schéma d'attributs dynamiques, et des instances qui portent les valeurs.

| Méta-type          | Définit le schéma de…                                      | Instance           | Exemple de valeurs dynamiques                |
| ------------------ | ---------------------------------------------------------- | ------------------ | -------------------------------------------- |
| **ItemType**       | Attributs d'un item                                        | **Item**           | titre, année, ISBN…                          |
| **RelationType**   | Attributs d'un lien inter-items + contrainte source→target | **ItemRelation**   | disque, piste, rôle…                         |
| **CollectionType** | Attributs du lien user↔item + sémantique de la collection  | **CollectionItem** | état physique, prix d'achat, date vue, note… |

### Schéma d'architecture

```
                    ┌──────────────────────────────┐
                    │  attributes_schema (JSONB)   │
                    │  = JSON Schema draft 2020-12 │
                    │  + extensions x-* (voir §3)  │
                    └──────────┬───────────────────┘
                               │ porté par chaque méta-type
              ┌────────────────┼────────────────┐
              ▼                ▼                ▼
        ┌──────────┐    ┌────────────────┐ ┌────────────────┐
        │ ItemType │    │ RelationType   │ │ CollectionType │
        │          │    │ source_types[] │ │                │
        │          │    │ target_types[] │ │                │
        └────┬─────┘    └──────┬─────────┘ └───────┬────────┘
             │                 │                  │
     définit le schéma  définit le schéma   définit le schéma
             │                 │                  │
             ▼                 ▼                  ▼
        ┌──────────┐    ┌──────────────┐   ┌────────────────┐
        │   Item   │    │ ItemRelation │   │ CollectionItem │
        │ (values) │◀───│   (values)   │──▶│    (values)    │
        └──────────┘    └──────────────┘   └───────┬────────┘
                                                   │
                                            ┌──────▼───────┐
                                            │  Collection  │
                                  ┌────────▶│   (name)     │
                                  │         └──────────────┘
                             ┌────┴───┐
                             │  User  │
                             └────────┘
```

### Entités du système

#### User

Utilisateur authentifié. Possède des collections.

#### Collection

Conteneur explicite. Associe un utilisateur à des items.
L'utilisateur ajoute manuellement des items à ses collections.

#### CollectionType

Définit la sémantique et les attributs dynamiques d'une collection.
Exemples : "Possession", "Visionnage", "Wishlist".
Les attributs dynamiques s'appliquent au lien `CollectionItem`.

#### Item

Nœud unique dans le graphe. Représente n'importe quel objet : une série, un épisode, un DVD, un auteur, un studio, etc.
Un item est unique dans le système — plusieurs utilisateurs peuvent le lier à leurs collections respectives.
Les données spécifiques à un utilisateur (état, prix d'achat, note…) sont portées par le lien `CollectionItem`, pas par l'item lui-même.

#### ItemType

Définit le schéma d'attributs dynamiques d'un item.
Exemples : "Série", "Épisode", "Livre", "Personne", "Studio".

> **Note** : Les Personnes et Entreprises (auteurs, éditeurs, studios…) sont gérées comme des ItemTypes standards. Un auteur est un Item de type "Personne" relié à des livres par une relation "écrit_par". Cela évite des tables spécifiques et permet des liaisons transversales (un auteur lié à des livres ET des BD).

#### RelationType

Définit les liens possibles entre items, avec :

- **Contraintes source→target** : **listes** d'ItemTypes autorisés de chaque côté (`source_item_type_ids`,
  `target_item_type_ids`, tables de jonction `relation_type_sources` / `relation_type_targets`). Ex. `diffuse_en`
  ne lie que Serie→Diffusion ; `contains_work` accepte Univers ou Saga → Manga ou BD. Permet de proposer les
  types pertinents lors de la création de liens.
- **Attributs dynamiques** : données portées par le lien (ex: disque, piste, rôle…)

#### ItemRelation

Instance d'un lien entre deux items, avec les valeurs des attributs dynamiques définis par le RelationType.

#### CollectionItem

Instance du lien entre une collection et un item, avec les valeurs des attributs dynamiques définis par le CollectionType (état, prix, note…).

#### Tag

Étiquettes libres associables aux items pour faciliter la recherche.

---

## 3. Attributs dynamiques

### Un JSON Schema par méta-type

Chaque méta-type (ItemType, RelationType, CollectionType) porte un **JSON Schema** (draft 2020-12) dans sa
colonne `attributes_schema` (JSONB). Les instances (Item, ItemRelation, CollectionItem) portent un objet
`attributes` (JSONB) validé contre ce schéma à chaque écriture, côté serveur
(`santhosh-tekuri/jsonschema`). Le schéma lui-même est vérifié à l'enregistrement du méta-type : objet JSON
de racine `"type": "object"`, compilable, sans `$ref` externe.

Les mots-clés standards couvrent les besoins du catalogue : `type` (string, integer, number, boolean, object,
array), `required`, `enum`, `format` (`date`, `uri`…), `minimum` / `maximum`, `minLength`, `pattern`,
`items`, `description`, `default`. L'« `AttributeDefinition` » des premières versions de ce document était
une abstraction de ce schéma ; elle n'existe pas dans l'implémentation.

### Extensions `x-*` (pilotent le front)

| Extension            | Niveau    | Rôle                                                                                                                                                                                                     |
| -------------------- | --------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `x-order`            | racine    | Ordre d'affichage des champs (tableau de clés ; les clés absentes suivent)                                                                                                                               |
| `x-labels`           | racine    | Libellés par langue : `{ "fr": { "title": "Titre" }, "en": { "title": "Title" } }`                                                                                                                       |
| `x-title`            | racine    | Nom affiché du méta-type par langue : `{ "fr": "Série", "en": "Series" }` ; repli sur la colonne `name` (identifiant technique anglais) |
| `x-description`      | racine    | Description du méta-type par langue ; repli sur la colonne `description` |
| `x-enum-labels`      | propriété | Libellés des valeurs d'un `enum` par langue : `{ "fr": { "tv": "Télévision" }, "en": { "tv": "TV" } }` ; repli sur la valeur brute |
| `x-widget`           | propriété | Widget dédié : `i18n-text` (objet `{fr, en, …}` : une chaîne par langue, codes supplémentaires acceptés), `image-gallery` (objet de catégories)                                                                                                                     |
| `x-image-categories` | propriété | Catégories d'images et ratios : `{ "poster": { "ratio": "2/3" } }` (avec `image-gallery`)                                                                                                                |
| `x-label`            | racine    | Gabarit du libellé des instances : `"{title}"`, `"{number}. {title}"`, `"{last_name} {first_name}"` ; sans gabarit, repli sur `title`, `name`, `label` (Go `internal/label`, TS `web/src/lib/schema.ts`) |

Règles d'affichage dérivées du schéma lui-même (pas d'extension) : un `integer` / `number` avec `minimum`
**et** `maximum` est rendu par un curseur ; un `enum` par une liste déroulante ; `format: date` / `uri` par
le champ adapté. Toute nouvelle extension doit être gérée par `DynamicForm.vue` **et** `SchemaBuilder.vue`,
et documentée ici et dans le README.

### Langues (i18n)

L'interface (vue-i18n) et les données suivent la même langue, choisie dans l'en-tête (mémorisée, sinon
langue du navigateur, sinon `fr`). Toute valeur multilingue (objet `i18n-text`, `x-title`, `x-labels`,
`x-enum-labels`) se lit avec le même repli : langue demandée → `fr` → autres langues connues → toute autre
clé (ordre alphabétique) — `i18nText` côté TS, `label.Stringify` côté Go. L'API renvoie des codes d'erreur
stables (`error`) traduits par le front ; son `message` est un texte technique en anglais.

Ajouter une langue (ex. `de`) : l'ajouter à `LANGS` et `LANG_NAMES` (`web/src/i18n/langs.ts`) et à `Langs`
(`server/internal/label/label.go`) ; créer `web/src/i18n/locales/de/` (même forme que `fr`, imposée par le
compilateur et vérifiée par `i18n.test.ts`) ; brancher catalogue et locale Element Plus dans
`web/src/i18n/index.ts`. Les schémas existants n'ont pas à migrer : les objets `i18n-text` acceptent déjà
d'autres codes de langue, et les traductions de types / champs / enums se complètent dans le SchemaBuilder.

---

## 4. Exemples concrets

### Série TV — Cas de référence MVP

**ItemTypes :**

| ItemType  | Attributs (clés JSON)                                | `x-label`           |
| --------- | ---------------------------------------------------- | ------------------- |
| Series    | title, synopsis (i18n), start_year, end_year, images | `{title}`           |
| Broadcast | name, type (tv/streaming/cable/web), channel         | `{name}`            |
| Season    | name, number, year                                   | `{name}`            |
| Episode   | number, title, duration, synopsis (i18n)             | `{number}. {title}` |
| DVD       | title, publisher, release_year, disc_count           | `{title}`           |

**RelationTypes :**

| RelationType     | Source → Target    | Attributs du lien         |
| ---------------- | ------------------ | ------------------------- |
| broadcast_on     | Series → Broadcast | broadcast_start_year      |
| contains_season  | Broadcast → Season | —                         |
| contains_episode | Season → Episode   | order (tri des relations) |
| released_as      | Series → DVD       | —                         |
| includes_episode | DVD → Episode      | disc, track               |

**CollectionTypes :**

| CollectionType | Attributs du lien CollectionItem                                           |
| -------------- | -------------------------------------------------------------------------- |
| Ownership      | condition (new/very_good/good/fair/poor), purchase_price, acquisition_date |
| Watching       | watched_at, rating, progress                                               |
| Wishlist       | priority, max_budget                                                       |

**Graphe de données :**

```
Breaking Bad (Series)
├──broadcast_on──▶ "AMC 2008-2013" (Broadcast)
│                 ├──contains_season──▶ Saison 1
│                 │                     ├──contains_episode──▶ Ep1 "Pilot"
│                 │                     └──contains_episode──▶ Ep2 "Cat's in the Bag"
│                 └──contains_season──▶ Saison 2 ──...
└──released_as──▶ "Coffret DVD Intégrale" (DVD)
                  ├──includes_episode {disc:1, track:1}──▶ Ep1 "Pilot"     ← même item !
                  └──includes_episode {disc:1, track:2}──▶ Ep2 "Cat's..."  ← même item !
```

Les épisodes sont les **mêmes nœuds**, accessibles par deux chemins. C'est la force du graphe.

**Collections utilisateur :**

```
User "shinsaber" :
├── Collection "Mes DVDs" (type: Ownership)
│     └── Coffret DVD Intégrale {condition: "new", purchase_price: 45, acquisition_date: 2020-12-25}
└── Collection "Séries vues" (type: Watching)
      └── Breaking Bad {watched_at: 2023-03-15, rating: 10, progress: 100}
```

### Autres cas d'usage (post-MVP)

- **Livres/BD** : gestion des versions, reprints, auteurs comme Items de type "Personne"
- **Cartes à collectionner** : sets, raretés, conditions, valeur estimée
- **Vinyles** : artiste, label, pressages
- **Sagas inter-média** : une Saga comme Item reliant livres, films, séries

---

## 5. Permissions et rôles

| Rôle             | Droits                                                                                                                  |
| ---------------- | ----------------------------------------------------------------------------------------------------------------------- |
| **Admin**        | Crée/modifie les ItemTypes, RelationTypes, CollectionTypes. Crée/modifie les Items et relations. Gère les utilisateurs. |
| **User**         | Crée ses collections, y ajoute des items existants. Consulte le catalogue.                                              |
| **User (futur)** | Soumet de nouveaux items ou types, validés par un admin.                                                                |

---

## 6. Performance et optimisation

La table unique d'items avec attributs dynamiques nécessite une attention particulière :

- **Stockage JSONB** (PostgreSQL) pour les attributs dynamiques : performant en lecture, indexable
- **Index GIN** sur les colonnes JSONB `attributes` (items, relations, collection items) — exploité par la
  recherche avancée (Phase 2, opérateur `@>`)
- **Index sur les relations** (source, cible, type) pour les traversées de graphe, qui se font en parcours en
  largeur borné (`GRAPH_MAX_NODES`, profondeur max, timeout)
- `statement_timeout` sur le pool de connexions
- _Piste_ : vues matérialisées par ItemType si des requêtes fréquentes l'exigent (aucune aujourd'hui)

---

## 7. Fonctionnalités détaillées

### Gestion de Collections

- Créer et organiser plusieurs collections avec des types différents
- Ajouter/retirer des items avec les attributs contextuels (état, prix, note…)
- Un item peut appartenir à plusieurs collections du même ou de différents utilisateurs
- Import/Export de données (CSV, JSON)

### Recherche et Statistiques

- Recherche multi-critères sur les attributs dynamiques
- Filtres avancés par type, tag, relation
- Statistiques sur les collections (valeur totale, tendances, complétude…)
- Graphiques et visualisations

### Fonctionnalités Sociales (post-MVP)

- Profils utilisateurs publics/privés
- Partage de collections
- Listes de souhaits
- Système d'échange entre collectionneurs

### Sources de données externes (long terme)

- Intégration d'APIs tierces : TMDB (films/séries), OpenLibrary (livres), Scryfall (Magic), TCGdex (Pokémon)…
- Scan de codes-barres / ISBN avec auto-complétion (app mobile)

---

## 8. Phases de développement

### Phase 1 : MVP — Valider le dynamisme

- [x] Schéma de BDD avec attributs dynamiques (PostgreSQL 16, JSONB, index GIN)
- [x] Tests des requêtes BDD : `db/queries_test.sql` (manuel) puis tests d'intégration Go sur PostgreSQL
      (`go test -tags integration`, Phase 1.5)
- [x] API REST (CRUD sur tous les objets) — spécification OpenAPI 3.1 dans `api/openapi.yaml`
- [x] Authentification utilisateur (admin/user) — JWT Bearer
- [x] Jeu de données de test : Séries TV (multi-chemin Série→Diffusion→Saison→Épisode + Série→DVD→Épisode)
- [x] Interface web de visualisation (SPA Vue 3 + PrimeVue)
- [x] Upload d'images (galerie, catégories avec ratios, onglets)
- [x] Visualisation de graphe (Cytoscape.js, traversée récursive avec profondeur configurable)
- [x] Gestion des collections utilisateur (CRUD, ajout/retrait d'items, attributs contextuels)
- [x] Gestion des relations (CRUD, arbre groupé par type, direction source/target)
- [x] Formulaires dynamiques générés depuis JSON Schema (DynamicForm)
- [x] Éditeur de JSON Schema visuel (SchemaBuilder) avec `x-order`, `x-labels`, `x-widget`, `x-image-categories`
- [x] Recherche et filtres (par type, texte libre `ILIKE` sur les attributs, tags, tri `created_at` / `updated_at`)
- [x] Statistiques (dashboard avec compteurs par type)
- [x] Tags libres sur les items
- [x] Déploiement Docker Compose (4 services : postgres, migrate, server, web)

### Phase 1.5 : Remise à niveau (audit du 2026-10-11, `docs/audit/`)

Lots de `docs/audit/03-feuille-de-route.md`, branche `remise-a-niveau` :

- [x] Lot 0 — sécurité : JWT courts + refresh tokens à rotation, secrets obligatoires, limites de corps,
      rate limit `/auth/*`, 422 sur PUT sans `attributes`
- [x] Lot 1 — outillage : flake suivi, gofmt/vet/golangci-lint, ESLint/Prettier/Vitest, CI GitHub Actions
- [x] Lot 2 — migrations tern (`db/migrations`), `/healthz`, conteneurs non-root, nginx durci, logs `slog`
- [x] Lot 3 — Go 1.27, pgx 5.11, chi 5.3, zéro vulnérabilité `govulncheck`
- [x] Lot 4 — front : axios, vue, cytoscape, vite, `/themes` ; `npm audit --omit=dev` propre
- [x] Lot 5 — API : graphe borné (BFS, `GRAPH_MAX_NODES`, `truncated`), schémas validés à l'écriture,
      `per_page` borné, UUID/tri invalides → 400, jokers échappés, `statement_timeout`
- [x] Lot 6 — tests : 100 % des handlers, intégration PostgreSQL, Vitest (DynamicForm, SchemaBuilder, LoginView)
- [x] Lot 7 — cohérence : spec OpenAPI = code (test routes ↔ spec, `redocly lint` en CI), docs à jour
- [x] Lot 8 — extension `x-label` (Go + TS), seed TV en anglais avec `x-order`, langue d'affichage fr/en, requêtes groupées (tags, jonctions)
- [x] Lot 9 — découpage d'`ItemDetailView` / `SchemaBuilder` / `DynamicForm` / `GraphView` (< 400 lignes), `notifyError` partout, accessibilité (aria-label, alt, label/for, clavier), cache des méta-types, `baseUrl` retiré
- [x] Lot 10 — Vite 8, TypeScript 6, vue-tsc 3, Pinia 4, vue-router 5 ; PrimeVue 5 d'abord, puis **remplacé par Element Plus** (lot 10 ter) : PrimeVue 5 et primeicons 8 sont passés sous la « PrimeUI License » (clé requise, usage gratuit sous conditions), incompatible avec un projet open source — primeicons reste en 7.0 (MIT)
- [x] Lot 11 — Tailwind 4 (@tailwindcss/vite, config en CSS, outil de migration officiel, `npm audit` complet à zéro)

### Phase 2 : Enrichissement

- [x] Couverture multilingue : interface entière traduite (vue-i18n, fr/en, langues ajoutables via un registre), locale Element Plus, noms de types (`x-title` / `x-description`), valeurs d'enum (`x-enum-labels`), erreurs API traduites par code (messages serveur en anglais), langue du navigateur détectée
- [ ] Recherche avancée : filtres `attributes[champ]` (index GIN `@>`), tri par attribut (lot 14)
- [ ] Statistiques et visualisation graphiques
- [ ] Import/Export CSV/JSON (lot 12)
- [ ] Domaine BD / Manga : seed `les.collection.md`, rôles multiples (lot 13)
- [ ] Soumission d'items par les users (validation admin)

### Phase 3 : Social

- [ ] Profils publics/privés
- [ ] Partage de collections
- [ ] Système d'échange

### Phase 4 : Intégrations

- [ ] APIs externes (TMDB, OpenLibrary…)
- [ ] App mobile avec scan de codes-barres

---

## 9. Stack technique

| Composant       | Choix                                                                                   | Version              |
| --------------- | --------------------------------------------------------------------------------------- | -------------------- |
| Backend         | Go (chi 5.3, pgx 5.11, JWT 5.3, jsonschema v5 — v6 différé, API incompatible, sans CVE) | 1.27 (go.mod ≥ 1.26) |
| Base de données | PostgreSQL                                                                              | 16                   |
| Frontend        | Vue 3 + TypeScript                                                                      | 3.5 / 6.0            |
| UI              | PrimeVue (@primeuix/themes, aura-compat) + Tailwind CSS                                 | 5.0 / 3.4            |
| Graphes         | Cytoscape.js                                                                            | 3.34                 |
| Build           | Vite (Rolldown), vue-tsc 3, Pinia 4, vue-router 5                                       | 8.3                  |
| Déploiement     | Docker Compose (4 services : postgres, migrate, server, web)                            | —                    |
| Dev             | Nix flake (Go, Node, PostgreSQL, golangci-lint) + direnv                                | flake.lock           |

---

## Contribuer

Ce projet est en phase de conception. Vos retours et suggestions sont les bienvenus !

---

**Dernière mise à jour** : 22/04/2026
