<template>
  <!-- Menu « Plus » d'une collection (retours v0.4.0) : export JSON / CSV, modification, suppression.
       Même gabarit que le menu de la fiche d'un item ; sur mobile, seule l'icône. -->
  <ElDropdown trigger="click" @command="onCommand">
    <button
      type="button"
      class="flex h-11 items-center gap-2 rounded-[10px] bg-soft px-4 text-[15px] font-semibold text-ink"
      :aria-label="t('collections.detail.moreActions')"
      :aria-busy="busy"
      data-test="collection-more"
    >
      <span v-if="busy" aria-hidden="true"
        ><i class="pi pi-spin pi-spinner"></i
      ></span>
      <span class="max-sm:hidden">{{ t("collections.detail.more") }}</span>
      <!-- Les classes responsives sur un <span> : la feuille de primeicons force le display des <i>. -->
      <span class="max-sm:hidden" aria-hidden="true"
        ><i class="pi pi-chevron-down text-xs"></i
      ></span>
      <span v-if="!busy" class="sm:hidden" aria-hidden="true"
        ><i class="pi pi-ellipsis-h"></i
      ></span>
    </button>
    <template #dropdown>
      <ElDropdownMenu>
        <ElDropdownItem
          v-for="f in formats"
          :key="f"
          :command="`export:${f}`"
          :disabled="busy"
        >
          <i class="pi pi-download mr-2" aria-hidden="true"></i>
          {{
            t("collections.detail.exportFormat", { format: f.toUpperCase() })
          }}
        </ElDropdownItem>
        <ElDropdownItem command="edit" divided data-test="collection-edit">
          <i class="pi pi-pencil mr-2" aria-hidden="true"></i>
          {{ t("collections.detail.editTitle") }}
        </ElDropdownItem>
        <ElDropdownItem
          command="delete"
          class="text-danger!"
          data-test="collection-delete"
        >
          <i class="pi pi-trash mr-2" aria-hidden="true"></i>
          {{ t("collections.detail.delete") }}
        </ElDropdownItem>
      </ElDropdownMenu>
    </template>
  </ElDropdown>
</template>

<script setup lang="ts">
import { ref } from "vue";
import { useI18n } from "vue-i18n";
import { useNotify } from "@/composables/useNotify";
import type { ExportFormat } from "@/components/common/ExportButton.vue";

const props = defineProps<{
  /** Lance le téléchargement (`importExportApi.exportCollection`). */
  exportRun: (format: ExportFormat) => Promise<void>;
}>();
const emit = defineEmits<{ edit: []; delete: [] }>();

const { t } = useI18n();
const { notifyError } = useNotify();
const formats: ExportFormat[] = ["json", "csv"];
const busy = ref(false);

async function onCommand(cmd: string) {
  if (cmd === "edit") return emit("edit");
  if (cmd === "delete") return emit("delete");
  if (!cmd.startsWith("export:")) return;
  busy.value = true;
  try {
    await props.exportRun(cmd.slice("export:".length) as ExportFormat);
  } catch (e) {
    notifyError(e, t("collections.detail.exportFailed"));
  } finally {
    busy.value = false;
  }
}
</script>
