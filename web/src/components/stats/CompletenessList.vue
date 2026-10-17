<template>
  <section
    class="flex flex-col gap-3 rounded-2xl bg-surface p-5 shadow-card"
    data-test="completeness"
  >
    <div class="flex flex-col gap-1">
      <h2 class="text-base font-extrabold">
        {{ t("stats.completeness.title") }}
      </h2>
      <p class="text-sm text-muted">{{ t("stats.completeness.help") }}</p>
    </div>
    <ul class="flex flex-col gap-3">
      <li
        v-for="c in shown"
        :key="c.item.id"
        class="flex flex-col gap-1"
        data-test="completeness-row"
      >
        <div class="flex items-baseline justify-between gap-3">
          <RouterLink
            :to="`/items/${c.item.id}`"
            class="min-w-0 truncate font-semibold text-link"
            >{{ label(c) }}</RouterLink
          >
          <span class="shrink-0 font-mono text-xs text-muted">
            <template v-if="c.owned >= c.total">{{
              t("stats.completeness.complete")
            }}</template>
            <template v-else>{{
              t("stats.completeness.count", { owned: c.owned, total: c.total })
            }}</template>
          </span>
        </div>
        <ElProgress
          :percentage="percent(c.owned, c.total)"
          :show-text="false"
          :stroke-width="8"
          :color="c.owned >= c.total ? 'var(--c-accent)' : 'var(--c-primary)'"
          :aria-label="
            t('stats.completeness.count', { owned: c.owned, total: c.total })
          "
        />
      </li>
    </ul>
    <ElButton
      v-if="entries.length > LIMIT"
      text
      class="self-start"
      @click="all = !all"
    >
      {{
        all
          ? t("stats.completeness.showLess")
          : t("stats.completeness.showAll", { n: entries.length })
      }}
    </ElButton>
    <p v-if="truncated" class="text-xs text-muted">
      {{ t("stats.completeness.truncated") }}
    </p>
  </section>
</template>

<script setup lang="ts">
// Complétude par le graphe : « Dragon Ball (Glénat) — 30 sur 42 », barre de progression.
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useLang } from "@/composables/useLang";
import { useMetaTypes } from "@/stores/metaTypes";
import { itemLabel } from "@/lib/schema";
import { percent } from "@/lib/stats";
import type { Completeness } from "@/api/types";

/** Nombre d'ensembles montrés avant « Tout afficher ». */
const LIMIT = 10;

const props = defineProps<{ entries: Completeness[]; truncated?: boolean }>();
const { t } = useI18n();
const { lang } = useLang();
const metaTypes = useMetaTypes();
const all = ref(false);

const shown = computed(() =>
  all.value ? props.entries : props.entries.slice(0, LIMIT),
);

function label(c: Completeness): string {
  const type = metaTypes.itemTypes.find((it) => it.id === c.item.item_type_id);
  return itemLabel({ ...c.item, item_type: type }, lang.value);
}
</script>
