<template>
  <!-- Recherche et tri des lots d'une collection (P7-1) : texte, type d'item, lots de plusieurs exemplaires ;
       tri par date d'ajout, quantité, ou attribut du type choisi (les lots d'un même item se suivent). -->
  <form
    class="mb-4 flex flex-wrap items-center gap-2"
    data-test="collection-toolbar"
    @submit.prevent="emitChange"
  >
    <label :for="`${uid}-search`" class="sr-only">{{
      t("collections.toolbar.search")
    }}</label>
    <ElInput
      :id="`${uid}-search`"
      v-model="model.search"
      class="w-full md:max-w-64"
      clearable
      :placeholder="t('collections.toolbar.search')"
      data-test="toolbar-search"
      @input="searchSoon"
      @keyup.enter="emitChange"
      @clear="emitChange"
    >
      <template #prefix>
        <i class="pi pi-search text-xs" aria-hidden="true"></i>
      </template>
    </ElInput>
    <ElSelect
      v-if="types.length > 1"
      v-model="model.item_type_id"
      class="w-44!"
      clearable
      :placeholder="t('collections.toolbar.allTypes')"
      :aria-label="t('common.type')"
      data-test="toolbar-type"
      @change="onTypeChange"
    >
      <ElOption
        v-for="it in types"
        :key="it.id"
        :label="typeTitle(it, lang)"
        :value="it.id"
      />
    </ElSelect>
    <ElSelect
      v-model="model.sort"
      class="w-52!"
      :aria-label="t('collections.toolbar.sort')"
      data-test="toolbar-sort"
      @change="emitChange"
    >
      <ElOption
        v-for="o in sortOptions"
        :key="o.value"
        :label="o.label"
        :value="o.value"
      />
    </ElSelect>
    <ElCheckbox
      v-model="model.multiple"
      data-test="toolbar-multiple"
      @change="emitChange"
      >{{ t("collections.toolbar.multiple") }}</ElCheckbox
    >
  </form>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, useId } from "vue";
import { useI18n } from "vue-i18n";
import { useLang } from "@/composables/useLang";
import { sortFields, ATTR_PREFIX } from "@/lib/itemSearch";
import { fieldLabel, typeTitle, type Schema } from "@/lib/schema";
import { DEFAULT_LOT_SORT, type LotQuery } from "@/lib/lots";
import type { ItemType } from "@/api/types";

const props = defineProps<{
  /** Types d'item proposés au filtre (ceux que la collection permet). */
  types: ItemType[];
}>();
const model = defineModel<LotQuery>({ required: true });
const emit = defineEmits<{ change: [] }>();

const { t } = useI18n();
const { lang } = useLang();
const uid = useId();

const sortOptions = computed(() => {
  const base = [
    { value: DEFAULT_LOT_SORT, label: t("collections.toolbar.sortRecent") },
    { value: "created_at", label: t("collections.toolbar.sortOldest") },
    { value: "-quantity", label: t("collections.toolbar.sortQuantity") },
  ];
  if (model.value.search.trim())
    base.unshift({
      value: "relevance",
      label: t("collections.toolbar.sortRelevance"),
    });
  const type = props.types.find((it) => it.id === model.value.item_type_id);
  const schema = type?.attributes_schema as Schema | undefined;
  for (const f of sortFields(schema)) {
    const label = fieldLabel(schema, f.key, lang.value);
    base.push(
      { value: `${ATTR_PREFIX}${f.key}`, label: `${label} ↑` },
      { value: `-${ATTR_PREFIX}${f.key}`, label: `${label} ↓` },
    );
  }
  return base;
});

function onTypeChange() {
  // Un tri par attribut n'a de sens que pour le type qui le porte.
  if (model.value.sort.includes(ATTR_PREFIX))
    model.value.sort = DEFAULT_LOT_SORT;
  emitChange();
}

// Recherche relancée 400 ms après la dernière frappe (Entrée la lance tout de suite).
let timer: ReturnType<typeof setTimeout> | undefined;
function searchSoon() {
  clearTimeout(timer);
  timer = setTimeout(emitChange, 400);
}
onBeforeUnmount(() => clearTimeout(timer));

function emitChange() {
  clearTimeout(timer);
  // Sans recherche, la pertinence n'a plus de sens.
  if (model.value.sort === "relevance" && !model.value.search.trim())
    model.value.sort = DEFAULT_LOT_SORT;
  emit("change");
}
</script>
