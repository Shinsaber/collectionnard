<template>
  <div class="max-w-4xl">
    <PageHeader :title="t('admin.importExport.title')" />

    <!-- Export -->
    <section class="mb-5 rounded-2xl bg-surface p-5 shadow-card sm:p-6">
      <h2 class="mb-1.5 text-lg font-extrabold tracking-tight">
        {{ t("admin.importExport.exportTitle") }}
      </h2>
      <p class="mb-4 text-sm text-muted">
        {{ t("admin.importExport.exportHelp") }}
      </p>
      <ElButton :loading="exporting" @click="doExport">
        <i v-if="!exporting" class="pi pi-download mr-2" aria-hidden="true"></i>
        {{ t("admin.importExport.exportButton") }}
      </ElButton>
    </section>

    <!-- Import -->
    <section class="rounded-2xl bg-surface p-5 shadow-card sm:p-6">
      <h2 class="mb-1.5 text-lg font-extrabold tracking-tight">
        {{ t("admin.importExport.importTitle") }}
      </h2>
      <p class="mb-4 text-sm text-muted">
        {{ t("admin.importExport.importHelp") }}
      </p>
      <label
        :for="fileId"
        class="mb-1.5 block font-mono text-[11px] uppercase tracking-wider text-muted"
        >{{ t("admin.importExport.file") }}</label
      >
      <input
        :id="fileId"
        type="file"
        accept=".json,application/json"
        class="mb-4 block w-full text-sm text-muted file:mr-3 file:h-10 file:rounded-[10px] file:border-0 file:bg-soft file:px-4 file:font-semibold file:text-ink"
        @change="onFileChange"
      />
      <div class="flex flex-wrap gap-2">
        <ElButton :disabled="!file" :loading="busy === 'dry'" @click="simulate">
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
      </div>

      <!-- Report -->
      <ImportReportView
        v-if="report"
        class="mt-6"
        :report="report"
        :entities="ENTITIES"
      />
    </section>
  </div>
</template>

<script setup lang="ts">
// Import / export du catalogue (admin) : export JSON complet, import en deux temps
// (simulation puis application, tout ou rien) avec rapport par entité.
import { computed, ref, useId } from "vue";
import { useI18n } from "vue-i18n";
import { importExportApi } from "@/api/client";
import { useNotify } from "@/composables/useNotify";
import { useConfirmDialog } from "@/composables/useConfirmDialog";
import { useMetaTypes } from "@/stores/metaTypes";
import ImportReportView from "@/components/common/ImportReportView.vue";
import PageHeader from "@/components/common/PageHeader.vue";
import type { ImportEntity, ImportReport } from "@/api/types";

/** Ordre d'application de l'import (pendant du serveur). */
const ENTITIES: ImportEntity[] = [
  "item_types",
  "collection_types",
  "relation_types",
  "items",
  "relations",
];

const { t } = useI18n();
const { notifyError, notifySuccess } = useNotify();
const { confirm } = useConfirmDialog();
const metaTypes = useMetaTypes();
const fileId = useId();

// ── Export ──────────────────────────────────────────────────────────
const exporting = ref(false);

async function doExport() {
  exporting.value = true;
  try {
    await importExportApi.exportCatalog();
  } catch (e) {
    notifyError(e, t("admin.importExport.exportFailed"));
  } finally {
    exporting.value = false;
  }
}

// ── Import ──────────────────────────────────────────────────────────
const file = ref<File | null>(null);
const report = ref<ImportReport | null>(null);
const busy = ref<"dry" | "apply" | null>(null);

/** Changer de fichier invalide le rapport (et donc le bouton « Importer »). */
function onFileChange(ev: Event) {
  file.value = (ev.target as HTMLInputElement).files?.[0] ?? null;
  report.value = null;
}

/** Importer n'est proposé qu'après une simulation sans erreur du fichier courant. */
const canApply = computed(
  () =>
    !!file.value && !!report.value?.dry_run && report.value.errors.length === 0,
);

async function run(dryRun: boolean): Promise<ImportReport | null> {
  if (!file.value) return null;
  busy.value = dryRun ? "dry" : "apply";
  try {
    report.value = await importExportApi.importCatalog(file.value, dryRun);
    return report.value;
  } catch (e) {
    notifyError(e, t("admin.importExport.importFailed"));
    return null;
  } finally {
    busy.value = null;
  }
}

function simulate() {
  return run(true);
}

async function apply() {
  if (!file.value) return;
  const ok = await confirm({
    title: t("admin.importExport.confirmTitle"),
    message: t("admin.importExport.confirmApply", { name: file.value.name }),
    confirmText: t("admin.importExport.apply"),
    danger: true,
  });
  if (!ok) return;
  const r = await run(false);
  if (r?.applied) {
    notifySuccess(t("admin.importExport.imported"));
    metaTypes.invalidate();
  }
}
</script>
