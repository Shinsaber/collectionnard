# Collections livres (BD, manga, romans) — Modèle de données

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

## Implémentation (lot 13)

Types : paquet de domaine « books » ([`server/internal/domains/packages/books.json`](server/internal/domains/packages/books.json),
page « Domaines » ou `colec-server install-domain books`) ; données de démonstration : [`db/seed_bd_manga.sql`](db/seed_bd_manga.sql)
(chargé après `db/seed.sql`). Noms techniques des ItemTypes :
`Universe`, `Saga`, `Manga`, `Comic` (BD), `Edition`, `Volume` (Tome), `Person`, `Publisher` ; noms affichés via `x-title`.
Valeurs d'enum en ASCII (`shonen`, `tankobon`, `ova`…) affichées avec leurs accents via `x-enum-labels`.
Exemples réels : Dragon Ball (univers, sagas, deux mangas, Toriyama / Toyotarō, édition Glénat 1993, tomes 1-2,
anime) et Astérix (BD franco-belge sans saga ; Uderzo `roles: [writer, artist]` ; Dargaud puis Albert René).

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
| `demographic`    | enum          |        | `shonen`, `shojo`, `seinen`, `josei`, `kodomo` |
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
| `format`       | enum          |        | `tankobon`, `grand_format`, `poche`, `perfect`, `deluxe`, `collector`, `omnibus`, `album` |
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
| `isbn`             | string        |        | ISBN-13 (chiffres seuls)         |
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
| `nationality` | string        |        | Code pays ISO 3166-1 alpha-2    |
| `birth_date`  | date          |        | Date de naissance               |
| `biography`   | i18n-text     |        | Bio courte                      |
| `images`      | image-gallery |        | portrait                        |

### Maison d'édition

Éditeur / publisher (Glénat, Kana, Pika, Dargaud, Dupuis…).

| Attribut      | Type          | Requis | Description      |
| ------------- | ------------- | ------ | ---------------- |
| `name`        | string        | ✅     | Nom de l'éditeur |
| `country`     | string        |        | Code pays ISO    |
| `website`     | string        |        | URL du site      |
| `description` | i18n-text     |        | Description      |
| `images`      | image-gallery |        | logo             |

---

## RelationTypes

| Nom               | Source → Target             | Attributs du lien                                                                     |
| ----------------- | --------------------------- | ------------------------------------------------------------------------------------- |
| `contains_saga`   | Univers → Saga              | `order` (integer)                                                                     |
| `contains_work`   | Univers / Saga → Manga / BD | `order` (integer)                                                                     |
| `created_by`      | Manga / BD → Personne       | `roles` (array d'enum : author, writer, artist, colorist, translator ; `minItems: 1`) |
| `published_as`    | Manga / BD → Édition        | —                                                                                     |
| `edited_by`       | Édition → Maison d'édition  | —                                                                                     |
| `contains_volume` | Édition → Tome              | `order` (integer)                                                                     |
| `adapted_as`      | Manga / BD → Série _(TV)_   | `adaptation_type` (enum: anime, film, ova, live_action)                               |

> `contains_work` a plusieurs sources (Univers, Saga) et plusieurs cibles (Manga, BD) : supporté nativement
> par les tables de jonction de `relation_types`.

---

## Sources de référence : Wikidata et BnF

Décision (2026-10-11) : le domaine s'appuie sur deux bases ouvertes, à licence compatible avec la reprise des
données dans le catalogue. Le détail des licences, des accès et des sources écartées ou gardées pour plus
tard (GCD, Open Library, AniList…) est dans [`docs/sources-externes.md`](docs/sources-externes.md).

| Niveau du graphe                                   | Source       | Licence                                         |
| -------------------------------------------------- | ------------ | ----------------------------------------------- |
| Univers, Manga, BD, Personne, Maison d'édition     | **Wikidata** | CC0 (aucune obligation)                         |
| Édition, Tome (+ autorités Personne / Éditeur)     | **BnF**      | Licence Ouverte : citer la source et la date    |

Le schéma reste **le nôtre** : on ne calque pas le modèle d'une source. Chaque item garde simplement de quoi
retrouver sa fiche d'origine : c'est la clé de rapprochement du connecteur (Phase 3, ci-dessous).

### Attributs communs (tous les ItemTypes du domaine)

| Attribut       | Type   | Description                                                                                       |
| -------------- | ------ | ------------------------------------------------------------------------------------------------- |
| `external_ids` | object | `{ "wikidata": "Q…", "bnf_ark": "ark:/12148/…" }` — chaque clé optionnelle, `additionalProperties: false` |
| `sources`      | array  | Provenance : `[{ "source": "bnf", "retrieved_at": "2026-10-11" }]` (`source` : enum `wikidata`, `bnf`, `manual`) — obligatoire pour la BnF (attribution) |

- Motifs de validation : `wikidata` → `^Q[1-9][0-9]*$` ; `bnf_ark` → `^ark:/12148/[a-z0-9]+$`.
- Les identifiants d'autres bases (MangaUpdates `P11149`, MyAnimeList `P4087`, AniList `P8731`, Comic Vine
  `P5905`) ne sont **pas** stockés : ils se retrouvent à la demande via le QID Wikidata.
