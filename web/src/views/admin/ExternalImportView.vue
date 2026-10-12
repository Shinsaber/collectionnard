<template>
  <div class="max-w-5xl">
    <h1 class="text-2xl font-bold text-gray-800 dark:text-gray-100 mb-2">
      {{ t("admin.external.title") }}
    </h1>
    <p class="text-sm text-gray-500 dark:text-gray-400 mb-4">
      {{ t("admin.external.intro") }}
    </p>
    <!-- Import appliqué : active = 4 pour que la dernière étape passe elle aussi au vert. -->
    <ElSteps
      :active="step === 3 ? 4 : step"
      finish-status="success"
      class="mb-6"
      data-test="steps"
    >
      <ElStep :title="t('admin.external.steps.search')" />
      <ElStep :title="t('admin.external.steps.selection')" />
      <ElStep :title="t('admin.external.steps.simulation')" />
      <ElStep :title="t('admin.external.steps.done')" />
    </ElSteps>

    <!-- 1. Recherche -->
    <div v-if="step === 0" class="space-y-4">
      <ExternalWorkSearch v-model="work" />
      <ExternalEditionSearch v-model="editions" :suggestion="work?.label" />
      <div class="flex items-center gap-3">
        <ElButton
          type="primary"
          :disabled="!hasSelection"
          :loading="busy"
          @click="loadPreview"
        >
          <i v-if="!busy" class="pi pi-eye mr-2" aria-hidden="true"></i>
          {{ t("admin.external.preview") }}
        </ElButton>
        <span
          v-if="!hasSelection"
          class="text-sm text-gray-500 dark:text-gray-400"
          >{{ t("admin.external.needSelection") }}</span
        >
      </div>
    </div>

    <!-- 2. Sélection -->
    <div v-else-if="step === 1 && plan">
      <ExternalPlanView
        v-model="selected"
        v-model:overrides="overrides"
        :plan="plan"
      />
      <div class="flex gap-2 mt-4">
        <ElButton @click="step = 0">{{ t("admin.external.back") }}</ElButton>
        <ElButton
          type="primary"
          :disabled="selected.length === 0"
          :loading="busy"
          @click="simulate"
        >
          <i v-if="!busy" class="pi pi-search mr-2" aria-hidden="true"></i>
          {{ t("admin.external.simulate") }}
        </ElButton>
      </div>
    </div>

    <!-- 3. Simulation / 4. Terminé -->
    <div v-else-if="report">
      <ImportReportView :report="report" :entities="ENTITIES" />
      <div class="flex gap-2 mt-4">
        <template v-if="step === 2">
          <ElButton @click="step = 1">{{ t("admin.external.back") }}</ElButton>
          <ElButton
            v-if="report.errors.length === 0"
            type="danger"
            :loading="busy"
            @click="apply"
          >
            <i v-if="!busy" class="pi pi-upload mr-2" aria-hidden="true"></i>
            {{ t("admin.external.apply") }}
          </ElButton>
        </template>
        <template v-else>
          <RouterLink v-if="workItemId" :to="`/items/${workItemId}`">
            <ElButton type="primary">{{
              t("admin.external.openWork")
            }}</ElButton>
          </RouterLink>
          <ElButton @click="restart">{{
            t("admin.external.restart")
          }}</ElButton>
        </template>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
// Import depuis les sources externes (admin, connecteur BD / Manga) : recherche d'une œuvre
// (Wikidata) et d'éditions (BnF), aperçu à cocher, simulation, import (tout ou rien).
import { computed, onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { externalApi } from "@/api/client";
import { useNotify } from "@/composables/useNotify";
import { useConfirmDialog } from "@/composables/useConfirmDialog";
import { useMetaTypes } from "@/stores/metaTypes";
import {
  defaultSelection,
  selectedOverrides,
  writtenCount,
  type Overrides,
} from "@/lib/externalPlan";
import ExternalWorkSearch from "@/components/external/ExternalWorkSearch.vue";
import ExternalEditionSearch from "@/components/external/ExternalEditionSearch.vue";
import ExternalPlanView from "@/components/external/ExternalPlanView.vue";
import ImportReportView from "@/components/common/ImportReportView.vue";
import type {
  ExternalEditionHit,
  ExternalImportReport,
  ExternalImportRequest,
  ExternalPlan,
  ExternalWorkHit,
  ImportEntity,
} from "@/api/types";

/** L'import externe n'écrit que des items et des relations. */
const ENTITIES: ImportEntity[] = ["items", "relations"];

const { t } = useI18n();
const { notifyError, notifySuccess } = useNotify();
const { confirm } = useConfirmDialog();
const metaTypes = useMetaTypes();

const step = ref(0);
const work = ref<ExternalWorkHit | null>(null);
const editions = ref<ExternalEditionHit[]>([]);
const plan = ref<ExternalPlan | null>(null);
const selected = ref<string[]>([]);
const overrides = ref<Overrides>({});
const report = ref<ExternalImportReport | null>(null);
const busy = ref(false);

onMounted(() => {
  // Types et relations : libellés des nœuds et des rôles.
  Promise.all([metaTypes.loadItemTypes(), metaTypes.loadRelationTypes()]).catch(
    (e: unknown) => notifyError(e),
  );
});

const hasSelection = computed(() => !!work.value || editions.value.length > 0);

function request(withSelection: boolean): ExternalImportRequest {
  return {
    work: work.value?.wikidata_id,
    editions: editions.value.map((e) => e.bnf_ark),
    ...(withSelection
      ? {
          selected: selected.value,
          overrides: selectedOverrides(overrides.value, selected.value),
        }
      : {}),
  };
}

async function loadPreview() {
  busy.value = true;
  try {
    plan.value = await externalApi.preview(request(false));
    selected.value = defaultSelection(plan.value);
    overrides.value = {};
    report.value = null;
    step.value = 1;
  } catch (e) {
    notifyError(e, t("admin.external.previewFailed"));
  } finally {
    busy.value = false;
  }
}

async function run(dryRun: boolean): Promise<ExternalImportReport | null> {
  busy.value = true;
  try {
    report.value = await externalApi.importSelection(request(true), dryRun);
    return report.value;
  } catch (e) {
    notifyError(e, t("admin.external.importFailed"));
    return null;
  } finally {
    busy.value = false;
  }
}

async function simulate() {
  if (await run(true)) step.value = 2;
}

async function apply() {
  const ok = await confirm({
    title: t("admin.external.confirmTitle"),
    message: t("admin.external.confirmApply", {
      written: plan.value
        ? writtenCount(plan.value, selected.value, overrides.value)
        : 0,
    }),
    confirmText: t("admin.external.apply"),
    danger: true,
  });
  if (!ok) return;
  const r = await run(false);
  if (r?.applied) {
    notifySuccess(t("admin.external.imported"));
    step.value = 3;
  }
}

/** Œuvre importée (ou complétée) : lien vers sa fiche une fois l'import appliqué. */
const workItemId = computed(() => {
  const key = plan.value?.nodes.find((n) => n.role === "work")?.key;
  return key && report.value?.applied ? report.value.item_ids[key] : undefined;
});

function restart() {
  step.value = 0;
  work.value = null;
  editions.value = [];
  plan.value = null;
  selected.value = [];
  overrides.value = {};
  report.value = null;
}
</script>
