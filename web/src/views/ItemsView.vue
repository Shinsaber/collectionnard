<template>
  <div>
    <div class="flex items-center justify-between mb-6">
      <h1 class="text-2xl font-bold text-gray-800 dark:text-gray-100">
        {{ t("items.list.title") }}
      </h1>
      <ElButton v-if="auth.isAdmin" type="primary" @click="openCreate">
        <i class="pi pi-plus mr-2" aria-hidden="true"></i>
        {{ t("items.list.newItem") }}
      </ElButton>
    </div>

    <!-- Search bar: state lives in the URL query (shareable, back button) -->
    <div class="flex gap-3 mb-3 flex-wrap items-center">
      <ElSelect
        :model-value="state.item_type_id ?? ''"
        :placeholder="t('items.list.allTypes')"
        clearable
        class="w-48!"
        :aria-label="t('items.list.filterByType')"
        @update:model-value="onTypeChange"
      >
        <ElOption
          v-for="type in itemTypes"
          :key="type.id"
          :label="typeTitle(type, lang)"
          :value="type.id"
        />
      </ElSelect>
      <ElInput
        v-model="searchDraft"
        :placeholder="t('items.list.searchPlaceholder')"
        clearable
        class="w-64!"
        :aria-label="t('common.search')"
        @keyup.enter="applyText"
        @clear="applyText"
      />
      <ElInput
        v-model="tagDraft"
        :placeholder="t('items.list.tagPlaceholder')"
        clearable
        class="w-32!"
        :aria-label="t('items.list.tag')"
        @keyup.enter="applyText"
        @clear="applyText"
      />
      <ElButton :aria-label="t('common.search')" @click="applyText">
        <i class="pi pi-search" aria-hidden="true"></i>
      </ElButton>
      <ElSelect
        :model-value="currentSort"
        class="w-56!"
        :aria-label="t('items.search.sortBy')"
        @update:model-value="update({ sort: $event || undefined })"
      >
        <ElOption
          v-for="o in sortOptions"
          :key="o.value"
          :label="o.label"
          :value="o.value"
        />
      </ElSelect>
      <ElButton
        :disabled="!selectedType"
        :title="selectedType ? undefined : t('items.search.chooseTypeFirst')"
        :aria-expanded="showFilters"
        @click="showFilters = !showFilters"
      >
        <i class="pi pi-filter mr-2" aria-hidden="true"></i>
        {{ t("items.search.filters") }}
        <ElTag v-if="attrCount > 0" size="small" round class="ml-2">{{
          attrCount
        }}</ElTag>
      </ElButton>
      <ElButton v-if="hasCriteria" text @click="resetAll">
        {{ t("items.search.reset") }}
      </ElButton>
    </div>

    <!-- Related-to filter (from an item page) -->
    <div v-if="state.related_to" class="mb-3">
      <RelatedToTag
        :item-id="state.related_to"
        :relation-type-id="state.relation_type_id"
        @close="update({ related_to: undefined, relation_type_id: undefined })"
      />
    </div>

    <!-- Attribute filters, generated from the selected type's schema -->
    <div
      v-if="showFilters && selectedType"
      class="mb-4 p-4 rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800"
    >
      <ItemSearchFilters
        :schema="selectedSchema"
        :model-value="state.attrs"
        @update:model-value="update({ attrs: $event })"
      />
    </div>

    <ElTable
      v-loading="pag.loading.value"
      :data="pag.data.value"
      stripe
      row-key="id"
      highlight-current-row
      class="cursor-pointer"
      @row-click="goToItem"
    >
      <ElTableColumn :label="t('common.type')" width="140">
        <template #default="{ row }">
          <ElTag type="info" size="small">{{
            typeTitle((row as Item).item_type, lang) || "?"
          }}</ElTag>
        </template>
      </ElTableColumn>
      <ElTableColumn :label="t('items.list.colLabel')">
        <template #default="{ row }">
          {{ itemLabel(row as Item) }}
        </template>
      </ElTableColumn>
      <ElTableColumn :label="t('items.list.colTags')">
        <template #default="{ row }">
          <div class="flex gap-1 flex-wrap">
            <ElTag
              v-for="tag in (row as Item).tags ?? []"
              :key="tag.id"
              type="info"
              effect="plain"
              size="small"
              >{{ tag.name }}</ElTag
            >
          </div>
        </template>
      </ElTableColumn>
      <ElTableColumn
        v-if="auth.isAdmin"
        :label="t('common.actions')"
        width="150"
      >
        <template #default="{ row }">
          <div class="flex gap-1">
            <ElButton
              text
              size="small"
              :title="t('items.list.graph')"
              :aria-label="t('items.list.viewGraph')"
              @click.stop="router.push(`/items/${(row as Item).id}/graph`)"
            >
              <i class="pi pi-sitemap" aria-hidden="true"></i>
            </ElButton>
            <ElButton
              text
              size="small"
              :aria-label="t('common.edit')"
              @click.stop="openEdit(row as Item)"
            >
              <i class="pi pi-pencil" aria-hidden="true"></i>
            </ElButton>
            <ElButton
              text
              size="small"
              type="danger"
              :aria-label="t('common.delete')"
              @click.stop="confirmDelete(row as Item)"
            >
              <i class="pi pi-trash" aria-hidden="true"></i>
            </ElButton>
          </div>
        </template>
      </ElTableColumn>
    </ElTable>
    <ElPagination
      class="mt-3 justify-end"
      layout="total, prev, pager, next"
      :total="pag.total.value"
      :page-size="pag.perPage.value"
      :current-page="pag.page.value"
      @current-change="onPageChange"
    />

    <!-- Create / Edit Dialog -->
    <ItemEditDialog
      v-model:visible="dialogVisible"
      :item="editing"
      :item-types="itemTypes"
      @saved="load"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import { itemApi } from "@/api/client";
