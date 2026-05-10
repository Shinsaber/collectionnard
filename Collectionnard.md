# Collectionnard

## Le document

Ce document est la **base documentaire de référence** du projet Collectionnard.
Il compile la vision, les décisions de conception et le modèle de données.
Il sera maintenu à jour au fil des échanges et itérations.

> **Voir aussi** :
>
> - [README.md](README.md) — Démarrage rapide, stack, structure du projet
> - [api/openapi.yaml](api/openapi.yaml) — Spécification complète de l'API REST
> - [db/schema.sql](db/schema.sql) — Schéma de base de données commenté

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
                    │    AttributeDefinition       │
                    │ (name, field_type, required, │
                    │  default, validation_rules)  │
                    └──────────┬───────────────────┘
                               │ utilisé par
              ┌────────────────┼────────────────┐
              ▼                ▼                ▼
        ┌──────────┐    ┌──────────────┐  ┌────────────────┐
        │ ItemType │    │ RelationType │  │ CollectionType │
        │          │    │ source_type  │  │                │
        │          │    │ target_type  │  │                │
        └────┬─────┘    └──────┬───────┘  └───────┬────────┘
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

- **Contraintes source→target** : quels ItemTypes peuvent être liés (ex: `diffusé_en` ne lie que Série→Diffusion). Permet de proposer les types pertinents lors de la création de liens.
- **Attributs dynamiques** : données portées par le lien (ex: disque, piste, rôle…)

#### ItemRelation

Instance d'un lien entre deux items, avec les valeurs des attributs dynamiques définis par le RelationType.

#### CollectionItem

Instance du lien entre une collection et un item, avec les valeurs des attributs dynamiques définis par le CollectionType (état, prix, note…).

#### Tag

Étiquettes libres associables aux items pour faciliter la recherche.

---

## 3. Attributs dynamiques

### Définition des champs

Chaque méta-type (ItemType, RelationType, CollectionType) définit une liste d'`AttributeDefinition` :

| Propriété          | Description                                                                                 |
| ------------------ | ------------------------------------------------------------------------------------------- |
| `name`             | Nom du champ                                                                                |
| `field_type`       | Type du champ (texte, nombre, date, booléen, liste déroulante, image, URL, référence item…) |
| `required`         | Champ obligatoire ou non                                                                    |
| `default`          | Valeur par défaut                                                                           |
| `validation_rules` | Règles de validation (format ISBN, prix > 0, regex…)                                        |

Le système de types de champs est conçu pour être extensible : de nouveaux types de champs pourront être ajoutés au fil du temps.

---

## 4. Exemples concrets

### Série TV — Cas de référence MVP

**ItemTypes :**

| ItemType  | Attributs                                       |
| --------- | ----------------------------------------------- |
| Série     | titre, synopsis, année_début, année_fin, poster |
| Diffusion | nom, type (TV/streaming/…), chaîne              |
| Saison    | numéro, année                                   |
| Épisode   | numéro, titre, durée, synopsis                  |
| DVD       | titre, éditeur, année_sortie, nb_disques        |

**RelationTypes :**

| RelationType     | Source → Target    | Attributs du lien |
| ---------------- | ------------------ | ----------------- |
| diffusé_en       | Série → Diffusion  | année_début_diff  |
| contient_saison  | Diffusion → Saison | —                 |
| contient_épisode | Saison → Épisode   | ordre             |
| édité_en         | Série → DVD        | —                 |
| inclut_épisode   | DVD → Épisode      | disque, piste     |

**CollectionTypes :**

| CollectionType | Attributs du lien CollectionItem            |
| -------------- | ------------------------------------------- |
| Possession     | état_physique, prix_achat, date_acquisition |
| Visionnage     | date_vue, note, progression                 |
| Wishlist       | priorité, budget_max                        |

**Graphe de données :**

```
Breaking Bad (Série)
├──diffusé_en──▶ "AMC 2008-2013" (Diffusion)
│                 ├──contient_saison──▶ Saison 1
│                 │                     ├──contient_épisode──▶ Ep1 "Pilot"
│                 │                     └──contient_épisode──▶ Ep2 "Cat's in the Bag"
│                 └──contient_saison──▶ Saison 2 ──...
└──édité_en────▶ "Coffret DVD Intégrale" (DVD)
                  ├──inclut_épisode {disque:1, piste:1}──▶ Ep1 "Pilot"     ← même item !
                  └──inclut_épisode {disque:1, piste:2}──▶ Ep2 "Cat's..."  ← même item !
```

Les épisodes sont les **mêmes nœuds**, accessibles par deux chemins. C'est la force du graphe.

**Collections utilisateur :**

```
User "shinsaber" :
├── Collection "Mes DVDs" (type: Possession)
│     └── Coffret DVD Intégrale {état: "neuf", prix: 45€, date: 2020-12-25}
└── Collection "Séries vues" (type: Visionnage)
      └── Breaking Bad {date_vue: 2023-03, note: 10, progression: 100%}
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
- **Vues matérialisées** par ItemType pour les requêtes fréquentes
- **Index GIN** sur les champs JSONB les plus filtrés (titre, année…)
- **Index sur les relations** pour les traversées de graphe

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
- [x] Tests des requêtes BDD (CRUD items, relations, collections, traversée de graphe)
- [x] API REST (CRUD sur tous les objets) — spécification OpenAPI 3.1 dans `api/openapi.yaml`
- [x] Authentification utilisateur (admin/user) — JWT Bearer
- [x] Jeu de données de test : Séries TV (multi-chemin Série→Diffusion→Saison→Épisode + Série→DVD→Épisode)
- [x] Interface web de visualisation (SPA Vue 3 + PrimeVue)
- [x] Upload d'images (galerie, catégories avec ratios, onglets)
- [x] Visualisation de graphe (Cytoscape.js, traversée récursive avec profondeur configurable)
- [x] Gestion des collections utilisateur (CRUD, ajout/retrait d'items, attributs contextuels)
- [x] Gestion des relations (CRUD, arbre groupé par type, direction source/target)
- [x] Formulaires dynamiques générés depuis JSON Schema (DynamicForm)
- [x] Éditeur de JSON Schema visuel (SchemaBuilder) avec `x-order`, `x-labels`, `x-widget`
- [x] Recherche et filtres (par type, texte libre, tags, tri)
- [x] Statistiques (dashboard avec compteurs par type)
- [x] Tags libres sur les items
- [x] Déploiement Docker Compose (3 services : postgres, server, web)

### Phase 2 : Enrichissement

- [ ] Recherche avancée et filtres
- [ ] Statistiques et graphiques
- [ ] Import/Export CSV/JSON
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

| Composant       | Choix                           | Version   |
| --------------- | ------------------------------- | --------- |
| Backend         | Go (chi, pgx, JWT, JSON Schema) | 1.22      |
| Base de données | PostgreSQL                      | 16        |
| Frontend        | Vue 3 + TypeScript              | 3.5 / 5.6 |
| UI              | PrimeVue + Tailwind CSS         | 4.3 / 3.4 |
| Graphes         | Cytoscape.js                    | 3.30      |
| Build           | Vite                            | 6.0       |
| Déploiement     | Docker Compose (3 services)     | —         |
| Packaging       | NixOS flake possible            | À voir    |

---

## Contribuer

Ce projet est en phase de conception. Vos retours et suggestions sont les bienvenus !

---

**Dernière mise à jour** : 22/04/2026
