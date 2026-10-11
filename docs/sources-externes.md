# Sources de données externes — BD / Manga / Comics

Recherche du 2026-10-11 (licences et conditions d'utilisation vérifiées ce jour-là ; elles évoluent :
**les revérifier avant tout import massif ou toute intégration**). Ce document n'est pas un avis juridique.

**Décision actuelle** : seules **Wikidata** et la **BnF** sont retenues pour le domaine BD / Manga
(voir [`les.collection.md`](../les.collection.md) § « Sources de référence »). Les autres sources sont
documentées ici pour plus tard (Phase 4 : intégrations).

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
  encore les index et formats ci-dessus le 2026-10-11, mais les zones UNIMARC et la structure des notices
  sont à revalider au moment de coder un connecteur.

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
