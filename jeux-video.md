# Collections de jeux vidéo — Modèle de données

Phase 3, lot P3-7 (2026-10-17). Paquet de domaine `games` (`server/internal/domains/packages/games.json`), requiert
`people` (personnes et sociétés). Sources de la V1 : **Wikidata** (le jeu, ses plateformes, ses sociétés) et **BnF** (les
éditions françaises du dépôt légal). La V1 se concentre sur les éditions françaises (choix du mainteneur) ;
libretro-database (CC BY-SA, éditions par plateforme et région, numéros de série) est une piste pour la suite.

## Décisions de conception

- **Jeu** (`Game`) = l'œuvre (Mario Kart 8), toutes plateformes et éditions confondues.
- **Édition de jeu** (`GameRelease`) = l'objet possédé : la cartouche Switch « Mario Kart 8 Deluxe », le disque Wii U, un
  code de téléchargement… (support, EAN, mention d'édition, PEGI, langues). C'est elle qu'on met en collection.
- **Plateforme** (`Platform`) = un item, **rattaché à la société qui la fabrique** (choix du mainteneur) : Nintendo
  Switch `produced_by` Nintendo {manufacturer}. Une collection « mes jeux Switch » se lit dans le graphe.
- **Sociétés** (développeur, éditeur, distributeur, constructeur) : type `Publisher` du paquet `people` (affiché
  « Société » depuis people v2), partagé avec les livres et la vidéo — le même Nintendo édite le jeu et fabrique la console.
- Pas d'images reprises (jaquettes et photos non libres) : les images restent des envois.

```
Jeu ──game_released_as ──▶ Édition de jeu ──on_platform ──▶ Plateforme ──produced_by {manufacturer} ──▶ Société
 ├── on_platform ──▶ Plateforme                     └── produced_by {publisher, distributor} ──▶ Société
 └── produced_by {developer, publisher} ──▶ Société
```

## ItemTypes

| Type          | UUID     | Attributs |
| ------------- | -------- | --------- |
| `Game`        | `b…0040` | `title`* (texte par langue), `original_title`, `release_year`, `genres` (30 genres), `game_modes` (solo, multijoueur, coopératif, écran partagé, joueur contre joueur), `pegi` (3, 7, 12, 16, 18), `description` (texte par langue), `images` (jaquette) |
| `GameRelease` | `b…0041` | `title`*, `format` (cartridge, optical_disc, card, download_code, digital, other), `edition`, `release_year`, `pegi`, `ean`, `languages`, `notes`, `images` (recto, verso, support) |
| `Platform`    | `b…0042` | `name`*, `kind` (home_console, handheld, hybrid, computer, mobile, arcade, other), `release_year`, `discontinued_year`, `images` (photo) |

`external_ids` : `wikidata` et `bnf_ark` (jeu), `bnf_ark` (édition), `wikidata` (plateforme). `sources` :
`[{ source: wikidata|bnf|manual, retrieved_at }]`.

## RelationTypes

| Nom                | UUID     | Source → Cible                       | Attributs |
| ------------------ | -------- | ------------------------------------ | --------- |
| `game_released_as` | `c…0040` | Jeu → Édition de jeu                 | — |
| `on_platform`      | `c…0041` | Jeu / Édition de jeu → Plateforme    | — |
| `produced_by`      | `c…0042` | Jeu / Édition / Plateforme → Société | `roles`* (developer, publisher, distributor, manufacturer) |

## Sources

Vérifié le 2026-10-17 (voir `docs/sources-externes.md`).

| Élément             | Wikidata (CC0)                                                                     | BnF (Licence Ouverte)                               |
| ------------------- | ---------------------------------------------------------------------------------- | --------------------------------------------------- |
| Jeu                 | `P31` jeu vidéo (`Q7889`), libellés, `P577`, `P136` genres, `P404` modes, `P908` PEGI | 500 `$a` (titre de l'œuvre, `$q` plateforme)     |
| Sociétés du jeu     | `P178` développeur, `P123` éditeur                                                 | 712 `$4` : 635 développement, 650 édition, 310 distribution, 630 production |
| Plateformes         | `P400` ; plateforme : `P176` fabricant, `P571` / `P2669` dates, `P31` classe        | 215 : « 1 cartouche (Nintendo Switch) », « 1 cartouche Super Nintendo » |
| Édition             | —                                                                                  | 073 EAN, 200 titre et mention, 215 support, 101 langues, 230 « Jeu multimédia », 300 notes |

## Import (connecteur `games`)

Page admin « Sources externes », sorte de collection « Jeux vidéo » (`/external/games/*`) :

1. **Jeu** cherché sur Wikidata (`P31` jeu vidéo) ; une fois choisi, ses **éditions françaises** sont listées depuis la
   BnF (notices titrées comme lui : « Mario kart 8 : deluxe » pour Mario Kart 8 Deluxe, pas le Mario Kart 8 de la Wii U),
   les plus récentes d'abord. Une édition se trouve aussi par **code-barres** (EAN ; la BnF a parfois perdu le zéro
   initial d'un UPC : les deux formes sont cherchées).
2. **Aperçu** : jeu (titres, année, genres, modes, PEGI), sociétés (`produced_by` : développeur, éditeur), plateformes
   (`on_platform`) chacune **rattachée à son constructeur** (`produced_by` {manufacturer}), éditions (support, EAN, année,
   langues, contenu), leur plateforme (lue dans la notice : « 1 cartouche (Nintendo Switch) » ; pour une notice brève sans
   plateforme, celle du jeu s'il n'en a qu'une) et leurs sociétés (712 : éditeur, distributeur, développeur).
3. **Une même société** n'est proposée qu'une fois : « Nintendo co. » de la BnF rejoint le « Nintendo » de Wikidata (nom
   comparé sans suffixe juridique) ; « Nintendo France » reste une société à part.
4. **Édition choisie seule** (code-barres) : son jeu est retrouvé sur Wikidata par le titre de la notice.
5. **Rapprochement** : QID ou ARK, sociétés et plateformes aussi par nom (sans casse ni accents).

Exemple : `db/seed_games.sql` (Mario Kart 8 Deluxe, la Switch et Nintendo, deux éditions françaises).

Limites de la V1 : éditions françaises seulement (BnF) ; libretro-database (CC BY-SA : éditions par région, numéros de
série) pour une V2 ; pas de générique de personnes (réalisateur, compositeur) ni de série de jeux.
