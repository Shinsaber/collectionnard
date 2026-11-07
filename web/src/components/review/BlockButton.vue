<template>
  <ElButton
    text
    size="small"
    class="text-muted!"
    :loading="busy"
    :aria-label="t('recommendations.block.actionLabel')"
    data-test="block-author"
    @click="block"
  >
    <i class="pi pi-ban mr-1.5" aria-hidden="true"></i>
    {{ t("recommendations.block.action") }}
  </ElButton>
</template>

<script setup lang="ts">
// Bloquer l'auteur d'un contenu, ou un membre nommé (I-4) : confirmation, puis le parent recharge ce qu'il montre.
import { ref } from "vue";
import { useI18n } from "vue-i18n";
import { blockApi } from "@/api/client";
import { useConfirmDialog } from "@/composables/useConfirmDialog";
import { useNotify } from "@/composables/useNotify";
import type { BlockContentKind } from "@/api/types";

const props = defineProps<{
  /** Contenu dont l'auteur est bloqué… */
  contentKind?: BlockContentKind;
  contentId?: string;
  /** …ou membre nommé. */
  username?: string;
}>();
const emit = defineEmits<{ blocked: [] }>();

const { t } = useI18n();
const { confirm } = useConfirmDialog();
const { notifyError, notifySuccess } = useNotify();
const busy = ref(false);

async function block() {
  const ok = await confirm({
    title: t("recommendations.block.confirmTitle"),
    message: t("recommendations.block.confirm"),
    confirmText: t("recommendations.block.action"),
    danger: true,
  });
  if (!ok) return;
  busy.value = true;
  try {
    await blockApi.create(
      props.username
        ? { username: props.username }
        : { content_kind: props.contentKind!, content_id: props.contentId! },
    );
    notifySuccess(t("recommendations.block.done"));
    emit("blocked");
  } catch (e) {
    notifyError(e);
  } finally {
    busy.value = false;
  }
}
</script>
