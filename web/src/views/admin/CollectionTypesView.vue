<template>
  <div>
    <div class="flex items-center justify-between mb-6">
      <h1 class="text-2xl font-bold text-gray-800 dark:text-gray-100">
        Types de collections
      </h1>
      <ElButton type="primary" @click="openCreate">
        <i class="pi pi-plus mr-2" aria-hidden="true"></i> Nouveau type
      </ElButton>
    </div>

    <ElTable
      v-loading="pag.loading.value"
      :data="pag.data.value"
      stripe
      row-key="id"
    >
      <ElTableColumn prop="name" label="Nom" width="200" />
      <ElTableColumn prop="description" label="Description" />
      <ElTableColumn label="Champs contextuels">
        <template #default="{ row }">
          <div class="text-xs text-gray-500 dark:text-gray-400">
            {{ schemaFieldNames((row as CollectionType).attributes_schema) }}
          </div>
        </template>
      </ElTableColumn>
      <ElTableColumn label="Actions" width="110">
        <template #default="{ row }">
          <div class="flex gap-1">
            <ElButton
              text
              size="small"
              aria-label="Modifier"
              @click="openEdit(row as CollectionType)"
            >
              <i class="pi pi-pencil" aria-hidden="true"></i>
            </ElButton>
            <ElButton
              text
              size="small"
              type="danger"
              aria-label="Supprimer"
              @click="confirmDelete(row as CollectionType)"
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
    <ElDialog
      v-model="dialogVisible"
      :title="editing ? 'Modifier le type' : 'Nouveau type'"
      width="48rem"
    >
      <div class="flex flex-col gap-4">
        <div class="flex flex-col gap-1">
          <label for="type-name" class="text-sm font-medium">Nom</label>
          <ElInput
            id="type-name"
            v-model="form.name"
            placeholder="Ex: Ownership, Watching, Wishlist..."
          />
        </div>
        <div class="flex flex-col gap-1">
          <label for="type-desc" class="text-sm font-medium">Description</label>
          <ElInput
            id="type-desc"
            v-model="form.description"
            type="textarea"
            :rows="2"
          />
        </div>
        <div class="flex flex-col gap-1">
          <span class="text-sm font-medium">Attributs contextuels</span>
          <SchemaBuilder ref="schemaBuilderRef" v-model="schemaValue" />
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
import { ref, onMounted } from "vue";
import { collectionTypeApi } from "@/api/client";
import { usePagination } from "@/composables/usePagination";
import { useNotify } from "@/composables/useNotify";
import { useConfirmDialog } from "@/composables/useConfirmDialog";
import { useMetaTypes } from "@/stores/metaTypes";
import { useLang } from "@/composables/useLang";
import { fieldLabel, orderedKeys } from "@/lib/schema";
import SchemaBuilder from "@/components/common/SchemaBuilder.vue";
import type { CollectionType } from "@/api/types";

const { notifyError, notifySuccess } = useNotify();
const { confirm } = useConfirmDialog();
const pag = usePagination(collectionTypeApi.list);
const metaTypes = useMetaTypes();
const { lang } = useLang();

const dialogVisible = ref(false);
const editing = ref<CollectionType | null>(null);
const saving = ref(false);
const form = ref({ name: "", description: "" });
const emptySchema = () => ({
  type: "object",
  properties: {},
  additionalProperties: false,
});
const schemaValue = ref<Record<string, unknown>>(emptySchema());
const schemaBuilderRef = ref<InstanceType<typeof SchemaBuilder> | null>(null);

onMounted(() => load());

function load() {
  pag.load().catch((e) => notifyError(e, "Chargement impossible"));
}

function onPageChange(p: number) {
  pag.goToPage(p);
  load();
}

function schemaFieldNames(schema: Record<string, unknown>): string {
  const keys = orderedKeys(schema);
  if (keys.length === 0) return "—";
  return keys.map((k) => fieldLabel(schema, k, lang.value)).join(", ");
}

function openCreate() {
  editing.value = null;
  form.value = { name: "", description: "" };
  schemaValue.value = emptySchema();
  schemaBuilderRef.value?.resetFromSchema(schemaValue.value);
  dialogVisible.value = true;
}

function openEdit(row: CollectionType) {
  editing.value = row;
  form.value = { name: row.name, description: row.description ?? "" };
  schemaValue.value = row.attributes_schema;
  schemaBuilderRef.value?.resetFromSchema(row.attributes_schema);
  dialogVisible.value = true;
}

async function save() {
  saving.value = true;
  try {
    const payload = { ...form.value, attributes_schema: schemaValue.value };
    if (editing.value) {
      await collectionTypeApi.update(editing.value.id, payload);
      notifySuccess("Type modifié");
    } else {
      await collectionTypeApi.create(payload);
      notifySuccess("Type créé");
    }
    dialogVisible.value = false;
    metaTypes.invalidate();
    load();
  } catch (e) {
    notifyError(e);
  } finally {
    saving.value = false;
  }
}

async function confirmDelete(row: CollectionType) {
  const ok = await confirm({
    message: `Supprimer le type « ${row.name} » ?`,
    confirmText: "Supprimer",
    danger: true,
  });
  if (!ok) return;
  try {
    await collectionTypeApi.delete(row.id);
    notifySuccess("Type supprimé");
    metaTypes.invalidate();
    load();
  } catch (e) {
    notifyError(e);
  }
}
</script>
