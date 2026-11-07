<template>
  <section
    class="flex flex-col gap-3 rounded-xl bg-soft p-3"
    data-test="comment-thread"
  >
    <AppSpinner v-if="loading" />
    <template v-else>
      <p v-if="comments.length === 0" class="text-sm text-muted">
        {{ t("reviews.comments.none") }}
      </p>
      <ul v-else class="flex flex-col gap-2.5">
        <li
          v-for="c in comments"
          :key="c.id"
          class="flex flex-col gap-1.5 rounded-lg bg-surface px-3 py-2"
          data-test="comment"
        >
          <div class="flex flex-wrap items-center gap-2 text-[13px]">
            <span class="font-bold" data-test="comment-author">{{
              c.author
                ? c.author.display_name || c.author.username
                : t("reviews.comments.reviewAuthor")
            }}</span>
            <span
              v-if="c.review_author && c.author"
              class="rounded-full bg-tint px-2 py-0.5 text-[11px] font-semibold text-tint-ink"
              >{{ t("reviews.comments.reviewAuthor") }}</span
            >
            <time
              class="font-mono text-[11px] text-muted"
              :datetime="c.created_at"
              >{{
                new Date(c.created_at).toLocaleString(lang, {
                  dateStyle: "short",
                  timeStyle: "short",
                })
              }}</time
            >
            <span class="ml-auto flex gap-1">
              <template v-if="c.mine">
                <ElButton text size="small" @click="startEdit(c)">{{
                  t("common.edit")
                }}</ElButton>
                <ElButton text size="small" type="danger" @click="remove(c)">{{
                  t("common.delete")
                }}</ElButton>
              </template>
              <ElButton
                v-else-if="canWrite"
                text
                size="small"
                class="text-muted!"
                data-test="comment-report"
                @click="reportingId = c.id"
              >
                <i class="pi pi-flag mr-1" aria-hidden="true"></i>
                {{ t("reviews.report.action") }}
              </ElButton>
            </span>
          </div>
          <p
            v-if="c.hidden_at"
            class="rounded bg-danger/12 px-2 py-1 text-[12px] text-danger"
          >
            {{
              t("reviews.comments.hidden", { reason: c.hidden_reason ?? "" })
            }}
          </p>
          <form
            v-if="editingId === c.id"
            class="flex flex-col gap-2"
            @submit.prevent="saveEdit(c)"
          >
            <ElInput
              v-model="editBody"
              type="textarea"
              :autosize="{ minRows: 2, maxRows: 10 }"
              maxlength="5000"
            />
            <div class="flex justify-end gap-2">
              <ElButton size="small" @click="editingId = null">{{
                t("common.cancel")
              }}</ElButton>
              <ElButton size="small" type="primary" native-type="submit">{{
                t("common.save")
              }}</ElButton>
            </div>
          </form>
          <!-- Markdown restreint, rendu puis assaini par lib/markdown.ts. -->
          <!-- eslint-disable vue/no-v-html -->
          <div
            v-else
            class="review-body text-sm"
            data-test="comment-body"
            v-html="renderMarkdown(c.body)"
          />
          <!-- eslint-enable vue/no-v-html -->
        </li>
      </ul>
      <ElPagination
        v-if="total > PER_PAGE"
        class="justify-end"
        layout="prev, pager, next"
        :total="total"
        :page-size="PER_PAGE"
        :current-page="page"
        @current-change="(p: number) => ((page = p), load())"
      />

      <p v-if="closed" class="text-sm text-muted" data-test="comments-closed">
        {{ t("reviews.comments.closed") }}
      </p>
      <form
        v-else-if="canWrite"
        class="flex flex-col gap-2"
        data-test="comment-form"
        @submit.prevent="send"
      >
        <ElInput
          v-model="body"
          type="textarea"
          :autosize="{ minRows: 2, maxRows: 8 }"
          maxlength="5000"
          :placeholder="t('reviews.comments.placeholder')"
          :aria-label="t('reviews.comments.placeholder')"
          data-test="comment-input"
        />
        <div class="flex justify-end">
          <ElButton
            type="primary"
            native-type="submit"
            :disabled="!body.trim()"
            :loading="sending"
            data-test="comment-send"
            >{{ t("reviews.comments.send") }}</ElButton
          >
        </div>
      </form>
      <div v-if="ownReview" class="flex justify-end">
        <ElButton
          text
          size="small"
          data-test="comments-toggle-closed"
          @click="toggleClosed"
          >{{
            closed ? t("reviews.comments.reopen") : t("reviews.comments.close")
          }}</ElButton
        >
      </div>
    </template>
    <ReportDialog
      v-if="reportingId"
      :visible="!!reportingId"
      :review-id="reportingId"
      target-kind="comment"
      @update:visible="(v) => !v && (reportingId = null)"
    />
  </section>
