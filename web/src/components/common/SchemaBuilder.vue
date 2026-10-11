<template>
  <div class="flex flex-col gap-4">
    <!-- Root: item label template (x-label) -->
    <div
      class="flex flex-col gap-1 border border-gray-200 dark:border-gray-600 rounded-lg px-3 py-2 bg-white dark:bg-gray-800"
    >
      <label class="text-xs font-medium text-gray-500 dark:text-gray-400"
        >Libellé des instances (<code>x-label</code>)</label
      >
      <InputText
        v-model="xLabel"
        placeholder="ex. {title} ou {number}. {title}"
        class="text-sm font-mono"
      />
      <span class="text-[11px] text-gray-400 dark:text-gray-500">
        Gabarit avec les clés des champs entre accolades ; sans gabarit, le
        premier champ <code>title</code>, <code>name</code> ou
        <code>label</code>
        est utilisé.
      </span>
    </div>

    <!-- Field list -->
    <div
      v-for="(field, idx) in fields"
      :key="idx"
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
          :disabled="idx === 0"
          class="!w-6 !h-6"
          @click="move(idx, -1)"
        />
        <Button
          icon="pi pi-chevron-down"
          text
          size="small"
          rounded
          :disabled="idx === fields.length - 1"
          class="!w-6 !h-6"
          @click="move(idx, 1)"
        />

        <InputText
          :model-value="field.key"
          placeholder="clé (ex: titre)"
          class="text-sm flex-1"
          @update:model-value="updateField(idx, 'key', $event)"
        />

        <Select
          :model-value="field.type"
          :options="availableTypeOptions"
          option-label="label"
          option-value="value"
          class="text-sm w-32"
          @update:model-value="updateField(idx, 'type', $event)"
        />

        <div v-tooltip="'Champ requis'" class="flex items-center gap-1">
          <Checkbox
            :model-value="field.required"
            :binary="true"
            :input-id="`req-${idx}`"
            @update:model-value="updateField(idx, 'required', $event)"
          />
          <label :for="`req-${idx}`" class="text-xs text-gray-500">Req.</label>
        </div>

        <Button
          icon="pi pi-trash"
          text
          severity="danger"
          size="small"
          rounded
          class="!w-7 !h-7"
          @click="removeField(idx)"
        />

        <Button
          v-tooltip="'Ajouter un champ après'"
          icon="pi pi-plus"
          text
          size="small"
          rounded
          class="!w-7 !h-7"
          @click="addFieldAfter(idx)"
        />
      </div>

      <!-- Body: labels + constraints -->
      <div class="p-3 grid grid-cols-1 md:grid-cols-2 gap-3">
        <!-- Label FR -->
        <div v-if="ft.labels" class="flex flex-col gap-1">
          <label class="text-xs font-medium text-gray-500">Label FR</label>
          <InputText
            :model-value="field.labelFr"
            placeholder="ex: Titre"
            class="text-sm"
            @update:model-value="updateField(idx, 'labelFr', $event)"
          />
        </div>

        <!-- Label EN -->
        <div v-if="ft.labels" class="flex flex-col gap-1">
          <label class="text-xs font-medium text-gray-500">Label EN</label>
          <InputText
            :model-value="field.labelEn"
            placeholder="ex: Title"
            class="text-sm"
            @update:model-value="updateField(idx, 'labelEn', $event)"
          />
        </div>

        <!-- Description -->
        <div v-if="ft.description" class="flex flex-col gap-1 md:col-span-2">
          <label class="text-xs font-medium text-gray-500">Description</label>
          <InputText
            :model-value="field.description ?? ''"
            placeholder="Description du champ"
            class="text-sm"
            @update:model-value="
              updateField(idx, 'description', $event || null)
            "
          />
        </div>

        <!-- ── String constraints ── -->
        <template v-if="field.type === 'string'">
          <div v-if="ft.minLength" class="flex flex-col gap-1">
            <label class="text-xs text-gray-500">Min. caractères</label>
            <InputNumber
              :model-value="field.minLength ?? null"
              :min="0"
              :use-grouping="false"
              class="text-sm"
              @update:model-value="updateField(idx, 'minLength', $event)"
            />
          </div>
          <div v-if="ft.maxLength" class="flex flex-col gap-1">
            <label class="text-xs text-gray-500">Max. caractères</label>
            <InputNumber
              :model-value="field.maxLength ?? null"
              :min="0"
              :use-grouping="false"
              class="text-sm"
              @update:model-value="updateField(idx, 'maxLength', $event)"
            />
          </div>
          <div v-if="ft.pattern" class="flex flex-col gap-1">
            <label class="text-xs text-gray-500">Pattern (regex)</label>
            <InputText
              :model-value="field.pattern ?? ''"
              placeholder="ex: ^[A-Z].*"
              class="text-sm font-mono"
              @update:model-value="updateField(idx, 'pattern', $event || null)"
            />
          </div>
          <div v-if="ft.format" class="flex flex-col gap-1">
            <label class="text-xs text-gray-500">Format</label>
            <Select
              :model-value="field.format ?? null"
              :options="formatOptions"
              option-label="label"
              option-value="value"
              placeholder="Aucun"
              show-clear
              class="text-sm"
              @update:model-value="updateField(idx, 'format', $event)"
            />
          </div>
          <div v-if="ft.enum" class="flex flex-col gap-1 md:col-span-2">
            <label class="text-xs text-gray-500"
              >Valeurs (enum, séparées par des virgules)</label
            >
            <InputText
              :model-value="(field.enum ?? []).join(', ')"
              placeholder="neuf, bon, mauvais"
              class="text-sm"
              @update:model-value="
                updateField(idx, 'enum', parseEnum(String($event ?? '')))
              "
            />
          </div>
          <div v-if="ft.default" class="flex flex-col gap-1">
            <label class="text-xs text-gray-500">Valeur par défaut</label>
            <InputText
              :model-value="String(field.defaultValue ?? '')"
              placeholder="ex: N/A"
              class="text-sm"
              @update:model-value="
                updateField(idx, 'defaultValue', $event ? $event : null)
              "
            />
          </div>
        </template>

        <!-- ── Number / Integer constraints ── -->
        <template v-if="field.type === 'integer' || field.type === 'number'">
          <div v-if="ft.minimum" class="flex flex-col gap-1">
            <label class="text-xs text-gray-500">Min</label>
            <InputNumber
              :model-value="field.minimum ?? null"
              :use-grouping="false"
              class="text-sm"
              @update:model-value="updateField(idx, 'minimum', $event)"
            />
          </div>
          <div v-if="ft.maximum" class="flex flex-col gap-1">
            <label class="text-xs text-gray-500">Max</label>
            <InputNumber
              :model-value="field.maximum ?? null"
              :use-grouping="false"
              class="text-sm"
              @update:model-value="updateField(idx, 'maximum', $event)"
            />
          </div>
          <div v-if="ft.exclusiveMinimum" class="flex flex-col gap-1">
            <label class="text-xs text-gray-500">Min exclusif</label>
            <InputNumber
              :model-value="field.exclusiveMinimum ?? null"
              :use-grouping="false"
              class="text-sm"
              @update:model-value="updateField(idx, 'exclusiveMinimum', $event)"
            />
          </div>
          <div v-if="ft.exclusiveMaximum" class="flex flex-col gap-1">
            <label class="text-xs text-gray-500">Max exclusif</label>
            <InputNumber
              :model-value="field.exclusiveMaximum ?? null"
              :use-grouping="false"
              class="text-sm"
              @update:model-value="updateField(idx, 'exclusiveMaximum', $event)"
            />
          </div>
          <div v-if="ft.multipleOf" class="flex flex-col gap-1">
            <label class="text-xs text-gray-500">Multiple de</label>
            <InputNumber
              :model-value="field.multipleOf ?? null"
              :use-grouping="false"
              :min="1"
              class="text-sm"
              @update:model-value="updateField(idx, 'multipleOf', $event)"
            />
          </div>
          <div v-if="ft.default" class="flex flex-col gap-1">
            <label class="text-xs text-gray-500">Valeur par défaut</label>
            <InputNumber
              :model-value="(field.defaultValue as number | null) ?? null"
              :use-grouping="false"
              class="text-sm"
              @update:model-value="updateField(idx, 'defaultValue', $event)"
            />
          </div>
        </template>

        <!-- ── Boolean constraints ── -->
        <template v-if="field.type === 'boolean' && ft.default">
          <div class="flex flex-col gap-1">
            <label class="text-xs text-gray-500">Valeur par défaut</label>
            <Select
              :model-value="field.defaultValue ?? null"
              :options="boolDefaultOptions"
              option-label="label"
              option-value="value"
              placeholder="Aucune"
              show-clear
              class="text-sm"
              @update:model-value="updateField(idx, 'defaultValue', $event)"
            />
          </div>
        </template>

        <!-- ── i18n-text info ── -->
        <template v-if="field.type === 'i18n-text'">
          <div class="md:col-span-2 text-xs text-gray-400 italic">
            Génère un objet
            <code class="bg-gray-200 px-1 rounded"
              >{ "fr": "…", "en": "…" }</code
            >
            — les utilisateurs saisiront une valeur par langue.
          </div>
        </template>

        <!-- ── Image gallery config ── -->
        <template v-if="field.type === 'image-gallery'">
          <div class="flex flex-col gap-2 md:col-span-2">
            <label class="text-xs text-gray-500">Catégories d'images</label>
            <div
              v-for="(cat, catIdx) in field.imageCategories ?? []"
              :key="catIdx"
              class="flex items-center gap-2"
            >
              <InputText
                :model-value="cat.name"
                placeholder="Nom (ex: poster)"
                class="text-sm flex-1"
                @update:model-value="
                  updateImageCategory(idx, catIdx, 'name', String($event ?? ''))
                "
              />
              <InputText
                :model-value="cat.ratio"
                placeholder="Ratio (ex: 2/3)"
                class="text-sm w-24"
                @update:model-value="
                  updateImageCategory(
                    idx,
                    catIdx,
                    'ratio',
                    String($event ?? ''),
                  )
                "
              />
              <button
                class="text-red-400 hover:text-red-600 text-sm"
                title="Supprimer"
                @click="removeImageCategory(idx, catIdx)"
              >
                ×
              </button>
            </div>
            <button
              class="text-blue-500 hover:text-blue-700 text-xs self-start"
              @click="addImageCategory(idx)"
            >
              + Ajouter une catégorie
            </button>
          </div>
          <div class="md:col-span-2 text-xs text-gray-400 italic">
            Génère un objet avec un tableau d'URLs par catégorie et un sélecteur
            d'image par défaut. Le ratio est au format CSS (ex: 2/3, 16/9, 3/1).
          </div>
        </template>

        <!-- ── Array constraints ── -->
        <template v-if="field.type === 'array'">
          <div class="flex flex-col gap-1">
            <label class="text-xs text-gray-500">Type des éléments</label>
            <Select
              :model-value="field.itemsType ?? 'string'"
              :options="arrayItemTypeOptions"
              option-label="label"
              option-value="value"
              class="text-sm"
              @update:model-value="updateField(idx, 'itemsType', $event)"
            />
          </div>
          <div v-if="ft.uniqueItems" class="flex items-center gap-2 pt-5">
            <Checkbox
              :model-value="field.uniqueItems ?? false"
              :binary="true"
              :input-id="`unique-${idx}`"
              @update:model-value="updateField(idx, 'uniqueItems', $event)"
            />
            <label :for="`unique-${idx}`" class="text-xs text-gray-500"
              >Éléments uniques</label
            >
          </div>
          <div v-if="ft.minItems" class="flex flex-col gap-1">
            <label class="text-xs text-gray-500">Min. éléments</label>
            <InputNumber
              :model-value="field.minItems ?? null"
              :min="0"
              :use-grouping="false"
              class="text-sm"
              @update:model-value="updateField(idx, 'minItems', $event)"
            />
          </div>
          <div v-if="ft.maxItems" class="flex flex-col gap-1">
            <label class="text-xs text-gray-500">Max. éléments</label>
            <InputNumber
              :model-value="field.maxItems ?? null"
              :min="0"
              :use-grouping="false"
              class="text-sm"
              @update:model-value="updateField(idx, 'maxItems', $event)"
            />
          </div>
        </template>
      </div>
    </div>

    <!-- Empty state -->
    <div
      v-if="fields.length === 0"
      class="text-sm text-gray-400 italic text-center py-4 border border-dashed border-gray-300 rounded-lg"
    >
      Aucun champ défini. Cliquez sur « Ajouter un champ » pour commencer.
    </div>

    <!-- Actions -->
    <div class="flex items-center gap-2">
      <Button
        label="Ajouter un champ"
        icon="pi pi-plus"
        size="small"
        outlined
        @click="addField"
      />
      <div class="flex-1"></div>
      <Button
        :label="showRaw ? 'Masquer JSON' : 'Voir JSON'"
        :icon="showRaw ? 'pi pi-eye-slash' : 'pi pi-eye'"
        size="small"
        text
        @click="showRaw = !showRaw"
      />
    </div>

    <!-- Raw JSON preview -->
    <div v-if="showRaw" class="flex flex-col gap-1">
      <Textarea
        :model-value="JSON.stringify(schema, null, 2)"
        readonly
        rows="12"
        class="font-mono text-xs bg-gray-100"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, type PropType } from "vue";
