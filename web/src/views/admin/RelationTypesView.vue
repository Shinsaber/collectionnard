<template>
  <div>
    <PageHeader
      :title="t('admin.relationTypes.title')"
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
      :summary-label="t('admin.relationTypes.sourcesTargets')"
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
import { relationTypeApi } from "@/api/client";
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
import type { ItemType, RelationType } from "@/api/types";

const BASE = "/admin/relation-types";

const { t } = useI18n();
const router = useRouter();
const { notifyError, notifySuccess } = useNotify();
const { confirm } = useConfirmDialog();
const pag = usePagination(relationTypeApi.list);
const metaTypes = useMetaTypes();
const { lang } = useLang();

const rows = computed<AdminTypeRow[]>(() =>
  pag.data.value.map((it: RelationType) => ({
    id: it.id,
    title: typeTitle(it, lang.value),
    name: it.name,
    description: typeDescription(it, lang.value),
    countLabel: `${typeNames(it.source_item_types)} → ${typeNames(it.target_item_types)}`,
    summary: "",
  })),
);

// ItemTypes complets (avec schéma, donc x-title) indexés par id, pour les extrémités.
const itemTypesById = computed(
  () => new Map(metaTypes.itemTypes.map((it) => [it.id, it])),
);

function typeNames(types?: ItemType[] | null): string {
  return (
    (types ?? [])
      .map((it) => typeTitle(itemTypesById.value.get(it.id) ?? it, lang.value))
      .join(", ") || "?"
  );
}

onMounted(() => {
  metaTypes
    .loadItemTypes()
    .catch((e) => notifyError(e, t("admin.relationTypes.itemTypesFailed")));
  load();
});

function load() {
  pag.load().catch((e) => notifyError(e, t("admin.shared.loadFailed")));
}

function onPageChange(p: number) {
  pag.goToPage(p);
  load();
}

async function confirmDelete(row: AdminTypeRow) {
  const ok = await confirm({
    message: t("admin.relationTypes.confirmDelete", { name: row.title }),
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
