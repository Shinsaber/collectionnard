<template>
  <AppSpinner v-if="loading" />
  <div v-else-if="collection">
    <RouterLink
      to="/collections"
      class="mb-4 inline-flex items-center gap-1.5 text-sm font-semibold text-link"
    >
      <i class="pi pi-arrow-left text-xs" aria-hidden="true"></i>
      {{ t("layout.nav.collections") }}
    </RouterLink>

    <PageHeader :title="collection.name" :meta="countLabel">
      <template #subtitle>
        <div class="flex flex-wrap items-center gap-2">
          <span
            class="rounded-full bg-tint px-2.5 py-0.5 font-mono text-[11px] uppercase text-tint-ink"
            >{{ typeTitle(collection.collection_type, lang) || "?" }}</span
          >
          <p v-if="collection.description" class="text-[15px] text-muted">
            {{ collection.description }}
          </p>
        </div>
      </template>
      <template #actions>
        <ElButton data-test="share-button" @click="shareVisible = true">
          <i
            class="pi mr-2"
            :class="
              collection.visibility === 'private' ? 'pi-lock' : 'pi-globe'
            "
            aria-hidden="true"
          ></i>
          {{ t("sharing.share.button") }}
        </ElButton>
        <ElButton
          type="primary"
          class="max-md:hidden!"
          @click="addVisible = true"
        >
          <i class="pi pi-plus mr-2" aria-hidden="true"></i>
          {{ t("collections.detail.addItem") }}
        </ElButton>
        <!-- Export, modification et suppression regroupés (retours v0.4.0). -->
        <CollectionMoreMenu
          :export-run="exportCollection"
          @edit="openEditCollection"
          @delete="confirmDeleteCollection"
        />
      </template>
    </PageHeader>

    <ElTabs v-model="tab">
      <ElTabPane :label="t('stats.itemsTab')" name="items">
        <CollectionItemsToolbar
          v-model="lotQuery"
          :types="filterTypes"
          :specialized-type-id="
            collection ? specializedTypeId(collection) : undefined
          "
          @change="onFilter"
        />
        <CollectionItemList
          :items="items.data.value"
          :empty-text="filtered ? t('collections.toolbar.noMatch') : undefined"
          :schema="schema"
          :loading="items.loading.value"
          @edit="(ci) => openLot('edit', ci)"
          @split="(ci) => openLot('split', ci)"
          @merge="(ci) => openLot('merge', ci)"
          @remove="confirmRemove"
        />
        <ElPagination
          class="mt-4 justify-end"
          hide-on-single-page
          :layout="isMobile ? 'prev, pager, next' : 'total, prev, pager, next'"
          :pager-count="isMobile ? 5 : 7"
          :total="items.total.value"
          :page-size="items.perPage.value"
          :current-page="items.page.value"
          @current-change="onPageChange"
        />
      </ElTabPane>
      <ElTabPane :label="t('stats.tab')" name="stats" lazy>
        <CollectionStatsPanel
          :load="loadStats"
          :schema="schema"
          :version="statsVersion"
        />
      </ElTabPane>
    </ElTabs>

    <!-- Mobile : ajout en bouton flottant, au-dessus de la barre d'onglets. -->
    <button
      type="button"
      class="fixed right-4 bottom-[calc(5.5rem+env(safe-area-inset-bottom))] z-10 flex h-14 items-center gap-2 rounded-2xl bg-primary px-5 font-semibold text-primary-ink shadow-primary md:hidden"
      @click="addVisible = true"
    >
      <i class="pi pi-plus" aria-hidden="true"></i>
      {{ t("collections.detail.addItem") }}
    </button>

    <CollectionShareDialog
      v-model:visible="shareVisible"
      :collection="collection"
      @saved="(c) => (collection = c)"
    />

    <CollectionAddItemDialog
      v-model:visible="addVisible"
      :collection-id="id"
      :schema="schema"
      :allowed-type-ids="accepted"
      @added="afterChange"
    />

    <CollectionEditDialog
      v-model:visible="colDialogVisible"
      :collection="collection"
      @saved="onCollectionSaved"
    />

    <!-- Lots (P7-1) : modifier, scinder, fusionner. -->
    <LotEditDialog
      v-model:visible="lotDialog.edit"
      :collection-id="id"
      :schema="schema"
      :lot="lot"
      @saved="afterChange"
    />
    <LotSplitDialog
      v-model:visible="lotDialog.split"
      :collection-id="id"
      :schema="schema"
      :lot="lot"
      @done="afterChange"
    />
    <LotMergeDialog
      v-model:visible="lotDialog.merge"
      :collection-id="id"
      :schema="schema"
      :lot="lot"
      @done="afterChange"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, defineAsyncComponent, onMounted } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { collectionApi, importExportApi, statsApi } from "@/api/client";
