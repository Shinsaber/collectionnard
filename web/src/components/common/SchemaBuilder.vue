<template>
  <!-- Champs à gauche ; identité du type (slot `aside`), libellé et JSON à droite. -->
  <div class="flex flex-wrap items-start gap-6">
    <section class="flex min-w-0 flex-[999_1_560px] flex-col gap-3">
      <h2 class="text-lg font-extrabold tracking-tight sm:text-xl">
        {{ t("schema.builder.fields") }}
        <span class="font-mono text-[13px] font-medium text-muted"
          >· {{ fields.length }}</span
        >
      </h2>

      <ul
        v-if="fields.length"
        class="overflow-hidden rounded-2xl bg-surface shadow-card"
      >
        <SchemaFieldCard
          v-for="(field, idx) in fields"
          :key="ids[idx]"
          :field="field"
          :index="idx"
          :count="fields.length"
          :features="ft"
          :type-options="availableTypeOptions"
          :open="expanded.has(ids[idx]!)"
          @toggle="toggle(ids[idx]!)"
          @update:field="patchField(idx, $event)"
          @move="move(idx, $event)"
          @remove="removeField(idx)"
          @add-after="addField(idx + 1)"
        />
      </ul>
      <p
        v-else
        class="rounded-2xl bg-surface px-5 py-6 text-center text-sm italic text-muted shadow-card"
      >
        {{ t("schema.builder.empty") }}
      </p>

      <button
        type="button"
        class="flex h-13 items-center justify-center gap-2 rounded-[14px] border-[1.5px] border-dashed border-dash text-[15px] font-semibold text-link hover:border-primary"
        @click="addField(fields.length)"
      >
        <i class="pi pi-plus text-sm" aria-hidden="true"></i>
        {{ t("schema.builder.addField") }}
      </button>
    </section>

    <aside
      class="flex min-w-0 flex-[1_1_320px] flex-col gap-4 lg:sticky lg:top-6"
    >
      <slot name="aside" />

      <!-- Nom affiché, description (x-title / x-description) et libellé des instances (x-label) -->
      <SchemaRootCard
        v-model:x-label="xLabel"
        v-model:title="xTitle"
        v-model:description="xDescription"
        v-model:completeness="xCompleteness"
        :labels="ft.labels"
        :show-completeness="ft.completeness"
      />

      <!-- JSON Schema généré (lecture seule) -->
      <div class="flex flex-col gap-2 rounded-2xl bg-surface p-4 shadow-card">
        <ElButton
          text
          class="self-start"
          :aria-expanded="showRaw"
          @click="showRaw = !showRaw"
        >
          <i
            :class="showRaw ? 'pi pi-eye-slash' : 'pi pi-code'"
            class="mr-2"
            aria-hidden="true"
          ></i>
          {{
            showRaw
              ? t("schema.builder.hideJson")
              : t("schema.builder.showJson")
          }}
        </ElButton>
        <ElInput
          v-if="showRaw"
          :model-value="JSON.stringify(schema, null, 2)"
          type="textarea"
          readonly
          :rows="14"
          :aria-label="t('schema.builder.generatedJson')"
          class="font-mono text-xs"
        />
      </div>
    </aside>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch, type PropType } from "vue";
