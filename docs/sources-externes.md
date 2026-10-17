# Sources de données externes — BD / Manga / Comics

Recherche du 2026-10-11 (licences et conditions d'utilisation vérifiées ce jour-là ; elles évoluent :
**les revérifier avant tout import massif ou toute intégration**). Ce document n'est pas un avis juridique.

**Décision actuelle** : seules **Wikidata** et la **BnF** sont retenues pour le domaine BD / Manga
(voir [`les.collection.md`](../les.collection.md) § « Sources de référence »). Les autres sources sont
documentées ici pour plus tard (Phase 3 : récupération automatique externe).

## Synthèse

| Source                       | Licence / conditions                                          | Couverture                                                                  | Statut                  |
| ---------------------------- | ------------------------------------------------------------- | --------------------------------------------------------------------------- | ----------------------- |
| Wikidata                     | **CC0** (données structurées)                                 | Œuvres, séries, auteurs, éditeurs, franchises ; identifiants croisés        | ✅ Retenue              |
| BnF (catalogue général)      | **Licence Ouverte Etalab** (attribution + date de récupération) | Dépôt légal : toutes les éditions françaises (tomes, ISBN, éditeurs, auteurs) | ✅ Retenue              |
| Grand Comics Database (GCD)  | CC BY-SA 4.0 (partage à l'identique)                          | Comics, surtout US, numéro par numéro                                       | ⏸ Plus tard (comics)    |
| Open Library                 | Aucun droit revendiqué, pas de licence nommée                 | Livres par ISBN ; manga inégal                                              | ⏸ Complément éventuel   |
| AniList                      | CGU propriétaires                                             | Très bon sur le manga                                                       | ❌ Incompatible         |
| Comic Vine                   | CGU propriétaires, non commercial                             | Comics                                                                      | ❌ Écarté               |
| MangaUpdates (Baka-Updates)  | Conditions non trouvées                                       | Manga                                                                       | ❌ Pas de base légale   |
| MyAnimeList                  | CGU propriétaires                                             | Manga / anime                                                               | ❌ Non étudié en détail |

## Sources retenues

### Wikidata

- **Licence** : les données structurées (espaces principal, Property, Lexeme) sont sous CC0 ; le texte
  libre des autres espaces est sous CC BY-SA 4.0.
  [Wikidata:Licensing](https://www.wikidata.org/wiki/Wikidata:Licensing)
- **Attention** : les extraits d'articles **Wikipédia** sont sous CC BY-SA, pas CC0. Ne reprendre que les
  données Wikidata (libellés, descriptions courtes, déclarations). Les images liées (`P18`, Wikimedia Commons)
  ont chacune leur propre licence, et les couvertures (œuvres protégées) n'y figurent généralement pas.
- **Accès** : API `wbgetentities` / `Special:EntityData/Qxxx.json`, SPARQL (query.wikidata.org), dumps.
- **Intérêt clé** : Wikidata sert de **table de correspondance** vers les autres bases, sans qu'on stocke
  leurs données : `P268` (BnF), `P11149` (MangaUpdates), `P4087` (MyAnimeList manga), `P8731` (AniList manga),
  `P5905` (Comic Vine). Un seul QID par item suffit pour retrouver les autres identifiants à la demande.

### BnF — catalogue général et data.bnf.fr

- **Licence** : métadonnées descriptives (bibliographiques et d'autorité) sous Licence Ouverte de l'État
  (Etalab) depuis le 1er janvier 2014 ; réutilisation gratuite en mentionnant la source et la date de
  récupération. [Mentions légales data.bnf.fr](https://data.bnf.fr/fr/licence),
  [Portail API et jeux de données](https://www.bnf.fr/fr/portail-bnf-api-et-jeux-de-donnees)
- **Accès** :
  - API SRU 1.2 : `https://catalogue.bnf.fr/api/SRU?version=1.2&operation=searchRetrieve&query=…`
    ([explain](https://catalogue.bnf.fr/api/SRU?version=1.2&operation=explain),
    [documentation](https://www.bnf.fr/fr/service-sru-catalogue-general-de-la-bnf)).
    Index utiles : `bib.isbn`, `bib.fuzzyIsbn` (ISBN-10 / 13 / EAN), `bib.title`, `bib.serialtitle`,
    `bib.author`, `bib.author2bib`, `bib.persistentid` (ARK), `bib.doctype`, `bib.col2bib`
    (notices liées à une collection éditoriale), `bib.ens2bib` (liées à un ensemble multivolume) ;
    autorités : `aut.persistentid`, `aut.status` (`any "validated sparse"` pour inclure les notices minimales).
    Formats : `unimarcXchange`, `intermarcXchange` (+ variantes `-anl`), `dublincore`.
  - data.bnf.fr : SPARQL, RDF ; [dumps](https://api.bnf.fr/fr/dumps-de-databnffr) ;
    [données complètes sur data.gouv.fr](https://www.data.gouv.fr/datasets/les-donnees-completes-des-catalogues-de-la-bnf).
- **Point de vigilance** : en **juin 2026**, la BnF a changé d'outil (Noemi), de format et de modèle de
  catalogage, ce qui « a des effets sur les données du Catalogue général »
  ([centre d'aide](https://www.bnf.fr/fr/centre-d-aide/recuperation-de-donnees)). L'`explain` SRU listait
  encore les index et formats ci-dessus le 2026-10-11 ; les zones UNIMARC lues par le connecteur ont été
  vérifiées ce jour-là sur des notices réelles (voir « Connecteur implémenté »).

## Connecteur implémenté (Phase 3)

Code : `server/internal/external/` (client HTTP commun, `wikidata/`, `bnf/`, connecteur `connectors/`) ; usage et limites
dans [`les.collection.md`](../les.collection.md) § « Import depuis les sources ». Le serveur interroge les sources
lui-même (pas le navigateur) : User-Agent identifié (`EXTERNAL_USER_AGENT`, nom + contact comme le demandent les
règles de Wikimedia), délai de 20 s par appel, réponses limitées à 20 Mio et gardées 15 min en mémoire (l'aperçu
puis l'import d'une même sélection n'interrogent les sources qu'une fois).

Constats faits en codant le connecteur (2026-10-11), utiles pour les prochains :

- **Wikidata** : la recherche filtrée par classe passe par CirrusSearch (`list=search`,
  `haswbstatement:P31=Q…|P31=Q…`), bien plus pertinente que `wbsearchentities` (qui mélange anime, jeux, films) ;
  `wbgetentities` accepte 50 identifiants par appel mais échoue en entier si l'un d'eux est inconnu ; le code ISO
  d'un pays (`P297`) se lit avec `wbgetclaims` sans télécharger l'entité (plusieurs Mio pour un pays). Le libellé
  `mul` d'un prénom japonais peut être en kana : préférer `en` / `fr`. `P268` (BnF) est rarement renseigné sur les
  séries de manga : le lien Wikidata → BnF passe par la recherche de l'édition.
- **BnF (SRU, après Noemi)** : un tome se rattache à son édition par la zone **461** (ensemble multivolume, index
  `bib.ens2bib`) ou **410** (collection éditoriale, index `bib.col2bib`) avec le numéro de notice en `$0` (l'ARK s'en
  déduit : caractère de contrôle NOID calculé sur `cb` + numéro) ; le numéro du tome est en `$v`, à défaut en 225 `$v`
  ou 200 `$h`. Le type de notice (position 6 du label) est parfois vide sur les notices d'ensemble : ne filtrer que
  les types non textuels (vidéo, son, multimédia…). Les titres portent des caractères de non-classement (U+0088 /
  U+0089) à retirer. Prix en 010 `$d` (francs pour les notices anciennes), pagination en 215 `$a`, EAN en 073 `$a`.
  `maximumRecords` accepte 1000 notices (≈ 1 Mio pour 150 tomes UNIMARC).

### Romans (connecteur `novels`, 2026-10-17)

- **Wikidata** : un roman est `P31` œuvre littéraire (`Q7725634`) avec `P7937` forme (`Q8261` roman) ; langue `P407`
  (code ISO 639-1 par `P218` de la langue) ; cycle `P179` avec l'ordre en qualificatif `P1545` — ne devient une Saga que si
  la série est une série d'œuvres (suite romanesque, série de livres, cycle, trilogie, tétralogie, heptalogie…). Libellés
  des genres et formes vérifiés un à un (un candidat s'est révélé « équation linéaire »).
- **BnF** : `P268` du roman = notice d'**œuvre** BnF ; l'index **`bib.work2bib`** liste toutes ses éditions, traductions
  comprises (L'Étranger : 65 notices, Les Misérables : 312). Zone 500 `$3` des éditions = numéro de cette notice d'œuvre.
  Contributeurs en 700-702 avec code de fonction `$4` : 070 auteur, **730 traducteur**, 440 illustrateur, **080 préfacier**,
  340 éditeur scientifique. Collection éditoriale en 225 `$a` / `$v`, reliure en 010 `$b` (« br. », « rel. »).
- Pièges : langue 101 parfois multiple (édition bilingue) ou indéterminée (`xxx`, `und`, `mul`) ; **compilations** sans titre en
  200 (titre de l'œuvre en 423 `$t`) ; notices quasi vides (sans label ni zones) à écarter ; un volume d'« Œuvres complètes »
  se rattache (461) à l'ensemble des œuvres complètes : ne le regrouper en édition multivolume que si l'ensemble porte le titre
  du roman.

## Musique (P3-4, recherche du 2026-10-17)

| Source            | Licence / conditions                                                                 | Statut         |
| ----------------- | ------------------------------------------------------------------------------------ | -------------- |
| MusicBrainz       | Données de base **CC0** ; tags, genres associés, notes, annotations : CC BY-NC-SA 3.0 | ✅ Retenue      |
| Cover Art Archive | Aucune licence affichée (« use at your own risk », respecter les droits)               | ❌ Pas d'images |
| Discogs           | Exports mensuels CC0 ; API : données « CC0 » et « restreintes », conditions non lisibles (403) | ⏸ À revoir    |

- **MusicBrainz** : [licence des données](https://musicbrainz.org/doc/About/Data_License) ; données de base
  ([liste](https://musicbrainz.org/doc/MusicBrainz_Database)) : zones, artistes, événements, genres, instruments, labels,
  supports, lieux, enregistrements, groupes de parutions, parutions, séries, œuvres, relations et URL. Supplémentaires (CC
  BY-NC-SA, **non reprises**) : annotations, tags (y compris les associations de genres) et notes des utilisateurs,
  statistiques. API `musicbrainz.org/ws/2` sans clé ; User-Agent « Application/version ( contact ) » obligatoire ;
  [~1 requête par seconde](https://musicbrainz.org/doc/MusicBrainz_API/Rate_Limiting) par adresse IP (503 au-delà).
- **Cover Art Archive** : [page](https://musicbrainz.org/doc/Cover_Art_Archive) sans licence ; les pochettes restent des
  envois, comme les couvertures des livres.
- **Discogs** : exports mensuels sous CC0, API gratuite avec enregistrement de l'application ; les
  [conditions de l'API](https://support.discogs.com/hc/en-us/articles/360009334593) distinguent données CC0 et données
  restreintes (page inaccessible le 2026-10-17) : à relire avant tout usage.

### Connecteur `music` (MusicBrainz, 2026-10-17)

- **Requêtes** : recherche `release-group` (chaque mot dans le titre **ou** l'artiste : `(releasegroup:discovery OR
  artist:discovery) AND (releasegroup:daft OR artist:daft) …` — la forme « titre OU artiste » sur toute la phrase classait
  « Daft Punk Live in NYC » avant « Discovery ») ; parutions d'un album par `release?release-group=…&inc=labels+media`
  (OK Computer : 39) ; recherche `release` par `barcode:` ou `release:(…)` ; parution complète avec
  `inc=artist-credits+labels+recordings+release-groups+isrcs` ; artiste avec `inc=url-rels` (lien Wikidata, dates).
  Les valeurs d'`inc` gardent leurs `+` non encodés. Un aperçu album + une parution coûte trois requêtes (≈ 2,5 s).
- **Débit** : le client commun espace les requêtes vers `musicbrainz.org` de 1,1 s (les réponses en cache ne sont pas
  retardées) ; un 503 (limite dépassée, service chargé) est retenté une fois.
- **Pièges** : un **code-barres** est partagé par plusieurs pressages (724385522925 : huit parutions d'OK Computer) —
  jamais une clé de rapprochement, seulement une recherche ; dates partielles (« 1997 », « 1997-06 ») : `release_date`
  seulement pour une date complète, l'année sinon ; pays pseudo-ISO **XE** (Europe) et **XW** (monde) ; formats de support
  nombreux (« 12" Vinyl », « Enhanced CD », « Hybrid SACD », « Digital Media »…) ramenés à l'énumération du type, le
  premier support d'une parution mixte (CD + DVD) donnant le format ; artiste spécial « Various Artists » (compilations) et
  label « [no label] » jamais importés ; `label-info` mêle labels et distributeurs, tous repris avec leur référence.
- **Rapprochement** : MBID partout (`external_ids.musicbrainz`), QID Wikidata en second pour l'album ; labels aussi par
  nom (sans casse ni accents), comme les éditeurs ; artistes **pas** par nom (homonymes : deux groupes « Nirvana »).

## Films (P3-5, recherche du 2026-10-17)

Mêmes sources que les romans, mêmes licences (Wikidata CC0, BnF Licence Ouverte) ; **TMDB écarté** (données non ouvertes,
choix du mainteneur). Modèle : `video.md`.

- **Wikidata** : un film est `P31` film (`Q11424`), film d'animation (`Q202866`), long métrage d'animation (`Q29168811`),
  documentaire (`Q93204`), court métrage (`Q24862`) ou téléfilm (`Q506240`). Générique : `P57` réalisation, `P58`
  scénario, `P86` musique, `P344` image, `P161` interprétation (Amélie : 41 acteurs, rarement avec le personnage `P453`) ;
  `P577` dates de sortie (une par pays, qualificatif `P291`), `P2047` durée en minutes, `P495` pays, `P364` langue
  originale, `P136` genres, `P345` IMDb, `P268` **notice d'œuvre BnF** (Amélie : `14404106q`). Libellés des genres vérifiés
  un à un : deux candidats étaient faux (`Q1342372` = film de monstre, pas fantastique `Q157394` ; `Q1146335` = film pour
  adolescents, pas sport `Q1339864`).
- **BnF** : les vidéogrammes du dépôt légal sont des notices de type `g` (guide `cgm`), filtrées par
  `bib.doctype any "h"` ; `bib.work2bib` sur la notice d'œuvre du film liste ses éditions (Amélie : 9, DVD, Blu-ray et VHS ;
  Chihiro : 9) ; **`bib.ean`** trouve une édition par son code-barres (`bib.fuzzyIsbn` ne le fait pas). Zones : 073 EAN,
  200 `$a` titre et `$e` mention (« édition limitée »), 205 mention (« Autorisé à la location »), 215 support (« DVD vidéo
  monoface simple couche zone 2 (2 h) », « BD vidéo » = Blu-ray, « cass. vidéo (VHS) »), nombre de disques en tête,
  matériel d'accompagnement en `$e` ; 101 langues (`$a` audio, `$c` originale, `$j` sous-titres) ; 225 collection et
  numéro (« Collection Studio Ghibli », 2) ; 300 notes (contenu, bonus) ; 500 `$3` numéro de la notice d'œuvre ; 702 `$4`
  fonction (300 réalisation, 690 scénario, 230 musique, 005 interprétation — les têtes d'affiche seulement) ; 712 `$4` 650
  éditeur, 310 distributeur, 630 producteur (pas toujours présente : 210 `$c` à défaut).
- Pas d'images : les jaquettes ne sont pas sous licence libre.

### Connecteur `films` (2026-10-17)

- **Requêtes** : film par `wbgetentities` (générique : une entité par personne, puis leurs noms et prénoms, et la
  nationalité de chacun par `wbgetclaims`) ; éditions par `bib.work2bib` de la notice d'œuvre `P268` (notices vidéo
  seulement, guide `g`) ; par EAN `bib.ean` ; par titre `bib.title … and bib.doctype any "h"`. Un aperçu film + une
  édition coûte une vingtaine de requêtes Wikidata (≈ 3 s), une édition seule ≈ 0,4 s de plus.
- **Édition seule** (scan du code-barres) : la notice d'œuvre de l'édition (500 `$3`, ex. `14404106` → `14404106q` avec le
  caractère de contrôle) est cherchée sur Wikidata par `haswbstatement:P268=14404106q`, limitée aux classes de films ; à
  défaut, le film est créé depuis la notice BnF (titre 500 `$a`, langue originale 101 `$c`, générique 7XX `$4`).
- **Générique** : rôles d'une même personne fusionnés en une relation (Jeunet : réalisation et scénario) ; Wikidata liste
  41 interprètes pour Amélie, sans ordre garanti ni personnage (`P453` rare) : dix proposés, les cinq premiers cochés.
- **Pièges** : `P577` porte une date par pays (année retenue : la plus ancienne) ; un même EAN pour deux VHS (location et
  vente) : jamais une clé de rapprochement ; éditeur souvent absent de 712 dans les notices récentes : repli sur 210 `$c`,
  où le nom est entre crochets avec son rôle (« [TF1 vidéo [éd., distrib.]] ») ; mention d'édition en 200 `$e`
  (« édition collector ») plutôt qu'en 205 (« Autorisé à la location »).

## Séries (P3-6, recherche du 2026-10-17)

- **Wikidata** : Breaking Bad (`Q1079`) a ses 5 saisons en `P527` (éléments « saison de série télévisée » `Q3464665`,
  ordre `P1545`, diffuseur `P449`, nombre d'épisodes `P1113`) et chaque saison ses épisodes en `P527` (éléments
  `Q21191270`, libellé français, `P577`, `P2047`, `P345`). Couverture inégale selon les séries : à compléter par TVmaze.
- **TVmaze** : [API](https://www.tvmaze.com/api) sans clé, **CC BY-SA** (citer TVmaze, lien vers la fiche ; partage à
  l'identique pour une rediffusion des données) ; au moins 20 requêtes par 10 secondes, 429 au-delà. Recherche par IMDb
  (`lookup/shows?imdb=`), saisons et épisodes (numéro, titre anglais, date, durée). Résumés et images non repris.
- **BnF** : les coffrets de séries sont des notices vidéo **sans lien vers une notice d'œuvre** (`bib.work2bib` vide pour
  Breaking Bad, notice `16963580`) : recherche par titre (21 notices, dont le film « El Camino » à écarter) ou par EAN ; la
  saison est en 200 `` (« Saison 5 », « Saison finale [Saison 5] », « Intégrale des saisons 1, 2 et 3 »,
  « L'intégrale de la série »).

### Connecteur `series` (2026-10-17)

- **Couverture Wikidata** : Breaking Bad, Kaamelott (459 épisodes), Lupin ont tous leurs épisodes ; Le Bureau des légendes
  et Dix pour cent n'en ont aucun (50 et 24 épisodes repris de TVmaze), Engrenages un seul sur 75. La durée `P2047` est
  parfois fausse (un épisode de Kaamelott de trois minutes noté 174 minutes) : reprise telle quelle.
- **Coût** : la série, ses saisons, puis épisodes, diffuseurs et personnes par lots de 50 entités : un aperçu de Breaking
  Bad (62 épisodes) prend ≈ 4 s ; TVmaze ajoute trois requêtes (série par IMDb, saisons, épisodes).
- **Coffrets** : sans notice d'œuvre, ils sont cherchés par titre puis filtrés sur le titre exact de la série (200 `$a`) ;
  une notice de coffret peut ne pas être typée vidéo (guide vide, « 21 DVD vidéo ») : reconnue à son support. La saison
  vient de 200 `$h` / `$i` / `$e` : « Intégrale de la saison 1 », « Saisons 1 à 4 », « Saison finale » (dernière saison),
  « L'intégrale de la série » (toutes).
- **Diffusion** : un élément « AMC 2008-2013 » propre à la série, sans identifiant : rapproché par la relation
  `broadcast_on` de la série déjà en base (même chaîne), jamais par son nom seul.

## Jeux vidéo (P3-7, recherche du 2026-10-17)

| Source            | Licence / conditions                                                               | Statut                 |
| ----------------- | ---------------------------------------------------------------------------------- | ---------------------- |
| Wikidata          | CC0                                                                                | ✅ Retenue (le jeu)    |
| BnF               | Licence Ouverte (dépôt légal des jeux vidéo)                                       | ✅ Retenue (éditions)  |
| libretro-database | CC BY-SA 4.0 (dépôt GitHub) ; données en partie issues de No-Intro / Redump         | ⏸ V2 (hors France)     |
| IGDB (Twitch)     | API gratuite en usage non commercial, contrat Twitch ; aucune licence ouverte       | ❌ Écartée             |
| RAWG              | Clé gratuite, lien obligatoire, redistribution interdite, payant au-delà d'un seuil | ❌ Écartée             |
| MobyGames, TheGamesDB | Conditions non lisibles (pages en JavaScript) ; API MobyGames payante à notre connaissance | ❌ Non retenues |

- **Wikidata** : Breath of the Wild (`Q17185964`) porte plateformes `P400`, développeurs `P178`, éditeur `P123`, genres
  `P136`, modes `P404`, série `P179`, PEGI `P908` (PEGI 3 `Q14915512`, 7 `Q14915514`, 12 `Q14915515`, 16 `Q14915516`, 18
  `Q14915517`), date `P577`, et une notice BnF `P268` qui **n'est pas** la notice d'œuvre des éditions (`bib.work2bib`
  vide). Genres et modes retenus : les plus fréquents des jeux Switch (requête SPARQL), libellés vérifiés. Les plateformes
  ont leur nom en libellé multilingue (`mul`), leur fabricant en `P176` ; leur classe varie (Wii U : « modèle de console
  de salon », Switch : « modèle d'objet manufacturé ») : le type de plateforme reste facultatif.
- **BnF** : deux sortes de notices. **Complètes** (guide `clm` / `nlm`, 230 « Jeu multimédia ») : 215 support et
  plateforme (« 1 cartouche (Nintendo Switch) », « 1 disque optique numérique (Wii U) », « 1 cartouche Super Nintendo »),
  500 `` notice d'œuvre (`` plateforme), 712 `` 650 éditeur, 310 distributeur, 635 développeur, 630 producteur.
  **Brèves** (guide sans type, 2026) : 073 EAN, 215 « 1 cartouche informatique » ou « 1 ressource dématérialisée » (code),
  sans plateforme. Pas de filtre `bib.doctype` propre aux jeux : recherche par titre puis tri sur le support. Un EAN peut
  avoir perdu son zéro initial (« 45496333133 », 11 chiffres). Environ 1 100 notices « jeu vidéo » datées de 2023.

### Connecteur `games` (2026-10-17)

- **Éditions** : pas de lien Wikidata → notice d'œuvre BnF exploitable (`P268` ne donne pas les éditions) : recherche par
  titre (« Mario Kart 8 Deluxe » : 4 notices dont 3 jeux), filtrée sur le support (guide `l` + 230 « Jeu », ou notice sans
  type avec « cartouche », « disque optique », « ressource dématérialisée ») et sur le titre (la notice contient le titre du
  jeu ; 200 `$i` et `$e` compris : « The legend of Zelda : Breath of the wild »).
- **Sociétés** : Wikidata donne le studio (`P178` : Nintendo EPD) et l'éditeur (`P123` : Nintendo) ; la BnF « Nintendo
  co. » (712 `$4` 635, développeur) — comparée sans suffixe juridique, la même société n'est proposée qu'une fois ; les
  notices brèves donnent la filiale française (« NINTENDO FRANCE »), gardée à part. La Switch a deux constructeurs
  (`P176` : Nintendo et sa division matérielle) : tous deux liés.
- **Plateformes** : nom en libellé multilingue (`mul`) ; dans la notice BnF, entre parenthèses (« (Nintendo Switch) »,
  « (Wii U) ») ou après le support (« 1 cartouche Super Nintendo »), sinon en 500 `$q` ; rapprochée d'une plateforme du
  jeu par la fin de son nom (« Wii U », « Switch » → « Nintendo Switch »).

## Sources pour plus tard

### Grand Comics Database (comics.org)

- **Licence** : toutes les parties de la GCD, sauf mention contraire, sous **CC BY-SA 4.0**, schéma de base et
  format de distribution compris. Conséquence : toute donnée qui en dérive doit rester sous CC BY-SA, avec
  attribution. [comics.org](https://www.comics.org/)
- **Accès** : exports CSV / JSON par page de listing (identifiants de clés étrangères seulement, sans relations
  many-to-many) ; un dump SQLite sans images serait proposé sur `comics.org/download/` (information lue dans la
  documentation d'un outil tiers, [gcd-talker](https://pypi.org/project/gcd-talker/), non vérifiée sur le site).
- **Usage envisagé** : comics US (séries, numéros, crédits détaillés), si le projet accepte le partage à
  l'identique pour ces données.

### Open Library

- **Conditions** : l'Internet Archive « ne revendique aucun nouveau droit » sur la base, mais signale de possibles
  droits existants sur certaines contributions ; pas de licence nommée, rien sur les couvertures.
  [Licensing](https://openlibrary.org/developers/licensing)
- **Accès** : [dumps mensuels JSON](https://openlibrary.org/developers/dumps) ; API par ISBN (1 req/s,
  3 req/s avec un User-Agent identifié), pas faite pour l'extraction massive.
  [API](https://openlibrary.org/developers/api)

### Écartées

- **AniList** : gratuit en dessous de 150 $ de revenus/mois, mais interdit la collecte massive, l'usage comme
  stockage et les services concurrents de type « list/tracker », ce qu'est Collectionnard.
  [Terms of use](https://docs.anilist.co/guide/terms-of-use)
- **Comic Vine** : strictement non commercial, 200 requêtes par ressource et par heure.
  [API](https://comicvine.gamespot.com/api)
- **MangaUpdates** : aucune condition d'utilisation des données trouvée ; ne pas stocker leurs données sans
  autorisation écrite. Leur identifiant reste accessible via Wikidata (`P11149`).
