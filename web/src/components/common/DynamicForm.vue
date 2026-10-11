<template>
  <div class="flex flex-col gap-3">
    <div v-for="field in fields" :key="field.name" class="flex flex-col gap-1">
      <label
        :id="`${uid}-${field.name}-label`"
        :for="`${uid}-${field.name}`"
        class="text-sm font-medium text-gray-700 dark:text-gray-300"
      >
        {{ field.label }}
        <span v-if="field.required" class="text-red-500" aria-hidden="true"
          >*</span
        >
      </label>

      <!-- String (texte, url, date) -->
      <ElInput
        v-if="field.type === 'string' && !field.enum"
        :id="`${uid}-${field.name}`"
        :model-value="getString(field.name)"
        :type="inputType(field)"
        :placeholder="field.label"
        :class="{ 'is-error': !!errors[field.name] }"
        @update:model-value="update(field.name, $event)"
      />

      <!-- Enum -->
      <ElSelect
        v-else-if="field.enum"
        :id="`${uid}-${field.name}`"
        :model-value="(model[field.name] as string | undefined) ?? undefined"
        :placeholder="t('schema.form.choose')"
        clearable
        :class="{ 'is-error': !!errors[field.name] }"
        @update:model-value="update(field.name, $event)"
      >
        <ElOption
          v-for="opt in field.enum"
          :key="opt"
          :label="enumLabel(schema, field.name, opt, lang)"
          :value="opt"
        />
      </ElSelect>

      <!-- Integer / Number with slider (when min & max defined) -->
      <div
        v-else-if="
          (field.type === 'integer' || field.type === 'number') &&
          field.minimum != null &&
          field.maximum != null
        "
        class="flex items-center gap-3"
      >
        <ElSlider
          :model-value="getNumber(field.name) ?? field.minimum"
          :min="field.minimum"
          :max="field.maximum"
          :step="field.type === 'integer' ? 1 : 0.01"
          class="flex-1"
          :aria-label="field.label"
          @update:model-value="update(field.name, $event)"
        />
        <output
          :id="`${uid}-${field.name}`"
          class="text-sm font-mono w-12 text-right"
          >{{ getNumber(field.name) ?? field.minimum }}</output
        >
      </div>

      <!-- Integer / Number (no slider) -->
      <ElInputNumber
        v-else-if="field.type === 'integer' || field.type === 'number'"
        :id="`${uid}-${field.name}`"
        :model-value="getNumber(field.name)"
        :min="field.minimum"
        :max="field.maximum"
        :precision="field.type === 'integer' ? 0 : 2"
        :step="field.type === 'integer' ? 1 : 0.1"
        controls-position="right"
        class="w-full!"
        :class="{ 'is-error': !!errors[field.name] }"
        @update:model-value="update(field.name, $event ?? undefined)"
      />

      <!-- Boolean -->
      <ElCheckbox
        v-else-if="field.type === 'boolean'"
        :id="`${uid}-${field.name}`"
        :model-value="!!model[field.name]"
        @update:model-value="update(field.name, $event)"
      />

      <!-- i18n text: one textarea per language -->
      <div
        v-else-if="field.widget === 'i18n-text'"
        :id="`${uid}-${field.name}`"
        class="grid grid-cols-1 sm:grid-cols-2 gap-2"
        role="group"
        :aria-labelledby="`${uid}-${field.name}-label`"
      >
        <div v-for="l in LANGS" :key="l" class="flex flex-col gap-1">
          <label
            :for="`${uid}-${field.name}-${l}`"
            class="text-xs text-gray-400 dark:text-gray-500 uppercase"
            >{{ l }}</label
          >
          <ElInput
            :id="`${uid}-${field.name}-${l}`"
            type="textarea"
            :model-value="getI18n(field.name, l)"
            :rows="2"
            :placeholder="`${field.label} (${l.toUpperCase()})`"
            @update:model-value="
              updateI18n(field.name, l, String($event ?? ''))
            "
          />
        </div>
      </div>

      <!-- Image gallery widget -->
      <ImageGalleryField
        v-else-if="field.widget === 'image-gallery'"
        :id="`${uid}-${field.name}`"
        :categories="field.imageCategories ?? []"
        :model-value="galleryValue(field.name)"
        @update:model-value="update(field.name, $event)"
      />

      <!-- Fallback: raw JSON for other objects/arrays -->
      <ElInput
        v-else
        :id="`${uid}-${field.name}`"
        type="textarea"
        :model-value="JSON.stringify(model[field.name] ?? '', null, 2)"
        :rows="3"
        class="font-mono text-xs"
        @update:model-value="updateJson(field.name, String($event ?? ''))"
      />

      <!-- Field error message -->
      <small
        v-if="errors[field.name]"
        class="text-red-500 text-xs"
        role="alert"
        >{{ errors[field.name] }}</small
      >
    </div>

    <div
      v-if="fields.length === 0"
      class="text-sm text-gray-400 dark:text-gray-500 italic"
    >
      {{ t("schema.form.noAttributes") }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, useId } from "vue";
