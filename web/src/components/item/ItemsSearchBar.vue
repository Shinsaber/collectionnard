<template>
  <!-- Recherche, scanner et tri du catalogue (ItemsView) : l'état vit dans l'URL de la page. -->
  <div class="mb-4 flex flex-wrap items-center gap-2.5">
    <ElInput
      :model-value="modelValue"
      :placeholder="t('items.list.searchPlaceholder')"
      clearable
      size="large"
      class="min-w-0 flex-[1_1_280px]"
      :aria-label="t('common.search')"
      @update:model-value="emit('update:modelValue', $event)"
      @keyup.enter="emit('apply')"
      @clear="emit('apply')"
    >
      <template #prefix>
        <i class="pi pi-search" aria-hidden="true"></i>
      </template>
    </ElInput>
    <!-- Code-barres (lot M2) : les identifiants sont indexés par la recherche plein texte. -->
    <ScanButton size="large" @detected="emit('scanned', $event)" />
    <ElSelect
      :model-value="sort"
      size="large"
      class="w-full! sm:w-64!"
      :aria-label="t('items.search.sortBy')"
      @update:model-value="emit('update:sort', $event || undefined)"
    >
      <ElOption
        v-for="o in sortOptions"
        :key="o.value"
        :label="o.label"
        :value="o.value"
      />
    </ElSelect>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from "vue-i18n";
import ScanButton from "@/components/common/ScanButton.vue";

defineProps<{
  /** Texte en cours de saisie (appliqué à l'URL par `apply`). */
  modelValue: string;
  /** Tri courant et options proposées (dépendent du type et de la recherche). */
  sort: string;
  sortOptions: { value: string; label: string }[];
}>();
const emit = defineEmits<{
  "update:modelValue": [value: string];
  apply: [];
  scanned: [code: string];
  "update:sort": [value: string | undefined];
}>();
const { t } = useI18n();
</script>
