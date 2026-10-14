import { ElMessageBox } from "element-plus";
import { i18n } from "@/i18n";

export interface ConfirmOptions {
  message: string;
  title?: string;
  confirmText?: string;
  cancelText?: string;
  /** Action destructive : bouton de confirmation rouge. */
  danger?: boolean;
}

/**
 * Boîte de confirmation (ElMessageBox) sous forme de promesse : `true` si l'utilisateur
 * confirme, `false` s'il annule ou ferme.
 */
export function useConfirmDialog() {
  async function confirm(opts: ConfirmOptions): Promise<boolean> {
    try {
      await ElMessageBox.confirm(
        opts.message,
        opts.title ?? i18n.global.t("common.confirmation"),
        {
          confirmButtonText:
            opts.confirmText ?? i18n.global.t("common.confirm"),
          cancelButtonText: opts.cancelText ?? i18n.global.t("common.cancel"),
          type: opts.danger ? "warning" : "info",
          confirmButtonClass: opts.danger ? "el-button--danger" : undefined,
        },
      );
      return true;
    } catch {
      return false;
    }
  }
  return { confirm };
}
