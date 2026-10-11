<template>
  <div>
    <div class="flex items-center justify-between mb-6">
      <h1 class="text-2xl font-bold text-gray-800 dark:text-gray-100">
        Types de relations
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
      <ElTableColumn label="Sources → Cibles">
        <template #default="{ row }">
          <div class="text-xs text-gray-500 dark:text-gray-400">
            <span class="text-sm">
              {{ typeNames((row as RelationType).source_item_types) }}
              <i
                class="pi pi-arrow-right mx-1 text-gray-400 dark:text-gray-500"
                aria-hidden="true"
              ></i>
              {{ typeNames((row as RelationType).target_item_types) }}
            </span>
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
              @click="openEdit(row as RelationType)"
            >
              <i class="pi pi-pencil" aria-hidden="true"></i>
            </ElButton>
            <ElButton
              text
              size="small"
              type="danger"
              aria-label="Supprimer"
              @click="confirmDelete(row as RelationType)"
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
            placeholder="Ex: contains_episode"
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
        <div class="grid grid-cols-2 gap-4">
          <div class="flex flex-col gap-1">
            <label for="rt-sources" class="text-sm font-medium"
              >Types source</label
            >
            <ElSelect
              id="rt-sources"
              v-model="form.source_item_type_ids"
              multiple
              collapse-tags
              collapse-tags-tooltip
              placeholder="Choisir..."
            >
              <ElOption
                v-for="t in itemTypes"
                :key="t.id"
                :label="t.name"
                :value="t.id"
              />
            </ElSelect>
          </div>
          <div class="flex flex-col gap-1">
            <label for="rt-targets" class="text-sm font-medium"
              >Types cible</label
            >
            <ElSelect
              id="rt-targets"
              v-model="form.target_item_type_ids"
              multiple
              collapse-tags
              collapse-tags-tooltip
              placeholder="Choisir..."
            >
              <ElOption
                v-for="t in itemTypes"
                :key="t.id"
                :label="t.name"
                :value="t.id"
              />
            </ElSelect>
          </div>
        </div>
        <div class="flex flex-col gap-1">
          <span class="text-sm font-medium">Attributs du lien</span>
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
import { computed, ref, onMounted } from "vue";
import { relationTypeApi } from "@/api/client";
import { usePagination } from "@/composables/usePagination";
import { useNotify } from "@/composables/useNotify";
import { useConfirmDialog } from "@/composables/useConfirmDialog";
import { useMetaTypes } from "@/stores/metaTypes";
import SchemaBuilder from "@/components/common/SchemaBuilder.vue";
import type { RelationType, ItemType } from "@/api/types";

const { notifyError, notifySuccess } = useNotify();
const { confirm } = useConfirmDialog();
const pag = usePagination(relationTypeApi.list);
const metaTypes = useMetaTypes();
const itemTypes = computed(() => metaTypes.itemTypes);

const dialogVisible = ref(false);
const editing = ref<RelationType | null>(null);
const saving = ref(false);
const form = ref({
  name: "",
  description: "",
  source_item_type_ids: [] as string[],
  target_item_type_ids: [] as string[],
});
const emptySchema = () => ({
  type: "object",
  properties: {},
  additionalProperties: false,
});
const schemaValue = ref<Record<string, unknown>>(emptySchema());
const schemaBuilderRef = ref<InstanceType<typeof SchemaBuilder> | null>(null);

onMounted(async () => {
  await Promise.all([
    load(),
    metaTypes
      .loadItemTypes()
      .catch((e) => notifyError(e, "Types d'items indisponibles")),
  ]);
});

function load() {
  return pag.load().catch((e) => notifyError(e, "Chargement impossible"));
}

function onPageChange(p: number) {
  pag.goToPage(p);
  load();
}

function typeNames(types?: ItemType[] | null): string {
  return (types ?? []).map((t) => t.name).join(", ") || "?";
}

function openCreate() {
  editing.value = null;
  form.value = {
    name: "",
    description: "",
    source_item_type_ids: [],
    target_item_type_ids: [],
  };
  schemaValue.value = emptySchema();
  schemaBuilderRef.value?.resetFromSchema(schemaValue.value);
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
  saving.value = true;
  try {
    const payload = { ...form.value, attributes_schema: schemaValue.value };
    if (editing.value) {
      await relationTypeApi.update(editing.value.id, payload);
      notifySuccess("Type de relation modifié");
    } else {
      await relationTypeApi.create(payload);
      notifySuccess("Type de relation créé");
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

async function confirmDelete(row: RelationType) {
  const ok = await confirm({
    message: `Supprimer le type de relation « ${row.name} » ?`,
    confirmText: "Supprimer",
    danger: true,
  });
  if (!ok) return;
  try {
    await relationTypeApi.delete(row.id);
    notifySuccess("Type de relation supprimé");
    metaTypes.invalidate();
    load();
  } catch (e) {
    notifyError(e);
  }
}
</script>
