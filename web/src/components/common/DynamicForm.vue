<template>
  <div class="flex flex-col gap-3">
    <div v-for="field in fields" :key="field.name" class="flex flex-col gap-1">
      <label class="text-sm font-medium text-gray-700">
        {{ field.label }}
        <span v-if="field.required" class="text-red-500">*</span>
      </label>

      <!-- String -->
      <InputText
        v-if="field.type === 'string' && !field.enum && !field.format"
        :modelValue="getString(field.name)"
        @update:modelValue="update(field.name, $event)"
        :placeholder="field.label"
      />

      <!-- String with format: uri -->
      <InputText
        v-else-if="field.type === 'string' && field.format === 'uri'"
        :modelValue="getString(field.name)"
        @update:modelValue="update(field.name, $event)"
        type="url"
        :placeholder="field.label"
        :invalid="!!errors[field.name]"
      />

      <!-- String with format: date -->
      <InputText
        v-else-if="field.type === 'string' && field.format === 'date'"
        :modelValue="getString(field.name)"
        @update:modelValue="update(field.name, $event)"
        type="date"
      />

      <!-- Enum -->
      <Select
        v-else-if="field.enum"
        :modelValue="model[field.name]"
        @update:modelValue="update(field.name, $event)"
        :options="field.enum"
        :placeholder="'Choisir...'"
      />

      <!-- Integer / Number with slider (when min & max defined) -->
      <div
        v-else-if="
          (field.type === 'integer' || field.type === 'number') &&
          field.minimum != null &&
          field.maximum != null
        "
        class="flex items-center gap-3"
      >
        <Slider
          :modelValue="getNumber(field.name) ?? field.minimum"
          @update:modelValue="update(field.name, $event)"
          :min="field.minimum"
          :max="field.maximum"
          :step="field.type === 'integer' ? 1 : 0.01"
          class="flex-1"
        />
        <span class="text-sm font-mono w-12 text-right">{{
          getNumber(field.name) ?? field.minimum
        }}</span>
      </div>

      <!-- Integer / Number (no slider) -->
      <InputNumber
        v-else-if="field.type === 'integer' || field.type === 'number'"
        :modelValue="getNumber(field.name)"
        @update:modelValue="update(field.name, $event)"
        :min="field.minimum"
        :max="field.maximum"
        :useGrouping="false"
        :minFractionDigits="field.type === 'number' ? 0 : undefined"
        :maxFractionDigits="field.type === 'number' ? 2 : 0"
      />

      <!-- Boolean -->
      <Checkbox
        v-else-if="field.type === 'boolean'"
        :modelValue="!!model[field.name]"
        @update:modelValue="update(field.name, $event)"
        :binary="true"
      />

      <!-- Fallback: textarea for objects/arrays -->
      <template v-else-if="field.widget === 'i18n-text'">
        <div class="grid grid-cols-2 gap-2">
          <div class="flex flex-col gap-1">
            <label class="text-xs text-gray-400">FR</label>
            <Textarea
              :modelValue="getI18n(field.name, 'fr')"
              @update:modelValue="
                updateI18n(field.name, 'fr', String($event ?? ''))
              "
              rows="2"
              :placeholder="field.label + ' (FR)'"
            />
          </div>
          <div class="flex flex-col gap-1">
            <label class="text-xs text-gray-400">EN</label>
            <Textarea
              :modelValue="getI18n(field.name, 'en')"
              @update:modelValue="
                updateI18n(field.name, 'en', String($event ?? ''))
              "
              rows="2"
              :placeholder="field.label + ' (EN)'"
            />
          </div>
        </div>
      </template>

      <!-- Image gallery widget -->
      <template v-else-if="field.widget === 'image-gallery'">
        <div class="flex flex-col gap-3">
          <div
            v-for="cat in field.imageCategories ?? []"
            :key="cat.name"
            class="flex flex-col gap-1"
          >
            <label class="text-xs font-medium text-gray-500 capitalize">{{
              cat.name
            }}</label>
            <div class="flex flex-wrap gap-2">
              <div
                v-for="(url, imgIdx) in getGalleryCategory(
                  field.name,
                  cat.name,
                )"
                :key="imgIdx"
                class="relative group"
              >
                <img
                  :src="url"
                  class="w-20 object-cover rounded border cursor-pointer"
                  :style="{ aspectRatio: cat.ratio || '1/1' }"
                  :class="{
                    'ring-2 ring-blue-500':
                      getGalleryDefault(field.name) === url,
                  }"
                  @click="setGalleryDefault(field.name, url)"
                  :title="'Cliquer pour définir comme image par défaut'"
                />
                <button
                  class="absolute -top-1 -right-1 bg-red-500 text-white rounded-full w-5 h-5 text-xs flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity"
                  @click="removeGalleryImage(field.name, cat.name, imgIdx)"
                  title="Supprimer"
                >
                  ×
                </button>
              </div>
            </div>
            <div class="flex items-center gap-2">
              <input
                type="file"
                accept="image/jpeg,image/png,image/gif,image/webp"
                :ref="(el) => setFileRef(field.name, cat.name, el)"
                class="hidden"
                @change="handleFileUpload(field.name, cat.name, $event)"
              />
              <Button
                :label="'Ajouter'"
                icon="pi pi-upload"
                size="small"
                outlined
                :loading="uploadingState[`${field.name}__${cat.name}`] ?? false"
                @click="triggerFileInput(field.name, cat.name)"
              />
            </div>
          </div>
          <div
            v-if="getGalleryDefault(field.name)"
            class="text-xs text-gray-400"
          >
            Image par défaut :
            <span class="font-medium text-gray-600">{{
              getGalleryDefault(field.name)
            }}</span>
          </div>
        </div>
      </template>

      <!-- Fallback: textarea for other objects/arrays -->
      <Textarea
        v-else
        :modelValue="JSON.stringify(model[field.name] ?? '', null, 2)"
        @update:modelValue="updateJson(field.name, $event)"
        rows="3"
        class="font-mono text-xs"
      />

      <!-- Field error message -->
      <small v-if="errors[field.name]" class="text-red-500 text-xs">{{
        errors[field.name]
      }}</small>
    </div>

    <div v-if="fields.length === 0" class="text-sm text-gray-400 italic">
      Aucun attribut défini pour ce type.
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from "vue";
import InputText from "primevue/inputtext";
import InputNumber from "primevue/inputnumber";
import Textarea from "primevue/textarea";
import Select from "primevue/select";
import Checkbox from "primevue/checkbox";
import Slider from "primevue/slider";
import Button from "primevue/button";
import { uploadApi } from "@/api/client";

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

