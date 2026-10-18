<template>
  <div
    class="@container flex flex-col gap-4 rounded-2xl bg-surface p-4.5 shadow-card"
  >
    <h2 class="text-base font-extrabold">{{ t("schema.root.heading") }}</h2>
    <!-- Nom et description affichés du méta-type (x-title / x-description) -->
    <div v-if="labels" class="grid grid-cols-1 gap-3 @md:grid-cols-2">
      <div v-for="l in LANGS" :key="l" class="flex flex-col gap-1">
        <label
          :for="`${uid}-title-${l}`"
          class="text-[13px] font-bold text-ink"
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
          class="text-[13px] font-bold text-ink"
          >{{ t("schema.root.description", { lang: l.toUpperCase() }) }}</label
        >
        <ElInput
          :id="`${uid}-desc-${l}`"
          :model-value="description[l] ?? ''"
          @update:model-value="patch('description', l, String($event ?? ''))"
        />
      </div>
      <span class="text-xs text-muted @md:col-span-2">{{
        t("schema.root.titleHelp")
      }}</span>
    </div>

    <!-- Gabarit du libellé des instances (x-label) -->
    <div class="flex flex-col gap-1">
      <label :for="`${uid}-xlabel`" class="text-[13px] font-bold text-ink"
        >{{ t("schema.root.xLabel") }} (<code>x-label</code>)</label
      >
      <ElInput
        :id="`${uid}-xlabel`"
        :model-value="xLabel"
        :placeholder="t('schema.root.xLabelPlaceholder')"
        class="font-mono"
        @update:model-value="emit('update:xLabel', String($event ?? ''))"
      />
      <span class="text-xs text-muted">{{ t("schema.root.xLabelHelp") }}</span>
    </div>

    <!-- Relation de composition (x-completeness), types de relation seulement -->
    <div v-if="showCompleteness" class="flex flex-col gap-1">
      <ElCheckbox
        :model-value="completeness"
        data-test="x-completeness"
        @update:model-value="emit('update:completeness', Boolean($event))"
      >
        {{ t("schema.root.completeness") }} (<code>x-completeness</code>)
      </ElCheckbox>
      <span class="text-xs text-muted">{{
        t("schema.root.completenessHelp")
      }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useId } from "vue";
import { useI18n } from "vue-i18n";
import { LANGS } from "@/i18n/langs";

// En-tête du SchemaBuilder : extensions racine du schéma — nom et description
// affichés du méta-type par langue (`x-title`, `x-description`), gabarit `x-label` et,
// pour un type de relation, la composition `x-completeness`.
// Les langues hors `LANGS` présentes dans les objets sont conservées par le parent.
const props = defineProps<{
  xLabel: string;
  title: Record<string, string>;
  description: Record<string, string>;
  /** Affiche les saisies par langue (feature `labels` du builder). */
  labels: boolean;
  completeness?: boolean;
  /** Affiche la case x-completeness (feature `completeness` : types de relation). */
  showCompleteness?: boolean;
}>();

const emit = defineEmits<{
  "update:xLabel": [value: string];
  "update:title": [value: Record<string, string>];
  "update:description": [value: Record<string, string>];
  "update:completeness": [value: boolean];
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
