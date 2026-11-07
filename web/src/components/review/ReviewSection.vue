<template>
  <div class="flex flex-col gap-4" data-test="review-section">
    <div class="flex flex-col gap-3 rounded-2xl bg-surface p-4 shadow-card">
      <div class="flex flex-wrap items-center gap-3">
        <template v-if="summary.rating != null">
          <ReviewStars
            :rating="summary.rating"
            :size="22"
            data-test="review-average"
          />
          <span class="text-sm text-muted">{{
            t("reviews.count", summary.count)
          }}</span>
        </template>
        <span v-else class="text-sm text-muted" data-test="review-threshold">{{
          t("reviews.belowThreshold", {
            count: summary.count,
            min: summary.min_ratings,
          })
        }}</span>
      </div>
      <dl
        v-if="criteriaAverages.length > 0"
        class="grid grid-cols-1 gap-1.5 sm:grid-cols-2"
        data-test="review-criteria-averages"
      >
        <div
          v-for="c in criteriaAverages"
          :key="c.key"
          class="flex items-center justify-between gap-3 text-sm"
        >
          <dt class="text-muted">{{ c.label }}</dt>
          <dd class="flex items-center gap-2">
            <ReviewStars :rating="c.rating" />
            <span class="font-mono text-[11px] text-muted">{{
              t("reviews.detailedCount", c.count)
            }}</span>
          </dd>
        </div>
      </dl>
    </div>

    <!-- Mon avis -->
    <ReviewForm
      v-if="editing"
      :item-id="itemId"
      :schema="schema"
      :existing="mine"
      :context-items="summary.context_items"
      :context-label-text="contextLabelText"
      @saved="onSaved"
      @cancel="editing = false"
      @remove="remove"
    />
    <div v-else-if="mine" class="flex flex-col gap-1.5" data-test="review-mine">
      <h3 class="text-sm font-bold">{{ t("reviews.mine") }}</h3>
      <ReviewCard
        :review="mine"
        :schema="schema"
        :context-label-text="contextLabelText"
        show-reactions
      >
        <template #heading>
          <span
            class="rounded-full bg-tint px-2.5 py-0.5 text-xs font-semibold text-tint-ink"
            >{{ t(`reviews.visibility.${mine.visibility}`) }}</span
          >
        </template>
        <template #actions>
          <ElButton data-test="review-edit" @click="editing = true">
            {{ t("common.edit") }}
          </ElButton>
        </template>
      </ReviewCard>
    </div>
    <ElButton
      v-else
      type="primary"
      class="self-start"
      data-test="review-write"
      @click="editing = true"
    >
      <i class="pi pi-pencil mr-2" aria-hidden="true"></i>
      {{ t("reviews.write") }}
    </ElButton>

    <!-- Avis des autres -->
    <div class="flex flex-wrap items-center gap-3">
      <ElSelect
        v-model="sort"
        class="w-48!"
        :aria-label="t('reviews.sort.label')"
        @change="reload"
      >
        <ElOption value="recent" :label="t('reviews.sort.recent')" />
        <ElOption value="rating" :label="t('reviews.sort.rating')" />
        <ElOption value="interesting" :label="t('reviews.sort.interesting')" />
      </ElSelect>
      <ElSelect
        v-if="summary.context_items.length > 0"
        v-model="contextFilter"
        clearable
        :value-on-clear="null"
        class="w-64!"
        :placeholder="t('reviews.allContexts')"
        :aria-label="contextLabelText"
        data-test="review-context-filter"
        @change="reload"
      >
        <ElOption
          v-for="ci in summary.context_items"
          :key="ci.id"
          :value="ci.id"
          :label="itemLabel(ci, lang)"
        />
      </ElSelect>
      <ElCheckbox
        v-model="myLangOnly"
        data-test="review-my-lang"
        @change="reload"
        >{{ t("reviews.myLangOnly") }}</ElCheckbox
      >
    </div>
    <AppSpinner v-if="loading" />
    <template v-else>
      <p
        v-if="reviews.length === 0"
        class="rounded-2xl bg-surface p-5 text-sm text-muted shadow-card"
      >
        {{ t("reviews.none") }}
      </p>
      <ReviewCard
        v-for="r in reviews"
        :key="r.id"
        :review="r"
        :schema="schema"
        :context-label-text="contextLabelText"
        reportable
        show-reactions
      />
      <p
        v-if="summary.note_only_count > 0"
        class="text-sm text-muted"
        data-test="review-note-only"
      >
        {{ t("reviews.noteOnly", summary.note_only_count) }}
      </p>
      <ElPagination
        class="justify-end"
        hide-on-single-page
        layout="prev, pager, next"
        :total="total"
        :page-size="PER_PAGE"
        :current-page="page"
        @current-change="onPage"
      />
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { reviewApi } from "@/api/client";
import { useConfirmDialog } from "@/composables/useConfirmDialog";
import { useLang } from "@/composables/useLang";
import { useNotify } from "@/composables/useNotify";
import { fieldLabel, itemLabel, type Schema } from "@/lib/schema";
import { contextLabel } from "@/lib/reviews";
import AppSpinner from "@/components/common/AppSpinner.vue";
import ReviewCard from "./ReviewCard.vue";
import ReviewForm from "./ReviewForm.vue";
import ReviewStars from "./ReviewStars.vue";
import type { Review, ReviewSummary } from "@/api/types";

