<template>
  <div class="max-w-4xl">
    <PageHeader :title="t('admin.images.title')" />

    <!-- Purge des images inutilisées : simulation, puis suppression confirmée -->
    <section class="rounded-2xl bg-surface p-5 shadow-card sm:p-6">
      <h2 class="mb-1.5 text-lg font-extrabold tracking-tight">
        {{ t("admin.images.unusedTitle") }}
      </h2>
      <p class="mb-4 text-sm text-muted">
        {{ t("admin.images.unusedHelp", { hours: report?.grace_hours ?? 24 }) }}
      </p>
      <div class="flex flex-wrap gap-2">
        <ElButton :loading="busy === 'dry'" @click="simulate">
          <i class="pi pi-search mr-2" aria-hidden="true"></i>
          {{ t("admin.images.search") }}
        </ElButton>
        <ElButton
          v-if="report?.files.length"
          type="danger"
          :loading="busy === 'apply'"
          @click="apply"
        >
          <i class="pi pi-trash mr-2" aria-hidden="true"></i>
          {{ t("admin.images.delete", report.files.length) }}
        </ElButton>
      </div>

      <template v-if="report">
        <p class="mt-5 text-sm font-semibold" role="status">
          {{
            report.files.length
              ? t("admin.images.found", {
                  n: report.files.length,
                  size: formatBytes(report.total_size, lang),
                })
              : t("admin.images.none")
          }}
        </p>
        <p class="mt-1 text-[13px] text-muted">
          {{
            t("admin.images.kept", {
              referenced: report.referenced,
              recent: report.recent,
            })
          }}
        </p>
        <ul
          v-if="report.files.length"
          class="mt-4 grid grid-cols-[repeat(auto-fill,minmax(7rem,1fr))] gap-3"
        >
          <li
            v-for="f in report.files"
            :key="f.filename"
            class="flex flex-col gap-1"
          >
            <img
              :src="`/api/v1/uploads/${f.filename}`"
              :alt="f.filename"
              loading="lazy"
              class="aspect-square w-full rounded-lg bg-soft object-cover shadow-cover"
            />
            <span class="truncate font-mono text-[11px]" :title="f.filename">{{
              f.filename
            }}</span>
            <span class="text-[11px] text-muted">
              {{ formatBytes(f.size, lang) }} ·
              {{
                t("admin.images.modified", {
                  date: new Date(f.modified_at).toLocaleDateString(lang),
                })
              }}
            </span>
          </li>
        </ul>
      </template>
    </section>
  </div>
</template>

<script setup lang="ts">
// Images envoyées (admin) : purge de celles qu'aucun attribut ne référence plus, en deux temps
// (simulation, puis suppression confirmée) comme l'import du catalogue.
import { ref } from "vue";
import { useI18n } from "vue-i18n";
import { uploadApi } from "@/api/client";
import { useLang } from "@/composables/useLang";
import { useNotify } from "@/composables/useNotify";
import { useConfirmDialog } from "@/composables/useConfirmDialog";
import { formatBytes } from "@/lib/format";
import PageHeader from "@/components/common/PageHeader.vue";
import type { UploadPurgeReport } from "@/api/types";

const { t } = useI18n();
const { lang } = useLang();
const { notifyError, notifySuccess } = useNotify();
const { confirm } = useConfirmDialog();

const report = ref<UploadPurgeReport | null>(null);
const busy = ref<"dry" | "apply" | null>(null);

async function simulate() {
  busy.value = "dry";
  try {
    report.value = await uploadApi.purge(true);
  } catch (e) {
    notifyError(e, t("admin.images.failed"));
  } finally {
    busy.value = null;
  }
}

async function apply() {
  if (!report.value?.files.length) return;
  const n = report.value.files.length;
  const ok = await confirm({
    message: t(
      "admin.images.confirm",
      { n, size: formatBytes(report.value.total_size, lang.value) },
      n,
    ),
    confirmText: t("admin.images.delete", n),
    danger: true,
  });
  if (!ok) return;
  busy.value = "apply";
  try {
    const done = await uploadApi.purge(false);
    notifySuccess(t("admin.images.deleted", done.files.length));
    report.value = await uploadApi.purge(true);
  } catch (e) {
    notifyError(e);
  } finally {
    busy.value = null;
  }
}
</script>
