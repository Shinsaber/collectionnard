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

    <!-- Filters -->
    <div class="flex gap-3 mb-4 flex-wrap">
      <ElSelect
        v-model="filters.item_type_id"
        :placeholder="t('items.list.allTypes')"
        clearable
        class="w-48!"
        :aria-label="t('items.list.filterByType')"
        @change="reload"
      >
        <ElOption
          v-for="type in itemTypes"
          :key="type.id"
          :label="typeTitle(type, lang)"
          :value="type.id"
        />
      </ElSelect>
      <ElInput
        v-model="filters.search"
        :placeholder="t('items.list.searchPlaceholder')"
        clearable
        class="w-64!"
        :aria-label="t('common.search')"
        @keyup.enter="reload"
      />
      <ElInput
        v-model="filters.tag"
        :placeholder="t('items.list.tagPlaceholder')"
        clearable
        class="w-32!"
        :aria-label="t('items.list.tag')"
        @keyup.enter="reload"
      />
      <ElButton :aria-label="t('common.search')" @click="reload">
        <i class="pi pi-search" aria-hidden="true"></i>
      </ElButton>
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
      @saved="pag.load(filters)"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import { itemApi } from "@/api/client";
import { useMetaTypes } from "@/stores/metaTypes";
import { useNotify } from "@/composables/useNotify";
import { useConfirmDialog } from "@/composables/useConfirmDialog";
import { usePagination } from "@/composables/usePagination";
import { useAuthStore } from "@/stores/auth";
import { useLang } from "@/composables/useLang";
import { itemLabel as renderItemLabel, typeTitle } from "@/lib/schema";
import ItemEditDialog from "@/components/item/ItemEditDialog.vue";
import type { Item } from "@/api/types";

const auth = useAuthStore();
const router = useRouter();
const { t } = useI18n();
const { lang } = useLang();
const { notifyError, notifySuccess } = useNotify();
const { confirm } = useConfirmDialog();

const metaTypes = useMetaTypes();
const itemTypes = computed(() => metaTypes.itemTypes);
const filters = ref<{ item_type_id?: string; search?: string; tag?: string }>(
  {},
);
const pag = usePagination(itemApi.list);

const dialogVisible = ref(false);
const editing = ref<Item | null>(null);

onMounted(async () => {
  await metaTypes
    .loadItemTypes()
    .catch((e) => notifyError(e, t("items.list.typesUnavailable")));
  reload();
});

function reload() {
  pag.page.value = 1;
  pag
    .load(filters.value)
    .catch((e) => notifyError(e, t("items.list.loadFailed")));
}

function onPageChange(p: number) {
  pag.goToPage(p);
  pag
    .load(filters.value)
    .catch((e) => notifyError(e, t("items.list.loadFailed")));
}

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
    pag.load(filters.value);
  } catch (e) {
    notifyError(e);
  }
}
</script>
