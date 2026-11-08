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
        v-model:lendable="xLendable"
        :labels="ft.labels"
        :show-completeness="ft.completeness"
        :show-lendable="ft.lendable"
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
import { isCompleteness, isLendable } from "@/lib/schema";
import SchemaFieldCard from "./SchemaFieldCard.vue";
import SchemaRootCard from "./SchemaRootCard.vue";
import { buildProperty, parseSchema } from "./schemaBuilderConvert";
import {
  MANAGED_ROOT_KEYS,
  arrayTypeOption,
  baseTypeOptions,
  defaultFeatures,
  emptyField,
  i18nTypeOption,
  imageGalleryTypeOption,
  linksTypeOption,
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
// Prêts possibles (x-lendable, types de collection, P7-3).
const xLendable = ref(false);
// Mots-clés racine inconnus, restitués à l'identique.
const rootExtra = ref<Record<string, unknown>>({});

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
  if (xLendable.value) result["x-lendable"] = true;
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
  xLendable.value = isLendable(value);
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
  fields.value.splice(
    at,
    0,
    ft.value.criteria
      ? { ...emptyField(), type: "integer", minimum: 0, maximum: 10 }
      : emptyField(),
  );
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
