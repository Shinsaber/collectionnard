<template>
  <!-- Un seul format : simple bouton ; sinon menu des formats. -->
  <ElButton
    v-if="formats.length === 1"
    :loading="loading"
    :disabled="disabled"
    :title="title"
    @click="pick(formats[0]!)"
  >
    <i v-if="!loading" class="pi pi-download mr-2" aria-hidden="true"></i>
    {{ label }}
  </ElButton>
  <ElDropdown
    v-else
    trigger="click"
    :disabled="disabled || loading"
    @command="pick"
  >
    <ElButton :loading="loading" :disabled="disabled" :title="title">
      <i v-if="!loading" class="pi pi-download mr-2" aria-hidden="true"></i>
      {{ label }}
      <i class="pi pi-angle-down ml-2" aria-hidden="true"></i>
    </ElButton>
    <template #dropdown>
      <ElDropdownMenu>
        <ElDropdownItem v-for="f in formats" :key="f" :command="f">
          {{ f.toUpperCase() }}
        </ElDropdownItem>
      </ElDropdownMenu>
    </template>
  </ElDropdown>
</template>

<script setup lang="ts">
// Bouton d'export réutilisable : lance `run(format)` (téléchargement via importExportApi),
// affiche l'état de chargement et notifie l'erreur éventuelle (`errorSummary`, déjà traduit).
import { ref } from "vue";
import { useNotify } from "@/composables/useNotify";

export type ExportFormat = "json" | "csv";

const props = defineProps<{
  /** Libellé du bouton (déjà traduit). */
  label: string;
  /** Formats proposés ; les noms (JSON, CSV) sont techniques, non traduits. */
  formats: ExportFormat[];
  run: (format: ExportFormat) => Promise<void>;
  errorSummary: string;
  disabled?: boolean;
  title?: string;
}>();

const { notifyError } = useNotify();
const loading = ref(false);

async function pick(format: ExportFormat) {
  loading.value = true;
  try {
    await props.run(format);
  } catch (e) {
    notifyError(e, props.errorSummary);
  } finally {
    loading.value = false;
  }
}
</script>
