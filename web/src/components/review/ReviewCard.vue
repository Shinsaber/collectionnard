<template>
  <article
    class="flex flex-col gap-2.5 rounded-2xl bg-surface p-4 shadow-card"
    data-test="review-card"
  >
    <header class="flex flex-wrap items-center gap-x-3 gap-y-1.5">
      <slot name="heading">
        <RouterLink
          v-if="review.author"
          :to="`/u/${review.author.username}`"
          class="flex items-center gap-2 font-bold text-ink"
          data-test="review-author"
        >
          <UserAvatar :user="review.author" :size="28" />
          {{ review.author.display_name || review.author.username }}
        </RouterLink>
        <span v-else class="font-bold text-muted" data-test="review-anonymous">
          {{ t("reviews.anonymousAuthor") }}
        </span>
      </slot>
      <ReviewStars :rating="review.rating" />
      <span
        class="text-sm"
        :title="LANG_NAMES[reviewLang]"
        :aria-label="LANG_NAMES[reviewLang]"
        role="img"
        data-test="review-lang"
        >{{ LANG_FLAGS[reviewLang] }}</span
      >
      <span class="ml-auto font-mono text-[11px] text-muted">
        <time :datetime="review.updated_at">{{ date }}</time>
      </span>
      <slot name="actions"></slot>
    </header>

    <p
      v-if="review.hidden_at"
      class="rounded-lg bg-danger/12 px-3 py-2 text-[13px] text-danger"
      role="status"
      data-test="review-hidden"
    >
      {{ t("reviews.hidden", { reason: review.hidden_reason ?? "" }) }}
    </p>
    <p
      v-if="review.context_item"
      class="text-[13px] text-muted"
      data-test="review-context"
    >
      {{ t("reviews.contextOf", { label: contextLabelText }) }}
      <RouterLink
        :to="`/items/${review.context_item.id}`"
        class="font-semibold text-link"
        >{{ itemLabel(review.context_item, lang) }}</RouterLink
      >
    </p>

    <!--
      Avis détaillé : les critères à gauche, le texte et le pied (réactions, signalement) à droite, le pied calé en
      bas, au niveau des critères quand le texte est court ; sur mobile, tout s'empile. Note globale (déjà dans
      l'en-tête) : le texte en retrait, marqué d'un filet, le pied aligné dessous.
    -->
    <div class="flex flex-col gap-3 sm:flex-row sm:gap-4">
      <!-- Une ligne par critère : libellé à gauche, étoiles et note alignées à droite. -->
      <dl
        v-if="criteria.length > 0"
        class="flex flex-col gap-2 rounded-xl bg-soft px-3 py-2.5 sm:w-[230px] sm:shrink-0 sm:self-start"
        data-test="review-criteria"
      >
        <div v-for="c in criteria" :key="c.key" class="flex items-center gap-3">
          <dt
            class="min-w-0 flex-1 truncate text-[13px] font-semibold"
            :title="c.label"
          >
            {{ c.label }}
          </dt>
          <dd>
            <ReviewStars :rating="Number(c.value)" :size="14" />
          </dd>
        </div>
      </dl>
      <div
        class="flex min-w-0 flex-1 flex-col gap-2.5"
        :class="{ 'pl-3': criteria.length === 0 }"
      >
        <div
          v-if="review.body"
          :class="{ '-ml-3 border-l border-line pl-3': criteria.length === 0 }"
        >
          <ElButton
            v-if="review.spoiler && !revealed"
            data-test="review-spoiler"
            @click="revealed = true"
          >
            <i class="pi pi-eye mr-2" aria-hidden="true"></i>
            {{ t("reviews.showSpoiler") }}
          </ElButton>
          <!-- Source Markdown rendu puis assaini par lib/markdown.ts. -->
          <!-- eslint-disable vue/no-v-html -->
          <div
            v-else
            class="review-body"
            data-test="review-body"
            v-html="html"
          />
          <!-- eslint-enable vue/no-v-html -->
        </div>
        <div
          v-if="reportable || (showReactions && hasReactions)"
          class="mt-auto flex flex-wrap items-center justify-between gap-2"
          data-test="review-footer"
        >
          <ReviewReactions
            v-if="showReactions"
            :review="review"
            :interactive="reportable"
          />
          <ElButton
            v-if="reportable"
            text
            size="small"
            class="ml-auto text-muted!"
            data-test="review-report"
            @click="reporting = true"
          >
            <i class="pi pi-flag mr-1.5" aria-hidden="true"></i>
            {{ t("reviews.report.action") }}
          </ElButton>
        </div>
      </div>
    </div>
    <ReportDialog
      v-if="reportable"
      v-model:visible="reporting"
      :review-id="review.id"
    />
  </article>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useLang } from "@/composables/useLang";
