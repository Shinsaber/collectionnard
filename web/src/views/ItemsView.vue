<template>
  <div>
    <PageHeader
      :title="t('items.list.title')"
      :meta="pag.loading.value ? undefined : t('items.list.count', total)"
    >
      <template #actions>
        <ExportButton
          :label="t('items.export.csv')"
          :formats="['csv']"
          :run="exportCsv"
          :error-summary="t('items.export.failed')"
          :disabled="!selectedType"
          :title="selectedType ? undefined : t('items.export.chooseTypeFirst')"
        />
        <ElButton v-if="auth.canEditCatalog" type="primary" @click="openCreate">
          <i class="pi pi-plus mr-2" aria-hidden="true"></i>
          {{ t("items.list.newItem") }}
        </ElButton>
      </template>
    </PageHeader>

    <!-- Recherche et tri : l'état vit dans l'URL (partageable, bouton retour). -->
    <div class="mb-4 flex flex-wrap items-center gap-2.5">
      <ElInput
        v-model="searchDraft"
        :placeholder="t('items.list.searchPlaceholder')"
        clearable
        size="large"
        class="min-w-0 flex-[1_1_280px]"
        :aria-label="t('common.search')"
        @keyup.enter="applyText"
        @clear="applyText"
      >
        <template #prefix>
          <i class="pi pi-search" aria-hidden="true"></i>
        </template>
      </ElInput>
      <ElSelect
        :model-value="currentSort"
        size="large"
        class="w-full! sm:w-64!"
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
    </div>

    <!-- Filtres actifs + interrupteur du panneau (à droite) -->
    <div
      class="mb-4 flex min-h-10 flex-wrap items-center justify-between gap-3"
    >
      <div class="flex flex-wrap items-center gap-2">
        <FilterChip
          v-if="selectedType"
          :label="
            t('items.search.typeChip', { type: typeTitle(selectedType, lang) })
          "
          @remove="onTypeChange(undefined)"
        />
        <FilterChip
          v-if="state.tag"
          :label="t('items.search.tagChip', { tag: state.tag })"
          @remove="update({ tag: undefined })"
        />
        <FilterChip
          v-if="attrCount > 0"
          :label="t('items.search.attrChip', attrCount)"
          @remove="update({ attrs: {} })"
        />
        <RelatedToTag
          v-if="state.related_to"
          :item-id="state.related_to"
          :relation-type-id="state.relation_type_id"
          @close="
            update({ related_to: undefined, relation_type_id: undefined })
          "
        />
      </div>
      <button
        type="button"
        class="ml-auto flex h-10 items-center gap-2.5 rounded-full bg-surface pr-2 pl-3.5 text-sm font-semibold text-ink shadow-control"
        v-bind="filterToggleAttrs"
        @click="toggleFilters"
      >
        <i class="pi pi-sliders-h text-sm" aria-hidden="true"></i>
        {{ t("items.search.filters") }}
        <span
          v-if="activeCount > 0"
          class="rounded-full bg-accent px-1.5 font-mono text-[11px] text-accent-ink"
          >{{ activeCount }}</span
        >
        <span
          v-if="isWide"
          aria-hidden="true"
          class="relative block h-5.5 w-9.5 rounded-full transition-colors"
          :class="panelOpen ? 'bg-primary' : 'bg-dash'"
        >
          <span
            class="absolute top-0.75 size-4 rounded-full bg-surface shadow-control transition-[left]"
            :class="panelOpen ? 'left-4.75' : 'left-0.75'"
          ></span>
        </span>
        <span v-else class="w-1.5"></span>
      </button>
    </div>

    <div class="flex items-start gap-5">
      <section class="min-w-0 flex-1">
        <ItemList
          :items="pag.data.value"
          :loading="pag.loading.value"
          :can-edit="auth.canEditCatalog"
          :can-delete="auth.isAdmin"
          :empty-text="
            hasCriteria ? t('items.list.empty') : t('items.list.emptyCatalog')
          "
          @edit="openEdit"
          @remove="confirmDelete"
        />
        <ElPagination
          class="mt-4 justify-end"
          hide-on-single-page
          :layout="isMobile ? 'prev, pager, next' : 'total, prev, pager, next'"
          :pager-count="isMobile ? 5 : 7"
          :total="total"
          :page-size="pag.perPage.value"
          :current-page="pag.page.value"
          @current-change="onPageChange"
        />
      </section>

      <!-- Grand écran : panneau de filtres à droite, masquable (choix mémorisé). -->
      <aside
        v-if="isWide && panelOpen"
        id="panneau-filtres"
        :aria-label="t('items.search.filters')"
        class="sticky top-6 w-75 shrink-0 rounded-2xl bg-surface p-4.5 shadow-card"
      >
        <div class="mb-4 flex items-center justify-between">
          <h2 class="text-base font-extrabold">
            {{ t("items.search.filters") }}
          </h2>
          <ElButton v-if="hasCriteria" link type="primary" @click="resetAll">
            {{ t("items.search.reset") }}
          </ElButton>
        </div>
        <CatalogFilters v-bind="filterProps" v-on="filterHandlers" />
      </aside>
    </div>

    <!-- Plus étroit : les mêmes filtres dans une feuille (du bas sur mobile, de droite sur tablette). -->
    <ElDrawer
      v-if="!isWide"
      v-model="sheetOpen"
      :direction="isMobile ? 'btt' : 'rtl'"
      :size="isMobile ? 'auto' : '380px'"
      :title="t('items.search.filters')"
      class="app-sheet"
    >
      <CatalogFilters v-bind="filterProps" v-on="filterHandlers" />
      <template #footer>
        <div class="flex gap-2.5">
          <ElButton class="flex-1" :disabled="!hasCriteria" @click="resetAll">
            {{ t("items.search.reset") }}
          </ElButton>
          <ElButton
            class="flex-[1.6]"
            type="primary"
            @click="sheetOpen = false"
          >
            {{ t("items.search.showResults") }}
            <span class="ml-1.5 font-mono text-xs">({{ total }})</span>
          </ElButton>
        </div>
      </template>
    </ElDrawer>

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
import { importExportApi, itemApi } from "@/api/client";
import { useMetaTypes } from "@/stores/metaTypes";
import { useNotify } from "@/composables/useNotify";
import { useConfirmDialog } from "@/composables/useConfirmDialog";
import { usePagination } from "@/composables/usePagination";
import {
  MOBILE_QUERY,
  WIDE_QUERY,
  useMediaQuery,
} from "@/composables/useMediaQuery";
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
import ItemList from "@/components/item/ItemList.vue";
import CatalogFilters from "@/components/item/CatalogFilters.vue";
import RelatedToTag from "@/components/item/RelatedToTag.vue";
import ExportButton from "@/components/common/ExportButton.vue";
import FilterChip from "@/components/common/FilterChip.vue";
import PageHeader from "@/components/common/PageHeader.vue";
import { defaultDelimiter } from "@/lib/csv";
import type { Item, ItemListParams } from "@/api/types";

