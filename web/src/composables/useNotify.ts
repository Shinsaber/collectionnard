import { ElMessage } from "element-plus";
import { extractApiError } from "@/api/client";

/**
 * Notifications uniformes (ElMessage, sans provider). `notifyError(e)` affiche le
 * message de l'API (`extractApiError`) : plus aucun `catch {}` silencieux dans les vues.
 */
export function useNotify() {
  function notifyError(e: unknown, summary = "Erreur") {
    const { message } = extractApiError(e);
    ElMessage({
      type: "error",
      message:
        message && message !== summary ? `${summary} : ${message}` : summary,
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