import { useMetaTypes } from "@/stores/metaTypes";
import { useNotify } from "@/composables/useNotify";
import { useConfirmDialog } from "@/composables/useConfirmDialog";
import { usePagination } from "@/composables/usePagination";
import { useAuthStore } from "@/stores/auth";
import { useLang } from "@/composables/useLang";
import {
  fieldLabel,
  itemLabel as renderItemLabel,
  typeTitle,
  type Schema,
} from "@/lib/schema";
import {
  activeAttrCount,
  fromQuery,
  sortFields,
  toQuery,
  type ItemSearch,
} from "@/lib/itemSearch";
import ItemEditDialog from "@/components/item/ItemEditDialog.vue";
import ItemSearchFilters from "@/components/item/ItemSearchFilters.vue";
import RelatedToTag from "@/components/item/RelatedToTag.vue";
import type { Item, ItemListParams } from "@/api/types";

const auth = useAuthStore();
const route = useRoute();
const router = useRouter();
const { t } = useI18n();
const { lang } = useLang();
const { notifyError, notifySuccess } = useNotify();
const { confirm } = useConfirmDialog();

const metaTypes = useMetaTypes();
const itemTypes = computed(() => metaTypes.itemTypes);
// Les paramètres viennent de toQuery (noms et valeurs déjà conformes à ItemListParams).
const pag = usePagination((p) => itemApi.list(p as ItemListParams));

const dialogVisible = ref(false);
const editing = ref<Item | null>(null);

// ── Search state = URL query (lib/itemSearch) ───────────────────────
const state = computed<ItemSearch>(() => fromQuery(route.query));

/** Applique un changement de critères ; toute modification hors page revient à la page 1. */
function update(patch: Partial<ItemSearch>) {
  const next: ItemSearch = { ...state.value, page: undefined, ...patch };
  router.replace({ query: toQuery(next) });
}

