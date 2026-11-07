<template>
  <div
    class="flex flex-wrap gap-1.5"
    role="group"
    :aria-label="t('reviews.reactions.label')"
    data-test="review-reactions"
  >
    <template v-for="r in shown" :key="r.kind">
      <button
        v-if="interactive"
        type="button"
        class="inline-flex h-8 items-center gap-1.5 rounded-full border px-2.5 text-[13px] transition-colors"
        :class="
          r.mine
            ? 'border-primary bg-tint font-semibold text-tint-ink'
            : 'border-line text-muted hover:bg-soft'
        "
        :aria-pressed="r.mine"
        :title="t(`reviews.reactions.${r.kind}`)"
        :disabled="busy"
        :data-test="`reaction-${r.kind}`"
        @click="toggle(r.kind)"
      >
        <span aria-hidden="true">{{ ICONS[r.kind] }}</span>
        {{ t(`reviews.reactions.${r.kind}`) }}
        <span v-if="r.count > 0" class="font-mono text-xs">{{ r.count }}</span>
      </button>
      <span
        v-else-if="r.count > 0"
        class="inline-flex h-7 items-center gap-1.5 rounded-full bg-soft px-2.5 text-[13px] text-muted"
        :title="t(`reviews.reactions.${r.kind}`)"
        :data-test="`reaction-${r.kind}`"
      >
        <span aria-hidden="true">{{ ICONS[r.kind] }}</span>
        <span class="sr-only">{{ t(`reviews.reactions.${r.kind}`) }}</span>
        <span class="font-mono text-xs">{{ r.count }}</span>
      </span>
    </template>
  </div>
</template>

<script setup lang="ts">
// Réactions à un avis (I-1) : une intention ou une émotion, jamais un signalement. Elles se cumulent ;
// en lecture seule (son propre avis, page publique), seules celles reçues s'affichent.
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { reviewApi } from "@/api/client";
import { useNotify } from "@/composables/useNotify";
import { REACTION_KINDS, type ReactionKind } from "@/lib/reviews";
import type { Review } from "@/api/types";

const ICONS: Record<ReactionKind, string> = {
  interesting: "💡",
  tempting: "✨",
  funny: "😄",
  disapprove: "👎",
};

const props = defineProps<{ review: Review; interactive: boolean }>();

const { t } = useI18n();
const { notifyError } = useNotify();
const counts = ref<Record<string, number>>({});
const mine = ref<string[]>([]);
const busy = ref(false);

watch(
  () => props.review,
  (r) => {
    counts.value = { ...(r.reactions ?? {}) };
    mine.value = [...(r.my_reactions ?? [])];
  },
  { immediate: true },
);

const shown = computed(() =>
  REACTION_KINDS.map((kind) => ({
    kind,
    count: counts.value[kind] ?? 0,
    mine: mine.value.includes(kind),
  })),
);

async function toggle(kind: ReactionKind) {
  const on = !mine.value.includes(kind);
  busy.value = true;
  try {
    await reviewApi.react(props.review.id, kind, on);
    counts.value[kind] = Math.max(0, (counts.value[kind] ?? 0) + (on ? 1 : -1));
    mine.value = on
      ? [...mine.value, kind]
      : mine.value.filter((k) => k !== kind);
  } catch (e) {
    notifyError(e);
  } finally {
    busy.value = false;
  }
}
</script>
