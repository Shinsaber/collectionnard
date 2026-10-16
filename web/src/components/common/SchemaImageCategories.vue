<template>
  <!-- Catégories d'une galerie (`x-image-categories`) : nom, ratio, puis contrôles à l'envoi. -->
  <div class="flex flex-col gap-2">
    <span class="text-xs text-muted">{{
      t("schema.constraints.imageCategories")
    }}</span>
    <div
      v-for="(cat, catIdx) in modelValue"
      :key="catIdx"
      class="flex flex-col gap-2 rounded-xl bg-soft p-2.5"
    >
      <div class="flex items-center gap-2">
        <ElInput
          :model-value="cat.name"
          :placeholder="t('schema.constraints.categoryName')"
          :aria-label="t('schema.constraints.categoryNameN', { n: catIdx + 1 })"
          class="min-w-0 flex-1 text-sm"
          @update:model-value="update(catIdx, { name: String($event ?? '') })"
        />
        <ElInput
          :model-value="cat.ratio"
          :placeholder="t('schema.constraints.categoryRatio')"
          :aria-label="
            t('schema.constraints.categoryRatioN', { n: catIdx + 1 })
          "
          class="w-24! shrink-0 text-sm"
          @update:model-value="update(catIdx, { ratio: String($event ?? '') })"
        />
        <ElButton
          text
          size="small"
          type="danger"
          :aria-label="
            t('schema.constraints.categoryRemoveN', { n: catIdx + 1 })
          "
          @click="remove(catIdx)"
        >
          <i class="pi pi-times" aria-hidden="true"></i>
        </ElButton>
      </div>
      <fieldset class="flex flex-wrap items-end gap-x-3 gap-y-2">
        <legend class="sr-only">
          {{ t("schema.constraints.categoryChecks", { n: catIdx + 1 }) }}
        </legend>
        <label
          v-for="num in NUMBERS"
          :key="num.key"
          class="flex flex-col gap-1 text-[11px] text-muted"
        >
          {{ t(num.labelKey) }}
          <ElInputNumber
            :model-value="cat[num.key]"
            :min="0"
            :max="num.max"
            :controls="false"
            size="small"
            class="w-28!"
            @update:model-value="
              update(catIdx, { [num.key]: $event ?? undefined })
            "
          />
        </label>
        <ElCheckbox
          :model-value="cat.strict ?? false"
          size="small"
          @update:model-value="
            update(catIdx, { strict: !!$event || undefined })
          "
        >
          {{ t("schema.constraints.strict") }}
        </ElCheckbox>
      </fieldset>
    </div>
    <ElButton text size="small" class="self-start" @click="add">
      <i class="pi pi-plus mr-1" aria-hidden="true"></i>
      {{ t("schema.constraints.addCategory") }}
    </ElButton>
    <p class="text-xs text-muted italic">
      {{ t("schema.constraints.checksHelp") }}
    </p>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from "vue-i18n";
import type { ImageCategory } from "@/lib/schema";

const props = defineProps<{ modelValue: ImageCategory[] }>();
const emit = defineEmits<{ "update:modelValue": [value: ImageCategory[]] }>();
const { t } = useI18n();

const NUMBERS = [
  { key: "tolerance", labelKey: "schema.constraints.tolerance", max: 100 },
  { key: "min_width", labelKey: "schema.constraints.minWidth", max: undefined },
  {
    key: "min_height",
    labelKey: "schema.constraints.minHeight",
    max: undefined,
  },
] as const;

function update(catIdx: number, patch: Partial<ImageCategory>) {
  const cats = [...props.modelValue];
  cats[catIdx] = { ...cats[catIdx]!, ...patch };
  emit("update:modelValue", cats);
}

function remove(catIdx: number) {
  const cats = [...props.modelValue];
  cats.splice(catIdx, 1);
  emit("update:modelValue", cats);
}

function add() {
  emit("update:modelValue", [...props.modelValue, { name: "", ratio: "1/1" }]);
}
</script>