import Button from "primevue/button";
import InputText from "primevue/inputtext";
import InputNumber from "primevue/inputnumber";
import Select from "primevue/select";
import Checkbox from "primevue/checkbox";
import Textarea from "primevue/textarea";

// ── Feature flags ─────────────────────────────────────────────────────
// Each flag controls the visibility of a constraint option in the UI.
// Pass a partial object to override defaults.
interface SchemaBuilderFeatures {
  // x-labels FR/EN
  labels: boolean;
  // per-field description
  description: boolean;
  // default value
  default: boolean;
  // string
  minLength: boolean;
  maxLength: boolean;
  pattern: boolean;
  format: boolean;
  enum: boolean;
  // number / integer
  minimum: boolean;
  maximum: boolean;
  exclusiveMinimum: boolean;
  exclusiveMaximum: boolean;
  multipleOf: boolean;
  // array
  array: boolean;
  minItems: boolean;
  maxItems: boolean;
  uniqueItems: boolean;
  // i18n
  i18nText: boolean;
  // images
  imageGallery: boolean;
}

const defaultFeatures: SchemaBuilderFeatures = {
  labels: true,
  description: false,
  default: false,
  minLength: true,
  maxLength: true,
  pattern: true,
  format: true,
  enum: true,
  minimum: true,
  maximum: true,
  exclusiveMinimum: false,
  exclusiveMaximum: false,
  multipleOf: false,
  array: false,
  minItems: true,
  maxItems: true,
  uniqueItems: true,
  i18nText: true,
  imageGallery: true,
};

