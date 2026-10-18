<template>
  <section class="flex flex-col gap-3" data-test="proposal-discussion">
    <h2 class="text-lg font-extrabold tracking-tight">
      {{ t("proposals.discussion.title") }}
    </h2>
    <p v-if="comments.length === 0" class="text-sm text-muted">
      {{ t("proposals.discussion.empty") }}
    </p>
    <ol v-else class="flex flex-col gap-2">
      <li
        v-for="c in comments"
        :key="c.id"
        class="flex gap-3 rounded-2xl bg-surface p-3 shadow-card"
      >
        <UserAvatar :user="c.author" :size="32" />
        <div class="flex min-w-0 flex-col">
          <span class="text-sm"
            ><strong>{{ c.author.display_name || c.author.username }}</strong>
            <span class="ml-2 font-mono text-xs text-muted">{{
              new Date(c.created_at).toLocaleString(lang, {
                dateStyle: "short",
                timeStyle: "short",
              })
            }}</span></span
          >
          <p class="whitespace-pre-line text-[15px]">{{ c.body }}</p>
        </div>
      </li>
    </ol>
    <form v-if="canWrite" class="flex flex-col gap-2" @submit.prevent="send">
      <label :for="id" class="sr-only">{{
        t("proposals.discussion.placeholder")
      }}</label>
      <ElInput
        :id="id"
        v-model="body"
        type="textarea"
        :autosize="{ minRows: 2, maxRows: 8 }"
        maxlength="10000"
        :placeholder="t('proposals.discussion.placeholder')"
      />
      <ElButton
        native-type="submit"
        class="self-end"
        :disabled="!body.trim()"
        :loading="sending"
        data-test="comment-send"
        >{{ t("proposals.discussion.send") }}</ElButton
      >
    </form>
  </section>
</template>

<script setup lang="ts">
// Fil de discussion d'une proposition (P4-3b) : auteur et modérateurs, jusqu'à la décision.
import { onMounted, ref, useId } from "vue";
import { useI18n } from "vue-i18n";
import { proposalApi } from "@/api/client";
import { useLang } from "@/composables/useLang";
import { useNotify } from "@/composables/useNotify";
import UserAvatar from "@/components/account/UserAvatar.vue";
import type { ProposalComment } from "@/api/types";

const props = defineProps<{ proposalId: string; canWrite: boolean }>();
const emit = defineEmits<{ posted: [] }>();
const { t } = useI18n();
const { lang } = useLang();
const { notifyError } = useNotify();
const id = useId();

const comments = ref<ProposalComment[]>([]);
const body = ref("");
const sending = ref(false);

async function load() {
  try {
    comments.value = await proposalApi.comments(props.proposalId);
  } catch (e) {
    notifyError(e);
  }
}

async function send() {
  sending.value = true;
  try {
    const c = await proposalApi.comment(props.proposalId, body.value.trim());
    comments.value = [...comments.value, c];
    body.value = "";
    emit("posted");
  } catch (e) {
    notifyError(e);
  } finally {
    sending.value = false;
  }
}

onMounted(load);
</script>
