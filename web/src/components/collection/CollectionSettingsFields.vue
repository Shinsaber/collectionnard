<template>
  <!-- Types d'item acceptés et tri par défaut d'une collection (P7-1b), à la création et dans « Modifier ». -->
  <div class="flex flex-col gap-4" data-test="collection-settings">
    <div class="flex flex-col gap-1">
      <label :for="`${uid}-types`" class="text-sm font-medium">{{
        t("collections.settings.itemTypes")
      }}</label>
      <ElSelect
        :id="`${uid}-types`"
        v-model="itemTypeIds"
        multiple
        clearable
        :placeholder="t('collections.settings.allTypes')"
        data-test="settings-types"
      >
        <ElOption
          v-for="it in typeOptions"
          :key="it.id"
          :label="typeTitle(it, lang)"
          :value="it.id"
        />
      </ElSelect>
      <p class="text-xs text-muted">
        {{
          specialized
            ? t("collections.settings.specializedHelp")
            : t("collections.settings.itemTypesHelp")
        }}
      </p>
    </div>
    <div class="flex flex-col gap-1">
      <label :for="`${uid}-sort`" class="text-sm font-medium">{{
        t("collections.settings.defaultSort")
      }}</label>
      <ElSelect
        :id="`${uid}-sort`"
        v-model="sortValue"
        data-test="settings-sort"
      >
        <ElOption
          v-for="o in sortOptions"
          :key="o.value"
          :label="o.label"
          :value="o.value"
        />
      </ElSelect>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, useId, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useLang } from "@/composables/useLang";
import { useNotify } from "@/composables/useNotify";
import { useMetaTypes } from "@/stores/metaTypes";
import { lotSortOptions, specializedTypeId } from "@/lib/collections";
import { DEFAULT_LOT_SORT } from "@/lib/lots";
import { typeTitle, type Schema } from "@/lib/schema";
import type { CollectionType } from "@/api/types";

const props = defineProps<{
  /** Type de la collection : il borne les types d'item proposés. */
  collectionType: CollectionType | undefined;
}>();
/** Types d'item acceptés ; vide = tous ceux du type de collection. */
const itemTypeIds = defineModel<string[]>("itemTypeIds", { required: true });
/** Tri par défaut ; vide = ajout le plus récent. */
const defaultSort = defineModel<string>("defaultSort", { required: true });

const { t } = useI18n();
const { lang } = useLang();
const { notifyError } = useNotify();
const metaTypes = useMetaTypes();
const uid = useId();

onMounted(() => metaTypes.loadItemTypes().catch((e) => notifyError(e)));

/** Types d'item que le type de collection permet (tous s'il ne limite rien). */
const typeOptions = computed(() => {
  const limit = props.collectionType?.item_type_ids ?? [];
  return limit.length
    ? metaTypes.itemTypes.filter((it) => limit.includes(it.id))
    : metaTypes.itemTypes;
});

/** Seul type accepté : la collection est spécialisée et se trie par ses attributs. */
const specialized = computed(() => {
  const id = specializedTypeId({
    item_type_ids: itemTypeIds.value,
    collection_type: props.collectionType,
  });
  return id ? metaTypes.itemTypes.find((it) => it.id === id) : undefined;
});

const sortOptions = computed(() =>
  lotSortOptions(
    t,
    specialized.value?.attributes_schema as Schema | undefined,
    lang.value,
  ),
);

// Le sélecteur montre le défaut de l'API (vide) comme « Ajoutés récemment ».
const sortValue = computed({
  get: () => defaultSort.value || DEFAULT_LOT_SORT,
  set: (v: string) => {
    defaultSort.value = v === DEFAULT_LOT_SORT ? "" : v;
  },
});

// Les types changent : un tri par un attribut qui n'est plus proposé revient au défaut.
watch(sortOptions, (options) => {
  if (defaultSort.value && !options.some((o) => o.value === defaultSort.value))
    defaultSort.value = "";
});
</script>