import { LANG_FLAGS, LANG_NAMES, isLang, DEFAULT_LANG } from "@/i18n/langs";
import { fieldLabel, itemLabel, type Schema } from "@/lib/schema";
import { renderMarkdown } from "@/lib/markdown";
import { criteriaKeys } from "@/lib/reviews";
import UserAvatar from "@/components/account/UserAvatar.vue";
import ReviewStars from "./ReviewStars.vue";
import ReviewReactions from "./ReviewReactions.vue";
import ReportDialog from "./ReportDialog.vue";
import type { Review } from "@/api/types";

const props = defineProps<{
  review: Review;
  /** Schéma du type d'avis (libellés des critères). */
  schema?: Record<string, unknown>;
  /** Libellé de l'édition utilisée (« Joué sur »). */
  contextLabelText?: string;
  /** Avis d'un autre membre, vu connecté : réactions et signalement possibles (I-1). */
  reportable?: boolean;
  /** Afficher les réactions reçues (en lecture seule si non `reportable`). */
  showReactions?: boolean;
}>();
const reporting = ref(false);
// Réactions et signalement en pied d'avis, sur toute la largeur ; rien en lecture seule sans réaction reçue.
const hasReactions = computed(() =>
  Object.values(props.review.reactions ?? {}).some((n) => n > 0),
);

const { t } = useI18n();
const { lang } = useLang();
const revealed = ref(false);

const reviewLang = computed(() =>
  isLang(props.review.lang) ? props.review.lang : DEFAULT_LANG,
);
const html = computed(() => renderMarkdown(props.review.body));
const date = computed(() =>
  new Date(props.review.updated_at).toLocaleDateString(lang.value),
);
const criteria = computed(() => {
  const values = props.review.criteria;
  if (!values) return [];
  const keys = criteriaKeys(props.schema);
  return (keys.length > 0 ? keys : Object.keys(values))
    .filter((key) => values[key] != null)
    .map((key) => ({
      key,
      label: fieldLabel(props.schema as Schema | undefined, key, lang.value),
      value: values[key],
    }));
});
</script>

<style scoped>
.review-body {
  font-size: 15px;
  overflow-wrap: anywhere;
}
.review-body :deep(p),
.review-body :deep(ul),
.review-body :deep(ol),
.review-body :deep(blockquote),
.review-body :deep(pre) {
  margin-block: 0.4em;
}
.review-body :deep(ul) {
  list-style: disc;
  padding-inline-start: 1.4em;
}
.review-body :deep(ol) {
  list-style: decimal;
  padding-inline-start: 1.4em;
}
.review-body :deep(blockquote) {
  border-inline-start: 3px solid var(--c-line);
  color: var(--c-muted);
  padding-inline-start: 0.8em;
}
.review-body :deep(a) {
  color: var(--c-link);
  text-decoration: underline;
}
.review-body :deep(code) {
  font-family: var(--font-mono, monospace);
  font-size: 0.9em;
}
.review-body :deep(pre) {
  overflow-x: auto;
}
</style>
