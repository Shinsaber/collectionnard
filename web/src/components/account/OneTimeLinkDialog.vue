<template>
  <!-- Lien à usage unique (invitation, réinitialisation) : montré cette seule fois, à transmettre soi-même. -->
  <ElDialog
    :model-value="!!url"
    :title="title"
    width="520px"
    @update:model-value="(v: boolean) => !v && emit('close')"
  >
    <div class="flex flex-col gap-3">
      <ElAlert
        type="warning"
        :closable="false"
        show-icon
        :title="t('account.link.once')"
      />
      <label :for="id" class="sr-only">{{ title }}</label>
      <ElInput
        :id="id"
        :model-value="url"
        readonly
        class="font-mono"
        data-test="one-time-link"
        @focus="($event.target as HTMLInputElement).select()"
      />
      <p v-if="expiresAt" class="text-sm text-muted">
        {{ t("account.link.expires", { date: formatDate(expiresAt) }) }}
      </p>
    </div>
    <template #footer>
      <ElButton text @click="emit('close')">{{ t("common.close") }}</ElButton>
      <ElButton type="primary" @click="copy">
        <i class="pi pi-copy mr-2" aria-hidden="true"></i>
        {{ copied ? t("account.link.copied") : t("account.link.copy") }}
      </ElButton>
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
import { ref, useId, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useLang } from "@/composables/useLang";
import { useNotify } from "@/composables/useNotify";

const props = defineProps<{ url: string; title: string; expiresAt?: string }>();
const emit = defineEmits<{ close: [] }>();
const { t } = useI18n();
const { lang } = useLang();
const { notifyError } = useNotify();
const id = useId();
const copied = ref(false);

watch(
  () => props.url,
  () => (copied.value = false),
);

function formatDate(iso: string): string {
  return new Date(iso).toLocaleString(lang.value, {
    dateStyle: "long",
    timeStyle: "short",
  });
}

async function copy() {
  try {
    await navigator.clipboard.writeText(props.url);
    copied.value = true;
  } catch (e) {
    notifyError(e);
  }
}
</script>
