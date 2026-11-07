// Element Plus (MIT) enregistré composant par composant, avec les seules feuilles de style
// utiles : `app.use(ElementPlus)` et `dist/index.css` embarquaient toute la bibliothèque
// (≈ 280 ko gzip de JS et 62 ko de CSS dans le chunk d'entrée, audit Q-002).
//
// Ajouter un composant : l'importer et l'ajouter à `components`, importer son style
// (`element-plus/es/components/<nom-kebab>/style/css`). Le test `elementPlus.test.ts` compare
// cette liste aux balises `<El…>` des gabarits : un oubli ou un composant devenu inutile
// fait échouer la suite.
import type { App, Plugin } from "vue";
import {
  ElAlert,
  ElAutocomplete,
  ElButton,
  ElCheckbox,
  ElConfigProvider,
  ElDialog,
  ElDrawer,
  ElDropdown,
  ElDropdownItem,
  ElDropdownMenu,
  ElInput,
  ElInputNumber,
  ElLoading,
  ElOption,
  ElPagination,
  ElPopover,
  ElProgress,
  ElRadio,
  ElRadioButton,
  ElRadioGroup,
  ElSelect,
  ElSlider,
  ElStep,
  ElSteps,
  ElSwitch,
  ElTabPane,
  ElTable,
  ElTableColumn,
  ElTabs,
  ElTag,
} from "element-plus";

// Variables et styles communs, puis un fichier par composant (les dépendances internes —
// popper, scrollbar, tag du select… — sont tirées par chacun).
import "element-plus/es/components/base/style/css";
import "element-plus/es/components/alert/style/css";
import "element-plus/es/components/autocomplete/style/css";
import "element-plus/es/components/button/style/css";
import "element-plus/es/components/checkbox/style/css";
import "element-plus/es/components/config-provider/style/css";
import "element-plus/es/components/dialog/style/css";
import "element-plus/es/components/drawer/style/css";
import "element-plus/es/components/dropdown/style/css";
import "element-plus/es/components/dropdown-item/style/css";
import "element-plus/es/components/dropdown-menu/style/css";
import "element-plus/es/components/input/style/css";
import "element-plus/es/components/input-number/style/css";
import "element-plus/es/components/option/style/css";
import "element-plus/es/components/pagination/style/css";
import "element-plus/es/components/popover/style/css";
import "element-plus/es/components/progress/style/css";
import "element-plus/es/components/radio/style/css";
import "element-plus/es/components/radio-button/style/css";
import "element-plus/es/components/radio-group/style/css";
import "element-plus/es/components/select/style/css";
import "element-plus/es/components/slider/style/css";
import "element-plus/es/components/step/style/css";
import "element-plus/es/components/steps/style/css";
import "element-plus/es/components/switch/style/css";
import "element-plus/es/components/tab-pane/style/css";
import "element-plus/es/components/table/style/css";
import "element-plus/es/components/table-column/style/css";
import "element-plus/es/components/tabs/style/css";
import "element-plus/es/components/tag/style/css";
// Services appelés depuis les composables (useNotify, useConfirmDialog) et directive v-loading.
import "element-plus/es/components/message/style/css";
import "element-plus/es/components/message-box/style/css";
import "element-plus/es/components/loading/style/css";

/** Composants globaux `<El…>` des gabarits, par nom. */
export const components: Record<string, Plugin> = {
  ElAlert,
  ElAutocomplete,
  ElButton,
  ElCheckbox,
  ElConfigProvider,
  ElDialog,
  ElDrawer,
  ElDropdown,
  ElDropdownItem,
  ElDropdownMenu,
  ElInput,
  ElInputNumber,
  ElOption,
  ElPagination,
  ElPopover,
  ElProgress,
  ElRadio,
  ElRadioButton,
  ElRadioGroup,
  ElSelect,
  ElSlider,
  ElStep,
  ElSteps,
  ElSwitch,
  ElTabPane,
  ElTable,
  ElTableColumn,
  ElTabs,
  ElTag,
};

/** Installe les composants ci-dessus et la directive `v-loading`. */
export function installElementPlus(app: App) {
  for (const component of Object.values(components)) app.use(component);
  app.use(ElLoading);
}
