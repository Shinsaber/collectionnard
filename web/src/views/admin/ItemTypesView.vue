<template>
  <div>
    <PageHeader
      :title="t('admin.itemTypes.title')"
      :meta="
        pag.loading.value
          ? undefined
          : t('admin.shared.typeCount', pag.total.value)
      "
    >
      <template #actions>
        <ElButton type="primary" @click="router.push(`${BASE}/new`)">
          <i class="pi pi-plus mr-2" aria-hidden="true"></i>
          {{ t("admin.shared.newType") }}
        </ElButton>
      </template>
    </PageHeader>

    <AdminTypeList
      :rows="rows"
      :loading="pag.loading.value"
      :base-path="BASE"
      :summary-label="t('admin.itemTypes.schemaFields')"
      @remove="confirmDelete"
    />
    <ElPagination
      class="mt-4 justify-end"
      hide-on-single-page
      layout="total, prev, pager, next"
      :total="pag.total.value"
      :page-size="pag.perPage.value"
      :current-page="pag.page.value"
      @current-change="onPageChange"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { itemTypeApi } from "@/api/client";
import { usePagination } from "@/composables/usePagination";
import { useNotify } from "@/composables/useNotify";
import { useConfirmDialog } from "@/composables/useConfirmDialog";
import { useMetaTypes } from "@/stores/metaTypes";
import { useLang } from "@/composables/useLang";
import { typeDescription, typeTitle } from "@/lib/schema";
import PageHeader from "@/components/common/PageHeader.vue";
import AdminTypeList, {
  type AdminTypeRow,
} from "@/components/admin/AdminTypeList.vue";
import { schemaSummary } from "./typeSummary";
import type { ItemType } from "@/api/types";

const BASE = "/admin/item-types";

const { t } = useI18n();
const router = useRouter();
const { notifyError, notifySuccess } = useNotify();
const { confirm } = useConfirmDialog();
const pag = usePagination(itemTypeApi.list);
const metaTypes = useMetaTypes();
const { lang } = useLang();

const rows = computed<AdminTypeRow[]>(() =>
  pag.data.value.map((it: ItemType) => ({
    id: it.id,
    title: typeTitle(it, lang.value),
    name: it.name,
    description: typeDescription(it, lang.value),
    ...schemaSummary(it.attributes_schema, lang.value, t),
  })),
);

onMounted(() => load());

function load() {
  pag.load().catch((e) => notifyError(e, t("admin.shared.loadFailed")));
}

function onPageChange(p: number) {
  pag.goToPage(p);
  load();
}

async function confirmDelete(row: AdminTypeRow) {
  const ok = await confirm({
    message: t("admin.shared.confirmDelete", { name: row.title }),
    confirmText: t("common.delete"),
    danger: true,
  });
  if (!ok) return;
  try {
    await itemTypeApi.delete(row.id);
    notifySuccess(t("admin.shared.deleted"));
    metaTypes.invalidate();
    load();
  } catch (e) {
    notifyError(e);
  }
}
</script>
