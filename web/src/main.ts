import { createApp } from "vue";
import { createPinia } from "pinia";
import ElementPlus from "element-plus";
import fr from "element-plus/es/locale/lang/fr";
import "element-plus/dist/index.css";
import "element-plus/theme-chalk/dark/css-vars.css";
import "primeicons/primeicons.css";
import "@/assets/main.css";

import App from "./App.vue";
import router from "./router";

const app = createApp(App);

app.use(createPinia());
app.use(router);
// Element Plus (MIT) : composants globaux, libellés en français ; le mode sombre
// suit la classe `html.dark` posée par useTheme (dark/css-vars.css).
app.use(ElementPlus, { locale: fr });

app.mount("#app");