// ── Field model ───────────────────────────────────────────────────────
interface SchemaField {
  key: string;
  type: string;
  labelFr: string;
  labelEn: string;
  required: boolean;
  description?: string | null;
  defaultValue?: unknown;
  // string
  minLength?: number | null;
  maxLength?: number | null;
  pattern?: string | null;
  format?: string | null;
  enum?: string[] | null;
  // number / integer
  minimum?: number | null;
  maximum?: number | null;
  exclusiveMinimum?: number | null;
  exclusiveMaximum?: number | null;
  multipleOf?: number | null;
  // array
  itemsType?: string | null;
  minItems?: number | null;
  maxItems?: number | null;
  uniqueItems?: boolean | null;
  // image-gallery
  imageCategories?: { name: string; ratio: string }[] | null;
  // mots-clés non gérés par le builder (futurs x-*, default sur un objet…), conservés tels quels
  extra?: Record<string, unknown>;
}

// Mots-clés d'une propriété pilotés par le builder ; tout le reste est conservé dans `extra`.
const MANAGED_PROPERTY_KEYS = new Set([
  "type",
  "description",
  "default",
  "minLength",
  "maxLength",
  "pattern",
  "format",
  "enum",
  "minimum",
  "maximum",
  "exclusiveMinimum",
  "exclusiveMaximum",
  "multipleOf",
  "items",
  "minItems",
  "maxItems",
  "uniqueItems",
  "properties",
  "additionalProperties",
  "x-widget",
  "x-image-categories",
]);

