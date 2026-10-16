<template>
  <div class="max-w-4xl">
    <PageHeader :title="t('admin.domains.title')">
      <template #subtitle>
        <p class="text-[15px] text-muted">{{ t("admin.domains.help") }}</p>
      </template>
    </PageHeader>

    <AppSpinner v-if="loading" />
    <div v-else class="flex flex-col gap-4">
      <section
        v-for="d in list"
        :key="d.id"
        class="rounded-2xl bg-surface p-5 shadow-card"
      >
        <div class="flex flex-wrap items-start justify-between gap-3">
          <div class="flex min-w-0 flex-col gap-1">
            <h2 class="text-lg font-extrabold tracking-tight">
              {{ text(d.title) || d.id }}
            </h2>
            <p class="text-sm text-muted">{{ text(d.description) }}</p>
            <p class="font-mono text-[11px] uppercase text-muted">
              {{ t("admin.domains.version", { v: d.package_version }) }}
              <template v-if="d.installed_version">
                ·
                {{
                  t("admin.domains.installedVersion", {
                    v: d.installed_version,
                  })
                }}</template
              >
              ·
              {{
                t("admin.domains.counts", {
                  items: d.item_types,
                  relations: d.relation_types,
                  collections: d.collection_types,
                })
              }}
            </p>
          </div>
          <div class="flex flex-wrap gap-2">
            <span
              class="rounded-full px-2.5 py-0.5 text-xs font-semibold"
              :class="
                d.status === 'up_to_date'
                  ? 'bg-tint text-tint-ink'
                  : d.status === 'update_available'
                    ? 'bg-accent text-accent-ink'
                    : 'bg-soft text-muted'
              "
              >{{ t(`admin.domains.status.${d.status}`) }}</span
            >
            <span
              v-if="d.modified_locally"
              class="rounded-full bg-soft px-2.5 py-0.5 text-xs font-semibold text-muted"
              >{{ t("admin.domains.modifiedLocally") }}</span
            >
          </div>
        </div>
        <div class="mt-4 flex flex-wrap gap-2">
          <ElButton :loading="busy === `dry:${d.id}`" @click="simulate(d)">
            <i class="pi pi-search mr-2" aria-hidden="true"></i>
            {{ t("admin.domains.simulate") }}
          </ElButton>
          <ElButton
            type="primary"
            :loading="busy === `apply:${d.id}`"
            @click="apply(d)"
          >
            <i class="pi pi-download mr-2" aria-hidden="true"></i>
            {{ actionLabel(d) }}
          </ElButton>
        </div>
        <DomainReport v-if="reports[d.id]" :report="reports[d.id]!" />
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
// Domaines installables (lot P3-0) : paquets de types embarqués dans le serveur, installés ou mis
// à jour par fusion (les modifications locales sont gardées) ; simulation, puis application confirmée.
import { onMounted, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import { domainApi } from "@/api/client";
import { useLang } from "@/composables/useLang";
import { useNotify } from "@/composables/useNotify";
import { useConfirmDialog } from "@/composables/useConfirmDialog";
import { useMetaTypes } from "@/stores/metaTypes";
import AppSpinner from "@/components/common/AppSpinner.vue";
import PageHeader from "@/components/common/PageHeader.vue";
import DomainReport from "@/components/admin/DomainReport.vue";
import type { DomainInfo, DomainInstallReport } from "@/api/types";

const { t } = useI18n();
const { lang } = useLang();
const { notifyError, notifySuccess } = useNotify();
const { confirm } = useConfirmDialog();
const metaTypes = useMetaTypes();

const list = ref<DomainInfo[]>([]);
const loading = ref(true);
const busy = ref<string | null>(null);
const reports = reactive<Record<string, DomainInstallReport>>({});

/** Texte du paquet dans la langue courante (repli : français, anglais). */
function text(map: Record<string, string>): string {
  return map[lang.value] || map.fr || map.en || "";
}

function actionLabel(d: DomainInfo): string {
  if (d.status === "not_installed") return t("admin.domains.install");
  if (d.status === "update_available") return t("admin.domains.update");
  return t("admin.domains.reinstall");
}

async function load() {
  try {
    list.value = await domainApi.list();
  } catch (e) {
    notifyError(e, t("admin.domains.failed"));
  } finally {
    loading.value = false;
  }
}

async function simulate(d: DomainInfo) {
  busy.value = `dry:${d.id}`;
  try {
    reports[d.id] = await domainApi.install(d.id, true);
  } catch (e) {
    notifyError(e);
  } finally {
    busy.value = null;
  }
}

async function apply(d: DomainInfo) {
  const name = text(d.title) || d.id;
  if (!(await confirm({ message: t("admin.domains.confirm", { name }) })))
    return;
  busy.value = `apply:${d.id}`;
  try {
    const rep = await domainApi.install(d.id, false);
    reports[d.id] = rep;
    if (rep.applied) {
      notifySuccess(t("admin.domains.done", { name }));
      metaTypes.invalidate();
      await load();
    }
  } catch (e) {
    notifyError(e);
  } finally {
    busy.value = null;
  }
}

onMounted(load);
</script>
