<template>
  <article
    class="flex flex-col gap-2 rounded-2xl bg-surface p-4 shadow-card"
    data-test="article-card"
  >
    <header class="flex flex-wrap items-center gap-x-3 gap-y-1">
      <a
        :href="article.url"
        target="_blank"
        rel="nofollow ugc noopener noreferrer"
        class="text-[17px] font-bold text-link hover:underline"
        data-test="article-link"
        >{{ article.title }}</a
      >
      <span class="font-mono text-[11px] text-muted">{{ host }}</span>
      <span
        v-if="statusLabel"
        class="rounded-full px-2.5 py-0.5 text-xs font-semibold"
        :class="
          article.status === 'pending'
            ? 'bg-tint text-tint-ink'
            : 'bg-danger/12 text-danger'
        "
        data-test="article-status"
        >{{ statusLabel }}</span
      >
    </header>
    <p
      v-if="article.message"
      class="whitespace-pre-line text-[15px]"
      data-test="article-message"
    >
      {{ article.message }}
    </p>
    <p
      v-if="article.mine && (article.reject_reason || article.hidden_reason)"
      class="rounded-lg bg-danger/12 px-3 py-2 text-[13px] text-danger"
    >
      {{ article.reject_reason || article.hidden_reason }}
    </p>
    <footer class="flex flex-wrap items-center gap-x-3 gap-y-1 text-[13px]">
      <span class="font-semibold">{{
        article.author
          ? article.author.display_name || article.author.username
          : t("reviews.comments.member")
      }}</span>
      <span
        role="img"
        :title="LANG_NAMES[lang_]"
        :aria-label="LANG_NAMES[lang_]"
        >{{ LANG_FLAGS[lang_] }}</span
      >
      <time
        class="font-mono text-[11px] text-muted"
        :datetime="article.created_at"
        >{{ new Date(article.created_at).toLocaleDateString(lang) }}</time
      >
      <span class="ml-auto flex flex-wrap items-center gap-1">
        <ElButton
          v-if="published"
          text
          size="small"
          class="text-muted!"
          :aria-expanded="threadOpen"
          data-test="article-comments"
          @click="threadOpen = !threadOpen"
        >
          <i class="pi pi-comment mr-1.5" aria-hidden="true"></i>
          {{ t("reviews.comments.count", commentCount) }}
        </ElButton>
        <template v-if="article.mine">
          <ElButton text size="small" @click="emit('edit')">{{
            t("common.edit")
          }}</ElButton>
          <ElButton text size="small" type="danger" @click="emit('remove')">{{
            t("common.delete")
          }}</ElButton>
        </template>
        <template v-else-if="canWrite && published">
          <ElButton
            text
            size="small"
            class="text-muted!"
            data-test="article-report"
            @click="reporting = true"
          >
            <i class="pi pi-flag mr-1.5" aria-hidden="true"></i>
            {{ t("reviews.report.action") }}
          </ElButton>
          <BlockButton
            content-kind="article"
            :content-id="article.id"
            @blocked="emit('blocked')"
          />
        </template>
      </span>
    </footer>
    <CommentThread
      v-if="published && threadOpen"
      :target="article"
      target-kind="article"
      :can-write="canWrite"
      :own-target="article.mine"
      @count="(n) => (commentCount = n)"
    />
    <ReportDialog
      v-if="reporting"
      :visible="reporting"
      :review-id="article.id"
      target-kind="article"
      @update:visible="(v) => (reporting = v)"
    />
  </article>
</template>

<script setup lang="ts">
// Un article partagé sur une fiche (I-3) : lien externe marqué comme contenu d'utilisateur, jamais récupéré
// par le serveur. Son auteur voit son état (en attente, refusé, masqué) et le motif.
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useLang } from "@/composables/useLang";
import { DEFAULT_LANG, LANG_FLAGS, LANG_NAMES, isLang } from "@/i18n/langs";
import CommentThread from "./CommentThread.vue";
import ReportDialog from "./ReportDialog.vue";
import BlockButton from "./BlockButton.vue";
import type { Article } from "@/api/types";

const props = defineProps<{ article: Article; canWrite: boolean }>();
const emit = defineEmits<{ edit: []; remove: []; blocked: [] }>();

const { t } = useI18n();
const { lang } = useLang();
const threadOpen = ref(false);
const reporting = ref(false);
const commentCount = ref(props.article.comment_count);

const lang_ = computed(() =>
  isLang(props.article.lang) ? props.article.lang : DEFAULT_LANG,
);
const published = computed(
  () => props.article.status === "published" && !props.article.hidden_at,
);
const host = computed(() => {
  try {
    return new URL(props.article.url).host.replace(/^www\./, "");
  } catch {
    return "";
  }
});
const statusLabel = computed(() => {
  if (props.article.hidden_at) return t("articles.status.hidden");
  if (props.article.status === "pending") return t("articles.status.pending");
  if (props.article.status === "rejected") return t("articles.status.rejected");
  return "";
});
</script>
