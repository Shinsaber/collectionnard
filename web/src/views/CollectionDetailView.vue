<template>
  <div v-if="loading" class="flex justify-center py-12">
    <ProgressSpinner />
  </div>

  <div v-else-if="collection">
    <div class="flex items-center gap-3 mb-6">
      <Button
        icon="pi pi-arrow-left"
        text
        @click="router.push('/collections')"
      />
      <div>
        <h1 class="text-2xl font-bold text-gray-800">{{ collection.name }}</h1>
        <div class="flex items-center gap-2 mt-1">
          <Tag
            :value="collection.collection_type?.name ?? '?'"
            severity="info"
          />
          <span class="text-sm text-gray-400"
            >{{ collection.item_count }} items</span
          >
        </div>
      </div>
      <div class="flex-1"></div>
      <Button label="Ajouter un item" icon="pi pi-plus" @click="openAddItem" />
    </div>

    <p v-if="collection.description" class="text-gray-500 mb-4">
      {{ collection.description }}
    </p>

    <DataTable
      :value="items.data.value"
      :loading="items.loading.value"
      :lazy="true"
      :paginator="true"
      :rows="items.perPage.value"
      :totalRecords="items.total.value"
      @page="onPageChange"
      stripedRows
      dataKey="id"
    >
      <Column header="Type" style="width: 100px">
        <template #body="{ data: ci }">
          <Tag :value="ci.item?.item_type?.name ?? '?'" severity="secondary" />
        </template>
      </Column>
      <Column header="Item">
        <template #body="{ data: ci }">
          <router-link
            :to="`/items/${ci.item_id}`"
            class="text-indigo-600 hover:underline"
          >
            {{ ciLabel(ci) }}
          </router-link>
        </template>
      </Column>
      <Column header="Attributs contextuels">
        <template #body="{ data: ci }">
          <div class="flex gap-2 flex-wrap text-xs">
            <span
              v-for="(val, key) in ci.attributes"
              :key="key"
              class="bg-gray-100 rounded px-2 py-0.5"
            >
              {{ ctxAttrLabel(String(key)) }}: {{ val }}
            </span>
          </div>
        </template>
      </Column>
      <Column header="Actions" style="width: 100px">
        <template #body="{ data: ci }">
          <div class="flex gap-1">
            <Button
              icon="pi pi-pencil"
              text
              size="small"
              @click="openEditItem(ci)"
            />
            <Button
              icon="pi pi-trash"
              text
              severity="danger"
              size="small"
              @click="confirmRemove(ci)"
            />
          </div>
        </template>
      </Column>
    </DataTable>

    <!-- Add item dialog -->
    <Dialog
      v-model:visible="addDialogVisible"
      header="Ajouter un item"
      modal
      class="w-full max-w-lg"
    >
      <div class="flex flex-col gap-4">
        <div class="flex flex-col gap-1">
          <label class="text-sm font-medium">Rechercher un item</label>
          <InputText
            v-model="searchText"
            placeholder="Rechercher..."
            @keyup.enter="searchItems"
          />
          <Button
            icon="pi pi-search"
            text
            size="small"
            @click="searchItems"
            class="self-start"
          />
        </div>

        <DataTable
          v-if="searchResults.length"
          :value="searchResults"
          selectionMode="single"
          v-model:selection="selectedItem"
          dataKey="id"
          class="text-sm"
        >
          <Column field="item_type.name" header="Type" style="width: 80px" />
          <Column header="Label">
            <template #body="{ data: row }">{{ itemLabel(row) }}</template>
          </Column>
        </DataTable>

        <div v-if="selectedItem && collectionTypeSchema">
          <h4 class="text-sm font-medium mb-2">Attributs contextuels</h4>
          <DynamicForm :schema="collectionTypeSchema" v-model="addAttributes" />
        </div>
      </div>
      <template #footer>
        <Button label="Annuler" text @click="addDialogVisible = false" />
        <Button
          label="Ajouter"
          icon="pi pi-plus"
          :disabled="!selectedItem"
          :loading="saving"
          @click="addItem"
        />
      </template>
    </Dialog>

    <!-- Edit item attributes dialog -->
    <Dialog
      v-model:visible="editDialogVisible"
      header="Modifier les attributs"
      modal
      class="w-full max-w-lg"
    >
      <DynamicForm
        v-if="collectionTypeSchema"
        :schema="collectionTypeSchema"
        v-model="editAttributes"
      />
      <template #footer>
        <Button label="Annuler" text @click="editDialogVisible = false" />
        <Button
          label="Enregistrer"
          icon="pi pi-check"
          :loading="saving"
          @click="updateItem"
        />
      </template>
    </Dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from "vue";