const model = computed(() => props.modelValue);

const lang = "fr"; // TODO: make configurable

const fields = computed<SchemaField[]>(() => {
  const properties = props.schema?.properties as
    | Record<string, Record<string, unknown>>
    | undefined;
  if (!properties) return [];

  const required = (props.schema?.required as string[]) ?? [];
  const labels = (
    props.schema?.["x-labels"] as
      | Record<string, Record<string, string>>
      | undefined
  )?.[lang];

  return Object.entries(properties).map(([name, prop]) => ({
    name,
    label:
      labels?.[name] ??
      name.replace(/_/g, " ").replace(/\b\w/g, (c) => c.toUpperCase()),
    type: (prop.type as string) ?? "string",
    required: required.includes(name),
    enum: prop.enum as string[] | undefined,
    format: prop.format as string | undefined,
    minimum: prop.minimum as number | undefined,
    maximum: prop.maximum as number | undefined,
    widget: (prop as Record<string, unknown>)["x-widget"] as string | undefined,
    imageCategories: parseImageCategories(
      (prop as Record<string, unknown>)["x-image-categories"],
    ),
  }));
});

function parseImageCategories(
  raw: unknown,
): { name: string; ratio: string }[] | undefined {
  if (!raw) return undefined;
  if (typeof raw === "object" && !Array.isArray(raw)) {
    return Object.entries(raw as Record<string, { ratio?: string }>).map(
      ([name, v]) => ({ name, ratio: v?.ratio ?? "1/1" }),
    );
  }
  if (Array.isArray(raw)) {
    return (raw as string[]).map((name) => ({ name, ratio: "1/1" }));
  }
  return undefined;
}

