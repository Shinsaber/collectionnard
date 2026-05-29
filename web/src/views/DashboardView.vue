<template>
  <div>
    <h1 class="text-2xl font-bold text-gray-800 dark:text-gray-100 mb-6">
      Dashboard
    </h1>

    <div v-if="loading" class="flex justify-center py-12">
      <ProgressSpinner />
    </div>

    <template v-else-if="stats">
      <div class="grid grid-cols-1 md:grid-cols-4 gap-4 mb-8">
        <div class="bg-white dark:bg-gray-800 rounded-lg shadow p-5">
          <div class="text-sm text-gray-500 dark:text-gray-400 mb-1">
            Items au catalogue
          </div>
          <div class="text-3xl font-bold text-indigo-600 dark:text-indigo-400">
            {{ stats.total_items }}
          </div>
        </div>
        <div class="bg-white dark:bg-gray-800 rounded-lg shadow p-5">
          <div class="text-sm text-gray-500 dark:text-gray-400 mb-1">
            Relations
          </div>
          <div
            class="text-3xl font-bold text-emerald-600 dark:text-emerald-400"
          >
            {{ stats.total_relations }}
          </div>
        </div>
        <div class="bg-white dark:bg-gray-800 rounded-lg shadow p-5">
          <div class="text-sm text-gray-500 dark:text-gray-400 mb-1">
            Collections
          </div>
          <div class="text-3xl font-bold text-amber-600 dark:text-amber-400">
            {{ stats.total_collections }}
          </div>
        </div>
        <div class="bg-white dark:bg-gray-800 rounded-lg shadow p-5">
          <div class="text-sm text-gray-500 dark:text-gray-400 mb-1">
            Types d'items
          </div>
          <div class="text-3xl font-bold text-rose-600 dark:text-rose-400">
            {{ stats.items_by_type?.length ?? 0 }}
          </div>
        </div>
      </div>

      <div
        class="bg-white dark:bg-gray-800 rounded-lg shadow p-6"
        v-if="stats.items_by_type?.length"
      >
        <h2 class="text-lg font-semibold text-gray-700 dark:text-gray-200 mb-4">
          Items par type
        </h2>
        <DataTable :value="stats.items_by_type" stripedRows>
          <Column field="type_name" header="Type" />
          <Column field="count" header="Nombre" />
        </DataTable>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from "vue";
import DataTable from "primevue/datatable";
import Column from "primevue/column";
import ProgressSpinner from "primevue/progressspinner";
import { statsApi } from "@/api/client";
import type { Stats } from "@/api/types";

const stats = ref<Stats | null>(null);
const loading = ref(true);

onMounted(async () => {
  try {
    stats.value = await statsApi.get();
  } finally {
    loading.value = false;
  }
});
</script>
