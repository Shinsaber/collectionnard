<template>
  <!-- Colonnes selon la largeur du conteneur (panneau étroit ou pleine largeur), pas de l'écran. -->
  <div class="@container">
    <div
      class="grid grid-cols-1 @md:grid-cols-2 @3xl:grid-cols-3 gap-3"
      role="group"
      :aria-label="t('items.search.filters')"
    >
      <div v-for="f in fields" :key="f.key" class="flex flex-col gap-1">
        <label
          :id="`${uid}-${f.key}-label`"
          :for="`${uid}-${f.key}`"
          class="text-xs font-medium text-muted"
          >{{ fieldLabel(schema, f.key, lang, "pretty") }}</label
        >

        <!-- Valeurs d'enum : plusieurs = OU -->
        <ElSelect
          v-if="f.control === 'enum'"
          :id="`${uid}-${f.key}`"
          :model-value="draft[attrParam(f.key)] ?? []"
          multiple
          collapse-tags
          clearable
          :placeholder="t('items.search.any')"
          @update:model-value="commit(attrParam(f.key), $event)"
        >
          <ElOption
            v-for="opt in f.options"
            :key="opt"
            :label="enumLabel(schema, f.key, opt, lang)"
            :value="opt"
          />
        </ElSelect>

        <ElSelect
          v-else-if="f.control === 'boolean'"
          :id="`${uid}-${f.key}`"
          :model-value="draft[attrParam(f.key)]?.[0] ?? ''"
          clearable
          :placeholder="t('items.search.any')"
          @update:model-value="commit(attrParam(f.key), [$event ?? ''])"
        >
          <ElOption :label="t('common.yes')" value="true" />
          <ElOption :label="t('common.no')" value="false" />
        </ElSelect>

        <!-- Intervalles : bornes incluses -->
        <div
          v-else-if="f.control === 'numberRange' || f.control === 'dateRange'"
          :id="`${uid}-${f.key}`"
          class="flex flex-wrap items-center gap-2"
          role="group"
          :aria-labelledby="`${uid}-${f.key}-label`"
        >
          <template v-for="op in RANGE_OPS" :key="op">
            <ElInputNumber
              v-if="f.control === 'numberRange'"
              :model-value="numberOf(attrParam(f.key, op))"
              :min="f.minimum"
              :max="f.maximum"
              :precision="f.integer ? 0 : undefined"
              controls-position="right"
              class="min-w-0 flex-1"
              :placeholder="t(`items.search.${op}`)"
              :aria-label="t(`items.search.${op}`)"
              @update:model-value="
                commit(attrParam(f.key, op), [
                  $event == null ? '' : String($event),
                ])
              "
            />
            <ElInput
              v-else
              type="date"
              :model-value="draft[attrParam(f.key, op)]?.[0] ?? ''"
              class="min-w-36 flex-1"
              :aria-label="t(`items.search.${op}`)"
              @update:model-value="
                commit(attrParam(f.key, op), [String($event ?? '')])
              "
            />
          </template>
        </div>

        <!-- Texte libre / multilingue : sous-chaîne, appliquée à la validation -->
        <ElInput
          v-else
          :id="`${uid}-${f.key}`"
          :model-value="draft[attrParam(f.key, 'contains')]?.[0] ?? ''"
          clearable
          :placeholder="
            f.control === 'i18nText'
              ? t('items.search.containsAnyLang')
              : t('items.search.contains')
          "
          @update:model-value="setText(attrParam(f.key, 'contains'), $event)"
          @change="emitDraft"
          @clear="emitDraft"
        />
      </div>

      <p v-if="fields.length === 0" class="text-sm text-muted italic">
        {{ t("items.search.noFilterableField") }}
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, useId, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useLang } from "@/composables/useLang";
import { attrParam, filterFields } from "@/lib/itemSearch";
import { enumLabel, fieldLabel, type Schema } from "@/lib/schema";

// Filtres d'attributs du catalogue, générés depuis le JSON Schema du type choisi
// (voir lib/itemSearch). `modelValue` : nom de paramètre `attr.…` → valeurs. Listes et
// bornes sont émises aussitôt ; le texte libre à la validation (Entrée / sortie du champ),
// pour ne pas relancer une recherche à chaque frappe.
const props = defineProps<{
  schema: Schema | null | undefined;
  modelValue: Record<string, string[]>;
}>();

const emit = defineEmits<{
  "update:modelValue": [value: Record<string, string[]>];
}>();

const RANGE_OPS = ["gte", "lte"] as const;

const { t } = useI18n();
const { lang } = useLang();
const uid = useId();

const fields = computed(() => filterFields(props.schema));

// Copie de travail : le texte en cours de saisie n'est pas encore émis.
const draft = ref<Record<string, string[]>>({});
watch(
  () => props.modelValue,
  (v) => {
    draft.value = { ...v };
  },
  { immediate: true, deep: true },
);

function numberOf(name: string): number | undefined {
  const raw = draft.value[name]?.[0];
  if (raw == null || raw === "") return undefined;
  const n = Number(raw);
  return Number.isFinite(n) ? n : undefined;
}

function clean(values: unknown[]): string[] {
  return values.map((v) => String(v ?? "")).filter((v) => v !== "");
}

function setText(name: string, value: unknown) {
  draft.value = { ...draft.value, [name]: clean([value]) };
}

function commit(name: string, values: unknown[]) {
  draft.value = { ...draft.value, [name]: clean(values) };
  emitDraft();
}

function emitDraft() {
  const out = Object.fromEntries(
    Object.entries(draft.value).filter(([, v]) => v.length > 0),
  );
  emit("update:modelValue", out);
}
</script>
