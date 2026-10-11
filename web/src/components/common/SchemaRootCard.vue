<template>
  <div
    class="flex flex-col gap-3 border border-gray-200 dark:border-gray-600 rounded-lg px-3 py-2 bg-white dark:bg-gray-800"
  >
    <!-- Nom et description affichés du méta-type (x-title / x-description) -->
    <div v-if="labels" class="grid grid-cols-1 md:grid-cols-2 gap-3">
      <div v-for="l in LANGS" :key="l" class="flex flex-col gap-1">
        <label
          :for="`${uid}-title-${l}`"
          class="text-xs font-medium text-gray-500 dark:text-gray-400"
          >{{ t("schema.root.title", { lang: l.toUpperCase() }) }}</label
        >
        <ElInput
          :id="`${uid}-title-${l}`"
          :model-value="title[l] ?? ''"
          :placeholder="t('schema.root.titlePlaceholder')"
          @update:model-value="patch('title', l, String($event ?? ''))"
        />
        <label
          :for="`${uid}-desc-${l}`"
          class="text-xs font-medium text-gray-500 dark:text-gray-400"
          >{{ t("schema.root.description", { lang: l.toUpperCase() }) }}</label
        >
        <ElInput
          :id="`${uid}-desc-${l}`"
          :model-value="description[l] ?? ''"
          @update:model-value="patch('description', l, String($event ?? ''))"
        />
      </div>
      <span
        class="md:col-span-2 text-[11px] text-gray-400 dark:text-gray-500"
        >{{ t("schema.root.titleHelp") }}</span
      >
    </div>

    <!-- Gabarit du libellé des instances (x-label) -->
    <div class="flex flex-col gap-1">
      <label
        :for="`${uid}-xlabel`"
        class="text-xs font-medium text-gray-500 dark:text-gray-400"
        >{{ t("schema.root.xLabel") }} (<code>x-label</code>)</label
      >
      <ElInput
        :id="`${uid}-xlabel`"
        :model-value="xLabel"
        :placeholder="t('schema.root.xLabelPlaceholder')"
        class="font-mono"
        @update:model-value="emit('update:xLabel', String($event ?? ''))"
      />
      <span class="text-[11px] text-gray-400 dark:text-gray-500">{{
        t("schema.root.xLabelHelp")
      }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useId } from "vue";
import { useI18n } from "vue-i18n";
import { LANGS } from "@/i18n/langs";

// En-tête du SchemaBuilder : extensions racine du schéma — nom et description
// affichés du méta-type par langue (`x-title`, `x-description`) et gabarit `x-label`.
// Les langues hors `LANGS` présentes dans les objets sont conservées par le parent.
const props = defineProps<{
  xLabel: string;
  title: Record<string, string>;
  description: Record<string, string>;
  /** Affiche les saisies par langue (feature `labels` du builder). */
  labels: boolean;
}>();

const emit = defineEmits<{
  "update:xLabel": [value: string];
  "update:title": [value: Record<string, string>];
  "update:description": [value: Record<string, string>];
}>();

const { t } = useI18n();
const uid = useId();

function patch(which: "title" | "description", l: string, value: string) {
  const next = { ...props[which], [l]: value };
  if (!value) delete next[l];
  if (which === "title") emit("update:title", next);
  else emit("update:description", next);
}
</script>
