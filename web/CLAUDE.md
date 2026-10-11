# Frontend Vue — conventions

Vue 3 `<script setup lang="ts">` · PrimeVue 4 (thème Aura via `@primeuix/themes`) · Tailwind 3 ·
Pinia · Vue Router (lazy imports) · alias `@` → `src/`.

## Organisation

| Chemin                         | Rôle                                                          |
| ------------------------------ | ------------------------------------------------------------- |
| `src/api/types.ts`             | Types TS **miroir des structs Go** (`server/internal/model`)  |
| `src/api/client.ts`            | **Seul** point d'appel HTTP (axios, `baseURL: /api/v1`, JWT)   |
| `src/stores/auth.ts`           | Store Pinia d'authentification                                |
| `src/router/index.ts`          | Routes ; `meta: { public: true }` pour les pages sans auth    |
| `src/views/`, `views/admin/`   | Pages (admin = gestion des méta-types)                        |
| `components/common/DynamicForm.vue`   | Formulaire généré depuis un JSON Schema           |
| `components/common/SchemaBuilder.vue` | Éditeur visuel de JSON Schema (admin)             |
| `src/composables/`             | `useTheme`, `useLang`, `usePagination`                        |
| `src/lib/schema.ts`            | Lecture des schémas et extensions x-* (pendant de `server/internal/label`) |

## Règles

- Pas d'`axios` direct dans les vues/composants : ajouter une fonction dans `client.ts`.
- Quand un modèle Go change, mettre `types.ts` à jour dans le même changement.
- Ne jamais coder en dur un ItemType/champ métier (« Series », « title »…) dans une vue :
  tout affichage d'attributs passe par `src/lib/schema.ts` (`itemLabel`, `fieldLabel`, `displayValue`,
  `orderedKeys`, `isImageGallery`, `defaultImage`) qui lit `x-label`, `x-order`, `x-labels`, `x-widget`.
- Libellés UI en français ; langue des libellés de champs via `useLang()` (fr/en, persistée).
- Après modification : `npm run lint && npm run format:check && npm run type-check && npm test`
  (`npm run lint:fix` et `npm run format` corrigent automatiquement). Tests unitaires : `vitest`,
  fichiers `src/**/*.test.ts` (environnement jsdom). Les composants PrimeVue se montent avec
  `global.plugins: [[PrimeVue, { unstyled: true }]]` ; les 13 schémas du seed sont disponibles dans
  `tests/fixtures/seed-schemas.json` (régénérer depuis `db/seed.sql` si le seed change).
