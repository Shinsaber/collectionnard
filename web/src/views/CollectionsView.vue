<template>
  <div>
    <div class="flex items-center justify-between mb-6">
      <h1 class="text-2xl font-bold text-gray-800 dark:text-gray-100">
        Mes collections
      </h1>
      <ElButton type="primary" @click="openCreate">
        <i class="pi pi-plus mr-2" aria-hidden="true"></i> Nouvelle collection
      </ElButton>
    </div>

    <!-- Filter -->
    <div class="flex gap-3 mb-4">
      <ElSelect
        v-model="filterTypeId"
        placeholder="Tous les types"
        clearable
        class="w-48!"
        aria-label="Filtrer par type de collection"
        @change="reload"
      >
        <ElOption
          v-for="t in collectionTypes"
          :key="t.id"
          :label="t.name"
          :value="t.id"
        />
      </ElSelect>
    </div>

    <AppSpinner v-if="pag.loading.value" />

    <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      <div
        v-for="col in pag.data.value"
        :key="col.id"
        class="bg-white dark:bg-gray-800 rounded-lg shadow-xs hover:shadow-md transition-shadow cursor-pointer p-5"
        role="link"
        tabindex="0"
        @click="router.push(`/collections/${col.id}`)"
        @keyup.enter="router.push(`/collections/${col.id}`)"
      >
        <div class="flex items-center justify-between mb-2">
          <h3 class="font-semibold text-gray-800 dark:text-gray-100">
            {{ col.name }}
          </h3>
          <ElTag type="info" size="small">{{
            col.collection_type?.name ?? "?"
          }}</ElTag>
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
        <div class="flex gap-1 mt-3 justify-end" @click.stop @keyup.stop>
          <ElButton
            text
            size="small"
            aria-label="Modifier"
            @click="openEdit(col)"
          >
            <i class="pi pi-pencil" aria-hidden="true"></i>
          </ElButton>
          <ElButton
            text
            size="small"
            type="danger"
            aria-label="Supprimer"
            @click="confirmDelete(col)"
          >
            <i class="pi pi-trash" aria-hidden="true"></i>
          </ElButton>
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
    <ElDialog
      v-model="dialogVisible"
      :title="editing ? 'Modifier' : 'Nouvelle collection'"
      width="28rem"
    >
      <div class="flex flex-col gap-4">
        <div class="flex flex-col gap-1">
          <label for="col-type" class="text-sm font-medium"
            >Type de collection</label
          >
          <ElSelect
            id="col-type"
            v-model="form.collection_type_id"
            placeholder="Choisir..."
            :disabled="!!editing"
          >
            <ElOption
              v-for="t in collectionTypes"
              :key="t.id"
              :label="t.name"
              :value="t.id"
            />
          </ElSelect>
        </div>
        <div class="flex flex-col gap-1">
          <label for="col-name" class="text-sm font-medium">Nom</label>
          <ElInput
            id="col-name"
            v-model="form.name"
            placeholder="Ma collection de..."
          />
        </div>
        <div class="flex flex-col gap-1">
          <label for="col-desc" class="text-sm font-medium">Description</label>
          <ElInput
            id="col-desc"
            v-model="form.description"
            type="textarea"
            :rows="2"
          />
        </div>
      </div>
      <template #footer>
        <ElButton text @click="dialogVisible = false">Annuler</ElButton>
        <ElButton type="primary" :loading="saving" @click="save">
          <i class="pi pi-check mr-2" aria-hidden="true"></i>
          {{ editing ? "Enregistrer" : "Créer" }}
        </ElButton>
      </template>
    </ElDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import { collectionApi } from "@/api/client";
import { usePagination } from "@/composables/usePagination";
import { useNotify } from "@/composables/useNotify";
import { useConfirmDialog } from "@/composables/useConfirmDialog";
import { useMetaTypes } from "@/stores/metaTypes";
import AppSpinner from "@/components/common/AppSpinner.vue";
import type { Collection } from "@/api/types";

const router = useRouter();
const { notifyError, notifySuccess } = useNotify();
const { confirm } = useConfirmDialog();
const pag = usePagination(collectionApi.list);
const metaTypes = useMetaTypes();
const collectionTypes = computed(() => metaTypes.collectionTypes);
const filterTypeId = ref<string>();

const dialogVisible = ref(false);
const editing = ref<Collection | null>(null);
const saving = ref(false);
const form = ref({ collection_type_id: "", name: "", description: "" });

onMounted(async () => {
  await metaTypes
    .loadCollectionTypes()
    .catch((e) => notifyError(e, "Types de collection indisponibles"));
  reload();
});

function reload() {
  pag.page.value = 1;
  pag
    .load(filterTypeId.value ? { collection_type_id: filterTypeId.value } : {})
    .catch((e) => notifyError(e, "Chargement impossible"));
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
      notifySuccess("Collection modifiée");
    } else {
      await collectionApi.create(form.value);
      notifySuccess("Collection créée");
    }
    dialogVisible.value = false;
    reload();
  } catch (e) {
    notifyError(e);
  } finally {
    saving.value = false;
  }
}

async function confirmDelete(col: Collection) {
  const ok = await confirm({
    message: `Supprimer la collection « ${col.name} » ? Les items ne seront pas supprimés.`,
    confirmText: "Supprimer",
    danger: true,
  });
  if (!ok) return;
  try {
    await collectionApi.delete(col.id);
    notifySuccess("Collection supprimée");
    reload();
  } catch (e) {
    notifyError(e);
  }
}
</script>
