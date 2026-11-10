<template>
  <!-- Un cadeau que je compte offrir (P7-4) : pour qui, combien ; offert, ou annulé. -->
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
          <span
            v-if="gift.status === 'reserved'"
            class="font-mono text-[11px] text-muted"
          >
            ·
            {{ t("gifts.untilOn", { date: formatDay(gift.expires_at, lang) }) }}
          </span>
        </span>
        <span
          v-if="gift.left_list && gift.status === 'reserved'"
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
          >{{ t("gifts.given") }}</span
        >
        <p v-if="gift.note" class="text-xs text-muted">{{ gift.note }}</p>
      </div>
    </div>
    <div class="flex flex-wrap items-center gap-2 sm:justify-end">
      <ElInput
        v-if="gift.status === 'reserved' && !gift.left_list"
        :model-value="gift.expires_at"
        type="date"
        size="small"
        class="w-40!"
        :min="today()"
        :max="maxExpiry()"
        :aria-label="t('gifts.until')"
        data-test="gift-until"
        @change="(v: string) => v && save({ expires_at: v }, 'gifts.extended')"
      />
      <ElButton
        size="small"
        :type="gift.status === 'reserved' ? 'primary' : 'default'"
        data-test="gift-given"
        @click="toggleGiven"
        >{{
          gift.status === "reserved"
            ? t("gifts.markGiven")
            : t("gifts.markReserved")
        }}</ElButton
      >
      <ElButton
        text
        size="small"
        :aria-label="t('gifts.cancel')"
        data-test="gift-cancel"
        @click="cancel"
      >
        <i class="pi pi-trash" aria-hidden="true"></i>
      </ElButton>
    </div>
  </li>
</template>

<script setup lang="ts">
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

/** Remplace les champs de ma réservation (PUT), en gardant ceux qu'on ne touche pas. */
async function save(
  change: { status?: GiftReservation["status"]; expires_at?: string },
  done = "gifts.saved",
) {
  const g = props.gift;
  try {
    await giftApi.update(g.id, {
      quantity: g.quantity,
      status: g.status,
      expires_at: g.expires_at,
      note: g.note ?? "",
      ...change,
    });
    notifySuccess(t(done));
    emit("changed");
  } catch (e) {
    notifyError(e);
  }
}

const toggleGiven = () =>
  save({ status: props.gift.status === "reserved" ? "given" : "reserved" });

async function cancel() {
  const ok = await confirm({
    message: t("gifts.cancelConfirm"),
    confirmText: t("gifts.cancel"),
    danger: true,
  });
  if (!ok) return;
  try {
    await giftApi.cancel(props.gift.id);
    notifySuccess(t("gifts.cancelled"));
    emit("changed");
  } catch (e) {
    notifyError(e);
  }
}
</script>
