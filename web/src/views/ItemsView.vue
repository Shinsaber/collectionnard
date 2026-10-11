<template>
  <div>
    <div class="flex items-center justify-between mb-6">
      <h1 class="text-2xl font-bold text-gray-800 dark:text-gray-100">
        Catalogue
      </h1>
      <Button
        v-if="auth.isAdmin"
        label="Nouvel item"
        icon="pi pi-plus"
        @click="openCreate"
      />
    </div>

    <!-- Filters -->
    <div class="flex gap-3 mb-4 flex-wrap">
      <Select
        v-model="filters.item_type_id"
        :options="itemTypes"
        option-label="name"
        option-value="id"
        placeholder="Tous les types"
        show-clear
        class="w-48"
        @change="reload"
      />
      <InputText
        v-model="filters.search"
        placeholder="Rechercher..."
        class="w-64"
        @keyup.enter="reload"
      />
      <InputText
        v-model="filters.tag"
        placeholder="Tag..."
        class="w-32"
        @keyup.enter="reload"
      />
      <Button icon="pi pi-search" @click="reload" />
    </div>

    <DataTable
      :value="pag.data.value"
      :loading="pag.loading.value"
      :lazy="true"
      :paginator="true"
      :rows="pag.perPage.value"
      :total-records="pag.total.value"
      striped-rows
      data-key="id"
      class="cursor-pointer"
      @page="onPageChange"
      @row-click="goToItem"
    >
      <Column header="Type" style="width: 120px">
        <template #body="{ data: row }">
          <Tag :value="row.item_type?.name ?? '?'" severity="info" />
        </template>
      </Column>
      <Column header="Label">
        <template #body="{ data: row }">
          {{ itemLabel(row) }}
        </template>
      </Column>
      <Column header="Tags">
        <template #body="{ data: row }">
          <div class="flex gap-1 flex-wrap">
            <Tag
              v-for="t in row.tags ?? []"
              :key="t.id"
              :value="t.name"
              severity="secondary"
              class="text-xs"
            />
          </div>
        </template>
      </Column>
      <Column v-if="auth.isAdmin" header="Actions" style="width: 140px">
        <template #body="{ data: row }">
          <div class="flex gap-1">
            <Button
              v-tooltip="'Graphe'"
              icon="pi pi-sitemap"
              text
              size="small"
              @click.stop="router.push(`/items/${row.id}/graph`)"
            />
            <Button
              icon="pi pi-pencil"
              text
              size="small"
              @click.stop="openEdit(row)"
            />
            <Button
              icon="pi pi-trash"
              text
              severity="danger"
              size="small"
              @click.stop="confirmDelete(row)"
            />
          </div>
        </template>
      </Column>
    </DataTable>

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
import { ref, onMounted } from "vue";
import { useRouter } from "vue-router";
import DataTable from "primevue/datatable";
import Column from "primevue/column";
import Button from "primevue/button";
import InputText from "primevue/inputtext";
import Select from "primevue/select";
import Tag from "primevue/tag";
import { useConfirm } from "primevue/useconfirm";
import { useToast } from "primevue/usetoast";
import { itemApi, itemTypeApi } from "@/api/client";
import { usePagination } from "@/composables/usePagination";
import { useAuthStore } from "@/stores/auth";
import { useLang } from "@/composables/useLang";
import { itemLabel as renderItemLabel } from "@/lib/schema";
import ItemEditDialog from "@/components/item/ItemEditDialog.vue";
import type { Item, ItemType } from "@/api/types";

const auth = useAuthStore();
const router = useRouter();
const { lang } = useLang();
const confirm = useConfirm();
const toast = useToast();

const itemTypes = ref<ItemType[]>([]);
const filters = ref<{ item_type_id?: string; search?: string; tag?: string }>(
  {},
);
const pag = usePagination(itemApi.list);

const dialogVisible = ref(false);
const editing = ref<Item | null>(null);

onMounted(async () => {
  await itemTypeApi
    .list({ per_page: 100 })
    .then((r) => (itemTypes.value = r.data));
  reload();
});

function reload() {
  pag.page.value = 1;
  pag.load(filters.value);
}

function onPageChange(e: { page: number; rows: number }) {
  pag.onPage(e);
  pag.load(filters.value);
}

function itemLabel(item: Item): string {
  return renderItemLabel(item, lang.value);
}

function goToItem(event: { data: Item }) {
  router.push(`/items/${event.data.id}`);
}

function openCreate() {
  editing.value = null;
  dialogVisible.value = true;
}

function openEdit(row: Item) {
  editing.value = row;
  dialogVisible.value = true;
}

function confirmDelete(row: Item) {
  confirm.require({
    message: `Supprimer cet item ?`,
    header: "Confirmation",
    icon: "pi pi-exclamation-triangle",
    acceptLabel: "Supprimer",
    rejectLabel: "Annuler",
    acceptClass: "p-button-danger",
    accept: async () => {
      await itemApi.delete(row.id);
      toast.add({ severity: "success", summary: "Supprimé", life: 3000 });
      pag.load(filters.value);
    },
  });
}
</script>
