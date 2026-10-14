<template>
  <div
    class="border border-gray-200 dark:border-gray-600 rounded-lg bg-gray-50 dark:bg-gray-800 relative overflow-hidden"
  >
    <!-- Header: reorder, key, type, required, delete -->
    <div
      class="flex items-center gap-2 px-3 py-2 bg-gray-100 dark:bg-gray-700 border-b border-gray-200 dark:border-gray-600"
    >
      <ElButton
        text
        size="small"
        circle
        :aria-label="t('schema.field.moveUp')"
        :disabled="index === 0"
        @click="$emit('move', -1)"
      >
        <i class="pi pi-chevron-up" aria-hidden="true"></i>
      </ElButton>
      <ElButton
        text
        size="small"
        circle
        :aria-label="t('schema.field.moveDown')"
        :disabled="index === count - 1"
        @click="$emit('move', 1)"
      >
        <i class="pi pi-chevron-down" aria-hidden="true"></i>
      </ElButton>

      <ElInput
        :model-value="field.key"
        :placeholder="t('schema.field.keyPlaceholder')"
        :aria-label="t('schema.field.key')"
        size="small"
        class="flex-1"
        @update:model-value="set('key', $event)"
      />

      <ElSelect
        :model-value="field.type"
        :aria-label="t('schema.field.type')"
        size="small"
        class="w-36!"
        @update:model-value="set('type', $event)"
      >
        <ElOption
          v-for="o in typeOptions"
          :key="o.value"
          :label="t(o.labelKey)"
          :value="o.value"
        />
      </ElSelect>

      <ElCheckbox
        :id="`${uid}-req`"
        :model-value="field.required"
        :title="t('schema.field.required')"
        size="small"
        @update:model-value="set('required', $event)"
        >{{ t("schema.field.requiredShort") }}</ElCheckbox
      >

      <ElButton
        text
        type="danger"
        size="small"
        circle
        :aria-label="t('schema.field.remove')"
        @click="$emit('remove')"
      >
        <i class="pi pi-trash" aria-hidden="true"></i>
      </ElButton>

      <ElButton
        text
        size="small"
        circle
        :title="t('schema.field.addAfter')"
        :aria-label="t('schema.field.addAfter')"
        @click="$emit('add-after')"
      >
        <i class="pi pi-plus" aria-hidden="true"></i>
      </ElButton>
    </div>

    <SchemaFieldConstraints
      :field="field"
      :features="features"
      @update:field="$emit('update:field', $event)"
    />
  </div>
</template>

<script setup lang="ts">
import { useId } from "vue";
import { useI18n } from "vue-i18n";
import SchemaFieldConstraints from "./SchemaFieldConstraints.vue";
import type {
  Option,
  SchemaBuilderFeatures,
  SchemaField,
} from "./schemaBuilderTypes";

// Carte d'édition d'un champ : en-tête (ordre, clé, type, requis, actions) et
// contraintes déléguées à SchemaFieldConstraints. Ne mute jamais `field` : chaque
// changement est émis en `update:field` (patch partiel) au parent, qui possède la liste.
defineProps<{
  field: SchemaField;
  index: number;
  count: number;
  features: SchemaBuilderFeatures;
  typeOptions: Option[];
}>();

const emit = defineEmits<{
  "update:field": [patch: Partial<SchemaField>];
  move: [dir: -1 | 1];
  remove: [];
  "add-after": [];
}>();

const { t } = useI18n();
const uid = useId();

function set<K extends keyof SchemaField>(key: K, value: unknown) {
  emit("update:field", { [key]: value } as Partial<SchemaField>);
}
</script>
