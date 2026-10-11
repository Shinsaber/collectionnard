<template>
  <div class="p-3 grid grid-cols-1 md:grid-cols-2 gap-3">
    <div v-if="features.labels" class="flex flex-col gap-1">
      <label :for="`${uid}-fr`" class="text-xs font-medium text-gray-500"
        >Label FR</label
      >
      <InputText
        :id="`${uid}-fr`"
        :model-value="field.labelFr"
        placeholder="ex: Titre"
        class="text-sm"
        @update:model-value="set('labelFr', $event)"
      />
    </div>
    <div v-if="features.labels" class="flex flex-col gap-1">
      <label :for="`${uid}-en`" class="text-xs font-medium text-gray-500"
        >Label EN</label
      >
      <InputText
        :id="`${uid}-en`"
        :model-value="field.labelEn"
        placeholder="ex: Title"
        class="text-sm"
        @update:model-value="set('labelEn', $event)"
      />
    </div>

    <div v-if="features.description" class="flex flex-col gap-1 md:col-span-2">
      <label :for="`${uid}-desc`" class="text-xs font-medium text-gray-500"
        >Description</label
      >
      <InputText
        :id="`${uid}-desc`"
        :model-value="field.description ?? ''"
        placeholder="Description du champ"
        class="text-sm"
        @update:model-value="set('description', $event || null)"
      />
    </div>

    <!-- ── String constraints ── -->
    <template v-if="field.type === 'string'">
      <div v-if="features.minLength" class="flex flex-col gap-1">
        <label :for="`${uid}-minlen`" class="text-xs text-gray-500"
          >Min. caractères</label
        >
        <InputNumber
          :input-id="`${uid}-minlen`"
          :model-value="field.minLength ?? null"
          :min="0"
          :use-grouping="false"
          class="text-sm"
          @update:model-value="set('minLength', $event)"
        />
      </div>
      <div v-if="features.maxLength" class="flex flex-col gap-1">
        <label :for="`${uid}-maxlen`" class="text-xs text-gray-500"
          >Max. caractères</label
        >
        <InputNumber
          :input-id="`${uid}-maxlen`"
          :model-value="field.maxLength ?? null"
          :min="0"
          :use-grouping="false"
          class="text-sm"
          @update:model-value="set('maxLength', $event)"
        />
      </div>
      <div v-if="features.pattern" class="flex flex-col gap-1">
        <label :for="`${uid}-pattern`" class="text-xs text-gray-500"
          >Pattern (regex)</label
        >
        <InputText
          :id="`${uid}-pattern`"
          :model-value="field.pattern ?? ''"
          placeholder="ex: ^[A-Z].*"
          class="text-sm font-mono"
          @update:model-value="set('pattern', $event || null)"
        />
      </div>
      <div v-if="features.format" class="flex flex-col gap-1">
        <label :for="`${uid}-format`" class="text-xs text-gray-500"
          >Format</label
        >
        <Select
          :input-id="`${uid}-format`"
          :model-value="field.format ?? null"
          :options="formatOptions"
          option-label="label"
          option-value="value"
          placeholder="Aucun"
          show-clear
          class="text-sm"
          @update:model-value="set('format', $event)"
        />
      </div>
      <div v-if="features.enum" class="flex flex-col gap-1 md:col-span-2">
        <label :for="`${uid}-enum`" class="text-xs text-gray-500"
          >Valeurs (enum, séparées par des virgules)</label
        >
        <InputText
          :id="`${uid}-enum`"
          :model-value="(field.enum ?? []).join(', ')"
          placeholder="new, good, poor"
          class="text-sm"
          @update:model-value="set('enum', parseEnum(String($event ?? '')))"
        />
      </div>
      <div v-if="features.default" class="flex flex-col gap-1">
        <label :for="`${uid}-default`" class="text-xs text-gray-500"
          >Valeur par défaut</label
        >
        <InputText
          :id="`${uid}-default`"
          :model-value="String(field.defaultValue ?? '')"
          placeholder="ex: N/A"
          class="text-sm"
          @update:model-value="set('defaultValue', $event ? $event : null)"
        />
      </div>
    </template>

    <!-- ── Number / Integer constraints ── -->
    <template v-if="field.type === 'integer' || field.type === 'number'">
      <div v-if="features.minimum" class="flex flex-col gap-1">
        <label :for="`${uid}-min`" class="text-xs text-gray-500">Min</label>
        <InputNumber
          :input-id="`${uid}-min`"
          :model-value="field.minimum ?? null"
          :use-grouping="false"
          class="text-sm"
          @update:model-value="set('minimum', $event)"
        />
      </div>
      <div v-if="features.maximum" class="flex flex-col gap-1">
        <label :for="`${uid}-max`" class="text-xs text-gray-500">Max</label>
        <InputNumber
          :input-id="`${uid}-max`"
          :model-value="field.maximum ?? null"
          :use-grouping="false"
          class="text-sm"
          @update:model-value="set('maximum', $event)"
        />
      </div>
      <div v-if="features.exclusiveMinimum" class="flex flex-col gap-1">
        <label :for="`${uid}-xmin`" class="text-xs text-gray-500"
          >Min exclusif</label
        >
        <InputNumber
          :input-id="`${uid}-xmin`"
          :model-value="field.exclusiveMinimum ?? null"
          :use-grouping="false"
          class="text-sm"
          @update:model-value="set('exclusiveMinimum', $event)"
        />
      </div>
      <div v-if="features.exclusiveMaximum" class="flex flex-col gap-1">
        <label :for="`${uid}-xmax`" class="text-xs text-gray-500"
          >Max exclusif</label
        >
        <InputNumber
          :input-id="`${uid}-xmax`"
          :model-value="field.exclusiveMaximum ?? null"
          :use-grouping="false"
          class="text-sm"
          @update:model-value="set('exclusiveMaximum', $event)"
        />
      </div>
      <div v-if="features.multipleOf" class="flex flex-col gap-1">
        <label :for="`${uid}-mult`" class="text-xs text-gray-500"
          >Multiple de</label
        >
        <InputNumber
          :input-id="`${uid}-mult`"
          :model-value="field.multipleOf ?? null"
          :use-grouping="false"
          :min-fraction-digits="0"
          :max-fraction-digits="4"
          class="text-sm"
          @update:model-value="set('multipleOf', $event)"
        />
      </div>
      <div v-if="features.default" class="flex flex-col gap-1">
        <label :for="`${uid}-ndefault`" class="text-xs text-gray-500"
          >Valeur par défaut</label
        >
        <InputNumber
          :input-id="`${uid}-ndefault`"
          :model-value="(field.defaultValue as number | null) ?? null"
          :use-grouping="false"
          class="text-sm"
          @update:model-value="set('defaultValue', $event)"
        />
      </div>
    </template>

    <!-- ── Boolean constraints ── -->
    <template v-if="field.type === 'boolean' && features.default">
      <div class="flex flex-col gap-1">
        <label :for="`${uid}-bdefault`" class="text-xs text-gray-500"
          >Valeur par défaut</label
        >
        <Select
          :input-id="`${uid}-bdefault`"
          :model-value="field.defaultValue ?? null"
          :options="boolDefaultOptions"
          option-label="label"
          option-value="value"
          placeholder="Aucune"
          show-clear
          class="text-sm"
          @update:model-value="set('defaultValue', $event)"
        />
      </div>
    </template>

    <!-- ── i18n-text info ── -->
    <div
      v-if="field.type === 'i18n-text'"
      class="md:col-span-2 text-xs text-gray-400 italic"
    >
      Génère un objet
      <code class="bg-gray-200 px-1 rounded">{ "fr": "…", "en": "…" }</code>
      — les utilisateurs saisiront une valeur par langue.
    </div>

    <!-- ── Image gallery config ── -->
    <template v-if="field.type === 'image-gallery'">
      <div class="flex flex-col gap-2 md:col-span-2">
        <span class="text-xs text-gray-500">Catégories d'images</span>
        <div
          v-for="(cat, catIdx) in field.imageCategories ?? []"
          :key="catIdx"
          class="flex items-center gap-2"
        >
          <InputText
            :model-value="cat.name"
            placeholder="Nom (ex: poster)"
            :aria-label="`Nom de la catégorie ${catIdx + 1}`"
            class="text-sm flex-1"
            @update:model-value="
              updateCategory(catIdx, 'name', String($event ?? ''))
            "
          />
          <InputText
            :model-value="cat.ratio"
            placeholder="Ratio (ex: 2/3)"
            :aria-label="`Ratio de la catégorie ${catIdx + 1}`"
            class="text-sm w-24"
            @update:model-value="
              updateCategory(catIdx, 'ratio', String($event ?? ''))
            "
          />
          <Button
            icon="pi pi-times"
            text
            size="small"
            severity="danger"
            :aria-label="`Supprimer la catégorie ${catIdx + 1}`"
            @click="removeCategory(catIdx)"
          />
        </div>
        <Button
          label="Ajouter une catégorie"
          icon="pi pi-plus"
          text
          size="small"
          class="self-start"
          @click="addCategory"
        />
      </div>
      <div class="md:col-span-2 text-xs text-gray-400 italic">
        Génère un objet avec un tableau d'URLs par catégorie et un sélecteur
        d'image par défaut. Le ratio est au format CSS (ex: 2/3, 16/9, 3/1).
      </div>
    </template>

    <!-- ── Array constraints ── -->
    <template v-if="field.type === 'array'">
      <div class="flex flex-col gap-1">
        <label :for="`${uid}-items`" class="text-xs text-gray-500"
          >Type des éléments</label
        >
        <Select
          :input-id="`${uid}-items`"
          :model-value="field.itemsType ?? 'string'"
          :options="arrayItemTypeOptions"
          option-label="label"
          option-value="value"
          class="text-sm"
          @update:model-value="set('itemsType', $event)"
        />
      </div>
      <div v-if="features.uniqueItems" class="flex items-center gap-2 pt-5">
        <Checkbox
          :model-value="field.uniqueItems ?? false"
          :binary="true"
          :input-id="`${uid}-unique`"
          @update:model-value="set('uniqueItems', $event)"
        />
        <label :for="`${uid}-unique`" class="text-xs text-gray-500"
          >Éléments uniques</label
        >
      </div>
      <div v-if="features.minItems" class="flex flex-col gap-1">
        <label :for="`${uid}-minitems`" class="text-xs text-gray-500"
          >Min. éléments</label
        >
        <InputNumber
          :input-id="`${uid}-minitems`"
          :model-value="field.minItems ?? null"
          :min="0"
          :use-grouping="false"
          class="text-sm"
          @update:model-value="set('minItems', $event)"
        />
      </div>
      <div v-if="features.maxItems" class="flex flex-col gap-1">
        <label :for="`${uid}-maxitems`" class="text-xs text-gray-500"
          >Max. éléments</label
        >
        <InputNumber
          :input-id="`${uid}-maxitems`"
          :model-value="field.maxItems ?? null"
          :min="0"
          :use-grouping="false"
          class="text-sm"
          @update:model-value="set('maxItems', $event)"
        />
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { useId } from "vue";
import Button from "primevue/button";
import InputText from "primevue/inputtext";
import InputNumber from "primevue/inputnumber";
import Select from "primevue/select";
import Checkbox from "primevue/checkbox";
import {
  arrayItemTypeOptions,
  boolDefaultOptions,
  formatOptions,
  parseEnum,
  type SchemaBuilderFeatures,
  type SchemaField,
} from "./schemaBuilderTypes";

// Corps d'une carte de champ : libellés, description et contraintes selon le type.
// Chaque changement est émis en `update:field` (patch partiel).
const props = defineProps<{
  field: SchemaField;
  features: SchemaBuilderFeatures;
}>();

const emit = defineEmits<{
  "update:field": [patch: Partial<SchemaField>];
}>();

const uid = useId();

function set<K extends keyof SchemaField>(key: K, value: unknown) {
  emit("update:field", { [key]: value } as Partial<SchemaField>);
}

function updateCategory(catIdx: number, key: "name" | "ratio", value: string) {
  const cats = [...(props.field.imageCategories ?? [])];
  cats[catIdx] = { ...cats[catIdx]!, [key]: value };
  set("imageCategories", cats);
}

function removeCategory(catIdx: number) {
  const cats = [...(props.field.imageCategories ?? [])];
  cats.splice(catIdx, 1);
  set("imageCategories", cats);
}

function addCategory() {
  set("imageCategories", [
    ...(props.field.imageCategories ?? []),
    { name: "", ratio: "1/1" },
  ]);
}
</script>
