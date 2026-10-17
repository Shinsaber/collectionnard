<template>
  <div class="p-3 grid grid-cols-1 md:grid-cols-2 gap-3">
    <div
      v-for="l in features.labels ? LANGS : []"
      :key="l"
      class="flex flex-col gap-1"
    >
      <label
        :for="`${uid}-label-${l}`"
        class="text-xs font-medium text-muted"
        >{{ t("schema.constraints.label", { lang: l.toUpperCase() }) }}</label
      >
      <ElInput
        :id="`${uid}-label-${l}`"
        :model-value="field.labels?.[l] ?? ''"
        :placeholder="t('schema.constraints.labelPlaceholder')"
        class="text-sm"
        @update:model-value="setLabel(l, String($event ?? ''))"
      />
    </div>

    <div v-if="features.description" class="flex flex-col gap-1 md:col-span-2">
      <label :for="`${uid}-desc`" class="text-xs font-medium text-muted">{{
        t("common.description")
      }}</label>
      <ElInput
        :id="`${uid}-desc`"
        :model-value="field.description ?? ''"
        :placeholder="t('schema.constraints.descriptionPlaceholder')"
        class="text-sm"
        @update:model-value="set('description', $event || null)"
      />
    </div>

    <!-- ── String constraints ── -->
    <template v-if="field.type === 'string'">
      <div v-if="features.minLength" class="flex flex-col gap-1">
        <label :for="`${uid}-minlen`" class="text-xs text-muted">{{
          t("schema.constraints.minLength")
        }}</label>
        <ElInputNumber
          :id="`${uid}-minlen`"
          :model-value="field.minLength ?? undefined"
          :min="0"
          @update:model-value="set('minLength', $event)"
        />
      </div>
      <div v-if="features.maxLength" class="flex flex-col gap-1">
        <label :for="`${uid}-maxlen`" class="text-xs text-muted">{{
          t("schema.constraints.maxLength")
        }}</label>
        <ElInputNumber
          :id="`${uid}-maxlen`"
          :model-value="field.maxLength ?? undefined"
          :min="0"
          @update:model-value="set('maxLength', $event)"
        />
      </div>
      <div v-if="features.pattern" class="flex flex-col gap-1">
        <label :for="`${uid}-pattern`" class="text-xs text-muted">{{
          t("schema.constraints.pattern")
        }}</label>
        <ElInput
          :id="`${uid}-pattern`"
          :model-value="field.pattern ?? ''"
          :placeholder="t('schema.example', { v: '^[A-Z].*' })"
          class="font-mono"
          @update:model-value="set('pattern', $event || null)"
        />
      </div>
      <div v-if="features.format" class="flex flex-col gap-1">
        <label :for="`${uid}-format`" class="text-xs text-muted">{{
          t("schema.constraints.format")
        }}</label>
        <ElSelect
          :id="`${uid}-format`"
          :model-value="field.format ?? undefined"
          :placeholder="t('common.none')"
          clearable
          @update:model-value="set('format', $event || null)"
        >
          <ElOption
            v-for="o in formatOptions"
            :key="o.value"
            :label="t(o.labelKey)"
            :value="o.value"
          />
        </ElSelect>
      </div>
      <div v-if="features.default" class="flex flex-col gap-1">
        <label :for="`${uid}-default`" class="text-xs text-muted">{{
          t("schema.constraints.default")
        }}</label>
        <ElInput
          :id="`${uid}-default`"
          :model-value="String(field.defaultValue ?? '')"
          :placeholder="t('schema.example', { v: 'N/A' })"
          @update:model-value="set('defaultValue', $event ? $event : null)"
        />
      </div>
    </template>

    <!-- ── Enum : chaîne, ou tableau de chaînes (valeurs multiples, ex. rôles) ── -->
    <div
      v-if="features.enum && acceptsEnum"
      class="flex flex-col gap-1 md:col-span-2"
    >
      <label :for="`${uid}-enum`" class="text-xs text-muted">{{
        t("schema.constraints.enum")
      }}</label>
      <ElInput
        :id="`${uid}-enum`"
        :model-value="(field.enum ?? []).join(', ')"
        placeholder="new, good, poor"
        @update:model-value="set('enum', parseEnum(String($event ?? '')))"
      />
      <SchemaEnumLabels
        v-if="features.labels && (field.enum ?? []).length > 0"
        :values="field.enum ?? []"
        :model-value="field.enumLabels ?? {}"
        @update:model-value="set('enumLabels', $event)"
      />
    </div>

    <!-- ── Number / Integer constraints ── -->
    <template v-if="field.type === 'integer' || field.type === 'number'">
      <div v-if="features.minimum" class="flex flex-col gap-1">
        <label :for="`${uid}-min`" class="text-xs text-muted">{{
          t("schema.constraints.minimum")
        }}</label>
        <ElInputNumber
          :id="`${uid}-min`"
          :model-value="field.minimum ?? undefined"
          @update:model-value="set('minimum', $event)"
        />
      </div>
      <div v-if="features.maximum" class="flex flex-col gap-1">
        <label :for="`${uid}-max`" class="text-xs text-muted">{{
          t("schema.constraints.maximum")
        }}</label>
        <ElInputNumber
          :id="`${uid}-max`"
          :model-value="field.maximum ?? undefined"
          @update:model-value="set('maximum', $event)"
        />
      </div>
      <div v-if="features.exclusiveMinimum" class="flex flex-col gap-1">
        <label :for="`${uid}-xmin`" class="text-xs text-muted">{{
          t("schema.constraints.exclusiveMinimum")
        }}</label>
        <ElInputNumber
          :id="`${uid}-xmin`"
          :model-value="field.exclusiveMinimum ?? undefined"
          @update:model-value="set('exclusiveMinimum', $event)"
        />
      </div>
      <div v-if="features.exclusiveMaximum" class="flex flex-col gap-1">
        <label :for="`${uid}-xmax`" class="text-xs text-muted">{{
          t("schema.constraints.exclusiveMaximum")
        }}</label>
        <ElInputNumber
          :id="`${uid}-xmax`"
          :model-value="field.exclusiveMaximum ?? undefined"
          @update:model-value="set('exclusiveMaximum', $event)"
        />
      </div>
      <div v-if="features.multipleOf" class="flex flex-col gap-1">
        <label :for="`${uid}-mult`" class="text-xs text-muted">{{
          t("schema.constraints.multipleOf")
        }}</label>
        <ElInputNumber
          :id="`${uid}-mult`"
          :model-value="field.multipleOf ?? undefined"
          :min-fraction-digits="0"
          :max-fraction-digits="4"
          @update:model-value="set('multipleOf', $event)"
        />
      </div>
      <div v-if="features.default" class="flex flex-col gap-1">
        <label :for="`${uid}-ndefault`" class="text-xs text-muted">{{
          t("schema.constraints.default")
        }}</label>
        <ElInputNumber
          :id="`${uid}-ndefault`"
          :model-value="(field.defaultValue as number | undefined) ?? undefined"
          @update:model-value="set('defaultValue', $event)"
        />
      </div>
    </template>

    <!-- ── Boolean constraints ── -->
    <template v-if="field.type === 'boolean' && features.default">
      <div class="flex flex-col gap-1">
        <label :for="`${uid}-bdefault`" class="text-xs text-muted">{{
          t("schema.constraints.default")
        }}</label>
        <ElSelect
          :id="`${uid}-bdefault`"
          :model-value="
            (field.defaultValue as boolean | undefined) ?? undefined
          "
          :placeholder="t('common.none')"
          clearable
          @update:model-value="
            set('defaultValue', $event === '' ? null : $event)
          "
        >
          <ElOption
            v-for="o in boolDefaultOptions"
            :key="String(o.value)"
            :label="t(o.labelKey)"
            :value="o.value"
          />
        </ElSelect>
      </div>
    </template>

    <!-- ── i18n-text info ── -->
    <div
      v-if="field.type === 'i18n-text'"
      class="md:col-span-2 text-xs text-muted italic"
    >
      {{ t("schema.constraints.i18nTextHelp", { langs: LANGS.join(", ") }) }}
    </div>

    <!-- ── Liens : structure fixe ── -->
    <div
      v-if="field.type === 'links'"
      class="md:col-span-2 text-xs text-muted italic"
    >
      {{ t("schema.constraints.linksHelp") }}
    </div>

    <!-- ── Image gallery config ── -->
    <template v-if="field.type === 'image-gallery'">
      <SchemaImageCategories
        class="md:col-span-2"
        :model-value="field.imageCategories ?? []"
        @update:model-value="set('imageCategories', $event)"
      />
      <div class="md:col-span-2 text-xs text-muted italic">
        {{ t("schema.constraints.imageGalleryHelp") }}
      </div>
    </template>

    <!-- ── Array constraints ── -->
    <template v-if="field.type === 'array'">
      <div class="flex flex-col gap-1">
        <label :for="`${uid}-items`" class="text-xs text-muted">{{
          t("schema.constraints.itemsType")
        }}</label>
        <ElSelect
          :id="`${uid}-items`"
          :model-value="field.itemsType ?? 'string'"
          @update:model-value="set('itemsType', $event)"
        >
          <ElOption
            v-for="o in arrayItemTypeOptions"
            :key="o.value"
            :label="t(o.labelKey)"
            :value="o.value"
          />
        </ElSelect>
      </div>
      <div v-if="features.uniqueItems" class="flex items-center gap-2 pt-5">
        <ElCheckbox
          :id="`${uid}-unique`"
          :model-value="field.uniqueItems ?? false"
          @update:model-value="set('uniqueItems', $event)"
        />
        <label :for="`${uid}-unique`" class="text-xs text-muted">{{
          t("schema.constraints.uniqueItems")
        }}</label>
      </div>
      <div v-if="features.minItems" class="flex flex-col gap-1">
        <label :for="`${uid}-minitems`" class="text-xs text-muted">{{
          t("schema.constraints.minItems")
        }}</label>
        <ElInputNumber
          :id="`${uid}-minitems`"
          :model-value="field.minItems ?? undefined"
          :min="0"
          @update:model-value="set('minItems', $event)"
        />
      </div>
      <div v-if="features.maxItems" class="flex flex-col gap-1">
        <label :for="`${uid}-maxitems`" class="text-xs text-muted">{{
          t("schema.constraints.maxItems")
        }}</label>
        <ElInputNumber
          :id="`${uid}-maxitems`"
          :model-value="field.maxItems ?? undefined"
          :min="0"
          @update:model-value="set('maxItems', $event)"
        />
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, useId } from "vue";
import { useI18n } from "vue-i18n";
import { LANGS } from "@/i18n/langs";
import SchemaEnumLabels from "./SchemaEnumLabels.vue";
import SchemaImageCategories from "./SchemaImageCategories.vue";
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

const { t } = useI18n();
const uid = useId();

function set<K extends keyof SchemaField>(key: K, value: unknown) {
  emit("update:field", { [key]: value } as Partial<SchemaField>);
}

// Valeurs d'enum proposées pour une chaîne et pour un tableau de chaînes (`items.enum`).
const acceptsEnum = computed(
  () =>
    props.field.type === "string" ||
    (props.field.type === "array" &&
      (props.field.itemsType ?? "string") === "string"),
);

function setLabel(l: string, value: string) {
  const labels = { ...(props.field.labels ?? {}), [l]: value };
  if (!value) delete labels[l];
  set("labels", labels);
}
</script>
