<template>
  <div class="flex flex-col gap-4">
    <!-- Root: item label template (x-label) -->
    <div
      class="flex flex-col gap-1 border border-gray-200 dark:border-gray-600 rounded-lg px-3 py-2 bg-white dark:bg-gray-800"
    >
      <label
        :for="`${uid}-xlabel`"
        class="text-xs font-medium text-gray-500 dark:text-gray-400"
        >Libellé des instances (<code>x-label</code>)</label
      >
      <ElInput
        :id="`-xlabel`"
        v-model="xLabel"
        placeholder="ex. {title} ou {number}. {title}"
        class="font-mono"
      />
      <span class="text-[11px] text-gray-400 dark:text-gray-500">
        Gabarit avec les clés des champs entre accolades ; sans gabarit, le
        premier champ <code>title</code>, <code>name</code> ou
        <code>label</code>
        est utilisé.
      </span>
    </div>

    <!-- Field list -->
    <SchemaFieldCard
      v-for="(field, idx) in fields"
      :key="idx"
      :field="field"
      :index="idx"
      :count="fields.length"
      :features="ft"
      :type-options="availableTypeOptions"
      @update:field="patchField(idx, $event)"
      @move="move(idx, $event)"
      @remove="fields.splice(idx, 1)"
      @add-after="fields.splice(idx + 1, 0, emptyField())"
    />

    <!-- Empty state -->
    <div
      v-if="fields.length === 0"
      class="text-sm text-gray-400 italic text-center py-4 border border-dashed border-gray-300 rounded-lg"
    >
      Aucun champ défini. Cliquez sur « Ajouter un champ » pour commencer.
    </div>

    <!-- Actions -->
    <div class="flex items-center gap-2">
      <ElButton size="small" plain @click="fields.push(emptyField())">
        <i class="pi pi-plus mr-2" aria-hidden="true"></i> Ajouter un champ
      </ElButton>
      <div class="flex-1"></div>
      <ElButton
        size="small"
        text
        :aria-expanded="showRaw"
        @click="showRaw = !showRaw"
      >
        <i
          :class="showRaw ? 'pi pi-eye-slash' : 'pi pi-eye'"
          class="mr-2"
          aria-hidden="true"
        ></i>
        {{ showRaw ? "Masquer JSON" : "Voir JSON" }}
      </ElButton>
    </div>

    <!-- Raw JSON preview -->
    <div v-if="showRaw" class="flex flex-col gap-1">
      <ElInput
        :model-value="JSON.stringify(schema, null, 2)"
        type="textarea"
        readonly
        :rows="12"
        aria-label="JSON Schema généré"
        class="font-mono text-xs"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, useId, watch, type PropType } from "vue";
import SchemaFieldCard from "./SchemaFieldCard.vue";
import {
  MANAGED_PROPERTY_KEYS,
  MANAGED_ROOT_KEYS,
  arrayTypeOption,
  baseTypeOptions,
  defaultFeatures,
  emptyField,
  i18nTypeOption,
  imageGalleryTypeOption,
  parseImageCategories,
  type SchemaBuilderFeatures,
  type SchemaField,
} from "./schemaBuilderTypes";

// Éditeur visuel d'un JSON Schema d'attributs : liste de champs ↔ schéma.
// Les mots-clés qu'il ne connaît pas (propriété ou racine) sont conservés.

const props = defineProps({
  modelValue: {
    type: Object as PropType<Record<string, unknown>>,
    required: true,
  },
  features: {
    type: Object as PropType<Partial<SchemaBuilderFeatures>>,
    default: () => ({}),
  },
});

const emit = defineEmits<{
  (e: "update:modelValue", value: Record<string, unknown>): void;
}>();

const uid = useId();

const ft = computed<SchemaBuilderFeatures>(() => ({
  ...defaultFeatures,
  ...props.features,
}));

const availableTypeOptions = computed(() => {
  const opts = [...baseTypeOptions];
  if (ft.value.i18nText) opts.push(i18nTypeOption);
  if (ft.value.imageGallery) opts.push(imageGalleryTypeOption);
  if (ft.value.array) opts.push(arrayTypeOption);
  return opts;
});

