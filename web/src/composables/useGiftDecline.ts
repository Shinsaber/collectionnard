import { useI18n } from "vue-i18n";
import { giftApi } from "@/api/client";
import { useConfirmDialog } from "@/composables/useConfirmDialog";
import { useNotify } from "@/composables/useNotify";
import { personName } from "@/lib/gifts";
import type { GiftReservation } from "@/api/types";

/**
 * « Pas reçu » (P7-4b) : après confirmation, la réservation remise est invalidée et l'offrant prévenu.
 * Renvoie `true` si le refus a été enregistré.
 */
export function useGiftDecline() {
  const { t } = useI18n();
  const { notifyError, notifySuccess } = useNotify();
  const { confirm } = useConfirmDialog();

  async function decline(
    collectionId: string,
    gift: GiftReservation,
  ): Promise<boolean> {
    const ok = await confirm({
      message: t("gifts.owner.declineConfirm", {
        name: personName(gift.giver),
      }),
      confirmText: t("gifts.owner.decline"),
      danger: true,
    });
    if (!ok) return false;
    try {
      await giftApi.decline(collectionId, gift.id);
      notifySuccess(t("gifts.owner.declined"));
      return true;
    } catch (e) {
      notifyError(e);
      return false;
    }
  }
  return { decline };
}
