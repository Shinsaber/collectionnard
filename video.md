# Collections vidéo (films en DVD, Blu-ray…) — Modèle de données

Phase 3, lot P3-5 (2026-10-17). Paquet de domaine `video` (`server/internal/domains/packages/video.json`), requis par
`tv` ; personnes et éditeurs : paquet commun `people`. Sources : **Wikidata** (le film) et **BnF** (les éditions vidéo
françaises du dépôt légal), comme les romans.

## Décisions de conception

- **Film** = l'œuvre (Le Fabuleux Destin d'Amélie Poulain), indépendante de ses éditions.
- **Édition vidéo** (`VideoRelease`) = l'objet qu'on possède : DVD TF1 Vidéo de 2010, Blu-ray de 2011, VHS de 2002…
  (support, disques, zone, EAN, langues, sous-titres, contenu). C'est l'ancien type `DVD` du paquet `tv` (même UUID
  `b…0005`, mêmes clés d'origine `title`, `publisher`, `release_year`, `disc_count`), enrichi et partagé : un coffret de
  série et un film sont tous deux des éditions vidéo, rangées dans la même collection (choix du mainteneur).
- **Personnes** (réalisateurs, scénaristes, acteurs, compositeurs…) et **éditeurs vidéo** : types `Person` et `Publisher`
  du paquet commun `people` (choix du mainteneur) — la même personne relie livres et films.
- Un coffret contenant plusieurs films : plusieurs films `released_as` la même édition (`order` = rang dans le coffret).
- Pas d'images reprises (jaquettes non libres) : les images restent des envois.

```
Film ──released_as {order} ──▶ Édition vidéo ──published_by {roles} ──▶ Éditeur (people)
  └── credited {roles, character, order} ──▶ Personne (people)
Série (tv) ──released_as ──▶ Édition vidéo ──includes_episode {disc, track} ──▶ Épisode (tv)
```

## ItemTypes

| Type           | UUID     | Attributs |
| -------------- | -------- | --------- |
| `Film`         | `b…0030` | `title`* (texte par langue), `original_title`, `kind` (feature, animated, documentary, short, tv_film), `release_year`, `duration` (min), `countries` (ISO 3166-1), `original_language`, `genres` (29 genres, libellés Wikidata vérifiés), `description` (texte par langue), `images` (affiche 2/3) |
| `VideoRelease` | `b…0005` | `title`*, `format` (dvd, blu_ray, uhd_blu_ray, vhs, hd_dvd, laserdisc, other), `disc_count`, `region`, `edition`, `collection`, `collection_number`, `publisher` (texte libre), `release_year`, `ean`, `runtime` (min), `audio_languages`, `subtitle_languages`, `aspect_ratio`, `notes` (contenu, bonus), `images` (recto, verso, disque) |

`external_ids` : `wikidata`, `bnf_ark` (notice d'œuvre BnF), `imdb` pour le film ; `bnf_ark` pour l'édition. `sources` :
`[{ source: wikidata|bnf|manual, retrieved_at }]`.

## RelationTypes

| Nom            | UUID     | Source → Cible               | Attributs |
| -------------- | -------- | ---------------------------- | --------- |
| `released_as`  | `c…0004` | Film / Série → Édition vidéo | `order` (rang dans un coffret) — ancienne relation du paquet `tv`, ouverte aux films |
| `credited`     | `c…0030` | Film → Personne              | `roles`* (director, screenwriter, actor, composer, cinematographer, producer, editor), `character`, `order` |
| `published_by` | `c…0031` | Édition vidéo → Éditeur      | `roles` (publisher, distributor) |

Mise à jour d'une instance en `tv` v1 : la v2 requiert `video`, installé d'abord, qui reprend le type `DVD` (renommé
`VideoRelease`, champs ajoutés, modifications locales gardées) et `released_as` ; `includes_episode` reste dans `tv`.

## Sources

Vérifié le 2026-10-17 (voir `docs/sources-externes.md`).

| Attribut / relation         | Wikidata (CC0)                                               | BnF (Licence Ouverte), notice vidéo                    |
| --------------------------- | ------------------------------------------------------------ | ------------------------------------------------------ |
| Film `title`, `description` | libellés et descriptions fr / en ; `P1476` titre original    | 500 `$a` (titre de l'œuvre)                             |
| `kind`, `genres`            | `P31` (film, film d'animation, documentaire…), `P136`        | —                                                      |
| `release_year`, `duration`  | `P577` (plus ancienne date), `P2047` (minutes)               | —                                                      |
| `countries`, langue         | `P495` → `P297`, `P364` → `P218`                             | —                                                      |
| Notice d'œuvre BnF          | `P268` (ex. `14404106q`)                                     | 500 `$3` des éditions ; index `bib.work2bib`           |
| `credited`                  | `P57` réalisation, `P58` scénario, `P86` musique, `P344` image, `P161` interprétation (ordre de l'élément) | 702 `$4` : 300 réalisation, 690 scénario, 230 musique, 005 interprétation |
| Édition                     | —                                                            | 073 EAN, 200 `$a` titre et `$e` mention (« édition limitée »), 205 mention, 215 support (« DVD vidéo », « BD vidéo » = Blu-ray, « cass. vidéo (VHS) ») / disques / zone / durée, 101 langues (`$a` audio, `$c` originale, `$j` sous-titres), 225 collection et numéro, 300 contenu, 712 `$4` 650 éditeur, 310 distributeur (à défaut 210 `$c`) |

## Import (connecteur `films`)

Page admin « Sources externes », sorte de collection « Films » (`/external/films/*`) :

1. **Film** cherché sur Wikidata parmi les classes de films ; une fois choisi, ses **éditions vidéo** sont listées depuis
   sa notice d'œuvre BnF (`P268` → `bib.work2bib`, notices vidéo seulement), des plus récentes aux plus anciennes
   (support, disques, zone, EAN, éditeur). Une édition se cherche aussi par titre ou par **code-barres** (`bib.ean`).
2. **Aperçu** : film (titres, type, année, durée, pays, langue, genres, IMDb), générique (réalisation, scénario, musique,
   image : cochés ; dix premiers interprètes, les cinq premiers cochés, ordre au générique), édition (support, disques,
   zone, durée, format d'image, EAN, mention d'édition, collection, langues audio et sous-titres, contenu et matériel
   d'accompagnement), éditeurs (`published_by` : éditeur, distributeur).
3. **Édition choisie seule** (code-barres) : son film est retrouvé par la notice d'œuvre BnF (recherche Wikidata
   `haswbstatement:P268=…`) ; à défaut, un film est créé depuis la notice (titre de l'œuvre, langue originale, générique
   des zones 7XX).
4. **Rapprochement** : QID Wikidata ou ARK BnF (le film par sa notice d'œuvre), éditeurs par ARK puis par nom (sans casse
   ni accents) ; jamais par EAN (TF1 Vidéo a réutilisé 3384442017220 pour deux VHS). Seuls les champs vides sont complétés.

Exemple : `db/seed_video.sql` (Amélie, son générique et le Blu-ray « édition collector » TF1 vidéo 2010).

## Séries en vidéo (P3-6)

Les séries gardent le modèle du paquet `tv` (diffusion, saisons, épisodes) ; leurs coffrets sont des éditions vidéo du
paquet `video`, liées aux épisodes qu'ils contiennent : un épisode est atteint par sa saison **et** par son coffret.

```
Série ──broadcast_on {broadcast_start_year} ──▶ Diffusion (AMC) ──contains_season ──▶ Saison ──contains_episode {order} ──▶ Épisode
  ├── credited {roles: creator, actor…} ──▶ Personne (people)                                            ▲
  └── released_as ──▶ Édition vidéo (coffret « Intégrale de la saison 1 ») ──includes_episode ─────────────┘
```

Paquet `tv` **v3** (ajouts seulement, aucune clé retirée) : `Series` gagne `original_title`, `countries`,
`original_language`, `season_count`, `episode_count`, `external_ids` (`wikidata`, `imdb`, `tvmaze`, `bnf_ark`) et `sources` ;
`Season` gagne `episode_count`, `external_ids` (`wikidata`, `tvmaze`), `sources` ; `Episode` gagne `original_title`,
`air_date`, `external_ids` (`wikidata`, `imdb`, `tvmaze`), `sources` ; `Broadcast` gagne `sources`. Sources possibles :
`wikidata`, `tvmaze`, `bnf`, `manual`. Paquet `video` **v2** : `credited` s'ouvre aux séries, avec le rôle `creator`.

| Élément            | Wikidata (CC0)                                                        | TVmaze (CC BY-SA), à défaut      | BnF (Licence Ouverte)                    |
| ------------------ | --------------------------------------------------------------------- | -------------------------------- | ---------------------------------------- |
| Série              | `P31` série télévisée, `P580` / `P582`, `P2437` saisons, `P1113` épisodes, `P495`, `P364`, `P345` | `lookup/shows?imdb=` | — |
| Diffusion          | `P449` diffuseur d'origine (de la saison, sinon de la série)          | `network`, `webChannel`          | —                                        |
| Saisons            | `P527` de la série (ordre `P1545`)                                    | `shows/{id}/seasons`             | —                                        |
| Épisodes           | `P527` de la saison (ordre `P1545`), libellés fr / en, `P577`, `P2047` | `shows/{id}/episodes` (titres anglais) | —                                  |
| Générique          | `P170` création, `P161` interprétation                                | —                                | —                                        |
| Coffrets           | —                                                                     | —                                | notices vidéo par titre ou EAN ; saisons lues en 200 `$h` (« Intégrale de la saison 1 », « Intégrale des saisons 1, 2 et 3 », « L'intégrale de la série ») |

### Import (connecteur `series`)

Page admin « Sources externes », sorte de collection « Séries » (`/external/series/*`) :

1. **Série** cherchée sur Wikidata (série télévisée, mini-série, web-série, séries d'animation) ; ses **coffrets** sont
   listés depuis la BnF (notices vidéo titrées comme la série : « Breaking bad — Intégrale de la saison 1 »), ou trouvés par
   titre ou **code-barres**.
2. **Aperçu** : série (titre français, titre original, années, pays, langue, nombres de saisons et d'épisodes, IMDb,
   TVmaze), diffusion (« AMC 2008-2013 », câble / streaming / télévision selon la classe du diffuseur), saisons et épisodes
   (titre français, titre original, numéro, date de première diffusion, durée) ; créateurs et dix premiers interprètes ;
   coffret (support, disques, zone, EAN, langues, contenu), éditeur, et `includes_episode` vers chaque épisode des saisons
   du coffret (« Saison finale » = dernière saison ; « L'intégrale de la série » = toutes).
3. **Épisodes manquants** : une saison sans épisodes sur Wikidata est complétée par TVmaze (titres anglais, source
   `tvmaze`, CC BY-SA) ; une saison inconnue de Wikidata est créée depuis TVmaze.
4. **Coffret choisi seul** (code-barres) : la série est retrouvée sur Wikidata par le titre du coffret (libellé identique).
5. **Rapprochement** : QID, IMDb ou identifiant TVmaze ; la diffusion (sans identifiant propre) par la relation
   `broadcast_on` de la série existante, même chaîne. Le Breaking Bad de `db/seed.sql` porte ses QID : un import le
   complète au lieu de le doubler.
