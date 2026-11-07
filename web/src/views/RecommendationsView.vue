<template>
  <div class="flex max-w-3xl flex-col gap-6">
    <PageHeader :title="t('recommendations.title')">
      <template #subtitle>
        <p class="text-[15px] text-muted">
          {{ t("recommendations.subtitle") }}
        </p>
      </template>
    </PageHeader>
    <AppSpinner v-if="list.loading.value && list.data.value.length === 0" />
    <p
      v-else-if="list.data.value.length === 0"
      class="rounded-2xl bg-surface p-5 text-muted shadow-card"
    >
      {{ t("recommendations.none") }}
    </p>
    <ul v-else class="flex flex-col gap-3">
      <li
        v-for="rc in list.data.value"
        :key="rc.id"
        class="flex gap-4 rounded-2xl bg-surface p-4 shadow-card"
        data-test="recommendation"
      >
        <RouterLink
          v-if="rc.item"
          :to="`/items/${rc.item_id}`"
          class="shrink-0"
          :aria-label="t('recommendations.open')"
        >
          <ItemCover :item="rc.item" :label="label(rc)" :width="64" />
        </RouterLink>
        <div class="flex min-w-0 flex-1 flex-col gap-1.5">
          <div class="flex flex-wrap items-center gap-2">
            <RouterLink
              :to="`/items/${rc.item_id}`"
              class="font-bold text-ink"
              data-test="recommendation-item"
              >{{ label(rc) }}</RouterLink
            >
            <span
              v-if="rc.item?.item_type"
              class="font-mono text-[11px] uppercase text-muted"
              >{{ typeTitle(rc.item.item_type, lang) }}</span
            >
            <span
              v-if="!rc.read_at"
              class="rounded-full bg-accent px-2 py-0.5 font-mono text-[10px] font-bold text-accent-ink"
              >{{ t("recommendations.new") }}</span
            >
          </div>
          <p class="flex items-center gap-2 text-[13px] text-muted">
            <UserAvatar :user="rc.sender ?? null" :size="20" />
            {{
              t("recommendations.from", {
                name: rc.sender?.display_name || rc.sender?.username || "",
              })
            }}
            <time class="font-mono text-[11px]" :datetime="rc.created_at">{{
              new Date(rc.created_at).toLocaleDateString(lang)
            }}</time>
          </p>
          <!-- Texte brut : pas de Markdown dans un message privé. -->
          <p
            v-if="rc.message"
            class="text-sm whitespace-pre-line"
            data-test="recommendation-message"
          >
            {{ rc.message }}
          </p>
          <div class="mt-1 flex flex-wrap items-center gap-1">
            <RouterLink
              :to="`/items/${rc.item_id}`"
              class="el-button el-button--small el-button--primary"
              >{{ t("recommendations.open") }}</RouterLink
            >
            <ElButton
              text
              size="small"
              data-test="recommendation-dismiss"
              @click="dismiss(rc)"
              >{{ t("recommendations.dismiss") }}</ElButton
            >
            <BlockButton
              content-kind="recommendation"
              :content-id="rc.id"
              @blocked="reload"
            />
          </div>
        </div>
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
// Recommandations reçues (I-4) : l'item, l'expéditeur et son mot ; ouvrir, écarter ou bloquer l'expéditeur.
// Les nouvelles gardent leur pastille le temps de la visite, puis sont marquées lues.
import { onMounted } from "vue";
import { useI18n } from "vue-i18n";
import { recommendationApi } from "@/api/client";
import { useLang } from "@/composables/useLang";
import { useNotify } from "@/composables/useNotify";
import { usePagination } from "@/composables/usePagination";
import { itemLabel, typeTitle } from "@/lib/schema";
import AppSpinner from "@/components/common/AppSpinner.vue";
import PageHeader from "@/components/common/PageHeader.vue";
import UserAvatar from "@/components/account/UserAvatar.vue";
import ItemCover from "@/components/item/ItemCover.vue";
import BlockButton from "@/components/review/BlockButton.vue";
import type { Recommendation } from "@/api/types";

const { t } = useI18n();
const { lang } = useLang();
const { notifyError, notifySuccess } = useNotify();
const list = usePagination((params: Record<string, unknown>) =>
  recommendationApi.mine(params),
);

const label = (rc: Recommendation) =>
  rc.item ? itemLabel(rc.item, lang.value) : "";

async function reload() {
  try {
    await list.load();
    const unread = list.data.value.filter((rc) => !rc.read_at);
    if (unread.length > 0)
      await recommendationApi.markRead(unread.map((rc) => rc.id));
  } catch (e) {
    notifyError(e);
  }
}

function onPage(p: number) {
  list.goToPage(p);
  reload();
}

async function dismiss(rc: Recommendation) {
  try {
    await recommendationApi.remove(rc.id);
    notifySuccess(t("recommendations.dismissed"));
    await reload();
  } catch (e) {
    notifyError(e);
  }
}

onMounted(reload);
</script>
