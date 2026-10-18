<template>
  <ElRadioGroup
    :model-value="modelValue"
    class="flex! flex-col! items-stretch! gap-2"
    :aria-label="label ?? t('sharing.visibility.label')"
    @update:model-value="emit('update:modelValue', $event as Visibility)"
  >
    <ElRadio
      v-for="v in LEVELS"
      :key="v"
      :value="v"
      border
      class="mr-0! h-auto! py-2!"
      :data-test="`visibility-${v}`"
    >
      <span class="flex flex-col leading-snug">
        <span class="font-semibold">{{ t(`sharing.visibility.${v}`) }}</span>
        <span class="text-xs whitespace-normal text-muted">{{
          t(`sharing.visibility.${v}Help`)
        }}</span>
      </span>
    </ElRadio>
  </ElRadioGroup>
</template>

<script setup lang="ts">
// Visibilité d'une collection ou d'un profil (P4-2) : privée, membres, publique.
import { useI18n } from "vue-i18n";
import type { Visibility } from "@/api/types";

const LEVELS: Visibility[] = ["private", "members", "public"];

defineProps<{ modelValue: Visibility; label?: string }>();
const emit = defineEmits<{ "update:modelValue": [value: Visibility] }>();
const { t } = useI18n();
</script>