// Mots-clés racine pilotés par le builder ; le reste (dont x-label édité à part) est conservé.
const MANAGED_ROOT_KEYS = new Set([
  "type",
  "properties",
  "required",
  "additionalProperties",
  "x-order",
  "x-labels",
  "x-label",
]);

// ── Props & Emits ─────────────────────────────────────────────────────
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

// Merged features
const ft = computed<SchemaBuilderFeatures>(() => ({
  ...defaultFeatures,
  ...props.features,
}));

// ── Options ───────────────────────────────────────────────────────────
const baseTypeOptions = [
  { label: "Texte", value: "string" },
  { label: "Entier", value: "integer" },
  { label: "Nombre", value: "number" },
  { label: "Booléen", value: "boolean" },
];

const arrayTypeOption = { label: "Tableau", value: "array" };
const i18nTypeOption = { label: "Texte i18n", value: "i18n-text" };
const imageGalleryTypeOption = {
  label: "Galerie d'images",
  value: "image-gallery",
};

const availableTypeOptions = computed(() => {
  const opts = [...baseTypeOptions];
  if (ft.value.i18nText) opts.push(i18nTypeOption);
  if (ft.value.imageGallery) opts.push(imageGalleryTypeOption);
  if (ft.value.array) opts.push(arrayTypeOption);
  return opts;
});

