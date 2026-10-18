<template>
  <li
    class="border-b border-line last:border-b-0"
    :class="{ 'bg-bar shadow-[inset_3px_0_0_var(--c-primary)]': open }"
  >
    <!-- Résumé : clé, libellé, type, obligatoire ; un clic déplie l'édition. -->
    <button
      type="button"
      class="flex min-h-14 w-full items-center gap-3 px-3.5 py-2 text-left"
      :aria-expanded="open"
      :aria-controls="`${uid}-body`"
      @click="emit('toggle')"
    >
      <span
        class="flex min-w-0 flex-1 flex-col gap-0.5 sm:flex-row sm:items-center sm:gap-3"
      >
        <span class="font-mono text-[13px] sm:w-40 sm:shrink-0 sm:truncate">{{
          field.key || t("schema.field.keyPlaceholder")
        }}</span>
        <span
          class="truncate text-[15px] font-semibold"
          :class="{ 'font-bold': open }"
          >{{ displayLabel || "—" }}</span
        >
      </span>
      <span
        class="hidden shrink-0 rounded-full bg-soft px-2.5 py-0.5 text-xs sm:inline"
        >{{ typeLabel }}</span
      >
      <span
        v-if="field.required"
        class="shrink-0 rounded-sm bg-danger/12 px-2 py-0.5 font-mono text-[11px] uppercase text-danger"
        >{{ t("schema.field.requiredShort") }}</span
      >
      <span
        v-if="field.private"
        class="shrink-0 rounded-sm bg-tint px-2 py-0.5 font-mono text-[11px] uppercase text-tint-ink"
        >{{ t("schema.field.privateShort") }}</span
      >
      <i
        class="pi shrink-0 text-xs"
        :class="open ? 'pi-chevron-up text-link' : 'pi-chevron-down text-muted'"
        aria-hidden="true"
      ></i>
    </button>

    <div
      v-if="open"
      :id="`${uid}-body`"
      class="flex flex-col gap-3 px-3.5 pb-4"
    >
      <!-- Ordre, clé, type, requis, actions -->
      <div class="flex flex-wrap items-center gap-2">
        <div class="flex gap-0.5">
          <ElButton
            text
            circle
            :aria-label="t('schema.field.moveUp')"
            :disabled="index === 0"
            @click="emit('move', -1)"
          >
            <i class="pi pi-chevron-up" aria-hidden="true"></i>
          </ElButton>
          <ElButton
            text
            circle
            :aria-label="t('schema.field.moveDown')"
            :disabled="index === count - 1"
            @click="emit('move', 1)"
          >
            <i class="pi pi-chevron-down" aria-hidden="true"></i>
          </ElButton>
        </div>

        <ElInput
          :model-value="field.key"
          :placeholder="t('schema.field.keyPlaceholder')"
          :aria-label="t('schema.field.key')"
          class="min-w-40 flex-1 font-mono"
          @update:model-value="set('key', $event)"
        />

        <ElSelect
          :model-value="field.type"
          :aria-label="t('schema.field.type')"
          class="w-full! sm:w-48!"
          @update:model-value="set('type', $event)"
        >
          <ElOption
            v-for="o in typeOptions"
            :key="o.value"
            :label="t(o.labelKey)"
            :value="o.value"
          />
        </ElSelect>

        <ElCheckbox
          :id="`${uid}-req`"
          :model-value="field.required"
          :title="t('schema.field.required')"
          @update:model-value="set('required', $event)"
          >{{ t("schema.field.requiredShort") }}</ElCheckbox
        >
        <ElCheckbox
          v-if="features.identifier && field.type === 'string'"
          :model-value="field.identifier ?? false"
          :title="t('schema.field.identifier')"
          data-test="x-identifier"
          @update:model-value="set('identifier', Boolean($event))"
          >{{ t("schema.field.identifierShort") }}</ElCheckbox
        >
        <ElCheckbox
          v-if="features.private"
          :model-value="field.private ?? false"
          :title="t('schema.field.private')"
          data-test="x-private"
          @update:model-value="set('private', Boolean($event))"
          >{{ t("schema.field.privateShort") }}</ElCheckbox
        >

        <div class="ml-auto flex gap-0.5">
          <ElButton
            text
            circle
            :title="t('schema.field.addAfter')"
            :aria-label="t('schema.field.addAfter')"
            @click="emit('add-after')"
          >
            <i class="pi pi-plus" aria-hidden="true"></i>
          </ElButton>
          <ElButton
            text
            type="danger"
            circle
            :aria-label="t('schema.field.remove')"
            @click="emit('remove')"
          >
            <i class="pi pi-trash" aria-hidden="true"></i>
          </ElButton>
        </div>
      </div>

      <SchemaFieldConstraints
        :field="field"
        :features="features"
        @update:field="emit('update:field', $event)"
      />
    </div>
  </li>
</template>

<script setup lang="ts">
import { computed, useId } from "vue";
import { useI18n } from "vue-i18n";
import { useLang } from "@/composables/useLang";
import SchemaFieldConstraints from "./SchemaFieldConstraints.vue";
import { arrayTypeOption, formatOptions } from "./schemaBuilderTypes";
import type {
  Option,
  SchemaBuilderFeatures,
  SchemaField,
} from "./schemaBuilderTypes";

// Ligne d'un champ du schéma : résumé repliable, puis en-tête (ordre, clé, type, requis,
// actions) et contraintes déléguées à SchemaFieldConstraints. Ne mute jamais `field` :
// chaque changement est émis en `update:field` (patch partiel) au parent, qui possède la liste.
const props = defineProps<{
  field: SchemaField;
  index: number;
  count: number;
  features: SchemaBuilderFeatures;
  typeOptions: Option[];
  open: boolean;
}>();

const emit = defineEmits<{
  "update:field": [patch: Partial<SchemaField>];
  move: [dir: -1 | 1];
  remove: [];
  "add-after": [];
  toggle: [];
}>();

const { t } = useI18n();
const { lang } = useLang();
const uid = useId();

/** Libellé dans la langue courante, sinon le premier renseigné. */
const displayLabel = computed(
  () =>
    props.field.labels?.[lang.value] ||
    Object.values(props.field.labels ?? {}).find(Boolean) ||
    "",
);

/** Type affiché : le format pour un texte formaté (Date, URL…), sinon le type ; objet compris. */
const typeLabel = computed(() => {
  const f = props.field;
  const fmt =
    f.type === "string" && formatOptions.find((o) => o.value === f.format);
  if (fmt) return t(fmt.labelKey);
  const o = [...props.typeOptions, arrayTypeOption].find(
    (x) => x.value === f.type,
  );
  if (o) return t(o.labelKey);
  return f.type === "object" ? t("schema.types.object") : f.type;
});

function set<K extends keyof SchemaField>(key: K, value: unknown) {
  emit("update:field", { [key]: value } as Partial<SchemaField>);
}
</script>
