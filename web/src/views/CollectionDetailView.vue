<template>
  <AppSpinner v-if="loading" />

  <div v-else-if="collection">
    <div class="flex items-center gap-3 mb-6">
      <ElButton
        text
        :aria-label="t('collections.detail.backToList')"
        @click="router.push('/collections')"
      >
        <i class="pi pi-arrow-left" aria-hidden="true"></i>
      </ElButton>
      <div>
        <h1 class="text-2xl font-bold text-gray-800 dark:text-gray-100">
          {{ collection.name }}
        </h1>
        <div class="flex items-center gap-2 mt-1">
          <ElTag type="info" size="small">{{
            typeTitle(collection.collection_type, lang) || "?"
          }}</ElTag>
          <span class="text-sm text-gray-400 dark:text-gray-500">{{
            t("collections.itemCount", collection.item_count)
          }}</span>
        </div>
      </div>
      <div class="flex-1"></div>
      <ExportButton
        :label="t('collections.detail.export')"
        :formats="['json', 'csv']"
        :run="exportCollection"
        :error-summary="t('collections.detail.exportFailed')"
      />
      <ElButton type="primary" @click="openAddItem">
        <i class="pi pi-plus mr-2" aria-hidden="true"></i>
        {{ t("collections.detail.addItem") }}
      </ElButton>
    </div>

    <p
      v-if="collection.description"
      class="text-gray-500 dark:text-gray-400 mb-4"
    >
      {{ collection.description }}
    </p>

    <ElTable
      v-loading="items.loading.value"
      :data="items.data.value"
      stripe
      row-key="id"
    >
      <ElTableColumn :label="t('common.type')" width="120">
        <template #default="{ row: ci }">
          <ElTag type="info" effect="plain" size="small">{{
            typeTitle((ci as CollectionItem).item?.item_type, lang) || "?"
          }}</ElTag>
        </template>
      </ElTableColumn>
      <ElTableColumn :label="t('collections.detail.item')">
        <template #default="{ row: ci }">
          <router-link
            :to="`/items/${(ci as CollectionItem).item_id}`"
            class="text-indigo-600 dark:text-indigo-400 hover:underline"
          >
            {{ ciLabel(ci as CollectionItem) }}
          </router-link>
        </template>
      </ElTableColumn>
      <ElTableColumn :label="t('collections.detail.contextAttributes')">
        <template #default="{ row: ci }">
          <div class="flex gap-2 flex-wrap text-xs">
            <span
              v-for="(val, key) in (ci as CollectionItem).attributes"
              :key="key"
              class="bg-gray-100 dark:bg-gray-700 dark:text-gray-300 rounded-xs px-2 py-0.5"
            >
              {{ ctxAttrLabel(String(key)) }}:
              {{ ctxAttrValue(String(key), val) }}
            </span>
          </div>
        </template>
      </ElTableColumn>
      <ElTableColumn :label="t('common.actions')" width="110">
        <template #default="{ row: ci }">
          <div class="flex gap-1">
            <ElButton
              text
              size="small"
              :aria-label="t('collections.detail.editAttributes')"
              @click="openEditItem(ci as CollectionItem)"
            >
              <i class="pi pi-pencil" aria-hidden="true"></i>
            </ElButton>
            <ElButton
              text
              size="small"
              type="danger"
              :aria-label="t('collections.detail.removeFromCollection')"
              @click="confirmRemove(ci as CollectionItem)"
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
      :total="items.total.value"
      :page-size="items.perPage.value"
      :current-page="items.page.value"
      @current-change="onPageChange"
    />

    <!-- Add item dialog -->
    <ElDialog
      v-model="addDialogVisible"
      :title="t('collections.detail.addItem')"
      width="34rem"
    >
      <div class="flex flex-col gap-4">
        <div class="flex flex-col gap-1">
          <label for="ci-search" class="text-sm font-medium">{{
            t("collections.detail.searchItem")
          }}</label>
          <div class="flex gap-2">
            <ElInput
              id="ci-search"
              v-model="searchText"
              :placeholder="t('collections.detail.searchPlaceholder')"
              clearable
              @keyup.enter="searchItems"
            />
            <ElButton :aria-label="t('common.search')" @click="searchItems">
              <i class="pi pi-search" aria-hidden="true"></i>
            </ElButton>
          </div>
        </div>

        <ElTable
          v-if="searchResults.length"
          :data="searchResults"
          highlight-current-row
          row-key="id"
          size="small"
          class="cursor-pointer"
          @current-change="selectedItem = $event"
        >
          <ElTableColumn :label="t('common.type')" width="110">
            <template #default="{ row }">{{
              typeTitle((row as Item).item_type, lang) || "?"
            }}</template>
          </ElTableColumn>
          <ElTableColumn :label="t('collections.detail.label')">
            <template #default="{ row }">{{ itemLabel(row as Item) }}</template>
          </ElTableColumn>
        </ElTable>

        <div v-if="selectedItem && collectionTypeSchema">
          <h4 class="text-sm font-medium mb-2">
            {{ t("collections.detail.contextAttributes") }}
          </h4>
          <DynamicForm
            ref="addFormRef"
            v-model="addAttributes"
            :schema="collectionTypeSchema"
          />
        </div>
      </div>
      <template #footer>
        <ElButton text @click="addDialogVisible = false">{{
          t("common.cancel")
        }}</ElButton>
        <ElButton
          type="primary"
          :disabled="!selectedItem"
          :loading="saving"
          @click="addItem"
        >
          <i class="pi pi-plus mr-2" aria-hidden="true"></i>
          {{ t("common.add") }}
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
        v-if="collectionTypeSchema"
        ref="editFormRef"
        v-model="editAttributes"
        :schema="collectionTypeSchema"
      />
      <template #footer>
        <ElButton text @click="editDialogVisible = false">{{
          t("common.cancel")
        }}</ElButton>
        <ElButton type="primary" :loading="saving" @click="updateItem">
          <i class="pi pi-check mr-2" aria-hidden="true"></i>
          {{ t("common.save") }}
        </ElButton>
      </template>
    </ElDialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { collectionApi, importExportApi, itemApi } from "@/api/client";
