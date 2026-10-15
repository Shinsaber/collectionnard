<template>
  <!-- Libellés des valeurs d'enum par langue (x-enum-labels) -->
  <table class="mt-2 text-xs w-full">
    <caption class="text-left text-muted pb-1">
      {{
        t("schema.enumLabels.caption")
      }}
    </caption>
    <thead>
      <tr>
        <th scope="col" class="text-left font-medium text-muted pr-2">
          {{ t("schema.enumLabels.value") }}
        </th>
        <th
          v-for="l in LANGS"
          :key="l"
          scope="col"
          class="text-left font-medium text-muted uppercase"
        >
          {{ l }}
        </th>
      </tr>
    </thead>
    <tbody>
      <tr v-for="v in values" :key="v">
        <th scope="row" class="text-left font-mono font-normal pr-2">
          {{ v }}
        </th>
        <td v-for="l in LANGS" :key="l" class="pr-1 pb-1">
          <ElInput
            size="small"
            :model-value="modelValue[l]?.[v] ?? ''"
            :placeholder="v"
            :aria-label="
              t('schema.enumLabels.inputLabel', { value: v, lang: l })
            "
            @update:model-value="patch(l, v, String($event ?? ''))"
          />
        </td>
      </tr>
    </tbody>
  </table>
</template>

<script setup lang="ts">
import { useI18n } from "vue-i18n";
import { LANGS } from "@/i18n/langs";

// Saisie de `x-enum-labels` : un libellé par valeur d'enum et par langue.
// Les langues hors `LANGS` déjà présentes sont conservées telles quelles.
const props = defineProps<{
  values: string[];
  modelValue: Record<string, Record<string, string>>;
}>();

const emit = defineEmits<{
  "update:modelValue": [value: Record<string, Record<string, string>>];
}>();

const { t } = useI18n();

function patch(l: string, value: string, text: string) {
  const row = { ...(props.modelValue[l] ?? {}), [value]: text };
  if (!text) delete row[value];
  emit("update:modelValue", { ...props.modelValue, [l]: row });
}
</script>