const PER_PAGE = 20;

const props = defineProps<{ itemId: string; summary: ReviewSummary }>();
const emit = defineEmits<{ changed: [] }>();

const { t } = useI18n();
const { lang } = useLang();
const { confirm } = useConfirmDialog();
const { notifyError, notifySuccess } = useNotify();

const schema = computed(
  () => props.summary.review_type?.attributes_schema ?? {},
);
const criteriaAverages = computed(() =>
  props.summary.criteria.flatMap((c) =>
    c.rating == null
      ? []
      : [
          {
            key: c.key,
            count: c.count,
            rating: c.rating,
            label: fieldLabel(
              schema.value as Schema | undefined,
              c.key,
              lang.value,
            ),
          },
        ],
  ),
);

const mine = ref<Review | null>(null);
const editing = ref(false);
const reviews = ref<Review[]>([]);
const total = ref(0);
const page = ref(1);
const sort = ref<"recent" | "rating" | "interesting">("recent");
const myLangOnly = ref(false);
const contextFilter = ref<string | null>(null);
const contextLabelText = computed(() =>
  contextLabel(
    props.summary.review_type?.context_label,
    lang.value,
    t("reviews.form.contextDefault"),
  ),
);
const loading = ref(true);

async function loadList() {
  loading.value = true;
  try {
    const res = await reviewApi.list(props.itemId, {
      page: page.value,
      per_page: PER_PAGE,
      sort: sort.value,
      lang: myLangOnly.value ? lang.value : undefined,
      context_item_id: contextFilter.value ?? undefined,
    });
    reviews.value = res.data;
    total.value = res.total;
  } catch (e) {
    notifyError(e);
  } finally {
    loading.value = false;
  }
}

function reload() {
  page.value = 1;
  loadList();
}

function onPage(p: number) {
  page.value = p;
  loadList();
}

async function loadMine() {
  try {
    mine.value = await reviewApi.mine(props.itemId);
  } catch (e) {
    notifyError(e);
  }
}

function onSaved(review: Review) {
  mine.value = review;
  editing.value = false;
  emit("changed");
}

async function remove() {
  const ok = await confirm({
    message: t("reviews.confirmDelete"),
    confirmText: t("common.delete"),
    danger: true,
  });
  if (!ok) return;
  try {
    await reviewApi.remove(props.itemId);
    mine.value = null;
    editing.value = false;
    notifySuccess(t("reviews.deleted"));
    emit("changed");
  } catch (e) {
    notifyError(e);
  }
}

function loadAll() {
  editing.value = false;
  page.value = 1;
  loadMine();
  loadList();
}

onMounted(loadAll);
watch(() => props.itemId, loadAll);
</script>
