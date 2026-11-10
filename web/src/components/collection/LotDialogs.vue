<template>
  <!-- Dialogues d'un lot de ma collection : modifier, scinder, fusionner (P7-1), prêter et prêts (P7-3), « Bien reçu »
       d'un cadeau remis (P7-4b). La page les ouvre par open() ; « Pas reçu » passe par decline(). -->
  <LotEditDialog
    v-model:visible="shown.edit"
    :collection-id="collectionId"
    :schema="schema"
    :lot="lot"
    @saved="emit('changed')"
  />
  <LotSplitDialog
    v-model:visible="shown.split"
    :collection-id="collectionId"
    :schema="schema"
    :lot="lot"
    @done="emit('changed')"
  />
  <LotMergeDialog
    v-model:visible="shown.merge"
    :collection-id="collectionId"
    :schema="schema"
    :lot="lot"
    @done="emit('changed')"
  />
  <LoanDialog
    v-model:visible="shown.lend"
    :collection-id="collectionId"
    :lot="lot"
    :label="label"
    @done="emit('changed')"
  />
  <LotLoansDialog
    v-model:visible="shown.loans"
    :collection-id="collectionId"
    :lot="lot"
    :label="label"
    @changed="emit('changed')"
  />
  <ReceiveGiftDialog
    v-model:visible="shown.receive"
    :collection-id="collectionId"
    :lot="lot"
    :gift="gift"
    :label="label"
    @done="emit('changed')"
  />
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { giftApi } from "@/api/client";
import { useConfirmDialog } from "@/composables/useConfirmDialog";
import { useLang } from "@/composables/useLang";
import { useNotify } from "@/composables/useNotify";
import { personName } from "@/lib/gifts";
import { itemLabel, type Schema } from "@/lib/schema";
import LoanDialog from "./LoanDialog.vue";
import LotEditDialog from "./LotEditDialog.vue";
import LotLoansDialog from "./LotLoansDialog.vue";
import LotMergeDialog from "./LotMergeDialog.vue";
import LotSplitDialog from "./LotSplitDialog.vue";
import ReceiveGiftDialog from "@/components/gifts/ReceiveGiftDialog.vue";
import type { CollectionItem, GiftReservation } from "@/api/types";

const props = defineProps<{
  collectionId: string;
  schema: Schema | undefined;
}>();
/** Un lot a changé : la page recharge sa liste et son compteur. */
const emit = defineEmits<{ changed: [] }>();

const { t } = useI18n();
const { lang } = useLang();
const { notifyError, notifySuccess } = useNotify();
const { confirm } = useConfirmDialog();

const shown = ref({
  edit: false,
  split: false,
  merge: false,
  lend: false,
  loans: false,
  receive: false,
});
type LotDialog = keyof typeof shown.value;

const lot = ref<CollectionItem | null>(null);
/** Cadeau remis visé par « Bien reçu ». */
const gift = ref<GiftReservation | null>(null);
const label = computed(() =>
  lot.value ? itemLabel(lot.value.item, lang.value) : "",
);

function open(kind: LotDialog, ci: CollectionItem, g?: GiftReservation) {
  lot.value = ci;
  gift.value = g ?? null;
  shown.value[kind] = true;
}

/** « Pas reçu » : la réservation est invalidée, l'offrant prévenu. */
async function decline(_ci: CollectionItem, g: GiftReservation) {
  const ok = await confirm({
    message: t("gifts.owner.declineConfirm", { name: personName(g.giver) }),
    confirmText: t("gifts.owner.decline"),
    danger: true,
  });
  if (!ok) return;
  try {
    await giftApi.decline(props.collectionId, g.id);
    notifySuccess(t("gifts.owner.declined"));
    emit("changed");
  } catch (e) {
    notifyError(e);
  }
}

defineExpose({ open, decline });
</script>
