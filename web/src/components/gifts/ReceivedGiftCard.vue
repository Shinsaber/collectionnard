<template>
  <!-- Un cadeau qu'on m'a offert (P7-4c) : remis à confirmer (« Bien reçu » / « Pas reçu »), ou reçu. -->
  <li
    class="flex flex-col gap-2 rounded-xl bg-soft px-3 py-2.5 sm:flex-row sm:items-center"
    data-test="received-gift"
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
          {{ t("gifts.received.by", { name: personName(gift.giver) }) }}
          <span v-if="gift.quantity > 1" class="font-mono text-xs text-muted">
            · ×{{ gift.quantity }}</span
          >
        </span>
        <span class="font-mono text-[11px] text-muted">{{ when }}</span>
      </div>
    </div>
    <div
      v-if="gift.status === 'given' && gift.collection_id"
      class="flex flex-wrap items-center gap-2 sm:justify-end"
    >
      <ElButton
        size="small"
        type="primary"
        data-test="gift-receive"
        @click="emit('receive', gift)"
        >{{ t("gifts.owner.receive") }}</ElButton
      >
      <ElButton
        size="small"
        text
        data-test="gift-decline"
        @click="emit('decline', gift)"
        >{{ t("gifts.owner.decline") }}</ElButton
      >
    </div>
  </li>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useLang } from "@/composables/useLang";
import { personName } from "@/lib/gifts";
import { itemLabel } from "@/lib/schema";
import ItemCover from "@/components/item/ItemCover.vue";
import type { GiftReservation } from "@/api/types";

const props = defineProps<{ gift: GiftReservation }>();
const emit = defineEmits<{
  receive: [gift: GiftReservation];
  decline: [gift: GiftReservation];
}>();
const { t } = useI18n();
const { lang } = useLang();

const day = (iso?: string) =>
  iso ? new Date(iso).toLocaleDateString(lang.value) : "";
/** « Remis le …, à confirmer » ou « Reçu le … ». */
const when = computed(() =>
  props.gift.status === "given"
    ? t("gifts.received.pending", { date: day(props.gift.given_at) })
    : t("gifts.received.on", { date: day(props.gift.received_at) }),
);
</script>
