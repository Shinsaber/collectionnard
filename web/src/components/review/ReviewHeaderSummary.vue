<template>
  <button
    type="button"
    class="flex flex-wrap items-center gap-x-3 gap-y-1.5 self-start rounded-xl bg-bar px-3 py-2 text-left text-sm transition-colors hover:bg-soft"
    :aria-label="t('reviews.header.open')"
    data-test="review-header"
    @click="emit('open')"
  >
    <span v-if="summary.rating != null" class="flex items-center gap-2">
      <ReviewStars :rating="summary.rating" :size="18" :show-value="false" />
      <span class="font-bold" data-test="review-header-average">{{
        t("reviews.header.average", { rating: format(summary.rating) })
      }}</span>
      <span class="text-muted">{{ t("reviews.count", summary.count) }}</span>
    </span>
    <span v-else class="text-muted" data-test="review-header-count">
      {{
        summary.count > 0
          ? t("reviews.header.notEnough", {
              count: t("reviews.count", summary.count),
              min: summary.min_ratings,
            })
          : t("reviews.header.none")
      }}
    </span>
    <span
      v-if="mine"
      class="flex items-center gap-1.5 border-line sm:border-l sm:pl-3"
      data-test="review-header-mine"
    >
      <span class="text-muted">{{ t("reviews.header.mine") }}</span>
      <ReviewStars :rating="mine.rating" :size="14" />
    </span>
    <span
      v-else
      class="font-semibold text-link"
      data-test="review-header-write"
    >
      {{ t("reviews.write") }}
    </span>
  </button>
</template>

<script setup lang="ts">
// Les notes dans le premier bloc de la fiche : moyenne (au-delà du seuil), nombre de notes et la mienne ;
// un clic ouvre l'onglet « Avis ».
import { useI18n } from "vue-i18n";
import { useLang } from "@/composables/useLang";
import ReviewStars from "./ReviewStars.vue";
import type { Review, ReviewSummary } from "@/api/types";

defineProps<{ summary: ReviewSummary; mine: Review | null }>();
const emit = defineEmits<{ open: [] }>();

const { t } = useI18n();
const { lang } = useLang();
const format = (v: number) =>
  v.toLocaleString(lang.value, { maximumFractionDigits: 1 });
</script>