import { useI18n } from "vue-i18n";
import { useLang } from "@/composables/useLang";
import {
  LANGS,
  enumLabel,
  fieldLabel,
  imageCategories,
  orderedKeys,
  widgetOf,
} from "@/lib/schema";
import ImageGalleryField from "./ImageGalleryField.vue";

interface SchemaField {
  name: string;
  label: string;
  type: string;
  required: boolean;
  enum?: string[];
  format?: string;
  minimum?: number;
  maximum?: number;
  widget?: string;
  imageCategories?: { name: string; ratio: string }[];
}

const props = defineProps<{
  schema: Record<string, unknown>;
  modelValue: Record<string, unknown>;
}>();

const emit = defineEmits<{
  "update:modelValue": [value: Record<string, unknown>];
}>();

// Préfixe unique des ids (plusieurs formulaires peuvent coexister dans une page).
const uid = useId();
const { lang } = useLang();
const { t } = useI18n();
const model = computed(() => props.modelValue);

const fields = computed<SchemaField[]>(() => {
  const properties = props.schema?.properties as
    Record<string, Record<string, unknown>> | undefined;
  if (!properties) return [];
  const required = (props.schema?.required as string[]) ?? [];

  return orderedKeys(props.schema)
    .filter((name) => name in properties)
    .map((name) => {
      const prop = properties[name]!;
      return {
        name,
        label: fieldLabel(props.schema, name, lang.value, "pretty"),
        type: (prop.type as string) ?? "string",
        required: required.includes(name),
        enum: prop.enum as string[] | undefined,
        format: prop.format as string | undefined,
        minimum: prop.minimum as number | undefined,
        maximum: prop.maximum as number | undefined,
        widget: widgetOf(props.schema, name),
        imageCategories: imageCategories(props.schema, name),
      };
    });
});

function getString(name: string): string {
  return (model.value[name] as string) ?? "";
}

// Type HTML de l'input texte selon `format` (uri → url, date → date).
function inputType(field: SchemaField): string {
  if (field.format === "uri") return "url";
  if (field.format === "date") return "date";
  return "text";
}

function getNumber(name: string): number | undefined {
  const v = model.value[name];
  return typeof v === "number" ? v : undefined;
}

function galleryValue(name: string): Record<string, unknown> | undefined {
  const v = model.value[name];
  return v && typeof v === "object" && !Array.isArray(v)
    ? (v as Record<string, unknown>)
    : undefined;
}

function update(name: string, value: unknown) {
  emit("update:modelValue", { ...model.value, [name]: value });
}

function updateJson(name: string, raw: string) {
  try {
    update(name, JSON.parse(raw));
  } catch {
    // JSON incomplet pendant la saisie : on attend une valeur valide
  }
}

function getI18n(name: string, l: string): string {
  const obj = model.value[name] as Record<string, string> | undefined;
  return obj?.[l] ?? "";
}

function updateI18n(name: string, l: string, value: string) {
  const current = (model.value[name] as Record<string, string>) ?? {};
  update(name, { ...current, [l]: value });
}

// ── Validation ──────────────────────────────────────────────────────
const errors = reactive<Record<string, string>>({});

function clearErrors() {
  for (const key of Object.keys(errors)) delete errors[key];
}

function isValidUrl(value: string): boolean {
  try {
    new URL(value);
    return true;
  } catch {
    return false;
  }
}

function validate(): boolean {
  clearErrors();
  let valid = true;
  for (const field of fields.value) {
    const val = model.value[field.name];
    if (field.required && (val === undefined || val === null || val === "")) {
      errors[field.name] = t("errors.required");
      valid = false;
    }
    if (
      field.format === "uri" &&
      typeof val === "string" &&
      val !== "" &&
      !isValidUrl(val)
    ) {
      errors[field.name] = t("errors.invalidUrl");
      valid = false;
    }
  }
  return valid;
}

function setErrors(fieldErrors: Record<string, string>) {
  clearErrors();
  Object.assign(errors, fieldErrors);
}

defineExpose({ validate, setErrors, clearErrors });
</script>
