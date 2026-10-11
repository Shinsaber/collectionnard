<template>
  <div
    class="border border-gray-200 dark:border-gray-600 rounded-lg bg-gray-50 dark:bg-gray-800 relative overflow-hidden"
  >
    <!-- Header: reorder, key, type, required, delete -->
    <div
      class="flex items-center gap-2 px-3 py-2 bg-gray-100 dark:bg-gray-700 border-b border-gray-200 dark:border-gray-600"
    >
      <Button
        icon="pi pi-chevron-up"
        text
        size="small"
        rounded
        aria-label="Monter le champ"
        :disabled="index === 0"
        class="!w-6 !h-6"
        @click="$emit('move', -1)"
      />
      <Button
        icon="pi pi-chevron-down"
        text
        size="small"
        rounded
        aria-label="Descendre le champ"
        :disabled="index === count - 1"
        class="!w-6 !h-6"
        @click="$emit('move', 1)"
      />

      <InputText
        :model-value="field.key"
        placeholder="clé (ex: title)"
        aria-label="Clé du champ"
        class="text-sm flex-1"
        @update:model-value="set('key', $event)"
      />

      <Select
        :model-value="field.type"
        :options="typeOptions"
        option-label="label"
        option-value="value"
        aria-label="Type du champ"
        class="text-sm w-32"
        @update:model-value="set('type', $event)"
      />

      <div v-tooltip="'Champ requis'" class="flex items-center gap-1">
        <Checkbox
          :model-value="field.required"
          :binary="true"
          :input-id="`${uid}-req`"
          @update:model-value="set('required', $event)"
        />
        <label :for="`${uid}-req`" class="text-xs text-gray-500">Req.</label>
      </div>

      <Button
        icon="pi pi-trash"
        text
        severity="danger"
        size="small"
        rounded
        aria-label="Supprimer le champ"
        class="!w-7 !h-7"
        @click="$emit('remove')"
      />

      <Button
        v-tooltip="'Ajouter un champ après'"
        icon="pi pi-plus"
        text
        size="small"
        rounded
        aria-label="Ajouter un champ après"
        class="!w-7 !h-7"
        @click="$emit('add-after')"
      />
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
import Button from "primevue/button";
import InputText from "primevue/inputtext";
import Select from "primevue/select";
import Checkbox from "primevue/checkbox";
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

const uid = useId();

function set<K extends keyof SchemaField>(key: K, value: unknown) {
  emit("update:field", { [key]: value } as Partial<SchemaField>);
}
</script>
