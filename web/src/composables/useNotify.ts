import { useToast } from "primevue/usetoast";
import type { ToastServiceMethods } from "primevue/toastservice";
import { extractApiError } from "@/api/client";

/**
 * Notifications uniformes. `notifyError(e)` affiche le message de l'API
 * (`extractApiError`) : plus aucun `catch {}` silencieux dans les vues.
 * Tolère l'absence de ToastService (tests unitaires) en retombant sur la console.
 */
export function useNotify() {
  let toast: ToastServiceMethods | null = null;
  try {
    toast = useToast();
  } catch {
    toast = null;
  }

  function notifyError(e: unknown, summary = "Erreur") {
    const { message } = extractApiError(e);
    if (toast) {
      toast.add({ severity: "error", summary, detail: message, life: 5000 });
    } else {
      console.error(summary, message);
    }
  }

  function notifySuccess(summary: string, detail?: string) {
    toast?.add({ severity: "success", summary, detail, life: 3000 });
  }

  return { notifyError, notifySuccess };
}