- `isbn` reste un attribut propre du Tome (identifiant métier, recherche par code-barres en Phase 5).
- Ni les textes Wikipédia (CC BY-SA) ni les images (licences propres, couvertures protégées) ne sont repris :
  `description` / `biography` sont écrites à la main ou reprises des **descriptions** Wikidata (CC0) ; les
  images restent des uploads.

### Correspondance des champs

**Manga / BD ← Wikidata** (classes : `Q21198342` série de manga, `Q14406742` série de bande dessinée,
`Q725377` roman graphique)

| Attribut / relation             | Wikidata                                                                    |
| ------------------------------- | --------------------------------------------------------------------------- |
| `title` (i18n)                  | libellés fr / en (ou `P1476` titre)                                         |
| `original_title`                | `P1559` nom dans la langue d'origine, ou libellé ja                         |
| `start_year` / `end_year`       | `P571` création ou `P577` date de publication / `P582` date de fin           |
| `volumes_vo`                    | `P2635` nombre de parties                                                   |
| `demographic`, genre            | `P136` genre (valeurs à mapper vers notre enum)                             |
| pays                            | `P495` pays d'origine                                                       |
| `created_by {roles}`            | `P50` auteur → `author`, `P58` scénariste → `writer`, `P110` illustrateur → `artist` |
| `contains_work` (Univers)       | `P8345` franchise médiatique (`Q196600`)                                    |
| `adapted_as`                    | `P4969` œuvre dérivée / `P144` basé sur                                     |
| éditeur d'origine               | `P123` éditeur                                                              |
| `external_ids.bnf_ark`          | `P268` identifiant BnF (quand Wikidata le connaît)                          |

**Personne ← Wikidata** (`Q5`), complétée par l'autorité BnF : `last_name` `P734`, `first_name` `P735`,
`native_name` `P1559`, `nationality` `P27`, `birth_date` `P569`, `external_ids.bnf_ark` `P268`.

**Maison d'édition ← Wikidata** (`Q2085381`) : `name` (libellé), `country` `P17`/`P495`, `website` `P856`,
`external_ids.bnf_ark` `P268`.

