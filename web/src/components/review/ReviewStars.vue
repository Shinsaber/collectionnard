<template>
  <span
    class="inline-flex items-center gap-1.5"
    role="img"
    :aria-label="t('reviews.outOfTen', { rating: label })"
    data-test="review-stars"
  >
    <span class="relative inline-block whitespace-nowrap leading-none">
      <span class="flex text-line" aria-hidden="true">
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
        :class="TONES[tone]"
        :style="{ width: `${fill}%` }"
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
    <span v-if="showValue" class="font-mono text-xs text-muted">{{
      label
    }}</span>
  </span>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useLang } from "@/composables/useLang";
import { ratingTone, starsFill } from "@/lib/reviews";

const props = withDefaults(
  defineProps<{ rating: number; size?: number; showValue?: boolean }>(),
  { size: 16, showValue: true },
);

const STAR =
  "M12 2l3.1 6.3 6.9 1-5 4.9 1.2 6.8-6.2-3.2-6.2 3.2L7 14.2l-5-4.9 6.9-1z";
const TONES = {
  danger: "text-danger",
  accent: "text-accent",
  primary: "text-primary",
};

const { t } = useI18n();
const { lang } = useLang();
const fill = computed(() => starsFill(props.rating));
const tone = computed(() => ratingTone(props.rating));
const label = computed(() =>
  props.rating.toLocaleString(lang.value, { maximumFractionDigits: 1 }),
);
</script>
