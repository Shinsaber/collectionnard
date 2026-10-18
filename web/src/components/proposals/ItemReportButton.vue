<template>
  <button
    type="button"
    v-bind="$attrs"
    data-test="item-report"
    @click="open = true"
  >
    <i class="pi pi-flag text-sm" aria-hidden="true"></i>
    {{ t("proposals.reportButton") }}
  </button>
  <TextProposalDialog
    v-model:visible="open"
    kind="report"
    :item-id="itemId"
    :item-label="itemLabel"
  />
</template>

<script setup lang="ts">
// « Signaler » une fiche (P4-3c) : erreur à corriger ou item à supprimer, relu par un modérateur.
import { ref } from "vue";
import { useI18n } from "vue-i18n";
import TextProposalDialog from "./TextProposalDialog.vue";

defineOptions({ inheritAttrs: false });
defineProps<{ itemId: string; itemLabel: string }>();
const { t } = useI18n();
const open = ref(false);
</script>