const fields = ref<SchemaField[]>([]);
const showRaw = ref(false);
// Gabarit du libellé des instances (extension racine x-label), ex. "{number}. {title}".
const xLabel = ref("");
// Mots-clés racine inconnus, restitués à l'identique.
const rootExtra = ref<Record<string, unknown>>({});

// ── Schema → fields ─────────────────────────────────────────────────
function parseSchema(schema: Record<string, unknown>): SchemaField[] {
  const properties = (schema.properties ?? {}) as Record<
    string,
    Record<string, unknown>
  >;
  const required = (schema.required ?? []) as string[];
  const xLabels = (schema["x-labels"] ?? {}) as Record<
    string,
    Record<string, string>
  >;
  const xOrder = (schema["x-order"] ?? []) as string[];
  const labelsFr = xLabels["fr"] ?? {};
  const labelsEn = xLabels["en"] ?? {};

  const keys =
    xOrder.length > 0
      ? [
          ...xOrder,
          ...Object.keys(properties).filter((k) => !xOrder.includes(k)),
        ]
      : Object.keys(properties);

  return keys
    .filter((key) => key in properties)
    .map((key) => {
      const prop = properties[key]!;
      let type = (prop.type as string) ?? "string";
      if (type === "object" && prop["x-widget"] === "i18n-text") {
        type = "i18n-text";
      }
      if (type === "object" && prop["x-widget"] === "image-gallery") {
        type = "image-gallery";
      }
      return {
        key,
        type,
        labelFr: labelsFr[key] ?? "",
        labelEn: labelsEn[key] ?? "",
        required: required.includes(key),
        description: (prop.description as string) ?? null,
        defaultValue: prop.default ?? null,
        minLength: (prop.minLength as number) ?? null,
        maxLength: (prop.maxLength as number) ?? null,
        pattern: (prop.pattern as string) ?? null,
        format: (prop.format as string) ?? null,
        enum: (prop.enum as string[]) ?? null,
        minimum: (prop.minimum as number) ?? null,
        maximum: (prop.maximum as number) ?? null,
        exclusiveMinimum: (prop.exclusiveMinimum as number) ?? null,
        exclusiveMaximum: (prop.exclusiveMaximum as number) ?? null,
        multipleOf: (prop.multipleOf as number) ?? null,
        itemsType:
          ((prop.items as Record<string, unknown>)?.type as string) ?? null,
        minItems: (prop.minItems as number) ?? null,
        maxItems: (prop.maxItems as number) ?? null,
        uniqueItems: (prop.uniqueItems as boolean) ?? null,
        imageCategories:
          type === "image-gallery"
            ? parseImageCategories(prop["x-image-categories"])
            : null,
        extra: Object.fromEntries(
          Object.entries(prop).filter(([k]) => !MANAGED_PROPERTY_KEYS.has(k)),
        ),
      };
    });
}

