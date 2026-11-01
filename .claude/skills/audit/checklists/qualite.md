# Checklist — Qualité et tests

Lire le code réel, citer `fichier:ligne`, constaté / supposé. Les outils de la CI ont déjà tourné (`RESUME.md`) :
interpréter leurs sorties, chercher ce qu'ils ne voient pas.

## Résultats outillés à interpréter

- `go-test.txt` : échecs, paquets « no test files », couverture par paquet (`-cover`) ; les plus bas sont-ils des
  paquets à risque (store, handler, privacy, itemquery, external) ?
- `golangci-lint.txt`, `go-vet.txt`, `gofmt.txt`, `npm-lint.txt`, `npm-type-check.txt`, `npm-format.txt` : tout écart
  est un constat (la CI devrait être rouge : expliquer pourquoi elle ne l'est pas).
- `npm-test.txt` : nombre de fichiers et de tests, durée, tests ignorés (`skip`, `todo`).
- `npm-build.txt` : taille des chunks ; ECharts, zxing-wasm, cropperjs, Cytoscape doivent être dans des chunks chargés à
  la demande, pas dans l'entrée.
- `metriques.txt` : fichiers > 400 lignes (règle `web/CLAUDE.md`), `fmt.Sprintf` dans le store, `w.Write` direct,
  `pool.Begin`, `catch` vides, `axios` hors `client.ts`, `any`, TODO.

## Architecture backend (`server/internal`)

- Couches respectées : handler → store (interface `Store`) → pgx ; pas de SQL dans les handlers, pas de HTTP dans le store.
- Erreurs : `store.ErrNotFound` / `ErrConflict` traduits par `handleStoreError` ; pas d'erreur avalée (`_ =`) ; contexte
  propagé partout (annulation client).
- Transactions : une par opération multi-table (import, proposition acceptée, inscription), `defer Rollback`, savepoints de
  l'import ; `s.begin` (acteur) sur toute écriture du catalogue.
- Duplication : connecteurs (`external/connectors/*.go`) — même code copié entre `films`, `series`, `games`, `novels` ?
  Handlers de listes ; conversion modèle ↔ ligne ; `scan` répétitifs.
- Performance : requêtes dans des boucles (N+1) dans le store ou les handlers (relations, jonctions, tags : « requêtes
  groupées » promises) ; index de `001_init.sql` et suivants vs `WHERE` / `ORDER BY` réels (`search_tsv`, GIN `attributes`,
  `item_relations(source, target, type)`, `collection_items`, `item_history`) ; `LIMIT` sur toutes les listes ; graphe en BFS
  borné.
- Concurrence : état partagé dans les handlers (cache des schémas compilés, rate limiter, cache HTTP externe) protégé ;
  fuites de goroutines ; `http.Client` avec délai.
- Configuration : valeurs par défaut sûres, erreurs explicites au démarrage, aucune lecture d'env hors `config` et `main`.

## Tests backend

- Règle : 100 % des handlers testés (`handler/*_test.go` via `MockStore`) : chaque route de `Router()` a-t-elle au moins
  un test succès + un test 401/403 + 404 + 422 si validation ? Lister les routes sans test.
- `testutil.MockStore` : méthodes ajoutées sans test correspondant ?
- Intégration (`store/integration_test.go`, `-tags integration`) : SQL récent couvert (visibilité, historique,
  propositions, notifications, hors ligne, statistiques, complétude) ? Déclencheurs d'historique testés ?
- Connecteurs : `connectorstest` avec réponses enregistrées ; chaque connecteur a un test d'aperçu et d'import ; réponses
  enregistrées datées (fraîcheur ?).
- Tests fragiles : dépendants de l'heure, de l'ordre des maps, du réseau.

## Frontend (`web/src`)

- Taille et découpage : composants > 400 lignes, logique métier dans les vues plutôt que `lib/` (testable à part).
- Typage : `any`, `as` abusifs, `!` non nuls, `types.ts` en retard sur `model` (champs manquants ou en trop : comparer
  quelques structs récentes : propositions, notifications, partage).
- État : stores Pinia (`auth`, `metaTypes`, `notifications`, `offline`) — invalidation du cache des méta-types après
  écriture admin ; une seule requête partagée pour les notifications ; fuites d'écouteurs (`onUnmounted`).
- Erreurs : `notifyError` partout ; promesses non attendues ; `catch` vides.
- Accessibilité : `aria-label` sur boutons icône, `alt` sur images, `label for`, navigation clavier des listes, focus des
  dialogues, contraste des couleurs sémantiques en sombre.
- Performance : imports statiques lourds (`StatsChart`, `cytoscape`, `cropperjs`, ponyfill du scanner) chargés au
  démarrage ? listes longues sans pagination ; images sans vignette `?w=240` ; re-rendus (computed vs méthodes).
- Hors ligne : cohérence de la copie IndexedDB (schémas inclus), éviction des couvertures, bandeau et actions masquées.
- i18n : clés orphelines (présentes dans `locales/fr` sans usage) ou manquantes (`i18n.test.ts` les compare fr/en, mais pas
  l'usage) ; textes composés par concaténation (pluriels, ordre des mots).

## Tests frontend

- Composants critiques avec test : `DynamicForm`, `SchemaBuilder`, validation (`validateValue`), recherche
  (`itemSearch`), csv, barcode, perspective, lightbox, proposals, stats, offline ; vues : lesquelles n'ont aucun test ?
- Fixtures `tests/fixtures/seed-schemas.json` à jour par rapport aux paquets (47 schémas, régénération documentée).

## Base de données et migrations

- Migrations : numérotation continue, chaque fichier a un bloc « down » ou est explicitement irréversible ; aucune
  migration passée modifiée (`git.txt`) ; contraintes `CHECK` et `FOREIGN KEY … ON DELETE` cohérentes avec les cascades
  attendues (supprimer un item : collection items, relations, historique ?).
- Paquets de domaine : `package_version` incrémenté quand un type change (comparer `git log -p` des `packages/*.json`
  depuis `--depuis` : un changement sans incrément est un constat).

## Documentation technique du code

- Commentaires de paquet Go, godoc des exportés principaux ; README et `CLAUDE.md` à jour sur les commandes et variables
  réellement lues (`config.go`, `main.go`, `cmd.go`).
