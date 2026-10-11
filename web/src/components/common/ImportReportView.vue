<template>
  <div data-test="import-report">
    <h3 class="font-semibold text-gray-800 dark:text-gray-100 mb-2">
      {{
        report.dry_run
          ? t("admin.importExport.reportSimulation")
          : t("admin.importExport.reportApplied")
      }}
    </h3>
    <ElTable :data="countRows" size="small" class="mb-4">
      <ElTableColumn :label="t('admin.importExport.entity')">
        <template #default="{ row }">
          {{ entityLabel((row as CountRow).entity) }}
        </template>
      </ElTableColumn>
      <ElTableColumn
        v-for="col in countCols"
        :key="col"
        :label="t(`admin.importExport.${col}`)"
        align="right"
        width="130"
      >
        <template #default="{ row }">
          {{ ((row as CountRow)[col] ?? 0).toLocaleString(lang) }}
        </template>
      </ElTableColumn>
    </ElTable>

    <template v-if="report.errors.length > 0">
      <ElAlert
        type="error"
        :closable="false"
        show-icon
        class="mb-3"
        :title="t('admin.importExport.hasErrors', report.errors.length)"
      />
      <ElTable :data="report.errors" size="small" data-test="import-errors">
        <ElTableColumn :label="t('admin.importExport.entity')" width="170">
          <template #default="{ row }">
            {{ entityLabel((row as ImportError).entity) }}
          </template>
        </ElTableColumn>
        <ElTableColumn :label="t('admin.importExport.position')" width="100">
          <template #default="{ row }">
            {{ position(row as ImportError) }}
          </template>
        </ElTableColumn>
        <ElTableColumn :label="t('admin.importExport.id')" width="300">
          <template #default="{ row }">
            <code class="text-xs">{{ (row as ImportError).id }}</code>
          </template>
        </ElTableColumn>
        <!-- Message technique de l'API, en anglais, affiché tel quel -->
        <ElTableColumn
          prop="message"
          :label="t('admin.importExport.message')"
        />
      </ElTable>
    </template>
    <ElAlert
      v-else-if="report.dry_run"
      type="success"
      :closable="false"
      show-icon
      :title="t('admin.importExport.simulationOk')"
    />
  </div>
</template>

<script setup lang="ts">
// Rapport d'import (catalogue ou collection) : compteurs par entité, puis erreurs.
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useLang } from "@/composables/useLang";
import type {
  ImportCounts,
  ImportEntity,
  ImportError,
  ImportReport,
} from "@/api/types";

type CountRow = ImportCounts & { entity: ImportEntity };
type CountCol = "created" | "updated" | "unchanged" | "removed";

const props = defineProps<{
  report: ImportReport;
  /** Entités affichées, dans l'ordre d'application du serveur. */
  entities: ImportEntity[];
  /** Ajoute la colonne « Retirés » (import de collection). */
  showRemoved?: boolean;
}>();

const { t } = useI18n();
const { lang } = useLang();

const countCols = computed<CountCol[]>(() =>
  props.showRemoved
    ? ["created", "updated", "unchanged", "removed"]
    : ["created", "updated", "unchanged"],
);

const countRows = computed<CountRow[]>(() =>
  props.entities.map((entity) => ({
    entity,
    created: 0,
    updated: 0,
    unchanged: 0,
    removed: 0,
    ...props.report.counts[entity],
  })),
);

function entityLabel(entity: ImportEntity): string {
  return t(`admin.importExport.entities.${entity}`);
}

function position(err: ImportError): string {
  return err.index < 0
    ? t("admin.importExport.wholeFile")
    : String(err.index + 1);
}
</script>
