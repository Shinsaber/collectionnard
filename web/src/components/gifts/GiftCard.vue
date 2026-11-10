<template>
  <!-- Un cadeau que je compte offrir (P7-4) : pour qui, combien, jusqu'à quand ; remis (P7-4b), reçu, ou retiré. -->
  <li
    class="flex flex-col gap-2 rounded-xl bg-soft px-3 py-2.5 sm:flex-row sm:items-center"
    data-test="gift"
  >
    <div class="flex min-w-0 flex-1 items-center gap-3">
      <ItemCover
        v-if="gift.item"
        :item="gift.item"
        :label="itemLabel(gift.item, lang)"
        :width="40"
      />
      <div class="flex min-w-0 flex-1 flex-col gap-0.5">
        <RouterLink
          v-if="gift.item"
          :to="`/items/${gift.item_id}`"
          class="truncate font-bold text-ink hover:text-link"
          >{{ itemLabel(gift.item, lang) }}</RouterLink
        >
        <span class="text-sm">
          <RouterLink
            v-if="gift.owner"
            :to="`/u/${gift.owner.username}`"
            class="text-link"
            >{{ t("gifts.for", { name: personName(gift.owner) }) }}</RouterLink
          >
          <span v-if="gift.quantity > 1" class="font-mono text-xs text-muted">
            · ×{{ gift.quantity }}</span
          >
          <span v-if="reserved" class="font-mono text-[11px] text-muted">
            ·
            {{ t("gifts.untilOn", { date: formatDay(gift.expires_at, lang) }) }}
          </span>
        </span>
        <span
          v-if="reserved && gift.left_list"
          class="self-start rounded-full bg-danger/10 px-2 py-0.5 font-mono text-[11px] text-danger"
          :title="t('gifts.leftHelp')"
          data-test="gift-left"
          >{{ t("gifts.leftList") }}</span
        >
        <span
          v-else-if="gift.expired"
          class="self-start rounded-full bg-danger/10 px-2 py-0.5 font-mono text-[11px] text-danger"
          :title="
            t('gifts.expiredHelp', { date: formatDay(gift.expires_at, lang) })
          "
          data-test="gift-expired"
          >{{ t("gifts.expired") }}</span
        >
        <span
          v-else-if="gift.status === 'given'"
          class="self-start rounded-full bg-tint px-2 py-0.5 font-mono text-[11px] text-tint-ink"
          data-test="gift-given"
          >{{ t("gifts.given") }}</span
        >
        <span
          v-else-if="gift.status === 'received' && gift.received_at"
          class="self-start rounded-full bg-tint px-2 py-0.5 font-mono text-[11px] text-tint-ink"
          data-test="gift-received"
          >{{
            t("gifts.received", {
              date: new Date(gift.received_at).toLocaleDateString(lang),
            })
          }}</span
        >
        <p v-if="gift.note" class="text-xs text-muted">{{ gift.note }}</p>
      </div>
    </div>
    <div class="flex flex-wrap items-center gap-2 sm:justify-end">
      <template v-if="reserved">
        <ElInput
          v-if="!gift.left_list"
          :model-value="gift.expires_at"
          type="date"
          size="small"
          class="w-40!"
          :min="today()"
          :max="maxExpiry()"
          :aria-label="t('gifts.until')"
          data-test="gift-until"
          @change="(v: string) => v && extend(v)"
        />
        <ElButton
          size="small"
          type="primary"
          data-test="gift-hand-over"
          @click="handOver"
          >{{ t("gifts.handOver") }}</ElButton
        >
      </template>
      <ElButton
        text
        size="small"
        :aria-label="reserved ? t('gifts.cancel') : t('gifts.remove')"
        data-test="gift-cancel"
        @click="remove"
      >
        <i class="pi pi-trash" aria-hidden="true"></i>
      </ElButton>
    </div>
  </li>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { giftApi } from "@/api/client";
import { useConfirmDialog } from "@/composables/useConfirmDialog";
import { useLang } from "@/composables/useLang";
import { useNotify } from "@/composables/useNotify";
import { maxExpiry, personName } from "@/lib/gifts";
import { formatDay, today } from "@/lib/loans";
import { itemLabel } from "@/lib/schema";
import ItemCover from "@/components/item/ItemCover.vue";
import type { GiftReservation } from "@/api/types";

const props = defineProps<{ gift: GiftReservation }>();
const emit = defineEmits<{ changed: [] }>();

const { t } = useI18n();
const { lang } = useLang();
const { notifyError, notifySuccess } = useNotify();
const { confirm } = useConfirmDialog();

/** Encore secrète : se prolonge, se remet ; une fois remise, elle ne se modifie plus. */
const reserved = computed(() => props.gift.status === "reserved");

async function run(action: () => Promise<unknown>, done: string) {
  try {
    await action();
    notifySuccess(t(done));
    emit("changed");
  } catch (e) {
    notifyError(e);
  }
}

/** Nouvelle échéance (PUT), le reste de la réservation gardé. */
const extend = (day: string) =>
  run(
    () =>
      giftApi.update(props.gift.id, {
        quantity: props.gift.quantity,
        expires_at: day,
        note: props.gift.note ?? "",
      }),
    "gifts.extended",
  );

/** « Je l'ai remis » : la surprise est levée, le propriétaire est prévenu. */
async function handOver() {
  const ok = await confirm({
    message: t("gifts.handOverConfirm"),
    confirmText: t("gifts.handOver"),
  });
  if (ok) await run(() => giftApi.handOver(props.gift.id), "gifts.handedOver");
}

/** Annuler une réservation encore secrète, ou retirer un cadeau remis ou reçu. */
async function remove() {
  const ok = await confirm({
    message: reserved.value
      ? t("gifts.cancelConfirm")
      : t("gifts.removeConfirm"),
    confirmText: reserved.value ? t("gifts.cancel") : t("gifts.remove"),
    danger: true,
  });
  if (ok)
    await run(
      () => giftApi.cancel(props.gift.id),
      reserved.value ? "gifts.cancelled" : "gifts.removed",
    );
}
</script>
