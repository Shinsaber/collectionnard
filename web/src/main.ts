import { createApp } from "vue";
import { createPinia } from "pinia";
import ElementPlus from "element-plus";
import "element-plus/dist/index.css";
import "element-plus/theme-chalk/dark/css-vars.css";
import "primeicons/primeicons.css";
import "@/assets/main.css";

import App from "./App.vue";
import router from "./router";
import { i18n } from "./i18n";

const app = createApp(App);

app.use(createPinia());
app.use(router);
app.use(i18n);
// Element Plus (MIT) : composants globaux ; la locale suit la langue de l'app
// (ElConfigProvider dans App.vue). Le mode sombre suit la classe `html.dark`
// posée par useTheme (dark/css-vars.css).
app.use(ElementPlus);

app.mount("#app");
