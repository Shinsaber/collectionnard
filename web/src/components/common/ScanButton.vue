<template>
  <!-- Bouton « scanner un code-barres » (lot M2) : ouvre le scanner et relaie le code lu. -->
  <ElButton
    :size="size"
    :type="type"
    :aria-label="t('common.scan.button')"
    :title="t('common.scan.button')"
    data-test="scan-button"
    @click="open = true"
  >
    <i class="pi pi-barcode" aria-hidden="true"></i>
    <span v-if="label" class="ml-2">{{ label }}</span>
  </ElButton>
  <BarcodeScannerDialog
    v-model:visible="open"
    :continuous="continuous"
    @detected="(code, format) => emit('detected', code, format)"
  />
</template>

<script setup lang="ts">
import { ref } from "vue";
import { useI18n } from "vue-i18n";
import BarcodeScannerDialog from "./BarcodeScannerDialog.vue";

defineProps<{
  size?: "large" | "default" | "small";
  type?: "primary" | "default";
  /** Texte à côté de l'icône (sinon icône seule, libellé accessible). */
  label?: string;
  continuous?: boolean;
}>();
const emit = defineEmits<{ detected: [code: string, format: string] }>();
const { t } = useI18n();
const open = ref(false);
</script>
