<template>
  <AppSpinner v-if="loading" />
  <p
    v-else-if="!proposal"
    class="rounded-2xl bg-surface p-6 text-muted shadow-card"
  >
    {{ t("proposals.notFound") }}
  </p>
  <div v-else class="flex max-w-4xl flex-col gap-6">
    <RouterLink
      to="/proposals"
      class="-mb-2 inline-flex items-center gap-1.5 text-sm font-semibold text-link"
    >
      <i class="pi pi-arrow-left text-xs" aria-hidden="true"></i>
      {{ t("proposals.title") }}
    </RouterLink>
    <PageHeader :title="proposal.title">
      <template #subtitle>
        <div class="flex flex-wrap items-center gap-2 text-sm text-muted">
          <span
            class="rounded-full px-2 py-0.5 font-mono text-[11px] uppercase"
            :class="statusClass(proposal.status)"
            data-test="proposal-status"
            >{{ t(`proposals.statuses.${proposal.status}`) }}</span
          >
          <span>{{ t(`proposals.kinds.${proposal.kind}`) }}</span>
          <span>{{
            t("proposals.by", {
              name: proposal.author.display_name || proposal.author.username,
            })
          }}</span>
          <span v-if="proposal.author_disabled" class="text-danger">{{
            t("proposals.authorDisabled")
          }}</span>
          <span v-if="proposal.submitted_at">{{
            t("proposals.submittedOn", { date: date(proposal.submitted_at) })
          }}</span>
        </div>
      </template>
    </PageHeader>

    <ElAlert
      v-if="proposal.review_comment"
      :type="proposal.status === 'rejected' ? 'error' : 'info'"
      :closable="false"
      show-icon
      :title="
        t('proposals.decidedBy', {
          name:
            proposal.reviewer?.display_name ||
            proposal.reviewer?.username ||
            '?',
        })
      "
      :description="proposal.review_comment"
    />

    <p
      v-if="proposal.description"
      class="whitespace-pre-line rounded-2xl bg-surface p-4 shadow-card"
    >
      {{ proposal.description }}
    </p>
    <RouterLink
      v-if="proposal.item_id && proposal.kind !== 'catalog'"
      :to="`/items/${proposal.item_id}`"
      class="font-semibold text-link"
      >{{ t("history.otherItem") }}</RouterLink
    >

    <ProposalChanges
      v-if="payload"
      v-model:payload="payload"
      :base="proposal.base"
      :editable="authorCanEdit || canModerate"
    />
    <ProposalConflicts
      v-if="canModerate && conflicts.length && payload"
      v-model:choices="choices"
      :conflicts="conflicts"
      :payload="payload"
    />

    <!-- Actions de l'auteur -->
    <div v-if="isAuthor && !final" class="flex flex-wrap gap-2">
      <ElButton
        v-if="authorCanEdit"
        type="primary"
        :loading="busy"
        data-test="proposal-submit"
        @click="submit"
      >
        {{
          proposal.status === "draft"
            ? t("proposals.actions.submit")
            : t("proposals.actions.resubmit")
        }}
      </ElButton>
      <ElButton
        :loading="busy"
        data-test="proposal-withdraw"
        @click="withdraw"
        >{{ t("proposals.actions.withdraw") }}</ElButton
      >
    </div>

    <!-- Modération -->
    <section
      v-if="canModerate"
      class="flex flex-col gap-3 rounded-2xl bg-surface p-5 shadow-card"
      data-test="moderation"
    >
      <label :for="`${uid}-comment`" class="text-sm font-bold">{{
        t("proposals.actions.commentLabel")
      }}</label>
      <ElInput
        :id="`${uid}-comment`"
        v-model="comment"
        type="textarea"
        :autosize="{ minRows: 2, maxRows: 8 }"
        maxlength="10000"
      />
      <div class="flex flex-wrap gap-2">
        <ElButton
          type="primary"
          :loading="busy"
          data-test="proposal-accept"
          @click="accept"
          >{{ t("proposals.actions.accept") }}</ElButton
        >
        <ElButton :loading="busy" @click="decide('changes')">{{
          t("proposals.actions.requestChanges")
        }}</ElButton>
        <ElButton
          type="danger"
          plain
          :loading="busy"
          @click="decide('reject')"
          >{{ t("proposals.actions.reject") }}</ElButton
        >
      </div>
    </section>

    <ProposalDiscussion
      :key="proposal.id"
      :proposal-id="proposal.id"
      :can-write="!final"
    />
  </div>
</template>

