<template>
  <!-- Texte multilingue (x-widget: i18n-text) : onglets par langue (● renseignée, ○ vide), une zone visible. -->
  <div
    :id="id"
    class="flex flex-col gap-2"
    role="group"
    :aria-labelledby="`${id}-label`"
  >
    <div
      role="tablist"
      :aria-label="t('schema.form.languages')"
      class="flex gap-0.5 self-start rounded-lg bg-soft p-0.75"
    >
      <button
        v-for="l in LANGS"
        :id="`${id}-tab-${l}`"
        :key="l"
        type="button"
        role="tab"
        :aria-selected="shown === l"
        :aria-controls="`${id}-panel-${l}`"
        class="h-7 rounded-md px-2.5 font-mono text-[11px] font-medium uppercase"
        :class="
          shown === l ? 'bg-surface text-ink shadow-control' : 'text-muted'
        "
        @click="active = l"
      >
        {{ text(l) ? "●" : "○" }} {{ l }}
      </button>
    </div>
    <div
      v-for="l in LANGS"
      v-show="shown === l"
      :id="`${id}-panel-${l}`"
      :key="l"
      role="tabpanel"
      :aria-labelledby="`${id}-tab-${l}`"
    >
      <label :for="`${id}-${l}`" class="sr-only">{{
        `${label} (${l.toUpperCase()})`
      }}</label>
      <ElInput
        :id="`${id}-${l}`"
        type="textarea"
        :model-value="text(l)"
        :autosize="{ minRows: 2, maxRows: 8 }"
        :placeholder="`${label} (${l.toUpperCase()})`"
        @update:model-value="set(l, String($event ?? ''))"
        @blur="emit('blur')"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useLang } from "@/composables/useLang";
import { LANGS } from "@/lib/schema";

const props = defineProps<{ modelValue: unknown; id: string; label: string }>();
const emit = defineEmits<{
  "update:modelValue": [value: Record<string, string>];
  blur: [];
}>();
const { t } = useI18n();
const { lang } = useLang();

// Une langue affichée à la fois, la langue courante d'abord.
const active = ref<string | null>(null);
const shown = computed(
  () =>
    active.value ??
    ((LANGS as readonly string[]).includes(lang.value) ? lang.value : LANGS[0]),
);

function value(): Record<string, string> {
  const v = props.modelValue;
  return v && typeof v === "object" && !Array.isArray(v)
    ? (v as Record<string, string>)
    : {};
}

function text(l: string): string {
  return value()[l] ?? "";
}

function set(l: string, v: string) {
  emit("update:modelValue", { ...value(), [l]: v });
}
</script>
