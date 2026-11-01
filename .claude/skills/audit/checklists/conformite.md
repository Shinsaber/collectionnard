# Checklist — Conformité aux conventions du projet

Les règles sont celles de `CLAUDE.md` (racine), `server/CLAUDE.md`, `web/CLAUDE.md` et `docs/conception.md` § 3. Pour
chaque règle : est-elle respectée partout ? Citer les écarts en `fichier:ligne`, constaté / supposé. `metriques.txt`,
`routes-openapi.txt` et `extensions.txt` donnent les premières pistes.

## Contrat d'API d'abord

- `routes-openapi.txt` : toute route du `Router()` est dans `api/openapi.yaml` et réciproquement ; pour un échantillon de
  routes récentes (propositions, notifications, partage, domaines, sources externes), comparer méthode, corps, codes
  d'erreur (`error` stables : `connector_unavailable`, `proposal_state`, `last_admin`…), sécurité déclarée
  (public / bearer / admin) et pagination (`PaginatedResponse`).
- `web/src/api/types.ts` **miroir** de `server/internal/model` : champs, noms snake_case, optionnels (`omitempty` ↔
  `?`), enums de rôles et d'états.
- `web/src/api/client.ts` : seul point d'appel HTTP ; chaque fonction appelle une route existante ; aucune route de la
  spec inutilisée par le front sans raison (CLI ou tiers).
- Messages d'erreur : l'API renvoie `error` (code) + `message` (anglais technique) ; le front traduit par code
  (`locales/*/errors.ts` ?) ; aucun message serveur affiché brut.

## Schéma BDD et paquets de domaine

- Aucune migration appliquée modifiée (`git.txt`) ; nouveau schéma = nouveau fichier `NNN_*.sql` au format tern.
- Types uniquement dans `server/internal/domains/packages/*.json` (jamais dans les seeds) ; UUID lisibles par préfixe
  (`a…` users, `b…` item_types, `c…` relation_types, `d…` collection_types, `e…` items), plages non renumérotées ;
  `package_version` incrémenté à chaque changement de type ; `requires` cohérent avec les types référencés.
- Convention d'anglais snake_case ASCII : clés JSON, noms de RelationTypes, valeurs d'enum ; libellés via `x-labels`,
  `x-title`, `x-enum-labels` en `fr` **et** `en` pour chaque champ, type et valeur (chercher les trous).
- Seeds : données de démonstration seulement, chargés après les paquets, ordre `defaultSeedNames`, `external_ids` /
  `sources` réels.

## Extensions `x-*`

- `extensions.txt` : chaque extension présente dans les paquets est gérée par `DynamicForm.vue` (ou ses champs),
  `SchemaBuilder.vue` (conserve ce qu'il ne connaît pas), lue via `lib/schema.ts`, documentée dans le README **et**
  `docs/conception.md` § 3. `DynamicForm` délègue la lecture à `lib/schema.ts` : un zéro à la fois dans `DynamicForm` **et** `lib/schema.ts`, ou dans une colonne de documentation, est un constat ; vérifier le reste en lisant le code.
- `x-label` : rendu identique entre Go `internal/label` et TS `lib/schema.ts` (mêmes replis `title` / `name` / `label`,
  même traitement d'un champ absent) ; `x-completeness` : `collstats` / `store/stats.go` ↔ `isCompleteness` ; `x-private`
  : `privacy` ↔ `CollectionShareDialog` ; `x-identifier` : `identifier` ↔ `identifierPrefill` / doublons.
- Pas de widget `slider` : curseur déduit de `minimum` + `maximum`.

## Backend Go

- `respondJSON` / `respondError` / `respondValidationError` partout (pas de `w.Write`), `decodeJSON` + `respondDecodeError`,
  `parsePagination`, `parseOptionalUUID`.
- Toute écriture d'attributs passe par `validator.ValidateAttributes` ; tout schéma par `checkAttributesSchema`.
- Nouvelle table du catalogue présente dans l'export **et** l'import (`store/export.go`) ; colonne référençant un envoi
  présente dans `ReferencedUploads`.
- Écritures du catalogue : `s.begin` (acteur, origine), rôles `RequireRole` / `RequireAdmin` selon la règle.
- Store : `ErrNotFound` / `ErrConflict`, un fichier par domaine, toutes les routes dans `handler.go`.
- `MockStore` : champ `XxxFn` **et** méthode pour chaque méthode de l'interface.
- Logs `slog`, aucun `fmt.Println` / `log.Printf` ; aucune lecture d'env hors `config` / `main` / `cmd`.

## Frontend Vue

- Aucun ItemType ni champ métier codé en dur (« Series », « title », « isbn »…) dans `views/` et `components/` : tout
  passe par `lib/schema.ts` (`itemLabel`, `fieldLabel`, `displayValue`, `orderedKeys`, `galleryUrls`…). Chercher les
  chaînes littérales suspectes.
- Aucun texte d'interface en dur : `t('…')` partout, clés présentes en `fr` et `en` ; listes dépendant de la langue en
  `computed` ; dates et nombres via `toLocale…(lang.value)`.
- Element Plus uniquement (pas de PrimeVue ≥ 5, pas d'autre bibliothèque UI) ; `ElTable` lignes castées ; pagination
  serveur `ElPagination` + `goToPage`.
- Identité « Étiquette » : pas de couleur Tailwind brute ni de variante `dark:` dans les vues et composants
  (`metriques.txt`) ; relief par ombres sémantiques ; polices `font-sans` / `font-mono` via `@fontsource` (aucune requête
  externe : vérifier `index.html`, CSS, CSP).
- Dialogues : pas de hauteur ni de marge propre ; actions dans `#footer` (« Annuler » puis principal) ; feuille du bas sous
  768 px héritée de `main.css`.
- Fichiers `.vue` ≤ ~400 lignes ; `ImageLightbox` seule surface hors thème.
- Accessibilité minimale (aria-label, alt, label/for, clavier) et `useId()` pour les champs de `DynamicForm`.
- PWA : service worker sans réponse d'API ; `.wasm` local ; nginx `no-cache` pour `sw.js` et le manifeste.

## Documentation et processus

- README : tableaux des variables d'environnement, des routes et de l'arborescence alignés avec `config.go`,
  `handler.go`, le dépôt ; tableau « Documentation » complet (tout fichier de `docs/` y figure, et dans `CLAUDE.md`).
- `docs/conception.md` § 8 : cases cochées réellement livrées (sonder deux ou trois), rien de livré non coché ; § 3 à
  jour des extensions ; § 9 versions alignées avec `go.mod`, `web/package.json`, Dockerfiles, `flake.nix`.
- `docs/domaines/*.md` ↔ paquets : attributs et relations décrits = ceux des `packages/*.json` (sonder un type par domaine).
- `web/CLAUDE.md` : avis `npm audit` acceptés encore exacts ; procédure de régénération des fixtures valide.
- CI (`.github/workflows/ci.yml`) : rejoue exactement les étapes de `CLAUDE.md` § Règles 4 (gofmt, vet, lint, test,
  govulncheck, intégration, lint, lint:openapi, format, type-check, test, build, audit).
- Langue : documentation, commits et commentaires destinés au projet en français ; identifiants, clés et messages
  techniques en anglais.
- Secrets : rien dans le dépôt ; `.env.example` à jour des variables réellement lues.
