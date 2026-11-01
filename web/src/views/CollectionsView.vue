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

    <!-- Cartes : le titre est un lien étendu à toute la carte ; modifier et supprimer sont dans la page
         de la collection, la place libérée va au compteur d'items (retours v0.4.0). -->
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
          <!-- Compteur à droite du titre, en grand, avec son unité dessous. -->
          <div
            class="flex shrink-0 flex-col items-end"
            data-test="collection-count"
          >
            <span class="font-mono text-2xl font-bold leading-none text-ink">{{
              col.item_count
            }}</span>
            <span
              class="mt-1 font-mono text-[11px] uppercase tracking-wider text-muted"
              >{{ t("collections.list.unit", col.item_count) }}</span
            >
          </div>
        </div>
        <p v-if="col.description" class="line-clamp-2 text-sm text-muted">
          {{ col.description }}
        </p>
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

    <!-- Création (la modification et la suppression sont dans la page de la collection, retours v0.4.0) -->
    <ElDialog
      v-model="dialogVisible"
      :title="t('collections.list.new')"
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
          {{ t("common.create") }}
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
import { useNotify } from "@/composables/useNotify";
import { useMetaTypes } from "@/stores/metaTypes";
import { typeTitle } from "@/lib/schema";
import AppSpinner from "@/components/common/AppSpinner.vue";
import PageHeader from "@/components/common/PageHeader.vue";
import NavIcon from "@/components/layout/NavIcon.vue";
import CollectionImportDialog from "@/components/collection/CollectionImportDialog.vue";

const router = useRouter();
const importVisible = ref(false);
const { t } = useI18n();
const { lang } = useLang();
const { notifyError, notifySuccess } = useNotify();
const pag = usePagination(collectionApi.list);
const metaTypes = useMetaTypes();
const collectionTypes = computed(() => metaTypes.collectionTypes);
const filterTypeId = ref<string>();

const dialogVisible = ref(false);
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
  form.value = { collection_type_id: "", name: "", description: "" };
  dialogVisible.value = true;
}

async function save() {
  saving.value = true;
  try {
    await collectionApi.create(form.value);
    notifySuccess(t("collections.list.created"));
    dialogVisible.value = false;
    reload();
  } catch (e) {
    notifyError(e);
  } finally {
    saving.value = false;
  }
}
</script>