import { usePagination } from "@/composables/usePagination";
import { useLang } from "@/composables/useLang";
import { useNotify } from "@/composables/useNotify";
import { useConfirmDialog } from "@/composables/useConfirmDialog";
import {
  fieldValue,
  fieldLabel,
  itemLabel as renderItemLabel,
  typeTitle,
} from "@/lib/schema";
import AppSpinner from "@/components/common/AppSpinner.vue";
import DynamicForm from "@/components/common/DynamicForm.vue";
import ExportButton, {
  type ExportFormat,
} from "@/components/common/ExportButton.vue";
import { defaultDelimiter } from "@/lib/csv";
import type { Collection, CollectionItem, Item } from "@/api/types";

const props = defineProps<{ id: string }>();
const router = useRouter();
const { t } = useI18n();
const { lang } = useLang();
const { notifyError, notifySuccess } = useNotify();
const { confirm } = useConfirmDialog();

const collection = ref<Collection | null>(null);
const loading = ref(true);

const items = usePagination((params: Record<string, unknown>) =>
  collectionApi.listItems(props.id, params),
);

const collectionTypeSchema = computed(
  () => collection.value?.collection_type?.attributes_schema ?? null,
);

const exportCollection = (format: ExportFormat) =>
  importExportApi.exportCollection(props.id, format, {
    delimiter: defaultDelimiter(lang.value),
    lang: lang.value,
  });

// Add item
const addDialogVisible = ref(false);
const searchText = ref("");
const searchResults = ref<Item[]>([]);
const selectedItem = ref<Item | null>(null);
const addAttributes = ref<Record<string, unknown>>({});
const addFormRef = ref<InstanceType<typeof DynamicForm> | null>(null);
const saving = ref(false);

// Edit item
const editDialogVisible = ref(false);
const editingItemId = ref("");
const editAttributes = ref<Record<string, unknown>>({});
const editFormRef = ref<InstanceType<typeof DynamicForm> | null>(null);

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

function ciLabel(ci: CollectionItem): string {
  return itemLabel(ci.item);
}

function itemLabel(item?: Item | null): string {
  return renderItemLabel(item, lang.value);
}

function ctxAttrLabel(key: string): string {
  return fieldLabel(ctxSchema(), key, lang.value);
}

/** Valeur d'un attribut contextuel (libellés d'enum du schéma du CollectionType). */
function ctxAttrValue(key: string, value: unknown): string {
  return fieldValue(ctxSchema(), key, value, lang.value);
}

function ctxSchema(): Record<string, unknown> | undefined {
  return collection.value?.collection_type?.attributes_schema as
    Record<string, unknown> | undefined;
}

function openAddItem() {
  searchText.value = "";
  searchResults.value = [];
  selectedItem.value = null;
  addAttributes.value = {};
  addDialogVisible.value = true;
}

async function searchItems() {
  try {
    const res = await itemApi.list({ search: searchText.value, per_page: 10 });
    searchResults.value = res.data;
  } catch (e) {
    notifyError(e, t("collections.detail.searchFailed"));
  }
}

async function addItem() {
  if (!selectedItem.value) return;
  if (addFormRef.value && !addFormRef.value.validate()) return;
  saving.value = true;
  try {
    await collectionApi.addItem(props.id, {
      item_id: selectedItem.value.id,
      attributes: addAttributes.value,
    });
    notifySuccess(t("collections.detail.itemAdded"));
    addDialogVisible.value = false;
    await items.load();
    await refreshCollection();
  } catch (e) {
    notifyError(e);
  } finally {
    saving.value = false;
  }
}

function openEditItem(ci: CollectionItem) {
  editingItemId.value = ci.item_id;
  editAttributes.value = { ...(ci.attributes ?? {}) };
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
    await items.load();
    await refreshCollection();
  } catch (e) {
    notifyError(e);
  }
}
</script>