import { useI18n } from "vue-i18n";
import { LANGS } from "@/i18n/langs";
import { isCompleteness, linksProperty } from "@/lib/schema";
import SchemaFieldCard from "./SchemaFieldCard.vue";
import SchemaRootCard from "./SchemaRootCard.vue";
import {
  MANAGED_PROPERTY_KEYS,
  MANAGED_ROOT_KEYS,
  LANG_CODE_PATTERN,
  arrayTypeOption,
  baseTypeOptions,
  cleanEnumLabels,
  column,
  defaultFeatures,
  emptyField,
  i18nTypeOption,
  imageGalleryTypeOption,
  linksTypeOption,
  parseImageCategories,
  stringMap,
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

const { t } = useI18n();

const ft = computed<SchemaBuilderFeatures>(() => ({
  ...defaultFeatures,
  ...props.features,
}));

const availableTypeOptions = computed(() => {
  const opts = [...baseTypeOptions];
  if (ft.value.i18nText) opts.push(i18nTypeOption);
  if (ft.value.imageGallery) opts.push(imageGalleryTypeOption);
  if (ft.value.links) opts.push(linksTypeOption);
  if (ft.value.array) opts.push(arrayTypeOption);
  return opts;
});

const fields = ref<SchemaField[]>([]);
// Identifiant stable de chaque champ (clé de rendu, état déplié), aligné sur `fields`.
const ids = ref<number[]>([]);
let nextId = 1;
const expanded = ref(new Set<number>());
const showRaw = ref(false);
// Gabarit du libellé des instances (extension racine x-label), ex. "{number}. {title}".
const xLabel = ref("");
// Nom et description affichés du méta-type, par langue (extensions racine x-title / x-description).
const xTitle = ref<Record<string, string>>({});
const xDescription = ref<Record<string, string>>({});
// Relation de composition (x-completeness) : compte des cibles possédées dans les statistiques.
const xCompleteness = ref(false);
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
      if (type === "array" && prop["x-widget"] === "links") {
        type = "links";
      }
      return {
        key,
        type,
        labels: column(xLabels, key),
        required: required.includes(key),
        description: (prop.description as string) ?? null,
        defaultValue: prop.default ?? null,
        minLength: (prop.minLength as number) ?? null,
        maxLength: (prop.maxLength as number) ?? null,
        pattern: (prop.pattern as string) ?? null,
        format: (prop.format as string) ?? null,
        // Tableau d'enum (ex. rôles) : les valeurs vivent dans `items.enum`, les libellés
        // (`x-enum-labels`) sur la propriété, comme pour un enum simple.
        enum:
          (prop.enum as string[]) ??
          (type === "array"
            ? ((prop.items as Record<string, unknown>)?.enum as string[])
            : undefined) ??
          null,
        enumLabels:
          (prop["x-enum-labels"] as Record<string, Record<string, string>>) ??
          null,
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
        private: prop["x-private"] === true,
        identifier: prop["x-identifier"] === true,
        extra: preservedKeys(prop, type),
        itemsExtra:
          type === "array"
            ? Object.fromEntries(
                Object.entries(
                  (prop.items ?? {}) as Record<string, unknown>,
                ).filter(([k]) => k !== "type" && k !== "enum"),
              )
            : null,
      };
    });
}

// Mots-clés conservés tels quels : ceux que le builder ne pilote pas, plus la structure
// d'un objet générique (sans widget), qu'il n'édite pas (ex. `external_ids`).
function preservedKeys(
  prop: Record<string, unknown>,
  type: string,
): Record<string, unknown> {
  const keep = (k: string) =>
    !MANAGED_PROPERTY_KEYS.has(k) ||
    (type === "object" &&
      ["properties", "additionalProperties", "propertyNames"].includes(k));
  return Object.fromEntries(Object.entries(prop).filter(([k]) => keep(k)));
}

