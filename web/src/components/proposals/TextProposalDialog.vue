<template>
  <ElDialog
    :model-value="visible"
    :title="
      kind === 'report'
        ? t('proposals.report.title')
        : t('proposals.suggestion.title')
    "
    width="32rem"
    @update:model-value="(v: boolean) => emit('update:visible', v)"
  >
    <form class="flex flex-col gap-3" @submit.prevent="send">
      <p class="text-sm text-muted">
        {{
          kind === "report"
            ? t("proposals.report.help")
            : t("proposals.suggestion.help")
        }}
      </p>
      <label :for="`${uid}-title`" class="text-sm font-medium">{{
        t("proposals.proposalTitle")
      }}</label>
      <ElInput
        :id="`${uid}-title`"
        v-model="title"
        maxlength="200"
        :placeholder="defaultTitle"
      />
      <label :for="`${uid}-text`" class="text-sm font-medium">{{
        t("proposals.report.text")
      }}</label>
      <ElInput
        :id="`${uid}-text`"
        v-model="text"
        type="textarea"
        :autosize="{ minRows: 3, maxRows: 10 }"
        maxlength="10000"
        data-test="text-proposal-body"
      />
    </form>
    <template #footer>
      <ElButton text @click="emit('update:visible', false)">{{
        t("common.cancel")
      }}</ElButton>
      <ElButton
        type="primary"
        :disabled="!text.trim()"
        :loading="sending"
        data-test="text-proposal-send"
        @click="send"
      >
        <i class="pi pi-send mr-2" aria-hidden="true"></i>
        {{ t("proposals.propose") }}
      </ElButton>
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
// Signalement d'une fiche (« à corriger », « à supprimer ») ou suggestion de champ (P4-3c) : un texte
// adressé aux modérateurs, sans modification du catalogue ; suivi comme toute proposition.
import { computed, ref, useId, watch } from "vue";
import { useI18n } from "vue-i18n";
import { proposalApi } from "@/api/client";
import { useNotify } from "@/composables/useNotify";

const props = defineProps<{
  visible: boolean;
  kind: "report" | "schema_suggestion";
  /** Signalement : l'item visé et son libellé. */
  itemId?: string;
  itemLabel?: string;
}>();
const emit = defineEmits<{ "update:visible": [value: boolean]; sent: [] }>();
const { t } = useI18n();
const { notifyError, notifySuccess } = useNotify();
const uid = useId();

const title = ref("");
const text = ref("");
const sending = ref(false);
const defaultTitle = computed(() =>
  props.kind === "report"
    ? t("proposals.report.defaultTitle", { label: props.itemLabel ?? "" })
    : t("proposals.suggestion.defaultTitle"),
);

watch(
  () => props.visible,
  (open) => {
    if (open) {
      title.value = "";
      text.value = "";
    }
  },
);

async function send() {
  if (!text.value.trim()) return;
  sending.value = true;
  try {
    await proposalApi.create({
      kind: props.kind,
      title: title.value.trim() || defaultTitle.value,
      description: text.value.trim(),
      item_id: props.itemId,
      submit: true,
    });
    notifySuccess(t("proposals.sent"));
    emit("update:visible", false);
    emit("sent");
  } catch (e) {
    notifyError(e);
  } finally {
    sending.value = false;
  }
}
</script>
