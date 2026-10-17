# Collections musique (CD, vinyles…) — Modèle de données

Phase 3, lot P3-4 (2026-10-17). Paquet de domaine `music` (`server/internal/domains/packages/music.json`),
connecteur `music` (source **MusicBrainz**).

## Décisions de conception

- **Album** = « groupe de parutions » MusicBrainz (*release group*) : l'œuvre (OK Computer), indépendante du support.
- **Édition musicale** (`Release`) = une parution précise, l'objet qu'on possède : CD européen de 2016, vinyle de 1997…
  (format, pays, date, label, référence catalogue, code-barres). C'est elle qu'on met en collection.
- **Piste** (`Track`) = un *enregistrement* MusicBrainz (*recording*), en item : la même piste présente sur l'album et
  sur une compilation est **le même item**, atteint par deux chemins (graphe libre, comme un épisode sur une saison et
  sur un DVD). Liée à l'édition par `contains_track` {`disc`, `position`, `order`}.
- **Artiste** (`Artist`) : personne ou groupe ; pas le type `Person` des livres (nom / prénom obligatoires, inadaptés
  à un groupe).
- **Label** (`Label`) : maison de disques, liée à l'édition avec la référence catalogue.
- Pas d'images reprises : la licence des pochettes du Cover Art Archive n'est pas établie (« use at your own risk ») ;
  les images restent des envois.

```
Album (release group)
├── performed_by {credited_as} ──▶ Artiste
└── issued_as ──▶ Édition musicale (release)
                  ├── released_by {catalog_number} ──▶ Label
                  ├── performed_by ──▶ Artiste        (si le crédit diffère de l'album)
                  └── contains_track {disc, position, order} ──▶ Piste (recording)
                                                           └── performed_by ──▶ Artiste (si différent)
```

## ItemTypes

| Type      | UUID      | Attributs                                                                                                     |
| --------- | --------- | ------------------------------------------------------------------------------------------------------------- |
| `Artist`  | `b…0020`  | `name`*, `sort_name`, `kind` (person, group, orchestra, choir, character, other), `country`, `begin_year`, `end_year`, `disambiguation`, `images` |
| `Album`   | `b…0021`  | `title`*, `primary_type` (album, single, ep, broadcast, other), `secondary_types` (compilation, soundtrack, live, remix, dj_mix, mixtape, demo…), `first_release_year`, `images` |
| `Release` | `b…0022`  | `title`*, `format` (cd, vinyl, cassette, digital, sacd, dvd, blu_ray, other), `media_count`, `release_date`, `release_year`, `country`, `barcode`, `catalog_number`, `status` (official, promotion, bootleg, pseudo_release), `packaging`, `images` (recto, verso, disque) |
| `Track`   | `b…0023`  | `title`*, `length` (secondes), `isrc` |
| `Label`   | `b…0024`  | `name`*, `country`, `disambiguation` |

Tous portent `external_ids` (`musicbrainz` : MBID ; `wikidata` pour Artiste et Album) et `sources`
(`[{ source: musicbrainz|manual, retrieved_at }]`).

## RelationTypes

| Nom               | Source → Cible                    | Attributs                                   |
| ----------------- | --------------------------------- | ------------------------------------------- |
| `performed_by`    | Album / Édition / Piste → Artiste | `credited_as` (nom crédité, s'il diffère)   |
| `issued_as`       | Album → Édition musicale          | —                                           |
| `contains_track`  | Édition musicale → Piste          | `disc`, `position`, `order` (tri)           |
| `released_by`     | Édition musicale → Label          | `catalog_number`                            |

## Source : MusicBrainz

Vérifié le 2026-10-17 (voir `docs/sources-externes.md`) : les **données de base** (artistes, groupes de parutions,
parutions, enregistrements, labels, relations) sont sous **CC0** ; les **tags, associations de genres, notes et
annotations** des utilisateurs sont sous CC BY-NC-SA 3.0 (non commercial) : **non repris** (pas de genre importé). API
sans clé, User-Agent avec contact obligatoire, 1 requête par seconde.

| Attribut / relation               | MusicBrainz                                                                 |
| --------------------------------- | --------------------------------------------------------------------------- |
| Album `title`, types, année        | release group : `title`, `primary-type`, `secondary-types`, `first-release-date` |
| Édition                            | release : `title`, `date`, `country`, `status`, `barcode`, `packaging`, `media[].format`, `label-info` |
| Pistes                             | `media[].tracks[]` : `position`, `recording.id`, `recording.title`, `length` (ms) |
| Artiste                            | artist : `name`, `sort-name`, `type`, `country`, `life-span`, `disambiguation` |
| Crédits                            | `artist-credit[]` (`name` crédité, `artist.id`, `joinphrase`)               |
| `external_ids.wikidata`            | relation URL `wikidata` de l'artiste ou du groupe de parutions             |
