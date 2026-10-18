<template>
  <div class="flex flex-col gap-1.5">
    <label :for="`${id}-new`" :class="labelClass">{{ label }}</label>
    <ElInput
      :id="`${id}-new`"
      v-model="first"
      size="large"
      type="password"
      show-password
      autocomplete="new-password"
      @blur="touched = true"
    />
    <small class="text-xs" :class="tooShort ? 'text-danger' : 'text-muted'">{{
      t("account.password.help", { n: MIN })
    }}</small>
  </div>
  <div class="flex flex-col gap-1.5">
    <label :for="`${id}-confirm`" :class="labelClass">{{
      t("account.password.confirm")
    }}</label>
    <ElInput
      :id="`${id}-confirm`"
      v-model="second"
      size="large"
      type="password"
      show-password
      autocomplete="new-password"
    />
    <small v-if="mismatch" class="text-xs text-danger" role="alert">{{
      t("account.password.mismatch")
    }}</small>
  </div>
</template>

<script setup lang="ts">
// Nouveau mot de passe saisi deux fois : n'émet la valeur que si elle est valide et confirmée.
import { computed, ref, useId, watch } from "vue";
import { useI18n } from "vue-i18n";

/** Longueur minimale, comme le serveur (model.MinPasswordLength). */
const MIN = 8;

const props = defineProps<{ modelValue: string; label?: string }>();
const emit = defineEmits<{ "update:modelValue": [value: string] }>();
const { t } = useI18n();
const id = useId();
const labelClass = "font-mono text-[11px] uppercase tracking-wider text-muted";
const label = computed(() => props.label ?? t("account.password.new"));

const first = ref("");
const second = ref("");
const touched = ref(false);
const tooShort = computed(() => touched.value && [...first.value].length < MIN);
const mismatch = computed(
  () => second.value !== "" && second.value !== first.value,
);

watch([first, second], () => {
  const ok = [...first.value].length >= MIN && first.value === second.value;
  emit("update:modelValue", ok ? first.value : "");
});
</script>
