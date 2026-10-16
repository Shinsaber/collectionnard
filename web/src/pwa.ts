// Service worker de l'application installable (lot E5a) : enregistré en production seulement ;
// quand une nouvelle version est publiée, l'utilisateur choisit de recharger maintenant ou plus tard.
import { ElMessageBox } from "element-plus";
import { registerSW } from "virtual:pwa-register";
import { i18n } from "@/i18n";

export function setupPwa() {
  if (!import.meta.env.PROD || !("serviceWorker" in navigator)) return;
  const { t } = i18n.global;
  const updateSW = registerSW({
    onNeedRefresh() {
      ElMessageBox.confirm(
        t("offline.update.message"),
        t("offline.update.title"),
        {
          confirmButtonText: t("offline.update.reload"),
          cancelButtonText: t("offline.update.later"),
          type: "info",
        },
      )
        .then(() => updateSW(true))
        .catch(() => {
          // « Plus tard » : la nouvelle version s'activera au prochain lancement.
        });
    },
  });
}