function getString(name: string): string {
  return (model.value[name] as string) ?? "";
}

function getNumber(name: string): number | undefined {
  const v = model.value[name];
  return typeof v === "number" ? v : undefined;
}

function update(name: string, value: unknown) {
  emit("update:modelValue", { ...model.value, [name]: value });
}

function updateJson(name: string, raw: string) {
  try {
    update(name, JSON.parse(raw));
  } catch {
    // ignore invalid JSON while typing
  }
}

function getI18n(name: string, lang: string): string {
  const obj = model.value[name] as Record<string, string> | undefined;
  return obj?.[lang] ?? "";
}

function updateI18n(name: string, lang: string, value: string) {
  const current = (model.value[name] as Record<string, string>) ?? {};
  update(name, { ...current, [lang]: value });
}

// ── Image gallery helpers ───────────────────────────────────────────
const uploadingState = reactive<Record<string, boolean>>({});
const fileInputRefs = ref<Record<string, HTMLInputElement | null>>({});

function setFileRef(fieldName: string, cat: string, el: unknown) {
  fileInputRefs.value[`${fieldName}__${cat}`] = el as HTMLInputElement | null;
}

function getGalleryData(name: string): Record<string, unknown> {
  return (model.value[name] as Record<string, unknown>) ?? {};
}

function getGalleryCategory(name: string, cat: string): string[] {
  const data = getGalleryData(name);
  return (data[cat] as string[]) ?? [];
}

function getGalleryDefault(name: string): string {
  const data = getGalleryData(name);
  return (data.default_image as string) ?? "";
}

function setGalleryDefault(name: string, url: string) {
  const data = { ...getGalleryData(name) };
  data.default_image = url;
  update(name, data);
}

function removeGalleryImage(name: string, cat: string, imgIdx: number) {
  const data = { ...getGalleryData(name) };
  const arr = [...((data[cat] as string[]) ?? [])];
  const removedUrl = arr[imgIdx];
  arr.splice(imgIdx, 1);
  data[cat] = arr;
  // If removed image was default, clear it
  if (data.default_image === removedUrl) {
    data.default_image = "";
  }
  update(name, data);
}

function triggerFileInput(fieldName: string, cat: string) {
  fileInputRefs.value[`${fieldName}__${cat}`]?.click();
}

async function handleFileUpload(fieldName: string, cat: string, event: Event) {
  const input = event.target as HTMLInputElement;
  const file = input.files?.[0];
  if (!file) return;

  const key = `${fieldName}__${cat}`;
  uploadingState[key] = true;
  try {
    const resp = await uploadApi.upload(file);
    const data = { ...getGalleryData(fieldName) };
    const arr = [...((data[cat] as string[]) ?? [])];
    arr.push(resp.url);
    data[cat] = arr;
    // Set first image as default if none set
    if (!data.default_image) {
      data.default_image = resp.url;
    }
    update(fieldName, data);
  } catch {
    // Upload error — handled silently for now
  } finally {
    uploadingState[key] = false;
    input.value = ""; // reset file input
  }
}

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
      errors[field.name] = "Ce champ est requis";
      valid = false;
    }
    if (
      field.format === "uri" &&
      typeof val === "string" &&
      val !== "" &&
      !isValidUrl(val)
    ) {
      errors[field.name] = "URL invalide";
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