import { useRouter } from "vue-router";
import DataTable from "primevue/datatable";
import Column from "primevue/column";
import Button from "primevue/button";
import Dialog from "primevue/dialog";
import InputText from "primevue/inputtext";
import Tag from "primevue/tag";
import ProgressSpinner from "primevue/progressspinner";
import { useConfirm } from "primevue/useconfirm";
import { useToast } from "primevue/usetoast";
import { collectionApi, itemApi, collectionTypeApi, extractApiError } from "@/api/client";
import { usePagination } from "@/composables/usePagination";
import DynamicForm from "@/components/common/DynamicForm.vue";
import type { Collection, CollectionItem, Item } from "@/api/types";

const props = defineProps<{ id: string }>();
const router = useRouter();
const confirm = useConfirm();
const toast = useToast();

const collection = ref<Collection | null>(null);
const loading = ref(true);

const items = usePagination((params: Record<string, unknown>) =>
  collectionApi.listItems(props.id, params),
);

const collectionTypeSchema = computed(() => {
  return collection.value?.collection_type?.attributes_schema ?? null;
});

// Add item
const addDialogVisible = ref(false);
const searchText = ref("");
const searchResults = ref<Item[]>([]);
const selectedItem = ref<Item | null>(null);
const addAttributes = ref<Record<string, unknown>>({});
const saving = ref(false);

// Edit item
const editDialogVisible = ref(false);
const editingItemId = ref("");
const editAttributes = ref<Record<string, unknown>>({});

onMounted(async () => {
  try {
    const col = await collectionApi.get(props.id);
    // Fetch collection type details if not embedded
    if (col.collection_type_id && !col.collection_type) {
      col.collection_type = await collectionTypeApi.get(col.collection_type_id);
    }
    collection.value = col;
    items.load();
  } finally {
    loading.value = false;
  }
});

function onPageChange(e: { page: number; rows: number }) {
  items.onPage(e);
  items.load();
}

function ciLabel(ci: CollectionItem): string {
  return itemLabel(ci.item);
}

function itemLabel(item?: Item | null): string {
  if (!item) return "?";
  for (const key of ["titre", "nom", "name", "title", "label"]) {
    const val = item.attributes?.[key];
    if (val == null) continue;
    if (typeof val === "object" && !Array.isArray(val)) {
      const obj = val as Record<string, unknown>;
      if (typeof obj["fr"] === "string") return obj["fr"];
      if (typeof obj["en"] === "string") return obj["en"];
    }
    return String(val);
  }
  return item.id.slice(0, 8);
}

function ctxAttrLabel(key: string): string {
  const schema = collection.value?.collection_type?.attributes_schema as
    | Record<string, unknown>
    | undefined;
  const labels = (
    schema?.["x-labels"] as Record<string, Record<string, string>> | undefined
  )?.["fr"];
  return labels?.[key] ?? key;
}

function openAddItem() {
  searchText.value = "";
  searchResults.value = [];
  selectedItem.value = null;
  addAttributes.value = {};
  addDialogVisible.value = true;
}

async function searchItems() {
  const res = await itemApi.list({ search: searchText.value, per_page: 10 });
  searchResults.value = res.data;
}

async function addItem() {
  if (!selectedItem.value) return;
  saving.value = true;
  try {
    await collectionApi.addItem(props.id, {
      item_id: selectedItem.value.id,
      attributes: addAttributes.value,
    });
    toast.add({ severity: "success", summary: "Item ajouté", life: 3000 });
    addDialogVisible.value = false;
    items.load();
    // Refresh collection to update count
    collection.value = await collectionApi.get(props.id);
  } catch (e: unknown) {
    const apiErr = extractApiError(e);
    toast.add({
      severity: "error",
      summary: "Erreur",
      detail: apiErr.message,
      life: 5000,
    });
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
  saving.value = true;
  try {
    await collectionApi.updateItem(props.id, editingItemId.value, {
      attributes: editAttributes.value,
    });
    toast.add({ severity: "success", summary: "Modifié", life: 3000 });
    editDialogVisible.value = false;
    items.load();
  } catch (e: unknown) {
    const apiErr = extractApiError(e);
    toast.add({ severity: "error", summary: "Erreur", detail: apiErr.message, life: 5000 });
  } finally {
    saving.value = false;
  }
}

function confirmRemove(ci: CollectionItem) {
  confirm.require({
    message: "Retirer cet item de la collection ?",
    header: "Confirmation",
    icon: "pi pi-exclamation-triangle",
    acceptLabel: "Retirer",
    rejectLabel: "Annuler",
    acceptClass: "p-button-danger",
    accept: async () => {
      await collectionApi.removeItem(props.id, ci.item_id);
      toast.add({ severity: "success", summary: "Retiré", life: 3000 });
      items.load();
      collection.value = await collectionApi.get(props.id);
    },
  });
}
</script>