const auth = useAuthStore();
const route = useRoute();
const router = useRouter();
const { t } = useI18n();
const { lang } = useLang();
const { notifyError, notifySuccess } = useNotify();
const { confirm } = useConfirmDialog();
const isWide = useMediaQuery(WIDE_QUERY);
const isMobile = useMediaQuery(MOBILE_QUERY);

const metaTypes = useMetaTypes();
const itemTypes = computed(() => metaTypes.itemTypes);
// Les paramètres viennent de toQuery (noms et valeurs déjà conformes à ItemListParams).
const pag = usePagination((p) => itemApi.list(p as ItemListParams));
const total = computed(() => pag.total.value);

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
watch(
  state,
  (s) => {
    searchDraft.value = s.search ?? "";
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
  update({ search, sort });
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
/** Filtres du panneau en vigueur (type, tag, attributs, lien) : pastille de l'interrupteur. */
const activeCount = computed(
  () =>
    (state.value.item_type_id ? 1 : 0) +
    (state.value.tag ? 1 : 0) +
    (state.value.related_to ? 1 : 0) +
    attrCount.value,
);
const hasCriteria = computed(
  () => Object.keys(toQuery(state.value)).length > 0,
);

function resetAll() {
  router.replace({ query: {} });
}

// ── Panneau de filtres ──────────────────────────────────────────────
const PANEL_KEY = "colec-catalog-filters";
function readPanel(): boolean {
  try {
    return localStorage.getItem(PANEL_KEY) !== "0";
  } catch {
    return true;
  }
}
const panelOpen = ref(readPanel());
watch(panelOpen, (open) => {
  try {
    localStorage.setItem(PANEL_KEY, open ? "1" : "0");
  } catch {
    /* stockage indisponible : le choix vaut pour la session */
  }
});
const sheetOpen = ref(false);

function toggleFilters() {
  if (isWide.value) panelOpen.value = !panelOpen.value;
  else sheetOpen.value = true;
}

/** Interrupteur sur grand écran (panneau), ouverture d'une feuille sinon. */
const filterToggleAttrs = computed(() =>
  isWide.value
    ? {
        role: "switch",
        "aria-checked": panelOpen.value,
        "aria-controls": "panneau-filtres",
      }
    : {
        "aria-haspopup": "dialog" as const,
        "aria-expanded": sheetOpen.value,
        "aria-label": t("items.search.showFilters"),
      },
);

const filterProps = computed(() => ({
  itemTypes: itemTypes.value,
  itemTypeId: state.value.item_type_id,
  tag: state.value.tag,
  attrs: state.value.attrs,
  schema: selectedSchema.value,
}));
const filterHandlers = {
  "update:type": onTypeChange,
  "update:tag": (tag: string | undefined) => update({ tag }),
  "update:attrs": (attrs: Record<string, string[]>) => update({ attrs }),
};

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

// Export CSV : recherche courante sans pagination, séparateur selon la langue.
function exportCsv() {
  const { page: _page, ...params } = toQuery(state.value);
  const opts = { delimiter: defaultDelimiter(lang.value), lang: lang.value };
  return importExportApi.exportItemsCsv({ ...params, ...opts });
}

function onPageChange(p: number) {
  router.replace({ query: toQuery({ ...state.value, page: p }) });
}

// ── Actions ─────────────────────────────────────────────────────────
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
    message: t("items.list.confirmDelete", {
      label: renderItemLabel(row, lang.value),
    }),
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