const formatOptions = [
  { label: "URI / URL", value: "uri" },
  { label: "Date", value: "date" },
  { label: "Date-Time", value: "date-time" },
  { label: "Email", value: "email" },
];

const boolDefaultOptions = [
  { label: "Vrai", value: true },
  { label: "Faux", value: false },
];

const arrayItemTypeOptions = [
  { label: "Texte", value: "string" },
  { label: "Entier", value: "integer" },
  { label: "Nombre", value: "number" },
  { label: "Booléen", value: "boolean" },
];

const fields = ref<SchemaField[]>([]);
const showRaw = ref(false);
// Gabarit du libellé des instances (extension racine x-label), ex. "{number}. {title}".
const xLabel = ref("");
// Mots-clés racine inconnus, restitués à l'identique.
const rootExtra = ref<Record<string, unknown>>({});

// ── Parse schema → fields ─────────────────────────────────────────────
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
      // Detect i18n-text widget
      if (
        type === "object" &&
        (prop as Record<string, unknown>)["x-widget"] === "i18n-text"
      ) {
        type = "i18n-text";
      }
      // Detect image-gallery widget
      if (
        type === "object" &&
        (prop as Record<string, unknown>)["x-widget"] === "image-gallery"
      ) {
        type = "image-gallery";
      }
      const field: SchemaField = {
        key,
        type,
        labelFr: labelsFr[key] ?? "",
        labelEn: labelsEn[key] ?? "",
        required: required.includes(key),
        description: (prop.description as string) ?? null,
        defaultValue: prop.default ?? null,
        // string
        minLength: (prop.minLength as number) ?? null,
        maxLength: (prop.maxLength as number) ?? null,
        pattern: (prop.pattern as string) ?? null,
        format: (prop.format as string) ?? null,
        enum: (prop.enum as string[]) ?? null,
        // number
        minimum: (prop.minimum as number) ?? null,
        maximum: (prop.maximum as number) ?? null,
        exclusiveMinimum: (prop.exclusiveMinimum as number) ?? null,
        exclusiveMaximum: (prop.exclusiveMaximum as number) ?? null,
        multipleOf: (prop.multipleOf as number) ?? null,
        // array
        itemsType:
          ((prop.items as Record<string, unknown>)?.type as string) ?? null,
        minItems: (prop.minItems as number) ?? null,
        maxItems: (prop.maxItems as number) ?? null,
        uniqueItems: (prop.uniqueItems as boolean) ?? null,
        // image-gallery
        imageCategories:
          type === "image-gallery"
            ? parseImageCategories(
                (prop as Record<string, unknown>)["x-image-categories"],
              )
            : null,
        extra: Object.fromEntries(
          Object.entries(prop).filter(([k]) => !MANAGED_PROPERTY_KEYS.has(k)),
        ),
      };
      return field;
    });
}

