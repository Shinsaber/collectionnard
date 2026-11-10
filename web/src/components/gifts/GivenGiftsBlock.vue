<template>
  <!-- Cadeaux remis sur un lot de ma liste (P7-4b) : la surprise est levée, je confirme ou je refuse. L'API ne
       joint au propriétaire que les réservations remises, jamais celles encore secrètes. -->
  <ul class="flex flex-col gap-1.5" data-test="given-gifts">
    <li
      v-for="g in lot.gifts"
      :key="g.id"
      class="flex flex-wrap items-center gap-2"
      data-test="given-gift"
    >
      <span
        class="rounded-full bg-tint px-2 py-0.5 font-mono text-[11px] text-tint-ink"
        >{{
          t(
            "gifts.owner.flag",
            { n: g.quantity, names: personName(g.giver) },
            g.quantity,
          )
        }}</span
      >
      <ElButton
        size="small"
        type="primary"
        data-test="gift-receive"
        @click="emit('receive', g)"
        >{{ t("gifts.owner.receive") }}</ElButton
      >
      <ElButton
        size="small"
        text
        data-test="gift-decline"
        @click="emit('decline', g)"
        >{{ t("gifts.owner.decline") }}</ElButton
      >
    </li>
  </ul>
</template>

<script setup lang="ts">
import { useI18n } from "vue-i18n";
import { personName } from "@/lib/gifts";
import type { CollectionItem, GiftReservation } from "@/api/types";

defineProps<{ lot: CollectionItem }>();
const emit = defineEmits<{
  receive: [gift: GiftReservation];
  decline: [gift: GiftReservation];
}>();
const { t } = useI18n();
</script>
