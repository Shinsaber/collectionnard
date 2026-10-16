<template>
  <!-- Deux colonnes dès que le conteneur le permet (dialogue large), une sinon (mobile). -->
  <div class="@container">
    <div class="grid grid-cols-1 gap-x-4 gap-y-4.5 @md:grid-cols-2">
      <div
        v-for="field in fields"
        :key="field.name"
        class="flex min-w-0 flex-col gap-1.5"
        :class="{ '@md:col-span-2': isWide(field) }"
      >
        <label
          :id="`${uid}-${field.name}-label`"
          :for="`${uid}-${field.name}`"
          class="text-[13px] font-bold text-ink"
        >
          {{ field.label }}
          <span v-if="field.required" class="text-danger" aria-hidden="true"
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

        <!-- Enum (tableau d'enum : choix multiples, ex. rôles) -->
        <ElSelect
          v-else-if="field.enum"
          :id="`${uid}-${field.name}`"
          :model-value="enumValue(field)"
          :multiple="field.multiple"
          :placeholder="t('schema.form.choose')"
          clearable
          :class="{ 'is-error': !!errors[field.name] }"
          @update:model-value="updateEnum(field, $event)"
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
          class="w-full! font-mono"
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

        <!-- Texte multilingue : onglets par langue (● renseignée, ○ vide), une zone visible. -->
        <div
          v-else-if="field.widget === 'i18n-text'"
          :id="`${uid}-${field.name}`"
          class="flex flex-col gap-2"
          role="group"
          :aria-labelledby="`${uid}-${field.name}-label`"
        >
          <div
            role="tablist"
            :aria-label="t('schema.form.languages')"
            class="flex gap-0.5 self-start rounded-lg bg-soft p-0.75"
          >
            <button
              v-for="l in LANGS"
              :id="`${uid}-${field.name}-tab-${l}`"
              :key="l"
              type="button"
              role="tab"
              :aria-selected="shownLang(field.name) === l"
              :aria-controls="`${uid}-${field.name}-panel-${l}`"
              class="h-7 rounded-md px-2.5 font-mono text-[11px] font-medium uppercase"
              :class="
                shownLang(field.name) === l
                  ? 'bg-surface text-ink shadow-control'
                  : 'text-muted'
              "
              @click="activeLang[field.name] = l"
            >
              {{ getI18n(field.name, l) ? "●" : "○" }} {{ l }}
            </button>
          </div>
          <div
            v-for="l in LANGS"
            v-show="shownLang(field.name) === l"
            :id="`${uid}-${field.name}-panel-${l}`"
            :key="l"
            role="tabpanel"
            :aria-labelledby="`${uid}-${field.name}-tab-${l}`"
          >
            <label :for="`${uid}-${field.name}-${l}`" class="sr-only">{{
              `${field.label} (${l.toUpperCase()})`
            }}</label>
            <ElInput
              :id="`${uid}-${field.name}-${l}`"
              type="textarea"
              :model-value="getI18n(field.name, l)"
              :autosize="{ minRows: 2, maxRows: 8 }"
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
          class="text-xs text-danger"
          role="alert"
          >{{ errors[field.name] }}</small
        >
      </div>

      <div v-if="fields.length === 0" class="text-sm italic text-muted">
        {{ t("schema.form.noAttributes") }}
      </div>
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
  /** Tableau dont les éléments sont un enum (`items.enum`) : sélection multiple. */
  multiple?: boolean;
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
      const itemsEnum =
        prop.type === "array"
          ? ((prop.items as Record<string, unknown> | undefined)?.enum as
              string[] | undefined)
          : undefined;
      return {
        name,
        label: fieldLabel(props.schema, name, lang.value, "pretty"),
        type: (prop.type as string) ?? "string",
        required: required.includes(name),
        enum: (prop.enum as string[] | undefined) ?? itemsEnum,
        multiple: !!itemsEnum,
        format: prop.format as string | undefined,
        minimum: prop.minimum as number | undefined,
        maximum: prop.maximum as number | undefined,
        widget: widgetOf(props.schema, name),
        imageCategories: imageCategories(props.schema, name),
      };
    });
});

/**
 * Champ « large » : toute la largeur dans la grille à deux colonnes (texte libre, multilingue,
 * galerie, curseur, choix multiples, JSON). Nombres, dates, booléens et choix simples tiennent
 * sur une demi-ligne.
 */
function isWide(field: SchemaField): boolean {
  if (field.widget === "i18n-text" || field.widget === "image-gallery")
    return true;
  if (field.enum) return !!field.multiple;
  if (field.type === "integer" || field.type === "number")
    return field.minimum != null && field.maximum != null;
  if (field.type === "boolean") return false;
  if (field.type === "string")
    return !["date", "date-time", "email"].includes(field.format ?? "");
  return true;
}

// Texte multilingue : une langue affichée à la fois (onglets), la langue courante d'abord.
const activeLang = reactive<Record<string, string>>({});
function shownLang(name: string): string {
  return (
    activeLang[name] ??
    ((LANGS as readonly string[]).includes(lang.value) ? lang.value : LANGS[0])
  );
}

function enumValue(field: SchemaField): string | string[] | undefined {
  const v = model.value[field.name];
  if (field.multiple) return Array.isArray(v) ? (v as string[]) : [];
  return (v as string | undefined) ?? undefined;
}

// Un tableau vidé redevient « absent » : `required` le signale, l'API ne reçoit pas [].
function updateEnum(field: SchemaField, value: unknown) {
  if (field.multiple && Array.isArray(value) && value.length === 0) {
    update(field.name, undefined);
    return;
  }
  update(field.name, value === "" ? undefined : value);
}

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
    if (
      field.required &&
      (val === undefined ||
        val === null ||
        val === "" ||
        (Array.isArray(val) && val.length === 0))
    ) {
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
