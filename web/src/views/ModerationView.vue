<template>
  <div class="flex flex-col gap-4">
    <PageHeader :title="t('moderation.title')">
      <template #subtitle>
        <p class="text-[15px] text-muted">{{ t("moderation.help") }}</p>
      </template>
    </PageHeader>

    <ElTabs v-model="tab" data-test="moderation-tabs">
      <ElTabPane
        name="queue"
        :label="`${t('moderation.queue')} · ${queueTotal}`"
      >
        <AppSpinner v-if="loadingQueue" />
        <p
          v-else-if="queue.length === 0"
          class="rounded-2xl bg-surface p-5 text-sm text-muted shadow-card"
        >
          {{ t("moderation.empty") }}
        </p>
        <div v-else class="flex flex-col gap-5">
          <section
            v-for="q in queue"
            :key="q.target_id"
            class="flex flex-col gap-3 rounded-2xl bg-bar p-4"
            data-test="moderation-item"
          >
            <div class="flex flex-wrap items-center justify-between gap-2">
              <RouterLink
                v-if="q.item"
                :to="`/items/${q.item.id}`"
                class="font-bold text-link"
                >{{ itemLabel(q.item, lang) }}</RouterLink
              >
              <button
                v-if="q.author_id"
                type="button"
                class="rounded-full bg-tint px-3 py-1 text-xs font-semibold text-tint-ink"
                data-test="author-recap"
                @click="openHistory(q.author_id)"
              >
                {{
                  t("moderation.recap", {
                    recent: q.author_recap.hidden_90d,
                    total: q.author_recap.hidden_total,
                  })
                }}
              </button>
            </div>
            <ReviewCard
              v-if="q.review"
              :review="q.review"
              :schema="q.review_type?.attributes_schema"
            >
              <template #heading>
                <span class="font-bold">{{
                  q.review.author?.display_name ||
                  q.review.author?.username ||
                  "?"
                }}</span>
                <span
                  class="rounded-full bg-tint px-2.5 py-0.5 text-xs font-semibold text-tint-ink"
                  >{{ t(`reviews.visibility.${q.review.visibility}`) }}</span
                >
              </template>
            </ReviewCard>
            <p v-else class="text-sm text-muted">{{ t("moderation.gone") }}</p>

            <ul class="flex flex-col gap-2" data-test="moderation-reports">
              <li
                v-for="r in q.reports"
                :key="r.id"
                class="flex flex-col gap-0.5 rounded-xl bg-surface px-3 py-2 text-sm"
              >
                <span>
                  <strong>{{ t(`reviews.report.reasons.${r.reason}`) }}</strong>
                  ·
                  <button
                    type="button"
                    class="text-link"
                    @click="openHistory(r.reporter_id)"
                  >
                    {{ r.reporter?.display_name || r.reporter?.username }}
                  </button>
                  <span
                    v-if="r.reporter_dismissed > 0"
                    class="ml-1 text-xs text-muted"
                    >{{
                      t("moderation.reporterDismissed", r.reporter_dismissed)
                    }}</span
                  >
                </span>
                <span v-if="r.details" class="text-muted">{{ r.details }}</span>
              </li>
            </ul>

            <div v-if="q.review" class="flex flex-wrap justify-end gap-2">
              <ElButton
                data-test="moderation-dismiss"
                @click="act(q.review.id, 'dismiss')"
                >{{ t("moderation.dismiss") }}</ElButton
              >
              <ElButton
                type="danger"
                data-test="moderation-hide"
                @click="act(q.review.id, 'hide')"
                >{{ t("moderation.hide") }}</ElButton
              >
            </div>
          </section>
          <ElPagination
            class="justify-end"
            hide-on-single-page
            layout="prev, pager, next"
            :total="queueTotal"
            :page-size="PER_PAGE"
            :current-page="queuePage"
            @current-change="(p: number) => ((queuePage = p), loadQueue())"
          />
        </div>
      </ElTabPane>

      <ElTabPane name="log" :label="t('moderation.log')" lazy>
        <AppSpinner v-if="loadingLog" />
        <ul v-else class="flex flex-col gap-2" data-test="moderation-log">
          <li
            v-for="e in log"
            :key="e.id"
            class="flex flex-col gap-1 rounded-2xl bg-surface p-3 text-sm shadow-card"
          >
            <div class="flex flex-wrap items-center gap-2">
              <span class="font-bold">{{
                t(`moderation.actions.${e.action}`)
              }}</span>
              <span class="text-muted">{{
                t("moderation.by", { name: e.moderator?.username ?? "?" })
              }}</span>
              <span class="text-muted">·</span>
              <button
                v-if="e.author_id"
                type="button"
                class="text-link"
                @click="openHistory(e.author_id)"
              >
                {{ e.target_author?.username }}
              </button>
              <span class="ml-auto font-mono text-[11px] text-muted">{{
                new Date(e.created_at).toLocaleString(lang, {
                  dateStyle: "short",
                  timeStyle: "short",
                })
              }}</span>
              <ElButton
                v-if="e.action === 'hide' && e.still_hidden"
                size="small"
                data-test="moderation-restore"
                @click="act(e.target_id, 'restore')"
                >{{ t("moderation.restore") }}</ElButton
              >
            </div>
            <p v-if="e.reason" class="text-muted">« {{ e.reason }} »</p>
            <p
              v-if="e.excerpt"
              class="line-clamp-3 border-l-2 border-line pl-2"
            >
              {{ e.excerpt }}
            </p>
          </li>
        </ul>
        <ElPagination
          class="mt-3 justify-end"
          hide-on-single-page
          layout="prev, pager, next"
          :total="logTotal"
          :page-size="PER_PAGE"
          :current-page="logPage"
          @current-change="(p: number) => ((logPage = p), loadLog())"
        />
      </ElTabPane>
    </ElTabs>

    <ElDialog
      v-model="historyOpen"
      :title="
        history
          ? t('moderation.historyOf', { name: history.user.username })
          : ''
      "
      width="min(640px, 94vw)"
      append-to-body
    >
      <div
        v-if="history"
        class="flex flex-col gap-3 text-sm"
        data-test="moderation-history"
      >
        <p>
          {{
            t("moderation.recap", {
              recent: history.recap.hidden_90d,
              total: history.recap.hidden_total,
            })
          }}
          ·
          {{
            t(
              "moderation.reporterDismissed",
              history.recap.dismissed_reports_90d,
            )
          }}
        </p>
        <p v-if="history.entries.length === 0" class="text-muted">
          {{ t("moderation.noHistory") }}
        </p>
        <ul class="flex flex-col gap-1.5">
          <li
            v-for="e in history.entries"
            :key="e.id"
            class="rounded-lg bg-soft px-3 py-2"
          >
            <strong>{{ t(`moderation.actions.${e.action}`) }}</strong>
            <span class="text-muted">
              · {{ new Date(e.created_at).toLocaleDateString(lang) }}
              <template v-if="e.reason"> · « {{ e.reason }} »</template></span
            >
            <p v-if="e.excerpt" class="mt-1 line-clamp-2">{{ e.excerpt }}</p>
          </li>
        </ul>
        <template v-if="history.dismissed_reports.length > 0">
          <h3 class="font-bold">{{ t("moderation.dismissedReports") }}</h3>
          <ul class="flex flex-col gap-1">
            <li
              v-for="r in history.dismissed_reports"
              :key="r.id"
              class="text-muted"
            >
              {{ t(`reviews.report.reasons.${r.reason}`) }} ·
              {{ new Date(r.created_at).toLocaleDateString(lang) }}
            </li>
          </ul>
        </template>
      </div>
    </ElDialog>
  </div>
