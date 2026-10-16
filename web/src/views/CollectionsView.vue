<template>
  <div>
    <PageHeader
      :title="t('collections.list.title')"
      :meta="
        pag.loading.value
          ? undefined
          : t('collections.list.count', pag.total.value)
      "
    >
      <template #actions>
        <ElButton @click="importVisible = true">
          <i class="pi pi-upload mr-2" aria-hidden="true"></i>
          {{ t("collections.import.button") }}
        </ElButton>
        <ElButton type="primary" @click="openCreate">
          <i class="pi pi-plus mr-2" aria-hidden="true"></i>
          {{ t("collections.list.new") }}
        </ElButton>
      </template>
    </PageHeader>

    <ElSelect
      v-model="filterTypeId"
      :placeholder="t('collections.list.allTypes')"
      clearable
      size="large"
      class="mb-5 w-full! sm:w-64!"
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

    <AppSpinner v-if="pag.loading.value" />

    <!-- Cartes : le titre est un lien étendu à toute la carte ; les actions restent cliquables. -->
    <ul
      v-else-if="pag.data.value.length"
      class="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3"
    >
      <li
        v-for="col in pag.data.value"
        :key="col.id"
        class="relative flex flex-col gap-3 rounded-2xl bg-surface p-4.5 shadow-card transition-shadow hover:shadow-raised"
      >
        <div class="flex items-start gap-3.5">
          <span
            class="flex size-10 shrink-0 items-center justify-center rounded-[10px] bg-tint text-tint-ink"
          >
            <NavIcon name="collections" />
          </span>
          <div class="flex min-w-0 flex-1 flex-col">
            <RouterLink
              :to="`/collections/${col.id}`"
              class="truncate text-base font-bold text-ink after:absolute after:inset-0 after:rounded-2xl"
              >{{ col.name }}</RouterLink
            >
            <span class="text-[13px] text-muted">{{
              col.collection_type ? typeTitle(col.collection_type, lang) : "?"
            }}</span>
          </div>
          <span class="shrink-0 font-mono text-[13px]">{{
            col.item_count
          }}</span>
        </div>
        <p v-if="col.description" class="line-clamp-2 text-sm text-muted">
          {{ col.description }}
        </p>
        <div class="relative z-10 mt-auto flex justify-end gap-1">
          <ElButton
            text
            circle
            :aria-label="`${t('common.edit')} : ${col.name}`"
            @click="openEdit(col)"
          >
            <i class="pi pi-pencil" aria-hidden="true"></i>
          </ElButton>
          <ElButton
            text
            circle
            type="danger"
            :aria-label="`${t('common.delete')} : ${col.name}`"
            @click="confirmDelete(col)"
          >
            <i class="pi pi-trash" aria-hidden="true"></i>
          </ElButton>
        </div>
      </li>
    </ul>

    <div
      v-else
      class="flex flex-col items-center gap-3 rounded-2xl bg-surface px-6 py-12 text-center text-muted shadow-card"
    >
      <NavIcon name="collections" :size="32" />
      <p>{{ t("collections.list.empty") }}</p>
      <ElButton type="primary" @click="openCreate">
        <i class="pi pi-plus mr-2" aria-hidden="true"></i>
        {{ t("collections.list.new") }}
      </ElButton>
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
        <ElButton
          type="primary"
          :loading="saving"
          :disabled="!!editing && !dirty"
          @click="save"
        >
          <i class="pi pi-check mr-2" aria-hidden="true"></i>
          {{ editing ? t("common.save") : t("common.create") }}
        </ElButton>
      </template>
    </ElDialog>

    <CollectionImportDialog
      v-model:visible="importVisible"
      @imported="(id) => router.push(`/collections/${id}`)"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { collectionApi } from "@/api/client";
import { usePagination } from "@/composables/usePagination";
import { useLang } from "@/composables/useLang";
import { useDirty } from "@/composables/useDirty";
import { useNotify } from "@/composables/useNotify";
import { useConfirmDialog } from "@/composables/useConfirmDialog";
import { useMetaTypes } from "@/stores/metaTypes";
import { typeTitle } from "@/lib/schema";
import AppSpinner from "@/components/common/AppSpinner.vue";
import PageHeader from "@/components/common/PageHeader.vue";
import NavIcon from "@/components/layout/NavIcon.vue";
import CollectionImportDialog from "@/components/collection/CollectionImportDialog.vue";
import type { Collection } from "@/api/types";

const router = useRouter();
const importVisible = ref(false);
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
// En modification, « Enregistrer » reste grisé tant que rien n'a changé.
const { dirty, markClean } = useDirty(() => form.value);

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
  markClean();
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
