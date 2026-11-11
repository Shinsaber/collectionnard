<template>
  <!-- Menu « ⋮ » d'un lot (P7-1) ; « Scinder » n'est proposé qu'à un lot de plusieurs exemplaires, « Prêter » qu'à
       un lot dont un exemplaire reste disponible (P7-3). -->
  <ElDropdown
    trigger="click"
    @command="(cmd: LotCommand) => emit('command', cmd)"
  >
    <button
      type="button"
      class="flex size-11 items-center justify-center rounded-[10px] text-muted md:size-8"
      :aria-label="t('collections.detail.itemActions', { label })"
      data-test="lot-actions"
    >
      <i class="pi pi-ellipsis-v" aria-hidden="true"></i>
    </button>
    <template #dropdown>
      <ElDropdownMenu>
        <ElDropdownItem v-if="commands.includes('edit')" command="edit">
          <i class="pi pi-pencil mr-2" aria-hidden="true"></i>
          {{ t("collections.detail.editAttributes") }}
        </ElDropdownItem>
        <ElDropdownItem
          v-if="commands.includes('split') && lot.quantity > 1"
          command="split"
          data-test="lot-split"
        >
          <i class="pi pi-clone mr-2" aria-hidden="true"></i>
          {{ t("collections.lots.split") }}
        </ElDropdownItem>
        <ElDropdownItem
          v-if="commands.includes('merge')"
          command="merge"
          data-test="lot-merge"
        >
          <i class="pi pi-sitemap mr-2" aria-hidden="true"></i>
          {{ t("collections.lots.merge") }}
        </ElDropdownItem>
        <ElDropdownItem
          v-if="commands.includes('lend') && availableCopies(lot) > 0"
          command="lend"
          data-test="lot-lend"
        >
          <i class="pi pi-arrow-up-right mr-2" aria-hidden="true"></i>
          {{ t("loans.lend") }}
        </ElDropdownItem>
        <ElDropdownItem
          v-if="commands.includes('loans')"
          command="loans"
          data-test="lot-loans"
        >
          <i class="pi pi-history mr-2" aria-hidden="true"></i>
          {{ t("loans.loansOf") }}
        </ElDropdownItem>
        <ElDropdownItem
          v-if="commands.includes('remove')"
          command="remove"
          class="text-danger!"
        >
          <i class="pi pi-trash mr-2" aria-hidden="true"></i>
          {{ t("collections.detail.removeFromCollection") }}
        </ElDropdownItem>
      </ElDropdownMenu>
    </template>
  </ElDropdown>
</template>

<script setup lang="ts">
import { useI18n } from "vue-i18n";
import type { LotCommand } from "@/lib/lots";
import { availableCopies } from "@/lib/loans";
import type { CollectionItem } from "@/api/types";

defineProps<{
  lot: CollectionItem;
  /** Libellé de l'item, pour le nom accessible du bouton. */
  label: string;
  commands: LotCommand[];
}>();
const emit = defineEmits<{ command: [cmd: LotCommand] }>();
const { t } = useI18n();
</script>
