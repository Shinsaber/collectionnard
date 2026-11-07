<template>
  <span
    class="inline-flex items-center gap-1.5"
    role="img"
    :aria-label="t('reviews.outOfTen', { rating: label })"
    data-test="review-stars"
  >
    <span class="relative inline-block whitespace-nowrap leading-none">
      <span class="flex text-dash" aria-hidden="true">
        <svg
          v-for="n in 5"
          :key="n"
          viewBox="0 0 24 24"
          :width="size"
          :height="size"
          fill="currentColor"
        >
          <path :d="STAR" />
        </svg>
      </span>
      <span
        class="absolute inset-y-0 left-0 flex overflow-hidden"
        :style="{ width: `${fill}%`, color }"
        aria-hidden="true"
        data-test="review-stars-fill"
      >
        <svg
          v-for="n in 5"
          :key="n"
          viewBox="0 0 24 24"
          :width="size"
          :height="size"
          class="shrink-0"
          fill="currentColor"
        >
          <path :d="STAR" />
        </svg>
      </span>
    </span>
    <span
      v-if="showValue"
      class="min-w-[2ch] text-right font-mono text-xs text-muted tabular-nums"
      >{{ label }}</span
    >
  </span>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useLang } from "@/composables/useLang";
import {
  STAR_COLOR_MODE,
  ratingColor,
  starsFill,
  type StarColorMode,
} from "@/lib/reviews";

const props = withDefaults(
  defineProps<{
    rating: number;
    size?: number;
    showValue?: boolean;
    /** Mode de couleur ; par défaut celui de l'application (`STAR_COLOR_MODE`). */
    mode?: StarColorMode;
  }>(),
  { size: 16, showValue: true, mode: STAR_COLOR_MODE },
);

const STAR =
  "M12 2l3.1 6.3 6.9 1-5 4.9 1.2 6.8-6.2-3.2-6.2 3.2L7 14.2l-5-4.9 6.9-1z";

const { t } = useI18n();
const { lang } = useLang();
const fill = computed(() => starsFill(props.rating));
const color = computed(() => ratingColor(props.rating, props.mode));
const label = computed(() =>
  props.rating.toLocaleString(lang.value, { maximumFractionDigits: 1 }),
);
</script>