**Édition / Tome ← BnF** (API SRU, voir `docs/sources-externes.md`) : un Tome = une notice bibliographique
(`bib.persistentid` → `bnf_ark`, recherche par `bib.fuzzyIsbn`) ; une Édition = la collection éditoriale ou
l'ensemble multivolume qui regroupe ces notices (`bib.col2bib`, `bib.ens2bib`). Champs du Tome : `isbn`,
`title` / `subtitle`, `number` (numéro dans la série), `publication_date`, `page_count`, `retail_price`,
éditeur (→ relation `edited_by` de l'Édition). Zones vérifiées sur des notices réelles pour le connecteur
(2026-10-11, après le changement de modèle de catalogage de juin 2026) : voir « Import depuis les sources ».

### Pour le seed (lot 13)

Les exemples (Dragon Ball, une BD franco-belge) sont repris de Wikidata (œuvres, auteurs, éditeurs) et de
la BnF (une édition française et quelques tomes), avec leurs `external_ids` et `sources` réels : données
légalement réutilisables et au format du connecteur (section suivante).

### Import depuis les sources (Phase 3)

Connecteur [`server/internal/external/bdmanga`](server/internal/external/bdmanga), page admin « Sources externes »
(`/admin/external-import`), API `/external/bd-manga/*` (voir `api/openapi.yaml`, tag `External`). Le connecteur
trouve les types du domaine par **nom technique** (`Manga`, `Comic`, `Universe`, `Edition`, `Volume`, `Person`,
`Publisher` ; `contains_work`, `created_by`, `published_as`, `edited_by`, `contains_volume`) et répond 409
`connector_unavailable` s'il en manque un ; les attributs proposés sont ajustés au schéma (clé ou valeur d'enum
absente du schéma : ignorée).

