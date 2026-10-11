<template>
  <div>
    <div class="flex items-center justify-between mb-6">
      <h1 class="text-2xl font-bold text-gray-800 dark:text-gray-100">
        {{ t("collections.list.title") }}
      </h1>
      <ElButton type="primary" @click="openCreate">
        <i class="pi pi-plus mr-2" aria-hidden="true"></i>
        {{ t("collections.list.new") }}
      </ElButton>
    </div>

    <!-- Filter -->
    <div class="flex gap-3 mb-4">
      <ElSelect
        v-model="filterTypeId"
        :placeholder="t('collections.list.allTypes')"
        clearable
        class="w-48!"
        :aria-label="t('collections.list.filterByType')"
        @change="reload"
      >
        <ElOption
          v-for="ct in collectionTypes"
          :key="ct.id"
          :label="typeTitle(ct, lang)"
          :value="ct.id"
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
            col.collection_type ? typeTitle(col.collection_type, lang) : "?"
          }}</ElTag>
        </div>
        <p
          v-if="col.description"
          class="text-sm text-gray-500 dark:text-gray-400 mb-3"
        >
          {{ col.description }}
        </p>
        <div class="text-sm text-gray-400 dark:text-gray-500">
          <i class="pi pi-box mr-1" aria-hidden="true"></i>
          {{ t("collections.itemCount", col.item_count) }}
        </div>
        <div class="flex gap-1 mt-3 justify-end" @click.stop @keyup.stop>
          <ElButton
            text
            size="small"
            :aria-label="t('common.edit')"
            @click="openEdit(col)"
          >
            <i class="pi pi-pencil" aria-hidden="true"></i>
          </ElButton>
          <ElButton
            text
            size="small"
            type="danger"
            :aria-label="t('common.delete')"
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
      <p>{{ t("collections.list.empty") }}</p>
    </div>

    <!-- Create / Edit Dialog -->
    <ElDialog
      v-model="dialogVisible"
      :title="
        editing ? t('collections.list.editTitle') : t('collections.list.new')
      "
      width="28rem"
    >
      <div class="flex flex-col gap-4">
        <div class="flex flex-col gap-1">
          <label for="col-type" class="text-sm font-medium">{{
            t("collections.list.collectionType")
          }}</label>
          <ElSelect
            id="col-type"
            v-model="form.collection_type_id"
            :placeholder="t('collections.list.choose')"
            :disabled="!!editing"
          >
            <ElOption
              v-for="ct in collectionTypes"
              :key="ct.id"
              :label="typeTitle(ct, lang)"
              :value="ct.id"
            />
          </ElSelect>
        </div>
        <div class="flex flex-col gap-1">
          <label for="col-name" class="text-sm font-medium">{{
            t("common.name")
          }}</label>
          <ElInput
            id="col-name"
            v-model="form.name"
            :placeholder="t('collections.list.namePlaceholder')"
          />
        </div>
        <div class="flex flex-col gap-1">
          <label for="col-desc" class="text-sm font-medium">{{
            t("common.description")
          }}</label>
          <ElInput
            id="col-desc"
            v-model="form.description"
            type="textarea"
            :rows="2"
          />
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
import { computed, onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { collectionApi } from "@/api/client";
import { usePagination } from "@/composables/usePagination";
import { useLang } from "@/composables/useLang";
import { useNotify } from "@/composables/useNotify";
import { useConfirmDialog } from "@/composables/useConfirmDialog";
import { useMetaTypes } from "@/stores/metaTypes";
import { typeTitle } from "@/lib/schema";
import AppSpinner from "@/components/common/AppSpinner.vue";
import type { Collection } from "@/api/types";

const router = useRouter();
const { t } = useI18n();
const { lang } = useLang();
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
    .catch((e) => notifyError(e, t("collections.typesUnavailable")));
  reload();
});

function reload() {
  pag.page.value = 1;
  pag
    .load(filterTypeId.value ? { collection_type_id: filterTypeId.value } : {})
    .catch((e) => notifyError(e, t("collections.loadFailed")));
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
      notifySuccess(t("collections.list.updated"));
    } else {
      await collectionApi.create(form.value);
      notifySuccess(t("collections.list.created"));
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
    message: t("collections.list.confirmDelete", { name: col.name }),
    confirmText: t("common.delete"),
    danger: true,
  });
  if (!ok) return;
  try {
    await collectionApi.delete(col.id);
    notifySuccess(t("collections.list.deleted"));
    reload();
  } catch (e) {
    notifyError(e);
  }
}
</script>
