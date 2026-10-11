import { createApp } from "vue";
import { createPinia } from "pinia";
import PrimeVue from "primevue/config";
// Preset « compat » : taille de base 14 px comme PrimeVue 4 (le preset aura natif de v5 est en 16 px).
import Aura from "@primeuix/themes/aura-compat";
import ToastService from "primevue/toastservice";
import ConfirmationService from "primevue/confirmationservice";
import "primeicons/primeicons.css";
import "@/assets/main.css";

import App from "./App.vue";
import router from "./router";

const app = createApp(App);

app.use(createPinia());
app.use(router);
app.use(PrimeVue, {
  theme: {
    preset: Aura,
    options: { darkModeSelector: ".dark" },
  },
});
app.use(ToastService);
app.use(ConfirmationService);

app.mount("#app");
