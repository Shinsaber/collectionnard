<template>
  <div>
    <div class="flex items-center justify-between mb-6">
      <h1 class="text-2xl font-bold text-gray-800 dark:text-gray-100">
        Mes collections
      </h1>
      <Button
        label="Nouvelle collection"
        icon="pi pi-plus"
        @click="openCreate"
      />
    </div>

    <!-- Filter -->
    <div class="flex gap-3 mb-4">
      <Select
        v-model="filterTypeId"
        :options="collectionTypes"
        option-label="name"
        option-value="id"
        placeholder="Tous les types"
        show-clear
        class="w-48"
        @change="reload"
      />
    </div>

    <div v-if="pag.loading.value" class="flex justify-center py-12">
      <ProgressSpinner />
    </div>

    <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      <div
        v-for="col in pag.data.value"
        :key="col.id"
        class="bg-white dark:bg-gray-800 rounded-lg shadow-sm hover:shadow-md transition-shadow cursor-pointer p-5"
        @click="router.push(`/collections/${col.id}`)"
      >
        <div class="flex items-center justify-between mb-2">
          <h3 class="font-semibold text-gray-800 dark:text-gray-100">
            {{ col.name }}
          </h3>
          <Tag
            :value="col.collection_type?.name ?? '?'"
            severity="info"
            class="text-xs"
          />
        </div>
        <p
          v-if="col.description"
          class="text-sm text-gray-500 dark:text-gray-400 mb-3"
        >
          {{ col.description }}
        </p>
        <div class="text-sm text-gray-400 dark:text-gray-500">
          <i class="pi pi-box mr-1"></i> {{ col.item_count }} items
        </div>
        <div class="flex gap-1 mt-3 justify-end" @click.stop>
          <Button
            icon="pi pi-pencil"
            text
            size="small"
            @click="openEdit(col)"
          />
          <Button
            icon="pi pi-trash"
            text
            severity="danger"
            size="small"
            @click="confirmDelete(col)"
          />
        </div>
      </div>
    </div>

    <div
      v-if="!pag.loading.value && pag.data.value.length === 0"
      class="text-center text-gray-400 dark:text-gray-500 py-12"
    >
      <i class="pi pi-bookmark text-4xl mb-2"></i>
      <p>Aucune collection. Créez-en une pour commencer !</p>
    </div>

    <!-- Create / Edit Dialog -->
    <Dialog
      v-model:visible="dialogVisible"
      :header="editing ? 'Modifier' : 'Nouvelle collection'"
      modal
      class="w-full max-w-md"
    >
      <div class="flex flex-col gap-4">
        <div class="flex flex-col gap-1">
          <label class="text-sm font-medium">Type de collection</label>
          <Select
            v-model="form.collection_type_id"
            :options="collectionTypes"
            option-label="name"
            option-value="id"
            placeholder="Choisir..."
            :disabled="!!editing"
          />
        </div>
        <div class="flex flex-col gap-1">
          <label class="text-sm font-medium">Nom</label>
          <InputText v-model="form.name" placeholder="Ma collection de..." />
        </div>
        <div class="flex flex-col gap-1">
          <label class="text-sm font-medium">Description</label>
          <Textarea v-model="form.description" rows="2" />
        </div>
      </div>
      <template #footer>
        <Button label="Annuler" text @click="dialogVisible = false" />
        <Button
          :label="editing ? 'Enregistrer' : 'Créer'"
          icon="pi pi-check"
          :loading="saving"
          @click="save"
        />
      </template>
    </Dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from "vue";
import { useRouter } from "vue-router";
import Button from "primevue/button";
import Dialog from "primevue/dialog";
import InputText from "primevue/inputtext";
import Textarea from "primevue/textarea";
import Select from "primevue/select";
import Tag from "primevue/tag";
import ProgressSpinner from "primevue/progressspinner";
import { useConfirm } from "primevue/useconfirm";
import { useToast } from "primevue/usetoast";
import { collectionApi, collectionTypeApi } from "@/api/client";
import { usePagination } from "@/composables/usePagination";
import { useNotify } from "@/composables/useNotify";
import type { Collection, CollectionType } from "@/api/types";

const router = useRouter();
const confirm = useConfirm();
const toast = useToast();
const { notifyError } = useNotify();
const pag = usePagination(collectionApi.list);
const collectionTypes = ref<CollectionType[]>([]);
const filterTypeId = ref<string>();

const dialogVisible = ref(false);
const editing = ref<Collection | null>(null);
const saving = ref(false);
const form = ref({ collection_type_id: "", name: "", description: "" });

onMounted(async () => {
  await collectionTypeApi
    .list({ per_page: 100 })
    .then((r) => (collectionTypes.value = r.data));
  reload();
});

function reload() {
  pag.page.value = 1;
  pag.load(
    filterTypeId.value ? { collection_type_id: filterTypeId.value } : {},
  );
}

function openCreate() {
  editing.value = null;
  form.value = { collection_type_id: "", name: "", description: "" };
  dialogVisible.value = true;
}

function openEdit(col: Collection) {
  editing.value = col;
  form.value = {
    collection_type_id: col.collection_type_id,
    name: col.name,
    description: col.description ?? "",
  };
  dialogVisible.value = true;
}

async function save() {
  saving.value = true;
  try {
    if (editing.value) {
      await collectionApi.update(editing.value.id, {
        name: form.value.name,
        description: form.value.description,
      });
      toast.add({
        severity: "success",
        summary: "Collection modifiée",
        life: 3000,
      });
    } else {
      await collectionApi.create(form.value);
      toast.add({
        severity: "success",
        summary: "Collection créée",
        life: 3000,
      });
    }
    dialogVisible.value = false;
    reload();
  } catch (e) {
    notifyError(e);
  } finally {
    saving.value = false;
  }
}

function confirmDelete(col: Collection) {
  confirm.require({
    message: `Supprimer la collection « ${col.name} » ? Les items ne seront pas supprimés.`,
    header: "Confirmation",
    icon: "pi pi-exclamation-triangle",
    acceptLabel: "Supprimer",
    rejectLabel: "Annuler",
    acceptClass: "p-button-danger",
    accept: async () => {
      await collectionApi.delete(col.id);
      toast.add({ severity: "success", summary: "Supprimée", life: 3000 });
      reload();
    },
  });
}
</script>
