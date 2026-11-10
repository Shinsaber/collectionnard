<template>
  <!-- Réservations d'un lot de la liste de souhaits d'un ami (P7-4) : qui a réservé, combien, et mon bouton.
       L'API ne les donne qu'aux amis du propriétaire, jamais à lui. -->
  <div class="flex flex-wrap items-center gap-2" data-test="reservations">
    <span
      v-if="names"
      class="rounded-full bg-tint px-2 py-0.5 font-mono text-[11px] text-tint-ink"
      data-test="reservation-flag"
      >{{
        t("gifts.flag", { names, n: reservedCopies(lot), total: lot.quantity })
      }}</span
    >
    <ElButton
      v-if="mine || reservableCopies(lot) > 0"
      size="small"
      :type="mine ? 'default' : 'primary'"
      data-test="reserve"
      @click="emit('reserve', lot)"
      >{{ mine ? t("gifts.mine") : t("gifts.reserve") }}</ElButton
    >
    <span v-else class="text-xs text-muted">{{ t("gifts.full") }}</span>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import {
  myReservation,
  personName,
  reservableCopies,
  reservedCopies,
} from "@/lib/gifts";
import type { CollectionItem } from "@/api/types";

const props = defineProps<{ lot: CollectionItem }>();
const emit = defineEmits<{ reserve: [lot: CollectionItem] }>();
const { t } = useI18n();

const mine = computed(() => myReservation(props.lot));
/** Offrants, moi compris (« vous »), dans l'ordre des réservations. */
const names = computed(() =>
  (props.lot.reservations ?? [])
    .map((r) => (r.mine ? t("gifts.you") : personName(r.giver)))
    .join(", "),
);
</script>
