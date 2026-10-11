<template>
  <div>
    <h1 class="text-2xl font-bold text-gray-800 dark:text-gray-100 mb-6">
      {{ t("dashboard.title") }}
    </h1>

    <AppSpinner v-if="loading" />

    <template v-else-if="stats">
      <div class="grid grid-cols-1 md:grid-cols-4 gap-4 mb-8">
        <div class="bg-white dark:bg-gray-800 rounded-lg shadow-xs p-5">
          <div class="text-sm text-gray-500 dark:text-gray-400 mb-1">
            {{ t("dashboard.totalItems") }}
          </div>
          <div class="text-3xl font-bold text-indigo-600 dark:text-indigo-400">
            {{ stats.total_items }}
          </div>
        </div>
        <div class="bg-white dark:bg-gray-800 rounded-lg shadow-xs p-5">
          <div class="text-sm text-gray-500 dark:text-gray-400 mb-1">
            {{ t("dashboard.totalRelations") }}
          </div>
          <div
            class="text-3xl font-bold text-emerald-600 dark:text-emerald-400"
          >
            {{ stats.total_relations }}
          </div>
        </div>
        <div class="bg-white dark:bg-gray-800 rounded-lg shadow-xs p-5">
          <div class="text-sm text-gray-500 dark:text-gray-400 mb-1">
            {{ t("dashboard.totalCollections") }}
          </div>
          <div class="text-3xl font-bold text-amber-600 dark:text-amber-400">
            {{ stats.total_collections }}
          </div>
        </div>
        <div class="bg-white dark:bg-gray-800 rounded-lg shadow-xs p-5">
          <div class="text-sm text-gray-500 dark:text-gray-400 mb-1">
            {{ t("dashboard.itemTypes") }}
          </div>
          <div class="text-3xl font-bold text-rose-600 dark:text-rose-400">
            {{ stats.items_by_type?.length ?? 0 }}
          </div>
        </div>
      </div>

      <div
        v-if="stats.items_by_type?.length"
        class="bg-white dark:bg-gray-800 rounded-lg shadow-xs p-6"
      >
        <h2 class="text-lg font-semibold text-gray-700 dark:text-gray-200 mb-4">
          {{ t("dashboard.itemsByType") }}
        </h2>
        <ElTable :data="stats.items_by_type" stripe>
          <ElTableColumn :label="t('common.type')">
            <template #default="{ row }">{{
              typeName((row as ItemsByType).type_name)
            }}</template>
          </ElTableColumn>
          <ElTableColumn
            prop="count"
            :label="t('dashboard.count')"
            width="140"
          />
        </ElTable>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from "vue";
import { useI18n } from "vue-i18n";
import { statsApi } from "@/api/client";
import { useLang } from "@/composables/useLang";
import { useNotify } from "@/composables/useNotify";
import { useMetaTypes } from "@/stores/metaTypes";
import { typeTitle } from "@/lib/schema";
import AppSpinner from "@/components/common/AppSpinner.vue";
import type { ItemsByType, Stats } from "@/api/types";

const { t } = useI18n();
const { lang } = useLang();
const { notifyError } = useNotify();
const metaTypes = useMetaTypes();
const stats = ref<Stats | null>(null);
const loading = ref(true);

/** Nom affiché d'un ItemType connu par son seul nom (stats) ; repli sur le nom brut. */
function typeName(name: string): string {
  const type = metaTypes.itemTypes.find((it) => it.name === name);
  return type ? typeTitle(type, lang.value) : name;
}

onMounted(async () => {
  // Méta-types en parallèle : tant qu'ils manquent, les noms bruts s'affichent.
  metaTypes.loadItemTypes().catch((e) => notifyError(e));
  try {
    stats.value = await statsApi.get();
  } catch (e) {
    notifyError(e, t("dashboard.statsUnavailable"));
  } finally {
    loading.value = false;
  }
});
</script>