const searchDraft = ref("");
const tagDraft = ref("");
watch(
  state,
  (s) => {
    searchDraft.value = s.search ?? "";
    tagDraft.value = s.tag ?? "";
  },
  { immediate: true },
);

function applyText() {
  const search = searchDraft.value.trim() || undefined;
  // Sans recherche, un tri par pertinence n'a plus de sens.
  const sort =
    !search && state.value.sort?.endsWith("relevance")
      ? undefined
      : state.value.sort;
  update({ search, tag: tagDraft.value.trim() || undefined, sort });
}

const selectedType = computed(() =>
  itemTypes.value.find((it) => it.id === state.value.item_type_id),
);
const selectedSchema = computed(
  () => selectedType.value?.attributes_schema as Schema | undefined,
);

// Les filtres et le tri par attribut dépendent du schéma : ils tombent avec le type.
function onTypeChange(id: string | undefined) {
  const sort = state.value.sort?.replace(/^-/, "").startsWith("attr.")
    ? undefined
    : state.value.sort;
  update({ item_type_id: id || undefined, attrs: {}, sort });
}

const attrCount = computed(() => activeAttrCount(state.value));
const showFilters = ref(attrCount.value > 0);
const hasCriteria = computed(
  () => Object.keys(toQuery(state.value)).length > 0,
);

function resetAll() {
  router.replace({ query: {} });
}

// ── Sort ────────────────────────────────────────────────────────────
const currentSort = computed(
  () => state.value.sort ?? (state.value.search ? "-relevance" : "-created_at"),
);

const sortOptions = computed(() => {
  const opts: { value: string; label: string }[] = [];
  if (state.value.search)
    opts.push({ value: "-relevance", label: t("items.search.sortRelevance") });
  opts.push(
    { value: "-created_at", label: t("items.search.sortNewest") },
    { value: "created_at", label: t("items.search.sortOldest") },
    { value: "-updated_at", label: t("items.search.sortUpdated") },
  );
  for (const f of sortFields(selectedSchema.value)) {
    const label = fieldLabel(selectedSchema.value, f.key, lang.value, "pretty");
    const [asc, desc] = f.numeric
      ? ["items.search.sortNumAsc", "items.search.sortNumDesc"]
      : ["items.search.sortTextAsc", "items.search.sortTextDesc"];
    opts.push(
      { value: `attr.${f.key}`, label: t(asc, { field: label }) },
      { value: `-attr.${f.key}`, label: t(desc, { field: label }) },
    );
  }
  return opts;
});

// ── Loading ─────────────────────────────────────────────────────────
function load() {
  pag.page.value = state.value.page ?? 1;
  const { page: _page, ...params } = toQuery(state.value);
  pag.load(params).catch((e) => notifyError(e, t("items.list.loadFailed")));
}

onMounted(async () => {
  await metaTypes
    .loadItemTypes()
    .catch((e) => notifyError(e, t("items.list.typesUnavailable")));
  load();
  watch(() => route.query, load);
});

function onPageChange(p: number) {
  router.replace({ query: toQuery({ ...state.value, page: p }) });
}

// ── Table actions ───────────────────────────────────────────────────
function itemLabel(item: Item): string {
  return renderItemLabel(item, lang.value);
}

function goToItem(row: Item) {
  router.push(`/items/${(row as Item).id}`);
}

function openCreate() {
  editing.value = null;
  dialogVisible.value = true;
}

function openEdit(row: Item) {
  editing.value = row;
  dialogVisible.value = true;
}

async function confirmDelete(row: Item) {
  const ok = await confirm({
    message: t("items.list.confirmDelete", { label: itemLabel(row as Item) }),
    confirmText: t("common.delete"),
    danger: true,
  });
  if (!ok) return;
  try {
    await itemApi.delete(row.id);
    notifySuccess(t("items.list.deleted"));
    load();
  } catch (e) {
    notifyError(e);
  }
}
</script>
