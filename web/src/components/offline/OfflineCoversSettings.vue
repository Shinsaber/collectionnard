<template>
  <!-- Réglages des couvertures hors ligne (propres à l'appareil) et espace utilisé. -->
  <section class="mt-6 rounded-2xl bg-surface p-5 shadow-card">
    <h2 class="mb-1.5 text-lg font-extrabold tracking-tight">
      {{ t("offline.coversTitle") }}
    </h2>
    <p class="mb-4 text-sm text-muted">{{ t("offline.coversHelp") }}</p>
    <div class="flex flex-wrap items-center gap-x-6 gap-y-3">
      <ElSwitch
        v-model="settings.enabled"
        :active-text="t('offline.coversEnabled')"
        @change="apply"
      />
      <label class="flex items-center gap-2 text-sm">
        {{ t("offline.coversLimit") }}
        <ElSelect
          v-model="settings.limitMb"
          :disabled="!settings.enabled"
          class="w-28!"
          @change="apply"
        >
          <ElOption
            v-for="mb in COVER_LIMITS_MB"
            :key="mb"
            :label="formatBytes(mb * MB, lang)"
            :value="mb"
          />
        </ElSelect>
      </label>
    </div>
    <p class="mt-4 text-[13px] text-muted" role="status">
      {{
        t("offline.coversUsage", {
          count: usage.count,
          size: formatBytes(usage.size, lang),
          limit: formatBytes(settings.limitMb * MB, lang),
        })
      }}
      <template v-if="storage !== null">
        · {{ t("offline.storageUsage", { size: formatBytes(storage, lang) }) }}
      </template>
    </p>
    <ElButton class="mt-4" type="danger" plain @click="clearAll">
      <i class="pi pi-trash mr-2" aria-hidden="true"></i>
      {{ t("offline.clear") }}
    </ElButton>
  </section>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useLang } from "@/composables/useLang";
import { useNotify } from "@/composables/useNotify";
import { useConfirmDialog } from "@/composables/useConfirmDialog";
import { useOfflineStore } from "@/stores/offline";
import { formatBytes } from "@/lib/format";
import {
  COVER_LIMITS_MB,
  coversSize,
  enforceCoverSettings,
  loadCoverIndex,
  loadCoverSettings,
  saveCoverSettings,
} from "@/lib/offlineCovers";

const MB = 1024 * 1024;
const { t } = useI18n();
const { lang } = useLang();
const { notifySuccess } = useNotify();
const { confirm } = useConfirmDialog();
const offline = useOfflineStore();

const settings = reactive(loadCoverSettings());
const usage = reactive({ count: 0, size: 0 });
const storage = ref<number | null>(null);

async function refreshUsage() {
  const index = await loadCoverIndex();
  usage.count = Object.keys(index).length;
  usage.size = coversSize(index);
  const estimate = await navigator.storage?.estimate?.().catch(() => null);
  storage.value = estimate?.usage ?? null;
}

async function apply() {
  saveCoverSettings({ ...settings });
  await enforceCoverSettings();
  // Plafond relevé ou couvertures réactivées : compléter dès maintenant si le serveur répond.
  if (offline.reachable) await offline.sync(true).catch(() => undefined);
  await refreshUsage();
}

async function clearAll() {
  if (!(await confirm({ message: t("offline.clearConfirm"), danger: true })))
    return;
  await offline.clear();
  notifySuccess(t("offline.cleared"));
  await refreshUsage();
}

onMounted(refreshUsage);
defineExpose({ refreshUsage });
</script>
