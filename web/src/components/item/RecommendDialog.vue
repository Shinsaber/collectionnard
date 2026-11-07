<template>
  <ElDialog
    :model-value="visible"
    :title="t('recommendations.send.title')"
    width="480px"
    @update:model-value="(v: boolean) => emit('update:visible', v)"
    @open="reset"
  >
    <form
      :id="`${id}-form`"
      class="flex flex-col gap-4"
      data-test="recommend-form"
      @submit.prevent="send"
    >
      <p class="text-sm text-muted">{{ t("recommendations.send.help") }}</p>
      <div class="flex flex-col gap-1.5">
        <label :for="`${id}-to`" :class="labelClass">{{
          t("recommendations.send.to")
        }}</label>
        <ElAutocomplete
          :id="`${id}-to`"
          v-model="to"
          :fetch-suggestions="suggest"
          value-key="username"
          :placeholder="t('recommendations.send.toPlaceholder')"
          clearable
        >
          <template #default="{ item }">
            <span class="font-semibold">{{
              (item as PublicProfile).display_name ||
              (item as PublicProfile).username
            }}</span>
            <span class="ml-2 font-mono text-[11px] text-muted">{{
              (item as PublicProfile).username
            }}</span>
          </template>
        </ElAutocomplete>
      </div>
      <div class="flex flex-col gap-1.5">
        <label :for="`${id}-message`" :class="labelClass">{{
          t("recommendations.send.message")
        }}</label>
        <ElInput
          :id="`${id}-message`"
          v-model="message"
          type="textarea"
          :autosize="{ minRows: 2, maxRows: 6 }"
          maxlength="500"
          show-word-limit
        />
      </div>
    </form>
    <template #footer>
      <ElButton text @click="emit('update:visible', false)">{{
        t("common.cancel")
      }}</ElButton>
      <ElButton
        type="primary"
        native-type="submit"
        :form="`${id}-form`"
        :loading="sending"
        :disabled="!to.trim()"
        data-test="recommend-send"
        >{{ t("recommendations.send.submit") }}</ElButton
      >
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
// Recommander un item à un membre de l'annuaire (I-4), avec un mot facultatif.
import { ref, useId } from "vue";
import { useI18n } from "vue-i18n";
import { memberApi, recommendationApi } from "@/api/client";
import { useAuthStore } from "@/stores/auth";
import { useNotify } from "@/composables/useNotify";
import type { PublicProfile } from "@/api/types";

const props = defineProps<{ visible: boolean; itemId: string }>();
const emit = defineEmits<{ "update:visible": [v: boolean] }>();

const { t } = useI18n();
const auth = useAuthStore();
const { notifyError, notifySuccess } = useNotify();
const id = useId();
const labelClass = "font-mono text-[11px] uppercase tracking-wider text-muted";

const to = ref("");
const message = ref("");
const sending = ref(false);

function reset() {
  to.value = "";
  message.value = "";
}

/** Annuaire chargé une fois à la première saisie, filtré ici (l'API n'a pas de recherche). */
let members: PublicProfile[] | null = null;
async function suggest(query: string, cb: (list: PublicProfile[]) => void) {
  try {
    members ??= (await memberApi.list({ per_page: 100 })).data;
  } catch (e) {
    notifyError(e);
    members = [];
  }
  const q = query.trim().toLowerCase();
  cb(
    members
      .filter((m) => m.username !== auth.user?.username)
      .filter(
        (m) =>
          !q ||
          m.username.toLowerCase().includes(q) ||
          (m.display_name ?? "").toLowerCase().includes(q),
      )
      .slice(0, 10),
  );
}

async function send() {
  const name = to.value.trim();
  if (!name) return;
  sending.value = true;
  try {
    await recommendationApi.create(props.itemId, {
      to: name,
      message: message.value.trim(),
    });
    notifySuccess(t("recommendations.send.sent", { name }));
    emit("update:visible", false);
  } catch (e) {
    notifyError(e);
  } finally {
    sending.value = false;
  }
}
</script>
