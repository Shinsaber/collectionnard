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
