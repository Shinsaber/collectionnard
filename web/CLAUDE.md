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
| `src/views/`, `views/admin/`          | Pages (admin = gestion des méta-types)                                                                                                      |
| `components/common/DynamicForm.vue`   | Formulaire généré depuis un JSON Schema (+ `ImageGalleryField.vue`)                                                                         |
| `components/common/SchemaBuilder.vue` | Éditeur visuel de JSON Schema (admin) : coquille + `SchemaRootCard.vue` (x-label, x-title, x-description) + `SchemaFieldCard.vue` + `SchemaFieldConstraints.vue` (+ `SchemaEnumLabels.vue`), types dans `schemaBuilderTypes.ts` |
| `components/item/`                    | `ItemGallery`, `RelationTree`, dialogues `ItemEditDialog`, `AddToCollectionDialog`, `RelationCreateDialog`, `RelationEditDialog`            |
| `components/graph/`                   | `GraphNodeDialog`                                                                                                                           |
| `components/collection/`              | `CollectionImportDialog` (import d'un export de collection : simulation, rapport, application), `CollectionItemList` (colonnes = attributs du CollectionType ; cartes sur mobile), `CollectionAddItemDialog` |
| `components/external/`                | Import depuis les sources externes : `ExternalWorkSearch` (Wikidata), `ExternalEditionSearch` (BnF, titre ou ISBN), `ExternalPlanView` + `ExternalNodeTable` (aperçu à cocher) + `ExternalNodeEditDialog` (correction d'un nœud via `DynamicForm`, envoyée en `overrides`) ; page `views/admin/ExternalImportView.vue` (étapes recherche → sélection → simulation → import) |
| `components/layout/`                  | Coquille : `AppLayout` (menu latéral `AppSidebar` dès `lg`, sinon `AppTopBar` + `AppTabBar` fixe + feuille `AppMoreSheet`), `AppUserCard` (langue, thème, déconnexion), `AppLogo`, `NavIcon` ; liens et onglet actif dans `navigation.ts`. Le contenu laisse `pb-28` sous `lg` pour la barre d'onglets |
| `src/stores/metaTypes.ts`             | Cache des ItemTypes / RelationTypes / CollectionTypes (`invalidate()` après toute écriture admin)                                           |
| `src/composables/`                    | `useTheme`, `useLang`, `useNotify` (ElMessage), `useConfirmDialog` (ElMessageBox → promesse booléenne), `usePagination`                   |
| `components/common/AppSpinner.vue`    | Indicateur de chargement                                                                                                                    |
| `views/admin/ImportExportView.vue` | Import / export du catalogue (JSON, simulation puis import, rapport) ; exports CSV/JSON via `components/common/ExportButton.vue` et `lib/csv.ts` (séparateur selon la langue) |
| `src/lib/schema.ts`                   | Lecture des schémas et extensions x-* (pendant de `server/internal/label`)                                                                  |
| `src/lib/itemSearch.ts`               | Recherche du catalogue : champs filtrables / triables d'un schéma, état ↔ query (URL = paramètres de `GET /items`) |
| `src/lib/externalPlan.ts`             | Aperçu d'un import externe : regroupement par édition, sélection par défaut, relation au parent (`role` = vocabulaire du connecteur, jamais un ItemType) |
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
- Identité graphique « Étiquette » (refonte en cours, branche `refonte-graphique`) : couleurs et ombres
  **sémantiques** définies dans `main.css` (`bg-canvas`, `bg-surface`, `bg-bar`, `bg-soft`, `text-ink`, `text-muted`,
  `bg-primary` / `text-primary-ink`, `text-link`, `bg-tint` / `text-tint-ink`, `bg-accent`, `shadow-card`, `shadow-bar`,
  `shadow-cover`…), redéfinies sous `html.dark` : pas de variante `dark:` pour ces utilitaires. Le relief passe par les
  ombres, pas par des bordures grises. Polices : `font-sans` (Schibsted Grotesk) et `font-mono` (IBM Plex Mono, pour les
  numéros, références et libellés en capitales), servies par `@fontsource` (aucune requête externe). Logo :
  `components/layout/AppLogo.vue` (même tracé que `public/favicon.svg`). Les gammes `gray` / `indigo` / `blue` sont
  réalignées sur la palette **à titre transitoire** : ne plus les utiliser dans du code neuf.
- Dialogues (`ElDialog`) et confirmations (`ElMessageBox`) : mise en forme globale dans `main.css` (corps qui défile,
  en-tête et pied fixes ; feuille du bas sous 768 px). Ne pas leur donner de hauteur ni de marges propres ; une `width`
  reste permise (ignorée sur mobile). Actions dans `#footer` : bouton texte « Annuler » puis bouton principal.
  `DynamicForm` passe en deux colonnes (container query) dès que son conteneur le permet.
- Taille : un fichier `.vue` ne dépasse pas ~400 lignes ; extraire un composant au-delà.
- Après modification : `npm run lint && npm run format:check && npm run type-check && npm test`
  (`npm run lint:fix` et `npm run format` corrigent automatiquement). Tests unitaires : `vitest`,
  fichiers `src/**/*.test.ts` (environnement jsdom, stubs `matchMedia` et `ResizeObserver` dans
  `tests/setup.ts`). Les composants se montent avec `global.plugins: [ElementPlus]` (vue-i18n est ajouté globalement par `tests/setup.ts`) ; les 28 schémas des seeds sont disponibles dans
  `tests/fixtures/seed-schemas.json` (régénérer si un seed change : `node scripts/seed-fixtures.mjs tests/fixtures/seed-schemas.json ../db/seed.sql ../db/seed_bd_manga.sql`).