// ── Fields → schema ─────────────────────────────────────────────────
function buildProperty(f: SchemaField): Record<string, unknown> {
  const prop: Record<string, unknown> = { ...(f.extra ?? {}), type: f.type };
  if (f.description) prop.description = f.description;
  if (f.private) prop["x-private"] = true;
  if (f.identifier) prop["x-identifier"] = true;

  if (f.type === "string") {
    if (f.minLength != null && f.minLength > 0) prop.minLength = f.minLength;
    if (f.maxLength != null && f.maxLength > 0) prop.maxLength = f.maxLength;
    if (f.pattern) prop.pattern = f.pattern;
    if (f.format) prop.format = f.format;
    if (f.enum && f.enum.length > 0) {
      prop.enum = f.enum;
      const enumLabels = cleanEnumLabels(f.enumLabels, f.enum);
      if (enumLabels) prop["x-enum-labels"] = enumLabels;
    }
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
  if (f.type === "links") {
    // Tableau de { label, url } : structure fixe (lib/schema.ts), seules les bornes du tableau varient.
    Object.assign(prop, linksProperty());
    if (f.minItems != null && f.minItems > 0) prop.minItems = f.minItems;
    if (f.maxItems != null && f.maxItems > 0) prop.maxItems = f.maxItems;
  }
  if (f.type === "i18n-text") {
    prop.type = "object";
    prop["x-widget"] = "i18n-text";
    // Une chaîne par langue connue ; d'autres codes de langue restent acceptés
    // (ajout futur d'une langue sans migrer les schémas existants).
    prop.properties = Object.fromEntries(
      LANGS.map((l) => [l, { type: "string" }]),
    );
    prop.propertyNames = { pattern: LANG_CODE_PATTERN };
    prop.additionalProperties = { type: "string" };
  }
  if (f.type === "image-gallery") {
    const cats = (
      f.imageCategories ?? [{ name: "poster", ratio: "1/1" }]
    ).filter((c) => c.name);
    prop.type = "object";
    prop["x-widget"] = "image-gallery";
    // Contrôles à l'envoi : seulement les valeurs renseignées (tolerance, strict, min_width…).
    prop["x-image-categories"] = Object.fromEntries(
      cats.map(({ name, ratio, ...checks }) => [
        name,
        {
          ratio: ratio || "1/1",
          ...Object.fromEntries(
            Object.entries(checks).filter(
              ([k, v]) =>
                v !== undefined && v !== null && (k !== "strict" || v),
            ),
          ),
        },
      ]),
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
    // `items` complet (ex. tableau d'objets `sources`) : seuls `type` et `enum` sont édités.
    const items: Record<string, unknown> = {
      ...(f.itemsExtra ?? {}),
      type: f.itemsType ?? "string",
    };
    if (f.enum && f.enum.length > 0) {
      items.enum = f.enum;
      const enumLabels = cleanEnumLabels(f.enumLabels, f.enum);
      if (enumLabels) prop["x-enum-labels"] = enumLabels;
    }
    prop.items = items;
    if (f.minItems != null) prop.minItems = f.minItems;
    if (f.maxItems != null) prop.maxItems = f.maxItems;
    if (f.uniqueItems) prop.uniqueItems = true;
  }
  return prop;
}

const schema = computed<Record<string, unknown>>(() => {
  const properties: Record<string, Record<string, unknown>> = {};
  const requiredKeys: string[] = [];
  const xLabels: Record<string, Record<string, string>> = {};

  const named = fields.value.filter((f) => f.key.trim());
  for (const f of named) {
    properties[f.key] = buildProperty(f);
    if (f.required) requiredKeys.push(f.key);
    for (const [l, text] of Object.entries(f.labels ?? {})) {
      if (text) (xLabels[l] ??= {})[f.key] = text;
    }
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
  if (Object.keys(xLabels).length > 0) result["x-labels"] = xLabels;
  if (Object.keys(xTitle.value).length > 0) result["x-title"] = xTitle.value;
  if (Object.keys(xDescription.value).length > 0)
    result["x-description"] = xDescription.value;
  if (xCompleteness.value) result["x-completeness"] = true;
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
  ids.value = fields.value.map(() => nextId++);
  expanded.value = new Set();
  xLabel.value = typeof value["x-label"] === "string" ? value["x-label"] : "";
  xTitle.value = stringMap(value["x-title"]);
  xDescription.value = stringMap(value["x-description"]);
  xCompleteness.value = isCompleteness(value);
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
  const id = ids.value;
  [id[idx], id[target]] = [id[target]!, id[idx]!];
}

/** Nouveau champ vide à la position donnée, déplié pour être saisi. */
function addField(at: number) {
  const id = nextId++;
  fields.value.splice(at, 0, emptyField());
  ids.value.splice(at, 0, id);
  expanded.value = new Set(expanded.value).add(id);
}

function removeField(idx: number) {
  fields.value.splice(idx, 1);
  ids.value.splice(idx, 1);
}

function toggle(id: number) {
  const next = new Set(expanded.value);
  if (next.has(id)) next.delete(id);
  else next.add(id);
  expanded.value = next;
}
</script>