import { usePagination } from "@/composables/usePagination";
import { useLang } from "@/composables/useLang";
import { useNotify } from "@/composables/useNotify";
import { useConfirmDialog } from "@/composables/useConfirmDialog";
import { MOBILE_QUERY, useMediaQuery } from "@/composables/useMediaQuery";
import { itemLabel, typeTitle, type Schema } from "@/lib/schema";
import { DEFAULT_LOT_SORT, emptyLotQuery, lotQueryParams } from "@/lib/lots";
import { acceptedTypeIds, specializedTypeId } from "@/lib/collections";
import { useMetaTypes } from "@/stores/metaTypes";
import AppSpinner from "@/components/common/AppSpinner.vue";
import PageHeader from "@/components/common/PageHeader.vue";
import type { ExportFormat } from "@/components/common/ExportButton.vue";
import CollectionMoreMenu from "@/components/collection/CollectionMoreMenu.vue";
import CollectionAddItemDialog from "@/components/collection/CollectionAddItemDialog.vue";
import CollectionItemList from "@/components/collection/CollectionItemList.vue";
import CollectionItemsToolbar from "@/components/collection/CollectionItemsToolbar.vue";
import CollectionEditDialog from "@/components/collection/CollectionEditDialog.vue";
import LotEditDialog from "@/components/collection/LotEditDialog.vue";
import LotMergeDialog from "@/components/collection/LotMergeDialog.vue";
import LotSplitDialog from "@/components/collection/LotSplitDialog.vue";
import CollectionShareDialog from "@/components/sharing/CollectionShareDialog.vue";
import { defaultDelimiter } from "@/lib/csv";
import type { Collection, CollectionItem } from "@/api/types";

// Onglet Statistiques : ECharts n'est téléchargé qu'à sa première ouverture.
const CollectionStatsPanel = defineAsyncComponent(
  () => import("@/components/stats/CollectionStatsPanel.vue"),
);

const props = defineProps<{ id: string }>();
const { t } = useI18n();
const router = useRouter();
const { lang } = useLang();
const { notifyError, notifySuccess } = useNotify();
const { confirm } = useConfirmDialog();
const isMobile = useMediaQuery(MOBILE_QUERY);

const collection = ref<Collection | null>(null);
const loading = ref(true);

// Recherche et tri des lots (P7-1) : type d'item, texte, lots de plusieurs exemplaires.
const lotQuery = ref(emptyLotQuery());
const filtered = computed(() =>
  Object.keys(lotQueryParams(lotQuery.value)).some((k) => k !== "sort"),
);
const metaTypes = useMetaTypes();
/** Types d'item proposés : ceux que la collection accepte (tous si rien ne les limite). */
const accepted = computed(() =>
  collection.value ? acceptedTypeIds(collection.value) : [],
);
const filterTypes = computed(() => {
  const allowed = accepted.value;
  return allowed.length
    ? metaTypes.itemTypes.filter((it) => allowed.includes(it.id))
    : metaTypes.itemTypes;
});

function onFilter() {
  items.goToPage(1);
  items.load().catch((e) => notifyError(e, t("collections.loadFailed")));
}