// ── Build fields → schema ─────────────────────────────────────────────
const schema = computed<Record<string, unknown>>(() => {
  const properties: Record<string, Record<string, unknown>> = {};
  const requiredKeys: string[] = [];
  const labelsFr: Record<string, string> = {};
  const labelsEn: Record<string, string> = {};
  let hasLabels = false;

  for (const f of fields.value) {
    if (!f.key.trim()) continue;
    const prop: Record<string, unknown> = { ...(f.extra ?? {}), type: f.type };

    // description
    if (f.description) prop.description = f.description;

    // string constraints
    if (f.type === "string") {
      if (f.minLength != null && f.minLength > 0) prop.minLength = f.minLength;
      if (f.maxLength != null && f.maxLength > 0) prop.maxLength = f.maxLength;
      if (f.pattern) prop.pattern = f.pattern;
      if (f.format) prop.format = f.format;
      if (f.enum && f.enum.length > 0) prop.enum = f.enum;
      if (f.defaultValue != null) prop.default = f.defaultValue;
    }

    // number / integer constraints
    if (f.type === "integer" || f.type === "number") {
      if (f.minimum != null) prop.minimum = f.minimum;
      if (f.maximum != null) prop.maximum = f.maximum;
      if (f.exclusiveMinimum != null)
        prop.exclusiveMinimum = f.exclusiveMinimum;
      if (f.exclusiveMaximum != null)
        prop.exclusiveMaximum = f.exclusiveMaximum;
      if (f.multipleOf != null) prop.multipleOf = f.multipleOf;
      if (f.defaultValue != null) prop.default = f.defaultValue;
    }

    // boolean
    if (f.type === "boolean") {
      if (f.defaultValue != null) prop.default = f.defaultValue;
    }

    // i18n-text → object with x-widget
    if (f.type === "i18n-text") {
      prop.type = "object";
      prop["x-widget"] = "i18n-text";
      prop.properties = {
        fr: { type: "string" },
        en: { type: "string" },
      };
      prop.additionalProperties = false;
    }

    // image-gallery → object with x-widget + categories
    if (f.type === "image-gallery") {
      const cats = (
        f.imageCategories ?? [{ name: "poster", ratio: "1/1" }]
      ).filter((c) => c.name);
      prop.type = "object";
      prop["x-widget"] = "image-gallery";
      const catsObj: Record<string, { ratio: string }> = {};
      for (const c of cats) catsObj[c.name] = { ratio: c.ratio || "1/1" };
      prop["x-image-categories"] = catsObj;
      const catProperties: Record<string, unknown> = {
        default_image: { type: "string" },
      };
      for (const c of cats) {
        catProperties[c.name] = { type: "array", items: { type: "string" } };
      }
      prop.properties = catProperties;
      prop.additionalProperties = false;
    }

    // array constraints
    if (f.type === "array") {
      prop.items = { type: f.itemsType ?? "string" };
      if (f.minItems != null) prop.minItems = f.minItems;
      if (f.maxItems != null) prop.maxItems = f.maxItems;
      if (f.uniqueItems) prop.uniqueItems = true;
    }

    properties[f.key] = prop;
    if (f.required) requiredKeys.push(f.key);
    if (f.labelFr) {
      labelsFr[f.key] = f.labelFr;
      hasLabels = true;
    }
    if (f.labelEn) {
      labelsEn[f.key] = f.labelEn;
      hasLabels = true;
    }
  }

  const result: Record<string, unknown> = {
    ...rootExtra.value,
    type: "object",
    properties,
    additionalProperties: false,
    "x-order": fields.value.filter((f) => f.key.trim()).map((f) => f.key),
  };
  if (xLabel.value.trim()) result["x-label"] = xLabel.value.trim();
  if (requiredKeys.length > 0) result.required = requiredKeys;
  if (hasLabels) {
    const xLabels: Record<string, Record<string, string>> = {};
    if (Object.keys(labelsFr).length > 0) xLabels.fr = labelsFr;
    if (Object.keys(labelsEn).length > 0) xLabels.en = labelsEn;
    result["x-labels"] = xLabels;
  }

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

// Load fields from initial modelValue
let initialized = false;

function loadRoot(schema: Record<string, unknown>) {
  xLabel.value =
    typeof schema["x-label"] === "string" ? (schema["x-label"] as string) : "";
  rootExtra.value = Object.fromEntries(
    Object.entries(schema ?? {}).filter(([k]) => !MANAGED_ROOT_KEYS.has(k)),
  );
}

watch(
  () => props.modelValue,
  (val) => {
    if (!initialized) {
      fields.value = parseSchema(val);
      loadRoot(val);
      initialized = true;
    }
  },
  { immediate: true },
);

// Expose a reset method for when parent needs to reinit
function resetFromSchema(schema: Record<string, unknown>) {
  initialized = false;
  fields.value = parseSchema(schema);
  loadRoot(schema);
  initialized = true;
}
defineExpose({ resetFromSchema });

function addField() {
  fields.value.push({
    key: "",
    type: "string",
    labelFr: "",
    labelEn: "",
    required: false,
    description: null,
    defaultValue: null,
    minLength: null,
    maxLength: null,
    pattern: null,
    minimum: null,
    maximum: null,
    exclusiveMinimum: null,
    exclusiveMaximum: null,
    multipleOf: null,
    format: null,
    enum: null,
    itemsType: null,
    minItems: null,
    maxItems: null,
    uniqueItems: null,
    imageCategories: null,
  });
}

function parseImageCategories(
  raw: unknown,
): { name: string; ratio: string }[] | null {
  if (!raw) return null;
  // New format: object { name: { ratio } }
  if (typeof raw === "object" && !Array.isArray(raw)) {
    return Object.entries(raw as Record<string, { ratio?: string }>).map(
      ([name, v]) => ({ name, ratio: v?.ratio ?? "1/1" }),
    );
  }
  // Legacy format: string[]
  if (Array.isArray(raw)) {
    return (raw as string[]).map((name) => ({ name, ratio: "1/1" }));
  }
  return null;
}

function removeField(idx: number) {
  fields.value.splice(idx, 1);
}

function addFieldAfter(idx: number) {
  fields.value.splice(idx + 1, 0, {
    key: "",
    type: "string",
    labelFr: "",
    labelEn: "",
    required: false,
    description: null,
    defaultValue: null,
    minLength: null,
    maxLength: null,
    pattern: null,
    minimum: null,
    maximum: null,
    exclusiveMinimum: null,
    exclusiveMaximum: null,
    multipleOf: null,
    format: null,
    enum: null,
    itemsType: null,
    minItems: null,
    maxItems: null,
    uniqueItems: null,
    imageCategories: null,
  });
}

function updateImageCategory(
  fieldIdx: number,
  catIdx: number,
  key: "name" | "ratio",
  value: string,
) {
  const cats = [...(fields.value[fieldIdx]!.imageCategories ?? [])];
  cats[catIdx] = { ...cats[catIdx]!, [key]: value };
  fields.value[fieldIdx]!.imageCategories = cats;
}

function removeImageCategory(fieldIdx: number, catIdx: number) {
  const cats = [...(fields.value[fieldIdx]!.imageCategories ?? [])];
  cats.splice(catIdx, 1);
  fields.value[fieldIdx]!.imageCategories = cats;
}

function addImageCategory(fieldIdx: number) {
  const cats = [...(fields.value[fieldIdx]!.imageCategories ?? [])];
  cats.push({ name: "", ratio: "1/1" });
  fields.value[fieldIdx]!.imageCategories = cats;
}

function move(idx: number, dir: -1 | 1) {
  const target = idx + dir;
  if (target < 0 || target >= fields.value.length) return;
  const arr = fields.value;
  [arr[idx], arr[target]] = [arr[target]!, arr[idx]!];
}

function updateField(idx: number, key: keyof SchemaField, value: unknown) {
  (fields.value[idx] as Record<string, unknown>)[key] = value;
}

function parseEnum(value: string): string[] | null {
  if (!value.trim()) return null;
  const items = value
    .split(",")
    .map((s) => s.trim())
    .filter(Boolean);
  return items.length > 0 ? items : null;
}
</script>
