<template>
  <div class="flex flex-col gap-6">
    <PageHeader :title="t('proposals.title')">
      <template #subtitle>
        <p class="text-[15px] text-muted">{{ t("proposals.help") }}</p>
      </template>
      <template #actions>
        <ElButton data-test="suggest-field" @click="suggestOpen = true">
          <i class="pi pi-lightbulb mr-2" aria-hidden="true"></i>
          {{ t("proposals.suggestField") }}
        </ElButton>
        <RouterLink to="/admin/external-import" class="el-button">
          <i class="pi pi-cloud-download mr-2" aria-hidden="true"></i>
          {{ t("proposals.fromSource") }}
        </RouterLink>
      </template>
    </PageHeader>

    <TextProposalDialog
      v-model:visible="suggestOpen"
      kind="schema_suggestion"
      @sent="reload"
    />

    <ElTabs v-model="scope" @tab-change="reload">
      <ElTabPane :label="t('proposals.mine')" name="mine" />
      <ElTabPane v-if="auth.canEditCatalog" name="review">
        <template #label>
          {{ t("proposals.review") }}
          <span
            v-if="toReview > 0"
            class="ml-1.5 rounded-full bg-accent px-1.5 font-mono text-[11px] text-accent-ink"
            data-test="review-count"
            >{{ toReview }}</span
          >
        </template>
      </ElTabPane>
    </ElTabs>

    <AppSpinner v-if="list.loading.value && list.data.value.length === 0" />
    <p
      v-else-if="list.data.value.length === 0"
      class="rounded-2xl bg-surface p-5 text-muted shadow-card"
    >
      {{
        scope === "review" ? t("proposals.noneToReview") : t("proposals.none")
      }}
    </p>
    <ul v-else class="flex flex-col gap-2.5" data-test="proposals">
      <li v-for="p in list.data.value" :key="p.id">
        <RouterLink
          :to="`/proposals/${p.id}`"
          class="flex flex-wrap items-center gap-3 rounded-2xl bg-surface p-4 shadow-card hover:shadow-raised"
        >
          <span
            class="rounded-full px-2 py-0.5 font-mono text-[11px] uppercase"
            :class="statusClass(p.status)"
            >{{ t(`proposals.statuses.${p.status}`) }}</span
          >
          <span class="min-w-0 flex-1 truncate font-bold text-ink">{{
            p.title
          }}</span>
          <span class="font-mono text-xs text-muted"
            >{{ t(`proposals.kinds.${p.kind}`) }} ·
            {{
              t("proposals.by", {
                name: p.author.display_name || p.author.username,
              })
            }}
            · {{ date(p.submitted_at ?? p.created_at) }}</span
          >
          <span v-if="p.comment_count" class="text-xs text-muted">
            <i class="pi pi-comment mr-1" aria-hidden="true"></i
            >{{ p.comment_count }}</span
          >
        </RouterLink>
      </li>
    </ul>
    <ElPagination
      class="justify-end"
      hide-on-single-page
      layout="prev, pager, next"
      :total="list.total.value"
      :page-size="list.perPage.value"
      :current-page="list.page.value"
      @current-change="onPage"
    />
  </div>
</template>

<script setup lang="ts">
// Propositions (P4-3b) : les miennes, et la file à relire pour les admins et contributeurs.
import { onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute } from "vue-router";
import { proposalApi } from "@/api/client";
import { useLang } from "@/composables/useLang";
import { useNotify } from "@/composables/useNotify";
import { usePagination } from "@/composables/usePagination";
import { useAuthStore } from "@/stores/auth";
import { statusClass } from "@/lib/proposals";
import AppSpinner from "@/components/common/AppSpinner.vue";
import PageHeader from "@/components/common/PageHeader.vue";
import TextProposalDialog from "@/components/proposals/TextProposalDialog.vue";

const { t } = useI18n();
const { lang } = useLang();
const { notifyError } = useNotify();
const auth = useAuthStore();

const route = useRoute();
const scope = ref<"mine" | "review">(
  route.query.scope === "review" && auth.canEditCatalog ? "review" : "mine",
);
const suggestOpen = ref(false);
const toReview = ref(0);
const list = usePagination((params: Record<string, unknown>) =>
  proposalApi.list({ ...params, scope: scope.value }),
);

function date(iso: string): string {
  return new Date(iso).toLocaleDateString(lang.value, { dateStyle: "medium" });
}

async function reload() {
  list.goToPage(1);
  await list.load().catch((e) => notifyError(e));
  if (auth.canEditCatalog) {
    toReview.value = await proposalApi.reviewCount().catch(() => 0);
  }
}

function onPage(p: number) {
  list.goToPage(p);
  list.load().catch((e) => notifyError(e));
}

onMounted(reload);
</script>
