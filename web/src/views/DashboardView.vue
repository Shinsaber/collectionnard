<template>
  <div>
    <h1 class="text-2xl font-bold text-gray-800 dark:text-gray-100 mb-6">
      Dashboard
    </h1>

    <AppSpinner v-if="loading" />

    <template v-else-if="stats">
      <div class="grid grid-cols-1 md:grid-cols-4 gap-4 mb-8">
        <div class="bg-white dark:bg-gray-800 rounded-lg shadow-xs p-5">
          <div class="text-sm text-gray-500 dark:text-gray-400 mb-1">
            Items au catalogue
          </div>
          <div class="text-3xl font-bold text-indigo-600 dark:text-indigo-400">
            {{ stats.total_items }}
          </div>
        </div>
        <div class="bg-white dark:bg-gray-800 rounded-lg shadow-xs p-5">
          <div class="text-sm text-gray-500 dark:text-gray-400 mb-1">
            Relations
          </div>
          <div
            class="text-3xl font-bold text-emerald-600 dark:text-emerald-400"
          >
            {{ stats.total_relations }}
          </div>
        </div>
        <div class="bg-white dark:bg-gray-800 rounded-lg shadow-xs p-5">
          <div class="text-sm text-gray-500 dark:text-gray-400 mb-1">
            Collections
          </div>
          <div class="text-3xl font-bold text-amber-600 dark:text-amber-400">
            {{ stats.total_collections }}
          </div>
        </div>
        <div class="bg-white dark:bg-gray-800 rounded-lg shadow-xs p-5">
          <div class="text-sm text-gray-500 dark:text-gray-400 mb-1">
            Types d'items
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
          Items par type
        </h2>
        <ElTable :data="stats.items_by_type" stripe>
          <ElTableColumn prop="type_name" label="Type" />
          <ElTableColumn prop="count" label="Nombre" width="140" />
        </ElTable>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from "vue";
import { statsApi } from "@/api/client";
import { useNotify } from "@/composables/useNotify";
import AppSpinner from "@/components/common/AppSpinner.vue";
import type { Stats } from "@/api/types";

const { notifyError } = useNotify();
const stats = ref<Stats | null>(null);
const loading = ref(true);

onMounted(async () => {
  try {
    stats.value = await statsApi.get();
  } catch (e) {
    notifyError(e, "Statistiques indisponibles");
  } finally {
    loading.value = false;
  }
});
</script>
