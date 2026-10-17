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
          @blur="touch(field.name)"
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
          @blur="touch(field.name)"
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
          @blur="touch(field.name)"
        />

        <!-- Boolean -->
        <ElCheckbox
          v-else-if="field.type === 'boolean'"
          :id="`${uid}-${field.name}`"
          :model-value="!!model[field.name]"
          @update:model-value="update(field.name, $event)"
        />

        <!-- Texte multilingue : onglets par langue -->
        <I18nTextField
          v-else-if="field.widget === 'i18n-text'"
          :id="`${uid}-${field.name}`"
          :label="field.label"
          :model-value="model[field.name]"
          @update:model-value="update(field.name, $event)"
          @blur="touch(field.name)"
        />

        <!-- Image gallery widget -->
        <ImageGalleryField
          v-else-if="field.widget === 'image-gallery'"
          :id="`${uid}-${field.name}`"
          :categories="field.imageCategories ?? []"
          :model-value="galleryValue(field.name)"
          @update:model-value="update(field.name, $event)"
        />

        <!-- Liens (x-widget: links) : libellé + adresse par ligne -->
        <LinksField
          v-else-if="field.widget === 'links'"
          :id="`${uid}-${field.name}`"
          :labelledby="`${uid}-${field.name}-label`"
          :model-value="model[field.name]"
          @update:model-value="update(field.name, $event)"
          @blur="touch(field.name)"
        />

        <!-- Fallback: raw JSON for other objects/arrays -->
        <ElInput
          v-else
          :id="`${uid}-${field.name}`"
          type="textarea"
          :model-value="jsonText(field.name)"
          :rows="3"
          class="font-mono text-xs"
          :class="{ 'is-error': !!errors[field.name] }"
          @update:model-value="updateJson(field.name, String($event ?? ''))"
          @blur="blurJson(field.name)"
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
import { computed, useId } from "vue";
import { useSchemaValidation } from "@/composables/useSchemaValidation";
import { useI18n } from "vue-i18n";
import { useLang } from "@/composables/useLang";
import {
  enumLabel,
  fieldLabel,
  imageCategories,
  type ImageCategory,
  orderedKeys,
  widgetOf,
} from "@/lib/schema";
import ImageGalleryField from "./ImageGalleryField.vue";
import LinksField from "./LinksField.vue";
import I18nTextField from "./I18nTextField.vue";

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
  imageCategories?: ImageCategory[];
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
  if (["i18n-text", "image-gallery", "links"].includes(field.widget ?? ""))
    return true;
  if (field.enum) return !!field.multiple;
  if (field.type === "integer" || field.type === "number")
    return field.minimum != null && field.maximum != null;
  if (field.type === "boolean") return false;
  if (field.type === "string")
    return !["date", "date-time", "email"].includes(field.format ?? "");
  return true;
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
  changed(name, value);
}

// Champ JSON : la saisie est gardée telle quelle ; une valeur valide est transmise, une
// invalide est signalée (« JSON invalide ») sans écraser la valeur précédente.
function jsonText(name: string): string {
  return drafts[name] ?? JSON.stringify(model.value[name] ?? "", null, 2);
}

function updateJson(name: string, raw: string) {
  drafts[name] = raw;
  try {
    update(name, raw.trim() === "" ? undefined : JSON.parse(raw));
  } catch {
    check(name, model.value[name]);
  }
}

function blurJson(name: string) {
  touch(name);
  try {
    JSON.parse(drafts[name] ?? "null");
    delete drafts[name]; // valide : réaffichée mise en forme
  } catch {
    // invalide : la saisie reste affichée avec son erreur
  }
}

// ── Validation (useSchemaValidation) ────────────────────────────────
const {
  errors,
  drafts,
  touch,
  changed,
  check,
  validate: validateFields,
  setErrors,
  clearErrors,
} = useSchemaValidation(
  computed(() => props.schema ?? {}),
  model,
);

/** Tous les champs, avant l'envoi (requis, motifs, longueurs, bornes, JSON…). */
function validate(): boolean {
  return validateFields(fields.value.map((f) => f.name));
}

defineExpose({ validate, setErrors, clearErrors });
</script>
