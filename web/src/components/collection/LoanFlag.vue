<template>
  <!-- Drapeau « prêté » d'un lot (P7-3) : vu de son propriétaire seulement (l'API ne joint les prêts qu'à sa liste). -->
  <span
    v-if="lot.loans?.length"
    class="self-start rounded-full px-2 py-0.5 font-mono text-[11px]"
    :class="overdue ? 'bg-danger/10 text-danger' : 'bg-tint text-tint-ink'"
    :title="overdue ? t('loans.overdue') : undefined"
    data-test="loan-flag"
  >
    <i
      v-if="overdue"
      class="pi pi-exclamation-triangle mr-1 text-[10px]"
      aria-hidden="true"
    ></i
    >{{ t("loans.flag", { n: lent, who }, lent) }}
  </span>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { borrowerName, lentCopies } from "@/lib/loans";
import type { CollectionItem } from "@/api/types";

const props = defineProps<{ lot: CollectionItem }>();
const { t } = useI18n();

const lent = computed(() => lentCopies(props.lot));
/** Emprunteurs distincts, dans l'ordre des prêts. */
const who = computed(() =>
  [...new Set((props.lot.loans ?? []).map(borrowerName))].join(", "),
);
const overdue = computed(() => (props.lot.loans ?? []).some((l) => l.overdue));
</script>
