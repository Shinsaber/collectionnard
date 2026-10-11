# Collections BD / Manga — Modèle de données

## Décisions de conception

- **Univers** : optionnel. Une Œuvre peut exister sans Univers parent. Utile long terme pour regrouper les médias (jeux vidéo, films, etc.)
- **Types dédiés** : Manga et BD sont des ItemTypes séparés (pas un type unique avec enum)
- **Personnes** : gérées comme Items de type Personne, liées par relation `created_by` (source = Manga / BD,
  cible = Personne, lecture « créé par ») avec attribut `roles` : **tableau** d'enum, une même personne pouvant
  cumuler plusieurs rôles sur une œuvre (ex. scénariste et dessinateur) sans violer l'unicité
  `(relation_type_id, source_item_id, target_item_id)` d'`item_relations`
- **Saga** : ItemType dédié (arcs narratifs)
- **CollectionType "Lecture"** : à définir plus tard
- ✅ **Traité** : `relation_types` accepte plusieurs types source et cible (tables de jonction
  `relation_type_sources` / `relation_type_targets`, champs `source_item_type_ids` / `target_item_type_ids`)
- **Convention** : clés JSON, noms de RelationTypes et valeurs d'enum en anglais snake_case ; les libellés
  affichés viennent de `x-labels` (fr/en) et le libellé d'un item de `x-label` ; le seed TV suit déjà cette convention

---

## Graphe

```
Univers (optionnel)
├── contains_saga ──▶ Saga
│                      └── contains_work ──▶ Manga / BD
│                                              ├── created_by {roles} ──▶ Personne
│                                              ├── published_as ──▶ Édition
│                                              │                  ├── edited_by ──▶ Maison d'édition
│                                              │                  └── contains_volume ──▶ Tome
│                                              └── adapted_as ──▶ Série (anime) ← modèle TV existant
├── contains_work ──▶ Manga / BD  (directement, sans saga)
└── adapted_as ──▶ Série / Jeu / Film (long terme)
```

---

## ItemTypes

### Univers

Regroupe plusieurs Œuvres/Sagas d'une même franchise. Optionnel.

| Attribut      | Type          | Requis | Description                              |
| ------------- | ------------- | ------ | ---------------------------------------- |
| `title`       | i18n-text     | ✅     | Nom de l'univers (peut être régionalisé) |
| `description` | i18n-text     |        | Description de l'univers                 |
| `images`      | image-gallery |        | poster, banner, logo                     |

### Saga

Arc narratif ou sous-ensemble d'un Univers (ex: "Saga des Saiyans", "Arc Impel Down").

| Attribut      | Type          | Requis | Description            |
| ------------- | ------------- | ------ | ---------------------- |
| `title`       | i18n-text     | ✅     | Nom de la saga         |
| `description` | i18n-text     |        | Description de la saga |
| `order`       | integer       |        | Ordre dans l'univers   |
| `images`      | image-gallery |        | illustration           |

### Manga

Œuvre de type manga (+ manhwa, manhua, webtoon).

| Attribut         | Type          | Requis | Description                                    |
| ---------------- | ------------- | ------ | ---------------------------------------------- |
| `title`          | i18n-text     | ✅     | Titre (régionalisable)                         |
| `original_title` | string        |        | Titre en VO (鳥山明 ドラゴンボール)            |
| `description`    | i18n-text     |        | Synopsis                                       |
| `sub_type`       | enum          |        | `manga`, `manhwa`, `manhua`, `webtoon`         |
| `demographic`    | enum          |        | `shōnen`, `shōjo`, `seinen`, `josei`, `kodomo` |
| `status`         | enum          |        | `ongoing`, `completed`, `on_hold`, `cancelled` |
| `start_year`     | integer       |        | Année de première publication                  |
| `end_year`       | integer       |        | Année de fin                                   |
| `volumes_vo`     | integer       |        | Nombre total de tomes en VO                    |
| `images`         | image-gallery |        | poster, couverture, banner                     |

### BD

Œuvre de type bande dessinée (franco-belge, comics, roman graphique).

| Attribut         | Type          | Requis | Description                                       |
| ---------------- | ------------- | ------ | ------------------------------------------------- |
| `title`          | i18n-text     | ✅     | Titre (régionalisable)                            |
| `original_title` | string        |        | Titre en VO                                       |
| `description`    | i18n-text     |        | Synopsis                                          |
| `sub_type`       | enum          |        | `franco_belge`, `comic`, `graphic_novel`, `strip` |
| `status`         | enum          |        | `ongoing`, `completed`, `on_hold`, `cancelled`    |
| `start_year`     | integer       |        | Année de première publication                     |
| `end_year`       | integer       |        | Année de fin                                      |
| `volumes_vo`     | integer       |        | Nombre total de tomes/albums en VO                |
| `images`         | image-gallery |        | poster, couverture, banner                        |

### Édition

Publication spécifique d'une Œuvre (édition originale, perfect edition, poche, collector…).

| Attribut       | Type          | Requis | Description                                                                               |
| -------------- | ------------- | ------ | ----------------------------------------------------------------------------------------- |
| `title`        | i18n-text     | ✅     | Ex: "Dragon Ball - Perfect Edition"                                                       |
| `language`     | enum          | ✅     | `fr`, `ja`, `en`, `ko`, `zh`, `es`, `de`, `it`                                            |
| `format`       | enum          |        | `tankōbon`, `grand_format`, `poche`, `perfect`, `deluxe`, `collector`, `omnibus`, `album` |
| `status`       | enum          |        | `ongoing`, `completed`, `on_hold`, `out_of_print`                                         |
| `volume_count` | integer       |        | Nombre de tomes dans cette édition                                                        |
| `start_year`   | integer       |        | Année de début de publication                                                             |
| `end_year`     | integer       |        | Année de fin de publication                                                               |
| `images`       | image-gallery |        | couverture type, visuels promo                                                            |