| Étape      | Ce qui se passe                                                                                                          |
| ---------- | ------------------------------------------------------------------------------------------------------------------------ |
| Recherche  | Œuvre par titre sur Wikidata (classes série de manga, série de BD, roman graphique, comic strip, webtoon…) ; éditions BnF par titre (`bib.serialtitle`) ou par ISBN d'un tome (`bib.fuzzyIsbn` → zone 461 ou 410 du tome) |
| Aperçu     | Œuvre + univers (`P8345`) + auteurs (`P50` → `author`, `P58` → `writer`, `P110` → `artist`, une relation par personne) ; pour chaque édition : notice d'ensemble, éditeurs (210 / 214 `$c`), tomes (`bib.ens2bib` **ou** `bib.col2bib`) |
| Rapprochement | Item existant du même type par `external_ids.wikidata`, `external_ids.bnf_ark`, `isbn` (tomes) ou `name` sans casse ni accents (éditeurs) : ses valeurs sont gardées, seuls les champs vides (et les langues manquantes d'un texte i18n) sont remplis, `sources` gagne l'entrée de la source ; une relation existante est réutilisée, ses rôles ne changent pas |
| Correction | Avant la simulation, chaque nœud peut être corrigé avec le formulaire de son type (ex. numéro d'un tome) : `overrides` (clé → attributs complets) remplace la proposition, l'ordre d'un tome dans son édition suit son numéro corrigé ; un nœud corrigé est coché |
| Import     | Les nœuds cochés passent par `POST /import` (tout ou rien, `dry_run` pour la simulation) ; les tags des items existants sont conservés ; une relation est écrite si chaque extrémité est cochée ou déjà en base |

Limites constatées sur les données réelles (2026-10-11) :

- Les tomes d'une même édition peuvent porter le même numéro (réimpressions, nouvelle édition dans le même ensemble) :
  le plus ancien est coché, les autres sont marqués « réimpression ».
- Les albums anciens rattachés à une **collection éditoriale** (zone 410, ex. « Une aventure d'Astérix le Gaulois »)
  n'ont souvent **pas de numéro** : ils sont proposés décochés avec le numéro 0, à corriger dans l'aperçu.
- La BnF ne donne que l'année de publication : `publication_date` (date complète) n'est pas rempli. Les prix en francs
  sont ignorés, seuls les prix en euros alimentent `retail_price`.
- Non repris pour l'instant : sagas (pas d'équivalent Wikidata), adaptations (`adapted_as`), traducteurs et autres
  contributeurs BnF (zones 700-702), format et statut des éditions, images.

---

## Romans (Phase 3)

Décision du 2026-10-16 : les romans **réutilisent le domaine livre** (Univers, Saga, Personne, Maison d'édition,
Édition, Tome) avec un ItemType d'œuvre dédié, `Novel` (« Roman »). La différence avec la BD tient au rapport
entre l'Édition et le Tome :

- **Un seul volume** (cas courant : « L'Étranger, Folio, 2005 ») : l'**Édition est le livre**. Elle porte l'ISBN,
  la date, la pagination, le prix, la collection éditoriale ; **aucun Tome**. C'est l'Édition qu'on met en collection.
- **Plusieurs volumes** (« Les Misérables » en 3 tomes, une intégrale en 2 volumes) : l'Édition regroupe des **Tomes**
  rattachés par `contains_volume`, comme en BD ; ISBN et prix sont sur chaque Tome.

```
Univers (optionnel) ── contains_work ──▶ Roman
Saga / cycle (optionnel, ex. « La Tour sombre ») ── contains_work {order} ──▶ Roman
Roman
├── created_by {roles: author} ──▶ Personne
└── published_as ──▶ Édition (le livre, si un seul volume)
                     ├── edited_by ──▶ Maison d'édition
                     ├── created_by {roles: translator | illustrator | preface} ──▶ Personne
                     └── contains_volume {order} ──▶ Tome   (seulement si plusieurs volumes)
```

### Roman (`Novel`)

| Attribut                 | Type          | Requis | Description                                                    |
| ------------------------ | ------------- | ------ | -------------------------------------------------------------- |
| `title`                  | i18n-text     | ✅     | Titre (par langue)                                             |
| `original_title`         | string        |        | Titre en VO                                                    |
| `original_language`      | string        |        | Code ISO 639-1 de la langue d'écriture                         |
| `form`                   | enum          |        | `novel`, `novella`, `short_stories`, `essay`, `play`, `poetry` |
| `genres`                 | array d'enum  |        | `fantasy`, `science_fiction`, `crime`, `thriller`, `romance`, `historical`, `horror`, `literary`, `young_adult`, `children`, `classic` |
| `first_publication_year` | integer       |        | Année de première publication                                 |
| `description`            | i18n-text     |        | Résumé (écrit à la main ou description Wikidata, CC0)          |
| `images`                 | image-gallery |        | couverture de référence                                        |
| `external_ids`, `sources` |              |        | Attributs communs du domaine (Wikidata, BnF)                   |

### Évolutions des types partagés

- **Édition** : nouveaux attributs pour le cas « un seul volume » — `isbn`, `publication_date`, `page_count`,
  `retail_price`, `collection` (collection éditoriale : « Folio »), `collection_number` (« n° 2 ») ;
  `format` gagne `hardcover` (relié) et `paperback` (broché) ; la langue d'une traduction est `language`.
- **`created_by`** : sources étendues à `Novel` et à `Edition` (traducteur, illustrateur, préfacier d'une édition) ;
  rôles ajoutés : `illustrator`, `preface`.
- **`contains_work`** et **`published_as`** : `Novel` ajouté aux sources / cibles.

### Sources

Les mêmes que pour la BD, licences vérifiées le 2026-10-11 (`docs/sources-externes.md`) :

| Niveau                                 | Source       | Correspondance                                                                 |
| -------------------------------------- | ------------ | ------------------------------------------------------------------------------ |
| Roman, Saga (cycle), Personne, Éditeur | **Wikidata** | Classes `Q7725634` œuvre littéraire, `Q8261` roman, `Q1667921` série de romans (→ Saga) ; `P50` auteur, `P577` date, `P407` langue, `P136` genre, `P179` partie de la série (→ Saga, ordre `P1545`) |
| Édition, Tome, traducteur              | **BnF**      | Notice bibliographique d'une monographie (une Édition) ; ensemble multivolume (zone 461) → Édition + Tomes ; zones 700-702 `$4` (code de fonction : traducteur 730…) → `created_by` de l'Édition ; 225 → `collection` / `collection_number` |

À valider en codant le connecteur, comme pour la BD : les classes Wikidata utiles (les romans sont souvent
déclarés `Q7725634` + `P7937` forme littéraire), et les zones UNIMARC des notices de romans depuis le
changement de modèle de la BnF (juin 2026).

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
