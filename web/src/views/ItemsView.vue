<template>
  <div>
    <div class="flex items-center justify-between mb-6">
      <h1 class="text-2xl font-bold text-gray-800 dark:text-gray-100">
        Catalogue
      </h1>
      <ElButton v-if="auth.isAdmin" type="primary" @click="openCreate">
        <i class="pi pi-plus mr-2" aria-hidden="true"></i> Nouvel item
      </ElButton>
    </div>

    <!-- Filters -->
    <div class="flex gap-3 mb-4 flex-wrap">
      <ElSelect
        v-model="filters.item_type_id"
        placeholder="Tous les types"
        clearable
        class="w-48!"
        aria-label="Filtrer par type"
        @change="reload"
      >
        <ElOption
          v-for="t in itemTypes"
          :key="t.id"
          :label="t.name"
          :value="t.id"
        />
      </ElSelect>
      <ElInput
        v-model="filters.search"
        placeholder="Rechercher..."
        clearable
        class="w-64!"
        aria-label="Recherche"
        @keyup.enter="reload"
      />
      <ElInput
        v-model="filters.tag"
        placeholder="Tag..."
        clearable
        class="w-32!"
        aria-label="Tag"
        @keyup.enter="reload"
      />
      <ElButton aria-label="Rechercher" @click="reload">
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
      <ElTableColumn label="Type" width="140">
        <template #default="{ row }">
          <ElTag type="info" size="small">{{
            (row as Item).item_type?.name ?? "?"
          }}</ElTag>
        </template>
      </ElTableColumn>
      <ElTableColumn label="Label">
        <template #default="{ row }">
          {{ itemLabel(row as Item) }}
        </template>
      </ElTableColumn>
      <ElTableColumn label="Tags">
        <template #default="{ row }">
          <div class="flex gap-1 flex-wrap">
            <ElTag
              v-for="t in (row as Item).tags ?? []"
              :key="t.id"
              type="info"
              effect="plain"
              size="small"
              >{{ t.name }}</ElTag
            >
          </div>
        </template>
      </ElTableColumn>
      <ElTableColumn v-if="auth.isAdmin" label="Actions" width="150">
        <template #default="{ row }">
          <div class="flex gap-1">
            <ElButton
              text
              size="small"
              title="Graphe"
              aria-label="Voir le graphe"
              @click.stop="router.push(`/items/${(row as Item).id}/graph`)"
            >
              <i class="pi pi-sitemap" aria-hidden="true"></i>
            </ElButton>
            <ElButton
              text
              size="small"
              aria-label="Modifier"
              @click.stop="openEdit(row as Item)"
            >
              <i class="pi pi-pencil" aria-hidden="true"></i>
            </ElButton>
            <ElButton
              text
              size="small"
              type="danger"
              aria-label="Supprimer"
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
import { itemApi } from "@/api/client";
import { useMetaTypes } from "@/stores/metaTypes";
import { useNotify } from "@/composables/useNotify";
import { useConfirmDialog } from "@/composables/useConfirmDialog";
import { usePagination } from "@/composables/usePagination";
import { useAuthStore } from "@/stores/auth";
import { useLang } from "@/composables/useLang";
import { itemLabel as renderItemLabel } from "@/lib/schema";
import ItemEditDialog from "@/components/item/ItemEditDialog.vue";
import type { Item } from "@/api/types";

const auth = useAuthStore();
const router = useRouter();
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
    .catch((e) => notifyError(e, "Types indisponibles"));
  reload();
});

function reload() {
  pag.page.value = 1;
  pag.load(filters.value).catch((e) => notifyError(e, "Chargement impossible"));
}

function onPageChange(p: number) {
  pag.goToPage(p);
  pag.load(filters.value).catch((e) => notifyError(e, "Chargement impossible"));
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
    message: `Supprimer « ${itemLabel(row as Item)} » ?`,
    confirmText: "Supprimer",
    danger: true,
  });
  if (!ok) return;
  try {
    await itemApi.delete(row.id);
    notifySuccess("Item supprimé");
    pag.load(filters.value);
  } catch (e) {
    notifyError(e);
  }
}
</script>