// ── Fields → schema ─────────────────────────────────────────────────
function buildProperty(f: SchemaField): Record<string, unknown> {
  const prop: Record<string, unknown> = { ...(f.extra ?? {}), type: f.type };
  if (f.description) prop.description = f.description;

  if (f.type === "string") {
    if (f.minLength != null && f.minLength > 0) prop.minLength = f.minLength;
    if (f.maxLength != null && f.maxLength > 0) prop.maxLength = f.maxLength;
    if (f.pattern) prop.pattern = f.pattern;
    if (f.format) prop.format = f.format;
    if (f.enum && f.enum.length > 0) prop.enum = f.enum;
    if (f.defaultValue != null) prop.default = f.defaultValue;
  }
  if (f.type === "integer" || f.type === "number") {
    if (f.minimum != null) prop.minimum = f.minimum;
    if (f.maximum != null) prop.maximum = f.maximum;
    if (f.exclusiveMinimum != null) prop.exclusiveMinimum = f.exclusiveMinimum;
    if (f.exclusiveMaximum != null) prop.exclusiveMaximum = f.exclusiveMaximum;
    if (f.multipleOf != null) prop.multipleOf = f.multipleOf;
    if (f.defaultValue != null) prop.default = f.defaultValue;
  }
  if (f.type === "boolean" && f.defaultValue != null) {
    prop.default = f.defaultValue;
  }
  if (f.type === "i18n-text") {
    prop.type = "object";
    prop["x-widget"] = "i18n-text";
    prop.properties = { fr: { type: "string" }, en: { type: "string" } };
    prop.additionalProperties = false;
  }
  if (f.type === "image-gallery") {
    const cats = (
      f.imageCategories ?? [{ name: "poster", ratio: "1/1" }]
    ).filter((c) => c.name);
    prop.type = "object";
    prop["x-widget"] = "image-gallery";
    prop["x-image-categories"] = Object.fromEntries(
      cats.map((c) => [c.name, { ratio: c.ratio || "1/1" }]),
    );
    const catProperties: Record<string, unknown> = {
      default_image: { type: "string" },
    };
    for (const c of cats) {
      catProperties[c.name] = { type: "array", items: { type: "string" } };
    }
    prop.properties = catProperties;
    prop.additionalProperties = false;
  }
  if (f.type === "array") {
    prop.items = { type: f.itemsType ?? "string" };
    if (f.minItems != null) prop.minItems = f.minItems;
    if (f.maxItems != null) prop.maxItems = f.maxItems;
    if (f.uniqueItems) prop.uniqueItems = true;
  }
  return prop;
}

const schema = computed<Record<string, unknown>>(() => {
  const properties: Record<string, Record<string, unknown>> = {};
  const requiredKeys: string[] = [];
  const labelsFr: Record<string, string> = {};
  const labelsEn: Record<string, string> = {};

  const named = fields.value.filter((f) => f.key.trim());
  for (const f of named) {
    properties[f.key] = buildProperty(f);
    if (f.required) requiredKeys.push(f.key);
    if (f.labelFr) labelsFr[f.key] = f.labelFr;
    if (f.labelEn) labelsEn[f.key] = f.labelEn;
  }

  const result: Record<string, unknown> = {
    ...rootExtra.value,
    type: "object",
    properties,
    additionalProperties: false,
    "x-order": named.map((f) => f.key),
  };
  if (xLabel.value.trim()) result["x-label"] = xLabel.value.trim();
  if (requiredKeys.length > 0) result.required = requiredKeys;
  const xLabels: Record<string, Record<string, string>> = {};
  if (Object.keys(labelsFr).length > 0) xLabels.fr = labelsFr;
  if (Object.keys(labelsEn).length > 0) xLabels.en = labelsEn;
  if (Object.keys(xLabels).length > 0) result["x-labels"] = xLabels;
  return result;
});

// Sync schema to parent on change
watch(
  schema,
  (val) => {
    emit("update:modelValue", val);
  },
  { deep: true },
);

// ── Init / reset ────────────────────────────────────────────────────
let initialized = false;

function loadFrom(value: Record<string, unknown>) {
  fields.value = parseSchema(value);
  xLabel.value = typeof value["x-label"] === "string" ? value["x-label"] : "";
  rootExtra.value = Object.fromEntries(
    Object.entries(value ?? {}).filter(([k]) => !MANAGED_ROOT_KEYS.has(k)),
  );
}

watch(
  () => props.modelValue,
  (val) => {
    if (!initialized) {
      loadFrom(val);
      initialized = true;
    }
  },
  { immediate: true },
);

/** Réinitialise l'éditeur depuis un schéma (ouverture d'un autre type). */
function resetFromSchema(value: Record<string, unknown>) {
  loadFrom(value);
  initialized = true;
}
defineExpose({ resetFromSchema });

// ── Field list operations ───────────────────────────────────────────
function patchField(idx: number, patch: Partial<SchemaField>) {
  fields.value[idx] = { ...fields.value[idx]!, ...patch };
}

function move(idx: number, dir: -1 | 1) {
  const target = idx + dir;
  if (target < 0 || target >= fields.value.length) return;
  const arr = fields.value;
  [arr[idx], arr[target]] = [arr[target]!, arr[idx]!];
}
</script>
