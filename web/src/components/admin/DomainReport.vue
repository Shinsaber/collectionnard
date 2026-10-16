<template>
  <!-- Rapport d'installation d'un domaine : un type par ligne, avec ce qui lui arrive. -->
  <div class="mt-4 flex flex-col gap-3">
    <p class="text-sm font-semibold" role="status">
      {{
        report.from_version
          ? t("admin.domains.reportTitle", {
              from: report.from_version,
              to: report.to_version,
            })
          : t("admin.domains.reportTitleNew", { to: report.to_version })
      }}
      ·
      <span
        v-for="(n, outcome) in counts"
        :key="outcome"
        class="mr-2 font-normal text-muted"
        >{{ t(`admin.domains.outcome.${outcome}`) }} : {{ n }}</span
      >
    </p>
    <div
      v-if="report.errors.length"
      class="rounded-xl bg-soft p-3 text-sm text-danger"
    >
      <p class="mb-1 font-semibold">{{ t("admin.domains.errorsTitle") }}</p>
      <p v-for="(e, i) in report.errors" :key="i">{{ e.message }}</p>
    </div>
    <ul class="flex flex-col divide-y divide-line rounded-xl bg-soft">
      <li
        v-for="e in shown"
        :key="e.entity + e.id"
        class="flex flex-col gap-0.5 px-3 py-2 text-sm"
      >
        <div class="flex items-center gap-2">
          <span
            class="rounded-full px-2 py-0.5 font-mono text-[11px] uppercase"
            :class="badgeClass(e.outcome)"
            >{{ t(`admin.domains.outcome.${e.outcome}`) }}</span
          >
          <span class="font-mono text-[13px]">{{ e.name }}</span>
        </div>
        <span v-if="e.conflicts?.length" class="text-xs text-muted">{{
          t("admin.domains.conflicts", { paths: e.conflicts.join(", ") })
        }}</span>
        <span v-if="e.missing_types?.length" class="text-xs text-muted">{{
          t("admin.domains.missingTypes", { n: e.missing_types.length })
        }}</span>
      </li>
    </ul>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import type { DomainInstallReport, DomainOutcome } from "@/api/types";

const props = defineProps<{ report: DomainInstallReport }>();
const { t } = useI18n();

const counts = computed(() => {
  const out: Partial<Record<DomainOutcome, number>> = {};
  for (const e of props.report.entries)
    out[e.outcome] = (out[e.outcome] ?? 0) + 1;
  return out;
});

/** Les types inchangés sans remarque ne sont pas listés (seulement comptés). */
const shown = computed(() =>
  props.report.entries.filter(
    (e) =>
      e.outcome !== "unchanged" ||
      e.conflicts?.length ||
      e.missing_types?.length,
  ),
);

function badgeClass(outcome: DomainOutcome): string {
  switch (outcome) {
    case "created":
    case "updated":
      return "bg-tint text-tint-ink";
    case "merged":
      return "bg-accent text-accent-ink";
    default:
      return "bg-surface text-muted";
  }
}
</script>
