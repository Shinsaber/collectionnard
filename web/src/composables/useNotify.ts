import { ElMessage } from "element-plus";
import { extractApiError } from "@/api/client";
import { i18n } from "@/i18n";

/**
 * Notifications uniformes (ElMessage, sans provider). `notifyError(e)` affiche l'erreur
 * de l'API traduite par son code (`extractApiError`) : plus aucun `catch {}` silencieux
 * dans les vues. `summary` est un texte déjà traduit (`t("…")`) ; défaut : « Erreur ».
 */
export function useNotify() {
  function notifyError(e: unknown, summary?: string) {
    const title = summary ?? i18n.global.t("common.error");
    const { message } = extractApiError(e);
    ElMessage({
      type: "error",
      message: message && message !== title ? `${title} : ${message}` : title,
      duration: 5000,
      showClose: true,
    });
  }

  function notifySuccess(summary: string, detail?: string) {
    ElMessage({
      type: "success",
      message: detail ? `${summary} : ${detail}` : summary,
      duration: 3000,
    });
  }

  return { notifyError, notifySuccess };
}
