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

    <PageHeader
      :title="collection.name"
      :meta="t('collections.itemCount', collection.item_count)"
    >
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
        <CollectionItemList
          :items="items.data.value"
          :schema="schema"
          :loading="items.loading.value"
          @edit="openEditItem"
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
      :allowed-type-ids="collection.collection_type?.item_type_ids"
      @added="afterChange"
    />

    <!-- Nom et description de la collection (le type ne change pas). -->
    <ElDialog
      v-model="colDialogVisible"
      :title="t('collections.detail.editTitle')"
      width="28rem"
    >
      <div class="flex flex-col gap-4">
        <div class="flex flex-col gap-1">
          <label for="col-name" class="text-sm font-medium">{{
            t("common.name")
          }}</label>
          <ElInput
            id="col-name"
            v-model="colForm.name"
            :placeholder="t('collections.list.namePlaceholder')"
          />
        </div>
        <div class="flex flex-col gap-1">
          <label for="col-desc" class="text-sm font-medium">{{
            t("common.description")
          }}</label>
          <ElInput
            id="col-desc"
            v-model="colForm.description"
            type="textarea"
            :rows="2"
          />
        </div>
      </div>
      <template #footer>
        <ElButton text @click="colDialogVisible = false">{{
          t("common.cancel")
        }}</ElButton>
        <ElButton
          type="primary"
          :loading="saving"
          :disabled="!colDirty || !colForm.name.trim()"
          data-test="collection-save"
          @click="saveCollection"
        >
          <i class="pi pi-check mr-2" aria-hidden="true"></i>
          {{ t("common.save") }}
        </ElButton>
      </template>
    </ElDialog>

    <!-- Edit item attributes dialog -->
    <ElDialog
      v-model="editDialogVisible"
      :title="t('collections.detail.editAttributes')"
      width="34rem"
    >
      <DynamicForm
        v-if="schema"
        ref="editFormRef"
        v-model="editAttributes"
        :schema="schema"
      />
      <template #footer>
        <ElButton text @click="editDialogVisible = false">{{
          t("common.cancel")
        }}</ElButton>
        <ElButton
          type="primary"
          :loading="saving"
          :disabled="!editDirty"
          @click="updateItem"
        >
          <i class="pi pi-check mr-2" aria-hidden="true"></i>
          {{ t("common.save") }}
        </ElButton>
      </template>
    </ElDialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, defineAsyncComponent, onMounted } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { collectionApi, importExportApi, statsApi } from "@/api/client";
import { usePagination } from "@/composables/usePagination";
import { useLang } from "@/composables/useLang";
import { useDirty } from "@/composables/useDirty";
import { useNotify } from "@/composables/useNotify";
import { useConfirmDialog } from "@/composables/useConfirmDialog";
import { MOBILE_QUERY, useMediaQuery } from "@/composables/useMediaQuery";
import { itemLabel, typeTitle, type Schema } from "@/lib/schema";
import AppSpinner from "@/components/common/AppSpinner.vue";
import DynamicForm from "@/components/common/DynamicForm.vue";
import PageHeader from "@/components/common/PageHeader.vue";
import type { ExportFormat } from "@/components/common/ExportButton.vue";
import CollectionMoreMenu from "@/components/collection/CollectionMoreMenu.vue";
import CollectionAddItemDialog from "@/components/collection/CollectionAddItemDialog.vue";
import CollectionItemList from "@/components/collection/CollectionItemList.vue";
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

const items = usePagination((params: Record<string, unknown>) =>
  collectionApi.listItems(props.id, params),
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
const saving = ref(false);

// Edit item
const editDialogVisible = ref(false);
const editingItemId = ref("");
const editAttributes = ref<Record<string, unknown>>({});
const editFormRef = ref<InstanceType<typeof DynamicForm> | null>(null);
// « Enregistrer » reste grisé tant que rien n'a changé.
const { dirty: editDirty, markClean: markEditClean } = useDirty(
  () => editAttributes.value,
);

// ── Collection elle-même : nom et description, suppression (menu « Plus ») ──
const colDialogVisible = ref(false);
const colForm = ref({ name: "", description: "" });
const { dirty: colDirty, markClean: markColClean } = useDirty(
  () => colForm.value,
);

function openEditCollection() {
  if (!collection.value) return;
  colForm.value = {
    name: collection.value.name,
    description: collection.value.description ?? "",
  };
  markColClean();
  colDialogVisible.value = true;
}

async function saveCollection() {
  saving.value = true;
  try {
    collection.value = await collectionApi.update(props.id, {
      name: colForm.value.name.trim(),
      description: colForm.value.description,
    });
    notifySuccess(t("collections.detail.updated"));
    colDialogVisible.value = false;
  } catch (e) {
    notifyError(e);
  } finally {
    saving.value = false;
  }
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

function openEditItem(ci: CollectionItem) {
  editingItemId.value = ci.item_id;
  editAttributes.value = { ...(ci.attributes ?? {}) };
  markEditClean();
  editDialogVisible.value = true;
}

async function updateItem() {
  if (editFormRef.value && !editFormRef.value.validate()) return;
  saving.value = true;
  try {
    await collectionApi.updateItem(props.id, editingItemId.value, {
      attributes: editAttributes.value,
    });
    notifySuccess(t("collections.detail.attributesUpdated"));
    editDialogVisible.value = false;
    statsVersion.value++;
    await items.load();
  } catch (e) {
    notifyError(e);
  } finally {
    saving.value = false;
  }
}

async function confirmRemove(ci: CollectionItem) {
  const ok = await confirm({
    message: t("collections.detail.confirmRemove", { label: ciLabel(ci) }),
    confirmText: t("collections.detail.remove"),
    danger: true,
  });
  if (!ok) return;
  try {
    await collectionApi.removeItem(props.id, ci.item_id);
    notifySuccess(t("collections.detail.itemRemoved"));
    await afterChange();
  } catch (e) {
    notifyError(e);
  }
}
</script>
