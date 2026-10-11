<template>
  <div>
    <div class="flex items-center justify-between mb-6">
      <h1 class="text-2xl font-bold text-gray-800 dark:text-gray-100">
        Types de relations
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
      <Column header="Sources → Cibles">
        <template #body="{ data: row }">
          <span class="text-sm">
            {{
              (row.source_item_types ?? [])
                .map((t: ItemType) => t.name)
                .join(", ") || "?"
            }}
            <i
              class="pi pi-arrow-right mx-1 text-gray-400 dark:text-gray-500"
            ></i>
            {{
              (row.target_item_types ?? [])
                .map((t: ItemType) => t.name)
                .join(", ") || "?"
            }}
          </span>
        </template>
      </Column>
      <Column header="Actions" style="width: 120px">
        <template #body="{ data: row }">
          <div class="flex gap-1">
            <Button
              icon="pi pi-pencil"
              text
              size="small"
              @click="openEdit(row)"
            />
            <Button
              icon="pi pi-trash"
              text
              severity="danger"
              size="small"
              @click="confirmDelete(row)"
            />
          </div>
        </template>
      </Column>
    </DataTable>

    <Dialog
      v-model:visible="dialogVisible"
      :header="editing ? 'Modifier' : 'Nouveau type de relation'"
      modal
      class="w-full max-w-2xl"
    >
      <div class="flex flex-col gap-4">
        <div class="flex flex-col gap-1">
          <label class="text-sm font-medium">Nom</label>
          <InputText v-model="form.name" />
        </div>
        <div class="flex flex-col gap-1">
          <label class="text-sm font-medium">Description</label>
          <Textarea v-model="form.description" rows="2" />
        </div>
        <div class="grid grid-cols-2 gap-4">
          <div class="flex flex-col gap-1">
            <label class="text-sm font-medium">Types source</label>
            <MultiSelect
              v-model="form.source_item_type_ids"
              :options="itemTypes"
              option-label="name"
              option-value="id"
              placeholder="Choisir..."
              display="chip"
            />
          </div>
          <div class="flex flex-col gap-1">
            <label class="text-sm font-medium">Types cible</label>
            <MultiSelect
              v-model="form.target_item_type_ids"
              :options="itemTypes"
              option-label="name"
              option-value="id"
              placeholder="Choisir..."
              display="chip"
            />
          </div>
        </div>
        <div class="flex flex-col gap-1">
          <label class="text-sm font-medium">Attributs du lien</label>
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
import MultiSelect from "primevue/multiselect";
import { useConfirm } from "primevue/useconfirm";
import { useToast } from "primevue/usetoast";
import { relationTypeApi, itemTypeApi, extractApiError } from "@/api/client";
import { usePagination } from "@/composables/usePagination";
import SchemaBuilder from "@/components/common/SchemaBuilder.vue";
import type { RelationType, ItemType } from "@/api/types";

const confirm = useConfirm();
const toast = useToast();
const pag = usePagination(relationTypeApi.list);
const itemTypes = ref<ItemType[]>([]);

const dialogVisible = ref(false);
const editing = ref<RelationType | null>(null);
const saving = ref(false);
const form = ref({
  name: "",
  description: "",
  source_item_type_ids: [] as string[],
  target_item_type_ids: [] as string[],
});
const schemaValue = ref<Record<string, unknown>>({
  type: "object",
  properties: {},
  additionalProperties: false,
});
const schemaBuilderRef = ref<InstanceType<typeof SchemaBuilder> | null>(null);

onMounted(async () => {
  await Promise.all([
    pag.load(),
    itemTypeApi.list({ per_page: 100 }).then((r) => (itemTypes.value = r.data)),
  ]);
});

function onPageChange(e: { page: number; rows: number }) {
  pag.onPage(e);
  pag.load();
}

function openCreate() {
  editing.value = null;
  form.value = {
    name: "",
    description: "",
    source_item_type_ids: [],
    target_item_type_ids: [],
  };
  const empty = { type: "object", properties: {}, additionalProperties: false };
  schemaValue.value = empty;
  schemaBuilderRef.value?.resetFromSchema(empty);
  dialogVisible.value = true;
}

function openEdit(row: RelationType) {
  editing.value = row;
  form.value = {
    name: row.name,
    description: row.description ?? "",
    source_item_type_ids: row.source_item_type_ids ?? [],
    target_item_type_ids: row.target_item_type_ids ?? [],
  };
  schemaValue.value = row.attributes_schema;
  schemaBuilderRef.value?.resetFromSchema(row.attributes_schema);
  dialogVisible.value = true;
}

async function save() {
  const schema = schemaValue.value;

  saving.value = true;
  try {
    const payload = { ...form.value, attributes_schema: schema };
    if (editing.value) {
      await relationTypeApi.update(editing.value.id, payload);
      toast.add({ severity: "success", summary: "Modifié", life: 3000 });
    } else {
      await relationTypeApi.create(payload);
      toast.add({ severity: "success", summary: "Créé", life: 3000 });
    }
    dialogVisible.value = false;
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

function confirmDelete(row: RelationType) {
  confirm.require({
    message: `Supprimer « ${row.name} » ?`,
    header: "Confirmation",
    icon: "pi pi-exclamation-triangle",
    acceptLabel: "Supprimer",
    rejectLabel: "Annuler",
    acceptClass: "p-button-danger",
    accept: async () => {
      try {
        await relationTypeApi.delete(row.id);
        toast.add({ severity: "success", summary: "Supprimé", life: 3000 });
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