</template>

<script setup lang="ts">
// Modération des avis (I-1), pour les admins et les modérateurs : file des contenus signalés, journal de
// toutes les actions, historique d'un membre. Aucune sanction automatique : les récapitulatifs éclairent la
// décision.
import { onMounted, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { ElMessageBox } from "element-plus";
import { moderationApi } from "@/api/client";
import { useLang } from "@/composables/useLang";
import { useNotify } from "@/composables/useNotify";
import { useConfirmDialog } from "@/composables/useConfirmDialog";
import { useNotificationsStore } from "@/stores/notifications";
import { itemLabel } from "@/lib/schema";
import AppSpinner from "@/components/common/AppSpinner.vue";
import PageHeader from "@/components/common/PageHeader.vue";
import ReviewCard from "@/components/review/ReviewCard.vue";
import type {
  ModerationHistory,
  ModerationItem,
  ModerationLogEntry,
} from "@/api/types";

const PER_PAGE = 20;

const { t } = useI18n();
const { lang } = useLang();
const { notifyError, notifySuccess } = useNotify();
const { confirm } = useConfirmDialog();
const notifications = useNotificationsStore();

const tab = ref<"queue" | "log">("queue");
const queue = ref<ModerationItem[]>([]);
const queueTotal = ref(0);
const queuePage = ref(1);
const loadingQueue = ref(true);
const log = ref<ModerationLogEntry[]>([]);
const logTotal = ref(0);
const logPage = ref(1);
const loadingLog = ref(true);
const history = ref<ModerationHistory | null>(null);
const historyOpen = ref(false);

async function loadQueue() {
  loadingQueue.value = true;
  try {
    const res = await moderationApi.queue({
      page: queuePage.value,
      per_page: PER_PAGE,
    });
    queue.value = res.data;
    queueTotal.value = res.total;
  } catch (e) {
    notifyError(e);
  } finally {
    loadingQueue.value = false;
  }
}

async function loadLog() {
  loadingLog.value = true;
  try {
    const res = await moderationApi.log({
      page: logPage.value,
      per_page: PER_PAGE,
    });
    log.value = res.data;
    logTotal.value = res.total;
  } catch (e) {
    notifyError(e);
  } finally {
    loadingLog.value = false;
  }
}

async function openHistory(userId: string) {
  try {
    history.value = await moderationApi.history(userId);
    historyOpen.value = true;
  } catch (e) {
    notifyError(e);
  }
}

/** Masquer demande un motif, montré à l'auteur ; classer et rétablir se confirment. */
async function act(reviewId: string, action: "hide" | "restore" | "dismiss") {
  let reason = "";
  if (action === "hide") {
    try {
      const res = await ElMessageBox.prompt(
        t("moderation.reasonPrompt"),
        t("moderation.hide"),
        {
          confirmButtonText: t("moderation.hide"),
          cancelButtonText: t("common.cancel"),
          confirmButtonClass: "el-button--danger",
          inputValidator: (v: string) => !!v?.trim() || t("errors.required"),
        },
      );
      reason = res.value.trim();
    } catch {
      return;
    }
  } else {
    const ok = await confirm({
      message: t(`moderation.confirm.${action}`),
      confirmText: t(`moderation.${action}`),
    });
    if (!ok) return;
  }
  try {
    await moderationApi.act(reviewId, action, reason);
    notifySuccess(t(`moderation.done.${action}`));
    await Promise.all([loadQueue(), tab.value === "log" ? loadLog() : null]);
    notifications.refresh().catch(() => undefined);
  } catch (e) {
    notifyError(e);
  }
}

onMounted(loadQueue);
watch(tab, (v) => {
  if (v === "log") loadLog();
});
</script>