</template>

<script setup lang="ts">
// Fil d'un avis (I-2) : à plat, les plus anciens d'abord. L'auteur d'un avis anonyme y reste « l'auteur de
// l'avis », sans nom (le serveur ne l'envoie pas). L'auteur de l'avis ferme ou rouvre son fil.
import { onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { commentApi } from "@/api/client";
import { useConfirmDialog } from "@/composables/useConfirmDialog";
import { useLang } from "@/composables/useLang";
import { useNotify } from "@/composables/useNotify";
import { renderMarkdown } from "@/lib/markdown";
import AppSpinner from "@/components/common/AppSpinner.vue";
import ReportDialog from "./ReportDialog.vue";
import type { Comment, Review } from "@/api/types";

const PER_PAGE = 50;

const props = defineProps<{
  review: Review;
  /** Lecteur connecté : écrire, signaler. */
  canWrite: boolean;
  /** Son propre avis : fermer ou rouvrir le fil. */
  ownReview?: boolean;
}>();
const emit = defineEmits<{ count: [n: number] }>();

const { t } = useI18n();
const { lang } = useLang();
const { confirm } = useConfirmDialog();
const { notifyError } = useNotify();

const comments = ref<Comment[]>([]);
const total = ref(0);
const page = ref(1);
const loading = ref(true);
const body = ref("");
const sending = ref(false);
const closed = ref(props.review.comments_closed ?? false);
const editingId = ref<string | null>(null);
const editBody = ref("");
const reportingId = ref<string | null>(null);

async function load() {
  loading.value = true;
  try {
    const res = await commentApi.list(props.review.id, {
      page: page.value,
      per_page: PER_PAGE,
    });
    comments.value = res.data;
    total.value = res.total;
    emit("count", res.data.filter((c) => !c.hidden_at).length);
  } catch (e) {
    notifyError(e);
  } finally {
    loading.value = false;
  }
}

async function send() {
  sending.value = true;
  try {
    await commentApi.create(props.review.id, body.value);
    body.value = "";
    page.value = Math.max(1, Math.ceil((total.value + 1) / PER_PAGE));
    await load();
  } catch (e) {
    notifyError(e);
  } finally {
    sending.value = false;
  }
}

function startEdit(c: Comment) {
  editingId.value = c.id;
  editBody.value = c.body;
}

async function saveEdit(c: Comment) {
  try {
    const updated = await commentApi.update(c.id, editBody.value);
    Object.assign(c, { body: updated.body, updated_at: updated.updated_at });
    editingId.value = null;
  } catch (e) {
    notifyError(e);
  }
}

async function remove(c: Comment) {
  const ok = await confirm({
    message: t("reviews.comments.confirmDelete"),
    confirmText: t("common.delete"),
    danger: true,
  });
  if (!ok) return;
  try {
    await commentApi.remove(c.id);
    await load();
  } catch (e) {
    notifyError(e);
  }
}

async function toggleClosed() {
  try {
    await commentApi.setClosed(props.review.id, !closed.value);
    closed.value = !closed.value;
  } catch (e) {
    notifyError(e);
  }
}

onMounted(load);
</script>
