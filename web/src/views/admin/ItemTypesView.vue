<template>
  <div>
    <div class="flex items-center justify-between mb-6">
      <h1 class="text-2xl font-bold text-gray-800 dark:text-gray-100">
        Types d'items
      </h1>
      <Button label="Nouveau type" icon="pi pi-plus" @click="openCreate" />
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
      @page="onPageChange"
    >
      <Column field="name" header="Nom" sortable />
      <Column field="description" header="Description" />
      <Column header="Champs schema">
        <template #body="{ data: row }">
          <div class="text-xs text-gray-500 dark:text-gray-400">
            {{ schemaFieldNames(row.attributes_schema) }}
          </div>
        </template>
      </Column>
      <Column header="Actions" style="width: 120px">
        <template #body="{ data: row }">
          <div class="flex gap-1">
            <Button
              icon="pi pi-pencil"
              text
              size="small"
              aria-label="Modifier"
              @click="openEdit(row)"
            />
            <Button
              icon="pi pi-trash"
              text
              severity="danger"
              size="small"
              aria-label="Supprimer"
              @click="confirmDelete(row)"
            />
          </div>
        </template>
      </Column>
    </DataTable>

    <!-- Create / Edit Dialog -->
    <Dialog
      v-model:visible="dialogVisible"
      :header="editing ? 'Modifier le type' : 'Nouveau type'"
      modal
      class="w-full max-w-2xl"
    >
      <div class="flex flex-col gap-4">
        <div class="flex flex-col gap-1">
          <label class="text-sm font-medium">Nom</label>
          <InputText
            v-model="form.name"
            placeholder="Ex: Serie, Episode, DVD..."
          />
        </div>
        <div class="flex flex-col gap-1">
          <label class="text-sm font-medium">Description</label>
          <Textarea v-model="form.description" rows="2" />
        </div>
        <div class="flex flex-col gap-1">
          <label class="text-sm font-medium">Attributs</label>
          <SchemaBuilder ref="schemaBuilderRef" v-model="schemaValue" />
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
import DataTable from "primevue/datatable";
import Column from "primevue/column";
import Button from "primevue/button";
import Dialog from "primevue/dialog";
import InputText from "primevue/inputtext";
import Textarea from "primevue/textarea";
import { useConfirm } from "primevue/useconfirm";
import { useToast } from "primevue/usetoast";
import { itemTypeApi, extractApiError } from "@/api/client";
import { usePagination } from "@/composables/usePagination";
import { useMetaTypes } from "@/stores/metaTypes";
import { useLang } from "@/composables/useLang";
import { fieldLabel, orderedKeys } from "@/lib/schema";
import SchemaBuilder from "@/components/common/SchemaBuilder.vue";
import type { ItemType } from "@/api/types";

const confirm = useConfirm();
const toast = useToast();
const pag = usePagination(itemTypeApi.list);
const metaTypes = useMetaTypes();
const { lang } = useLang();

const dialogVisible = ref(false);
const editing = ref<ItemType | null>(null);
const saving = ref(false);
const form = ref({ name: "", description: "" });
const schemaValue = ref<Record<string, unknown>>({
  type: "object",
  properties: {},
  additionalProperties: false,
});
const schemaBuilderRef = ref<InstanceType<typeof SchemaBuilder> | null>(null);

onMounted(() => pag.load());

function onPageChange(e: { page: number; rows: number }) {
  pag.onPage(e);
  pag.load();
}

function schemaFieldNames(schema: Record<string, unknown>): string {
  const keys = orderedKeys(schema);
  if (keys.length === 0) return "—";
  return keys.map((k) => fieldLabel(schema, k, lang.value)).join(", ");
}

function openCreate() {
  editing.value = null;
  form.value = { name: "", description: "" };
  const empty = { type: "object", properties: {}, additionalProperties: false };
  schemaValue.value = empty;
  schemaBuilderRef.value?.resetFromSchema(empty);
  dialogVisible.value = true;
}

function openEdit(row: ItemType) {
  editing.value = row;
  form.value = { name: row.name, description: row.description ?? "" };
  schemaValue.value = row.attributes_schema;
  schemaBuilderRef.value?.resetFromSchema(row.attributes_schema);
  dialogVisible.value = true;
}

async function save() {
  const schema = schemaValue.value;

  saving.value = true;
  try {
    if (editing.value) {
      await itemTypeApi.update(editing.value.id, {
        ...form.value,
        attributes_schema: schema,
      });
      toast.add({ severity: "success", summary: "Type modifié", life: 3000 });
    } else {
      await itemTypeApi.create({ ...form.value, attributes_schema: schema });
      toast.add({ severity: "success", summary: "Type créé", life: 3000 });
    }
    dialogVisible.value = false;
    metaTypes.invalidate();
    pag.load();
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

function confirmDelete(row: ItemType) {
  confirm.require({
    message: `Supprimer le type « ${row.name} » ?`,
    header: "Confirmation",
    icon: "pi pi-exclamation-triangle",
    acceptLabel: "Supprimer",
    rejectLabel: "Annuler",
    acceptClass: "p-button-danger",
    accept: async () => {
      try {
        await itemTypeApi.delete(row.id);
        toast.add({
          severity: "success",
          summary: "Type supprimé",
          life: 3000,
        });
        metaTypes.invalidate();
        pag.load();
      } catch (e: unknown) {
        const apiErr = extractApiError(e);
        toast.add({
          severity: "error",
          summary: "Erreur",
          detail: apiErr.message,
          life: 5000,
        });
      }
    },
  });
}
</script>
