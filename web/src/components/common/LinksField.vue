<template>
  <!-- Champ « liens » (x-widget: links) : une ligne libellé + adresse par lien. -->
  <div class="flex flex-col gap-2" role="group" :aria-labelledby="labelledby">
    <div
      v-for="(link, i) in rows"
      :key="i"
      class="flex flex-wrap items-center gap-2"
      data-test="link-row"
    >
      <ElInput
        :id="`${id}-${i}-label`"
        :aria-label="t('schema.form.linkLabel', { n: i + 1 })"
        :model-value="link.label ?? ''"
        :placeholder="t('schema.form.linkLabelPlaceholder')"
        class="min-w-32 flex-1"
        @update:model-value="set(i, 'label', String($event ?? ''))"
        @blur="emit('blur')"
      />
      <ElInput
        :id="`${id}-${i}-url`"
        :aria-label="t('schema.form.linkUrl', { n: i + 1 })"
        :model-value="link.url ?? ''"
        type="url"
        :placeholder="t('schema.form.linkUrlPlaceholder')"
        class="min-w-48 flex-[2]"
        @update:model-value="set(i, 'url', String($event ?? ''))"
        @blur="emit('blur')"
      />
      <ElButton
        text
        :aria-label="t('schema.form.linkRemove', { n: i + 1 })"
        @click="remove(i)"
      >
        <i class="pi pi-times" aria-hidden="true"></i>
      </ElButton>
    </div>
    <ElButton class="self-start" size="small" @click="add">
      <i class="pi pi-plus mr-2" aria-hidden="true"></i>
      {{ t("schema.form.linkAdd") }}
    </ElButton>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";

type Row = { label?: string; url?: string };

const props = defineProps<{
  modelValue: unknown;
  id: string;
  labelledby: string;
}>();
const emit = defineEmits<{
  "update:modelValue": [value: Row[] | undefined];
  blur: [];
}>();
const { t } = useI18n();

const rows = computed<Row[]>(() =>
  Array.isArray(props.modelValue) ? (props.modelValue as Row[]) : [],
);

// Libellé vide : clé absente (le schéma n'accepte que des chaînes, pas de libellé imposé).
function set(i: number, key: "label" | "url", value: string) {
  const next = rows.value.map((r) => ({ ...r }));
  const row = next[i] ?? {};
  if (value === "") delete row[key];
  else row[key] = value;
  next[i] = row;
  emit("update:modelValue", next);
}

function add() {
  emit("update:modelValue", [...rows.value, { url: "" }]);
}

// Plus aucun lien : champ absent (pas de tableau vide envoyé).
function remove(i: number) {
  const next = rows.value.filter((_, j) => j !== i);
  emit("update:modelValue", next.length > 0 ? next : undefined);
}
</script>
