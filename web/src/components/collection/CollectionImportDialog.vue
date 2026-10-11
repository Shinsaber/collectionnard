<template>
  <ElDialog
    :model-value="visible"
    :title="t('collections.import.title')"
    width="760px"
    @update:model-value="emit('update:visible', $event)"
    @closed="reset"
  >
    <p class="text-sm text-gray-500 dark:text-gray-400 mb-3">
      {{ t("collections.import.help") }}
    </p>
    <label
      :for="fileId"
      class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
      >{{ t("collections.import.file") }}</label
    >
    <input
      :id="fileId"
      :key="inputKey"
      type="file"
      accept=".json,application/json"
      class="block mb-3 text-sm text-gray-700 dark:text-gray-300"
      @change="onFileChange"
    />

    <ImportReportView
      v-if="report"
      class="mt-4"
      :report="report"
      :entities="ENTITIES"
      show-removed
    />

    <template #footer>
      <ElButton @click="emit('update:visible', false)">{{
        t("common.close")
      }}</ElButton>
      <ElButton :disabled="!file" :loading="busy === 'dry'" @click="run(true)">
        <i class="pi pi-search mr-2" aria-hidden="true"></i>
        {{ t("admin.importExport.simulate") }}
      </ElButton>
      <ElButton
        v-if="canApply"
        type="danger"
        :loading="busy === 'apply'"
        @click="apply"
      >
        <i class="pi pi-upload mr-2" aria-hidden="true"></i>
        {{ t("admin.importExport.apply") }}
      </ElButton>
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
// Import d'un export de collection (JSON) dans le compte courant : simulation, puis
// application après confirmation. Une collection existante du compte devient exactement
// le contenu du fichier (items absents retirés) ; sinon une collection est créée.
import { computed, ref, useId } from "vue";
import { useI18n } from "vue-i18n";
import { importExportApi } from "@/api/client";
import type { ImportEntity, ImportReport } from "@/api/types";
import { useConfirmDialog } from "@/composables/useConfirmDialog";
import { useNotify } from "@/composables/useNotify";
import ImportReportView from "@/components/common/ImportReportView.vue";

defineProps<{ visible: boolean }>();
const emit = defineEmits<{
  "update:visible": [value: boolean];
  /** Import appliqué : id de la collection créée ou mise à jour. */
  imported: [collectionId: string];
}>();

const ENTITIES: ImportEntity[] = ["collection", "items"];

const { t } = useI18n();
const { notifyError, notifySuccess } = useNotify();
const { confirm } = useConfirmDialog();
const fileId = useId();

const file = ref<File | null>(null);
const report = ref<ImportReport | null>(null);
const busy = ref<"dry" | "apply" | null>(null);
// Change de clé à la fermeture pour vider l'input fichier.
const inputKey = ref(0);

function onFileChange(ev: Event) {
  file.value = (ev.target as HTMLInputElement).files?.[0] ?? null;
  report.value = null;
}

function reset() {
  file.value = null;
  report.value = null;
  inputKey.value++;
}

const canApply = computed(
  () =>
    !!file.value && !!report.value?.dry_run && report.value.errors.length === 0,
);

async function run(dryRun: boolean): Promise<ImportReport | null> {
  if (!file.value) return null;
  busy.value = dryRun ? "dry" : "apply";
  try {
    report.value = await importExportApi.importCollection(file.value, dryRun);
    return report.value;
  } catch (e) {
    notifyError(e, t("admin.importExport.importFailed"));
    return null;
  } finally {
    busy.value = null;
  }
}

async function apply() {
  if (!file.value) return;
  const removed = report.value?.counts.items?.removed ?? 0;
  const ok = await confirm({
    title: t("collections.import.title"),
    message:
      removed > 0
        ? t("collections.import.confirmWithRemoval", { n: removed }, removed)
        : t("collections.import.confirm", { name: file.value.name }),
    confirmText: t("admin.importExport.apply"),
    danger: true,
  });
  if (!ok) return;
  const r = await run(false);
  if (r?.applied && r.collection_id) {
    notifySuccess(t("collections.import.done"));
    emit("imported", r.collection_id);
    emit("update:visible", false);
  }
}
</script>