<script setup lang="ts">
// Une proposition (P4-3b) : changements, conflits et décision pour les modérateurs (jamais sur les
// siennes), soumission et retrait pour l'auteur, discussion.
import { computed, onMounted, ref, useId, watch } from "vue";
import { useI18n } from "vue-i18n";
import { proposalApi } from "@/api/client";
import { useLang } from "@/composables/useLang";
import { useNotify } from "@/composables/useNotify";
import { useConfirmDialog } from "@/composables/useConfirmDialog";
import { useAuthStore } from "@/stores/auth";
import { useMetaTypes } from "@/stores/metaTypes";
import { resolvePayload, statusClass, type Side } from "@/lib/proposals";
import AppSpinner from "@/components/common/AppSpinner.vue";
import PageHeader from "@/components/common/PageHeader.vue";
import ProposalChanges from "@/components/proposals/ProposalChanges.vue";
import ProposalConflicts from "@/components/proposals/ProposalConflicts.vue";
import ProposalDiscussion from "@/components/proposals/ProposalDiscussion.vue";
import type { Proposal, ProposalConflict, ProposalPayload } from "@/api/types";

const props = defineProps<{ id: string }>();
const { t } = useI18n();
const { lang } = useLang();
const { notifyError, notifySuccess } = useNotify();
const { confirm } = useConfirmDialog();
const auth = useAuthStore();
const metaTypes = useMetaTypes();
const uid = useId();

const proposal = ref<Proposal | null>(null);
const payload = ref<ProposalPayload | null>(null);
const conflicts = ref<ProposalConflict[]>([]);
const choices = ref<Record<string, Record<string, Side>>>({});
const comment = ref("");
const loading = ref(true);
const busy = ref(false);

const isAuthor = computed(
  () => proposal.value?.author.username === auth.user?.username,
);
const final = computed(() =>
  ["accepted", "rejected", "withdrawn"].includes(proposal.value?.status ?? ""),
);
const authorCanEdit = computed(
  () =>
    isAuthor.value &&
    ["draft", "changes_requested"].includes(proposal.value?.status ?? ""),
);
const canModerate = computed(
  () =>
    auth.canEditCatalog &&
    !isAuthor.value &&
    proposal.value?.status === "pending",
);

function date(iso: string): string {
  return new Date(iso).toLocaleDateString(lang.value, { dateStyle: "medium" });
}

async function load() {
  loading.value = true;
  try {
    const [p] = await Promise.all([
      proposalApi.get(props.id),
      metaTypes.loadItemTypes(),
      metaTypes.loadRelationTypes(),
    ]);
    proposal.value = p;
    payload.value = p.payload ? structuredClone(p.payload) : null;
    choices.value = {};
    conflicts.value =
      canModerate.value && p.kind === "catalog"
        ? await proposalApi.conflicts(p.id)
        : [];
  } catch (e) {
    proposal.value = null;
    const status = (e as { response?: { status?: number } })?.response?.status;
    if (status !== 404) notifyError(e);
  } finally {
    loading.value = false;
  }
}

async function run(fn: () => Promise<Proposal>, message: string) {
  busy.value = true;
  try {
    proposal.value = await fn();
    notifySuccess(message);
    await load();
  } catch (e) {
    notifyError(e);
  } finally {
    busy.value = false;
  }
}

function submit() {
  const p = proposal.value!;
  return run(async () => {
    if (payload.value)
      await proposalApi.update(p.id, {
        title: p.title,
        description: p.description,
        payload: payload.value,
        item_id: p.item_id,
      });
    return proposalApi.submit(p.id);
  }, t("proposals.sent"));
}

async function withdraw() {
  if (
    !(await confirm({
      message: t("proposals.actions.withdrawConfirm"),
      danger: true,
    }))
  )
    return;
  await run(
    () => proposalApi.withdraw(proposal.value!.id),
    t("proposals.actions.done"),
  );
}

function accept() {
  const p = proposal.value!;
  const body = payload.value
    ? {
        payload: resolvePayload(payload.value, conflicts.value, choices.value),
        conflicts_resolved: conflicts.value.length > 0,
        comment: comment.value.trim(),
      }
    : { comment: comment.value.trim() };
  return run(
    () => proposalApi.accept(p.id, body),
    t("proposals.actions.accepted"),
  );
}

function decide(kind: "changes" | "reject") {
  const text = comment.value.trim();
  if (!text) {
    notifyError(new Error(t("proposals.actions.commentRequired")));
    return;
  }
  const id = proposal.value!.id;
  return run(
    () =>
      kind === "reject"
        ? proposalApi.reject(id, text)
        : proposalApi.requestChanges(id, text),
    t("proposals.actions.done"),
  );
}

onMounted(load);
watch(() => props.id, load);
</script>
