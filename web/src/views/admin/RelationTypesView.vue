<template>
  <div>
    <div class="flex items-center justify-between mb-6">
      <h1 class="text-2xl font-bold text-gray-800 dark:text-gray-100">
        {{ t("admin.relationTypes.title") }}
      </h1>
      <ElButton type="primary" @click="openCreate">
        <i class="pi pi-plus mr-2" aria-hidden="true"></i>
        {{ t("admin.shared.newType") }}
      </ElButton>
    </div>

    <ElTable
      v-loading="pag.loading.value"
      :data="pag.data.value"
      stripe
      row-key="id"
    >
      <ElTableColumn :label="t('common.name')" width="200">
        <template #default="{ row }">
          {{ typeTitle(row as RelationType, lang) }}
          <span
            v-if="
              typeTitle(row as RelationType, lang) !==
              (row as RelationType).name
            "
            class="ml-1 text-xs text-gray-400 dark:text-gray-500"
            >{{ (row as RelationType).name }}</span
          >
        </template>
      </ElTableColumn>
      <ElTableColumn :label="t('common.description')">
        <template #default="{ row }">
          {{ typeDescription(row as RelationType, lang) }}
        </template>
      </ElTableColumn>
      <ElTableColumn :label="t('admin.relationTypes.sourcesTargets')">
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
      <ElTableColumn :label="t('common.actions')" width="110">
        <template #default="{ row }">
          <div class="flex gap-1">
            <ElButton
              text
              size="small"
              :aria-label="t('common.edit')"
              @click="openEdit(row as RelationType)"
            >
              <i class="pi pi-pencil" aria-hidden="true"></i>
            </ElButton>
            <ElButton
              text
              size="small"
              type="danger"
              :aria-label="t('common.delete')"
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
      :title="editing ? t('admin.shared.editType') : t('admin.shared.newType')"
      width="48rem"
    >
      <div class="flex flex-col gap-4">
        <div class="flex flex-col gap-1">
          <label for="type-name" class="text-sm font-medium">{{
            t("admin.shared.technicalName")
          }}</label>
          <ElInput
            id="type-name"
            v-model="form.name"
            :placeholder="t('admin.relationTypes.namePlaceholder')"
            aria-describedby="type-name-help"
          />
          <small
            id="type-name-help"
            class="text-xs text-gray-500 dark:text-gray-400"
            >{{ t("admin.shared.technicalNameHelp") }}</small
          >
        </div>
        <div class="flex flex-col gap-1">
          <label for="type-desc" class="text-sm font-medium">{{
            t("common.description")
          }}</label>
          <ElInput
            id="type-desc"
            v-model="form.description"
            type="textarea"
            :rows="2"
          />
        </div>
        <div class="grid grid-cols-2 gap-4">
          <div
            v-for="side in sides"
            :key="side.field"
            class="flex flex-col gap-1"
          >
            <label :for="side.id" class="text-sm font-medium">{{
              side.label
            }}</label>
            <ElSelect
              :id="side.id"
              v-model="form[side.field]"
              multiple
              collapse-tags
              collapse-tags-tooltip
              :placeholder="t('admin.relationTypes.choose')"
            >
              <ElOption
                v-for="it in itemTypes"
                :key="it.id"
                :label="typeTitle(it, lang)"
                :value="it.id"
              >
                {{ typeTitle(it, lang) }}
                <span
                  v-if="typeTitle(it, lang) !== it.name"
                  class="ml-1 text-xs text-gray-400 dark:text-gray-500"
                  >{{ it.name }}</span
                >
              </ElOption>
            </ElSelect>
          </div>
        </div>
        <div class="flex flex-col gap-1">
          <span class="text-sm font-medium">{{
            t("admin.relationTypes.linkAttributes")
          }}</span>
          <SchemaBuilder ref="schemaBuilderRef" v-model="schemaValue" />
        </div>
      </div>
      <template #footer>
        <ElButton text @click="dialogVisible = false">{{
          t("common.cancel")
        }}</ElButton>
        <ElButton type="primary" :loading="saving" @click="save">
          <i class="pi pi-check mr-2" aria-hidden="true"></i>
          {{ editing ? t("common.save") : t("common.create") }}
        </ElButton>
      </template>
    </ElDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, onMounted } from "vue";
import { useI18n } from "vue-i18n";
import { relationTypeApi } from "@/api/client";
import { usePagination } from "@/composables/usePagination";
import { useNotify } from "@/composables/useNotify";
import { useConfirmDialog } from "@/composables/useConfirmDialog";
import { useMetaTypes } from "@/stores/metaTypes";
import { useLang } from "@/composables/useLang";
import { typeDescription, typeTitle } from "@/lib/schema";
import SchemaBuilder from "@/components/common/SchemaBuilder.vue";
import type { RelationType, ItemType } from "@/api/types";

const { t } = useI18n();
const { notifyError, notifySuccess } = useNotify();
const { confirm } = useConfirmDialog();
const pag = usePagination(relationTypeApi.list);
const metaTypes = useMetaTypes();
const { lang } = useLang();
const itemTypes = computed(() => metaTypes.itemTypes);
// ItemTypes complets (avec schéma, donc x-title) indexés par id.
const itemTypesById = computed(
  () => new Map(metaTypes.itemTypes.map((it) => [it.id, it])),
);

const dialogVisible = ref(false);
const editing = ref<RelationType | null>(null);
const saving = ref(false);
const form = ref({
  name: "",
  description: "",
  source_item_type_ids: [] as string[],
  target_item_type_ids: [] as string[],
});
// Sélecteurs des types source / cible (même gabarit).
const sides = computed(() => [
  {
    id: "rt-sources",
    field: "source_item_type_ids" as const,
    label: t("admin.relationTypes.sourceTypes"),
  },
  {
    id: "rt-targets",
    field: "target_item_type_ids" as const,
    label: t("admin.relationTypes.targetTypes"),
  },
]);
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
      .catch((e) => notifyError(e, t("admin.relationTypes.itemTypesFailed"))),
  ]);
});

function load() {
  return pag.load().catch((e) => notifyError(e, t("admin.shared.loadFailed")));
}

function onPageChange(p: number) {
  pag.goToPage(p);
  load();
}

function typeNames(types?: ItemType[] | null): string {
  return (
    (types ?? [])
      .map((it) => typeTitle(itemTypesById.value.get(it.id) ?? it, lang.value))
      .join(", ") || "?"
  );
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
      notifySuccess(t("admin.relationTypes.updated"));
    } else {
      await relationTypeApi.create(payload);
      notifySuccess(t("admin.relationTypes.created"));
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
    message: t("admin.relationTypes.confirmDelete", {
      name: typeTitle(row, lang.value),
    }),
    confirmText: t("common.delete"),
    danger: true,
  });
  if (!ok) return;
  try {
    await relationTypeApi.delete(row.id);
    notifySuccess(t("admin.relationTypes.deleted"));
    metaTypes.invalidate();
    load();
  } catch (e) {
    notifyError(e);
  }
}
</script>