### Tome

Volume physique individuel. Item le plus souvent mis en collection.

| Attribut           | Type          | Requis | Description                      |
| ------------------ | ------------- | ------ | -------------------------------- |
| `number`           | integer       | ✅     | Numéro dans l'édition            |
| `title`            | i18n-text     | ✅     | Titre du tome                    |
| `subtitle`         | string        |        | Sous-titre éventuel              |
| `isbn`             | string        |        | ISBN-13                          |
| `publication_date` | date          |        | Date de publication              |
| `page_count`       | integer       |        | Nombre de pages                  |
| `retail_price`     | number        |        | Prix catalogue (€)               |
| `images`           | image-gallery |        | couverture recto, verso, tranche |

### Personne

Créateur : auteur, scénariste, dessinateur, coloriste, traducteur…

| Attribut      | Type          | Requis | Description                     |
| ------------- | ------------- | ------ | ------------------------------- |
| `last_name`   | string        | ✅     | Nom de famille                  |
| `first_name`  | string        |        | Prénom                          |
| `native_name` | string        |        | Nom en écriture native (鳥山明) |
| `nationality` | string        |        | Nationalité                     |
| `birth_date`  | date          |        | Date de naissance               |
| `biography`   | i18n-text     |        | Bio courte                      |
| `images`      | image-gallery |        | portrait                        |

### Maison d'édition

Éditeur / publisher (Glénat, Kana, Pika, Dargaud, Dupuis…).

| Attribut      | Type          | Requis | Description      |
| ------------- | ------------- | ------ | ---------------- |
| `name`        | string        | ✅     | Nom de l'éditeur |
| `country`     | string        |        | Pays d'origine   |
| `website`     | string        |        | URL du site      |
| `description` | i18n-text     |        | Description      |
| `images`      | image-gallery |        | logo             |

---

## RelationTypes

| Nom               | Source → Target             | Attributs du lien                                           |
| ----------------- | --------------------------- | ----------------------------------------------------------- |
| `contains_saga`   | Univers → Saga              | `order` (integer)                                           |
| `contains_work`   | Univers / Saga → Manga / BD | `order` (integer)                                           |
| `created_by`      | Manga / BD → Personne       | `roles` (array d'enum : author, writer, artist, colorist, translator ; `minItems: 1`) |
| `published_as`    | Manga / BD → Édition        | —                                                           |
| `edited_by`       | Édition → Maison d'édition  | —                                                           |
| `contains_volume` | Édition → Tome              | `order` (integer)                                           |
| `adapted_as`      | Manga / BD → Série _(TV)_   | `adaptation_type` (enum: anime, film, OAV, live_action)     |

> `contains_work` a plusieurs sources (Univers, Saga) et plusieurs cibles (Manga, BD) : supporté nativement
> par les tables de jonction de `relation_types`.

---

## CollectionTypes (existants réutilisables)

| CollectionType | Usage BD/Manga                                      | Items typiques  |
| -------------- | --------------------------------------------------- | --------------- |
| **Possession** | état_physique, prix_achat, date_acquisition         | Tome, Édition   |
| **Wishlist**   | priorité, budget_max                                | Tome, Édition   |
| **Visionnage** | date_vue, note, progression                         | Œuvre (lecture) |
| _Lecture ?_    | _tome_en_cours, progression, relecture — à définir_ | _Manga, BD_     |

---

## Exemple concret : Dragon Ball

```
"Dragon Ball" (Univers)
├── contains_saga {order:1} ──▶ "Saga de l'Enfance" (Saga)
│     └── contains_work ──▶ "Dragon Ball" (Manga, shōnen, 1984-1995, 42 tomes VO)
│           ├── created_by {roles: [author]} ──▶ Akira Toriyama (Personne)
│           ├── published_as ──▶ "Édition originale FR" (Édition, fr, tankōbon, 42 tomes)
│           │     ├── edited_by ──▶ Glénat (Maison d'édition)
│           │     ├── contains_volume {order:1} ──▶ Tome 1 "Son Gokū et ses amis"
│           │     ├── contains_volume {order:2} ──▶ Tome 2 ...
│           │     └── ... (42 tomes)
│           ├── published_as ──▶ "Perfect Edition FR" (Édition, fr, perfect, 34 tomes)
│           │     ├── edited_by ──▶ Glénat
│           │     └── ...
│           └── adapted_as {type: anime} ──▶ "Dragon Ball" (Série TV) ← modèle TV
│
├── contains_saga {order:2} ──▶ "Dragon Ball Super" (Saga)
│     └── contains_work ──▶ "Dragon Ball Super" (Manga, shōnen, 2015-)
│           ├── created_by {roles: [writer]} ──▶ Akira Toriyama ← MÊME personne
│           ├── created_by {roles: [artist]} ──▶ Toyotarō (Personne)
│           └── published_as ──▶ ...
```

**Collections utilisateur :**

```
User "shinsaber" :
├── Collection "Mes Mangas" (type: Possession)
│     ├── DB Perfect Tome 1 {état: "très bon", prix: 9.60€, date: 2021-03-10}
│     ├── DB Perfect Tome 2 {état: "neuf", prix: 9.60€, date: 2021-03-10}
│     └── ...
└── Collection "Wishlist Manga" (type: Wishlist)
      └── Dragon Ball Super Tome 20 {priorité: 3, budget_max: 7.50}
```
