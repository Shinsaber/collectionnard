<template>
  <ElDialog
    :model-value="visible"
    :title="t(`reviews.report.titles.${targetKind}`)"
    width="min(480px, 92vw)"
    append-to-body
    @update:model-value="emit('update:visible', $event)"
  >
    <form
      :id="`${uid}-form`"
      class="flex flex-col gap-4"
      data-test="report-form"
      @submit.prevent="send"
    >
      <p class="text-sm text-muted">{{ t("reviews.report.help") }}</p>
      <fieldset class="flex flex-col gap-1">
        <legend class="mb-1 text-[13px] font-bold">
          {{ t("reviews.report.reason") }}
        </legend>
        <ElRadioGroup v-model="reason" class="flex-col! items-start!">
          <ElRadio v-for="r in REPORT_REASONS" :key="r" :value="r">{{
            t(`reviews.report.reasons.${r}`)
          }}</ElRadio>
        </ElRadioGroup>
      </fieldset>
      <div class="flex flex-col gap-1.5">
        <label :for="`${uid}-details`" class="text-[13px] font-bold">{{
          t("reviews.report.details")
        }}</label>
        <ElInput
          :id="`${uid}-details`"
          v-model="details"
          type="textarea"
          :autosize="{ minRows: 2, maxRows: 6 }"
          maxlength="1000"
          show-word-limit
        />
      </div>
    </form>
    <template #footer>
      <ElButton @click="emit('update:visible', false)">{{
        t("common.cancel")
      }}</ElButton>
      <ElButton
        type="danger"
        native-type="submit"
        :form="`${uid}-form`"
        :disabled="!reason"
        :loading="sending"
        data-test="report-send"
        >{{ t("reviews.report.send") }}</ElButton
      >
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
// Signalement d'un avis (I-1) : un motif et un mot facultatif, lus par les modérateurs.
import { ref, useId, watch } from "vue";
import { useI18n } from "vue-i18n";
import { reportApi } from "@/api/client";
import { useNotify } from "@/composables/useNotify";
import { REPORT_REASONS, type ReportReason } from "@/lib/reviews";

const props = withDefaults(
  defineProps<{
    visible: boolean;
    /** Contenu signalé : un avis, ou un commentaire (`targetKind`). */
    reviewId: string;
    targetKind?: "review" | "comment" | "article";
  }>(),
  { targetKind: "review" },
);
const emit = defineEmits<{ "update:visible": [value: boolean] }>();

const { t } = useI18n();
const { notifyError, notifySuccess } = useNotify();
const uid = useId();
const reason = ref<ReportReason | "">("");
const details = ref("");
const sending = ref(false);

watch(
  () => props.visible,
  (open) => {
    if (open) {
      reason.value = "";
      details.value = "";
    }
  },
);

async function send() {
  if (!reason.value) return;
  sending.value = true;
  try {
    await reportApi.create({
      target_kind: props.targetKind,
      target_id: props.reviewId,
      reason: reason.value,
      details: details.value,
    });
    notifySuccess(t("reviews.report.sent"));
    emit("update:visible", false);
  } catch (e) {
    notifyError(e);
  } finally {
    sending.value = false;
  }
}
</script>
