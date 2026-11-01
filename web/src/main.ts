import { createApp } from "vue";
import { createPinia } from "pinia";
import { installElementPlus } from "./plugins/elementPlus";
import "element-plus/theme-chalk/dark/css-vars.css";
import "primeicons/primeicons.css";
import "@/assets/main.css";

import App from "./App.vue";
import router from "./router";
import { i18n } from "./i18n";
import { setupPwa } from "./pwa";
// Thème mémorisé (ou du système) posé dès le démarrage : les pages publiques, sans carte
// utilisateur, l'oubliaient au rechargement (retours v0.4.0).
import "./composables/useTheme";

const app = createApp(App);

app.use(createPinia());
app.use(router);
app.use(i18n);
// Element Plus (MIT) : seuls les composants utilisés, enregistrés globalement par
// plugins/elementPlus.ts ; la locale suit la langue de l'app (ElConfigProvider dans
// App.vue). Le mode sombre suit la classe `html.dark` posée par useTheme (dark/css-vars.css).
installElementPlus(app);

app.mount("#app");
setupPwa();