const items = usePagination((params: Record<string, unknown>) =>
  collectionApi.listItems(props.id, {
    ...params,
    ...lotQueryParams(lotQuery.value),
  }),
);

const schema = computed(
  () =>
    collection.value?.collection_type?.attributes_schema as Schema | undefined,
);

const exportCollection = (format: ExportFormat) =>
  importExportApi.exportCollection(props.id, format, {
    delimiter: defaultDelimiter(lang.value),
    lang: lang.value,
  });

const loadStats = () => statsApi.collection(props.id);
const addVisible = ref(false);
const shareVisible = ref(false);
const tab = ref<"items" | "stats">("items");
// Incrémenté à chaque modification : l'onglet Statistiques se recharge.
const statsVersion = ref(0);

// Lot visé par les dialogues « Modifier », « Scinder » et « Fusionner » (P7-1).
const lot = ref<CollectionItem | null>(null);
const lotDialog = ref({ edit: false, split: false, merge: false });

function openLot(kind: "edit" | "split" | "merge", ci: CollectionItem) {
  lot.value = ci;
  lotDialog.value[kind] = true;
}

/** « 12 items », ou « 12 items · 45 exemplaires » quand des lots en comptent plusieurs. */
const countLabel = computed(() => {
  const c = collection.value;
  if (!c) return "";
  const items = t("collections.itemCount", c.item_count);
  return c.copy_count > c.item_count
    ? `${items} · ${t("collections.lots.copies", c.copy_count)}`
    : items;
});

// ── Collection elle-même : modification, suppression (menu « Plus ») ──
const colDialogVisible = ref(false);

function openEditCollection() {
  colDialogVisible.value = true;
}

/** Collection modifiée : son tri par défaut et ses types acceptés s'appliquent à la liste. */
function onCollectionSaved(c: Collection) {
  collection.value = c;
  lotQuery.value = {
    ...emptyLotQuery(),
    sort: c.default_sort || DEFAULT_LOT_SORT,
  };
  onFilter();
}

async function confirmDeleteCollection() {
  if (!collection.value) return;
  const ok = await confirm({
    message: t("collections.detail.confirmDelete", {
      name: collection.value.name,
    }),
    confirmText: t("common.delete"),
    danger: true,
  });
  if (!ok) return;
  try {
    await collectionApi.delete(props.id);
    notifySuccess(t("collections.detail.deleted"));
    await router.push("/collections");
  } catch (e) {
    notifyError(e);
  }
}

onMounted(async () => {
  try {
    collection.value = await collectionApi.get(props.id); // collection_type est toujours joint par l'API
    lotQuery.value.sort = collection.value.default_sort || DEFAULT_LOT_SORT;
    metaTypes.loadItemTypes().catch((e) => notifyError(e));
    await items.load();
  } catch (e) {
    notifyError(e, t("collections.detail.notFound"));
  } finally {
    loading.value = false;
  }
});

function onPageChange(p: number) {
  items.goToPage(p);
  items.load().catch((e) => notifyError(e, t("collections.loadFailed")));
}

async function refreshCollection() {
  try {
    collection.value = await collectionApi.get(props.id);
  } catch (e) {
    notifyError(e, t("collections.detail.unavailable"));
  }
}

/** Après un ajout ou un retrait : liste et compteur de la collection. */
async function afterChange() {
  statsVersion.value++;
  await items.load().catch((e) => notifyError(e, t("collections.loadFailed")));
  await refreshCollection();
}

function ciLabel(ci: CollectionItem): string {
  return itemLabel(ci.item, lang.value);
}

async function confirmRemove(ci: CollectionItem) {
  const ok = await confirm({
    message: t("collections.detail.confirmRemove", { label: ciLabel(ci) }),
    confirmText: t("collections.detail.remove"),
    danger: true,
  });
  if (!ok) return;
  try {
    await collectionApi.removeLot(props.id, ci.id);
    notifySuccess(t("collections.detail.itemRemoved"));
    await afterChange();
  } catch (e) {
    notifyError(e);
  }
}
</script>
