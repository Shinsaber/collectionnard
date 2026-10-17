# Frontend Vue — conventions

Vue 3 `<script setup lang="ts">` · Element Plus 2 (MIT, composants globaux `El*`, locale suivant la langue de l'app via `ElConfigProvider`, mode sombre via `html.dark`) · Tailwind 4 (`@tailwindcss/vite`, thème et variante `dark` dans `src/assets/main.css`, pas de `tailwind.config.js`) ·
Pinia · Vue Router (lazy imports) · alias `@` → `src/`.

## Organisation

| Chemin                                | Rôle                                                                                                                                        |
| ------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------- |
| `src/api/types.ts`                    | Types TS **miroir des structs Go** (`server/internal/model`)                                                                                |
| `src/api/client.ts`                   | **Seul** point d'appel HTTP (axios, `baseURL: /api/v1`, JWT)                                                                                |
| `src/stores/auth.ts`                  | Store Pinia d'authentification                                                                                                              |
| `src/router/index.ts`                 | Routes ; `meta: { public: true }` pour les pages sans auth                                                                                  |
| `src/views/`, `views/admin/`          | Pages (admin = gestion des méta-types). Listes `ItemTypesView` / `RelationTypesView` / `CollectionTypesView` (via `components/admin/AdminTypeList.vue`) ; édition sur une page dédiée commune `TypeEditorView` (`/admin/<item\|relation\|collection>-types/<id\|new>`, configuration par sorte dans `typeKinds.ts`, garde de sortie si non enregistré) |
| `components/common/DynamicForm.vue`   | Formulaire généré depuis un JSON Schema (+ `ImageGalleryField.vue`)                                                                         |
| `components/common/SchemaBuilder.vue` | Éditeur visuel de JSON Schema (admin) : deux colonnes (champs à gauche ; slot `aside` pour l'identité, puis `SchemaRootCard.vue` (x-label, x-title, x-description) et JSON à droite) + `SchemaFieldCard.vue` (ligne repliable ; identifiants stables dans `ids`) + `SchemaFieldConstraints.vue` (+ `SchemaEnumLabels.vue`), types dans `schemaBuilderTypes.ts` |
| `components/item/`                    | `ItemGallery`, `RelationTree`, dialogues `ItemEditDialog`, `AddToCollectionDialog`, `RelationCreateDialog`, `RelationEditDialog`            |
| `components/graph/`                   | `GraphNodeDialog`                                                                                                                           |
| `components/collection/`              | `CollectionImportDialog` (import d'un export de collection : simulation, rapport, application), `CollectionItemList` (colonnes = attributs du CollectionType ; cartes sur mobile), `CollectionAddItemDialog` |
| `components/external/`                | Import depuis les sources externes : `ExternalWorkSearch` (Wikidata, MusicBrainz), `ExternalEditionSearch` (titre, ISBN ou code-barres selon `edition_code`, éditions d'une œuvre si `editions_by_work`), `ExternalPlanView` + `ExternalNodeTable` (aperçu à cocher) + `ExternalNodeEditDialog` (correction d'un nœud via `DynamicForm`, envoyée en `overrides`) ; page `views/admin/ExternalImportView.vue` (étapes recherche → sélection → simulation → import). Textes propres à chaque connecteur (titres, intro, attribution, groupes) : `admin.external.connectors.<id>.*` ; identifiants génériques `id` des résultats |
| `components/layout/`                  | Coquille : `AppLayout` (menu latéral `AppSidebar` dès `lg`, sinon `AppTopBar` + `AppTabBar` fixe + feuille `AppMoreSheet`), `AppUserCard` (langue, thème, déconnexion), `AppLogo`, `NavIcon` ; liens et onglet actif dans `navigation.ts`. Le contenu laisse `pb-28` sous `lg` pour la barre d'onglets |
| `src/stores/metaTypes.ts`             | Cache des ItemTypes / RelationTypes / CollectionTypes (`invalidate()` après toute écriture admin)                                           |
| `src/composables/`                    | `useTheme`, `useLang`, `useNotify` (ElMessage), `useConfirmDialog` (ElMessageBox → promesse booléenne), `usePagination`, `useDirty` (bouton « Enregistrer » grisé sans modification)                   |
| `src/pwa.ts`, `vite.config.ts` (VitePWA) | Application installable (lot E5a) : manifeste, icônes `public/pwa-*.png`, service worker Workbox (application seule, aucune réponse d'API) ; mise à jour proposée à l'utilisateur. nginx sert `sw.js` et le manifeste en `no-cache` |
| `stores/offline.ts`, `lib/offlineDb.ts`, `lib/offlineSearch.ts`, `views/OfflineView.vue` | Consultation hors ligne (lot E5) : copie `GET /me/offline` dans IndexedDB, `reachable` (faux sur erreur réseau → le garde du routeur mène à `/offline`), recherche sans accents (ISBN avec ou sans tirets) ; copie effacée à la déconnexion. `fetchMe` ne déconnecte plus sur une erreur réseau ; couvertures : `lib/offlineCovers.ts` (vignettes `?w=240` dans Cache Storage, plafond réglable, éviction des moins récemment consultées) |
| `components/common/AppSpinner.vue`    | Indicateur de chargement                                                                                                                    |
| `views/admin/ImportExportView.vue` | Import / export du catalogue (JSON, simulation puis import, rapport) ; exports CSV/JSON via `components/common/ExportButton.vue` et `lib/csv.ts` (séparateur selon la langue) ; `ImagesView.vue` : purge des images inutilisées (simulation puis suppression) |
| `src/lib/schema.ts`                   | Lecture des schémas et extensions x-* (pendant de `server/internal/label`)                                                                  |
| `src/lib/itemSearch.ts`               | Recherche du catalogue : champs filtrables / triables d'un schéma, état ↔ query (URL = paramètres de `GET /items`) |
| `src/lib/externalPlan.ts`             | Aperçu d'un import externe : regroupement par édition (et par saison d'une série : rôle `season`), sélection par défaut, relation au parent (`role` = vocabulaire du connecteur, jamais un ItemType) |
| `src/i18n/`                          | vue-i18n : registre des langues (`langs.ts`), instance + locales Element Plus (`index.ts`), catalogues `locales/<lang>/<ns>.ts` (fr = référence typée) |

## Règles

- Pas d'`axios` direct dans les vues/composants : ajouter une fonction dans `client.ts`.
- Quand un modèle Go change, mettre `types.ts` à jour dans le même changement.
- Ne jamais coder en dur un ItemType/champ métier (« Series », « title »…) dans une vue :
  tout affichage d'attributs passe par `src/lib/schema.ts` (`itemLabel`, `fieldLabel`, `displayValue`,
  `orderedKeys`, `isImageGallery`, `defaultImage`) qui lit `x-label`, `x-order`, `x-labels`, `x-widget`.
- Aucun texte d'interface en dur : `const { t } = useI18n()` et une clé dans `src/i18n/locales/fr/<ns>.ts` **et** `en/<ns>.ts`
  (le compilateur et `i18n.test.ts` exigent les mêmes clés). Hors composant : `i18n.global.t`. Listes d'options ou libellés
  dépendant de la langue : `computed`. Ne pas nommer `t` une variable de boucle.
- Langue courante : `useLang()` (`lang`, `setLang` ; persistée, sinon langue du navigateur). Noms de méta-types affichés :
  `typeTitle` / `typeDescription` ; valeurs d'attributs : `fieldValue` (libellés d'enum) — jamais `name` brut à l'écran.
  Dates et nombres : `toLocale…(lang.value)`. Les tests tournent en français (`tests/setup.ts`).
- Erreurs : jamais de `catch {}` silencieux — `const { notifyError } = useNotify()` puis `notifyError(e)`
  (affiche le message de l'API, y compris le 413 d'un upload).
- Accessibilité : un `aria-label` sur tout bouton sans texte (icônes `pi pi-*` dans le slot d'`ElButton`,
  `aria-hidden`), un `alt` sur toute image, `label for` / `id` sur les champs (DynamicForm génère ses ids
  avec `useId()`), tableaux cliquables en `highlight-current-row` (navigation clavier).
- Composants UI : Element Plus uniquement (MIT). Lignes d'`ElTable` : le slot `#default="{ row }"` n'est
  pas générique, caster `(row as Item)`. Pagination serveur : `ElPagination` + `pag.goToPage(p)`.
  Jamais de dépendance UI sous licence non libre (PrimeVue ≥ 5 est sous PrimeUI License : exclu).
- Identité graphique « Étiquette » : couleurs et ombres
  **sémantiques** définies dans `main.css` (`bg-canvas`, `bg-surface`, `bg-bar`, `bg-soft`, `text-ink`, `text-muted`,
  `bg-primary` / `text-primary-ink`, `text-link`, `bg-tint` / `text-tint-ink`, `bg-accent`, `shadow-card`, `shadow-bar`,
  `shadow-cover`…), redéfinies sous `html.dark` : pas de variante `dark:` pour ces utilitaires. Le relief passe par les
  ombres, pas par des bordures grises. Polices : `font-sans` (Schibsted Grotesk) et `font-mono` (IBM Plex Mono, pour les
  numéros, références et libellés en capitales), servies par `@fontsource` (aucune requête externe). Logo :
  `components/layout/AppLogo.vue` (même tracé que `public/favicon.svg`). Pas de couleur Tailwind brute (`gray-*`, `indigo-*`…) ni de variante
  `dark:` dans les vues : uniquement les utilitaires sémantiques.
- Dialogues (`ElDialog`) et confirmations (`ElMessageBox`) : mise en forme globale dans `main.css` (corps qui défile,
  en-tête et pied fixes ; feuille du bas sous 768 px). Ne pas leur donner de hauteur ni de marges propres ; une `width`
  reste permise (ignorée sur mobile). Actions dans `#footer` : bouton texte « Annuler » puis bouton principal.
  `DynamicForm` passe en deux colonnes (container query) dès que son conteneur le permet.
- Taille : un fichier `.vue` ne dépasse pas ~400 lignes ; extraire un composant au-delà.
- Après modification : `npm run lint && npm run format:check && npm run type-check && npm test`
  (`npm run lint:fix` et `npm run format` corrigent automatiquement). Tests unitaires : `vitest`,
  fichiers `src/**/*.test.ts` (environnement jsdom, stubs `matchMedia` et `ResizeObserver` dans
  `tests/setup.ts`). Les composants se montent avec `global.plugins: [ElementPlus]` (vue-i18n est ajouté globalement par `tests/setup.ts`) ; les 41 schémas des paquets de domaine sont disponibles dans
  `tests/fixtures/seed-schemas.json` (régénérer si un paquet de domaine change : `node scripts/seed-fixtures.mjs tests/fixtures/seed-schemas.json ../server/internal/domains/packages/{collections,tv,people,video,books,music,games}.json`).
