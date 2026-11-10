<template>
  <div class="flex flex-col gap-6">
    <PageHeader :title="t('loans.title')">
      <template #subtitle>
        <p class="text-[15px] text-muted">{{ t("loans.help") }}</p>
      </template>
    </PageHeader>
    <ElTabs v-model="tab">
      <ElTabPane
        v-for="side in SIDES"
        :key="side"
        :label="tabLabel(side)"
        :name="side"
      />
    </ElTabs>
    <ReceivedGiftsPanel v-if="tab === 'received'" />
    <template v-else>
      <ElCheckbox v-model="all" data-test="loans-all" @change="reload">{{
        tab === "gifts" ? t("gifts.receivedAll") : t("loans.status.all")
      }}</ElCheckbox>
      <AppSpinner v-if="current.loading.value && empty" />
      <p
        v-else-if="empty"
        class="rounded-2xl bg-surface p-5 text-muted shadow-card"
        data-test="loans-none"
      >
        {{ tab === "gifts" ? t("gifts.none") : t(`loans.none.${tab}`) }}
      </p>
      <ul
        v-else
        class="flex flex-col gap-2 rounded-2xl bg-surface p-3 shadow-card"
      >
        <template v-if="tab === 'gifts'">
          <GiftCard
            v-for="g in gifts.data.value"
            :key="g.id"
            :gift="g"
            @changed="reload"
          />
        </template>
        <template v-else>
          <LoanCard
            v-for="l in loans.data.value"
            :key="l.id"
            :loan="l"
            :borrower-side="tab === 'borrowed'"
            @changed="reload"
          />
        </template>
      </ul>
      <ElPagination
        class="justify-end"
        hide-on-single-page
        layout="prev, pager, next"
        :total="current.total.value"
        :page-size="current.perPage.value"
        :current-page="current.page.value"
        @current-change="onPage"
      />
    </template>
  </div>
</template>

<script setup lang="ts">
// Prêts et cadeaux (P7-3, P7-4) : ce que j'ai prêté (actions du prêteur), ce que mes amis m'ont prêté (lecture), les
// cadeaux que je compte leur offrir, et ceux qu'on m'a offerts (P7-4c). L'onglet suit `?tab=borrowed|gifts|received`,
// cible des notifications.
import { computed, onMounted, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute, useRouter } from "vue-router";
import { giftApi, loanApi } from "@/api/client";
import { useNotify } from "@/composables/useNotify";
import { usePagination } from "@/composables/usePagination";
import AppSpinner from "@/components/common/AppSpinner.vue";
import PageHeader from "@/components/common/PageHeader.vue";
import GiftCard from "@/components/gifts/GiftCard.vue";
import ReceivedGiftsPanel from "@/components/gifts/ReceivedGiftsPanel.vue";
import LoanCard from "@/components/loans/LoanCard.vue";

const SIDES = ["lent", "borrowed", "gifts", "received"] as const;
type Side = (typeof SIDES)[number];

const tabLabel = (side: Side) =>
  side === "gifts"
    ? t("gifts.tab")
    : side === "received"
      ? t("gifts.received.tab")
      : t(`loans.tabs.${side}`);

const { t } = useI18n();
const { notifyError } = useNotify();
const route = useRoute();
const router = useRouter();

const tab = computed<Side>({
  get: () =>
    SIDES.includes(route.query.tab as Side)
      ? (route.query.tab as Side)
      : "lent",
  set: (v) => {
    router.replace({ query: v === "lent" ? {} : { tab: v } });
  },
});
/** Sans cette case : les prêts en cours, les cadeaux pas encore reçus. */
const all = ref(false);

const loans = usePagination((params: Record<string, unknown>) => {
  const p = { ...params, ...(all.value ? {} : { status: "active" as const }) };
  return tab.value === "borrowed" ? loanApi.borrowings(p) : loanApi.mine(p);
});
const gifts = usePagination((params: Record<string, unknown>) =>
  giftApi.mine({
    ...params,
    ...(all.value ? {} : { status: "open" as const }),
  }),
);
const current = computed(() => (tab.value === "gifts" ? gifts : loans));
const empty = computed(() =>
  tab.value === "gifts"
    ? gifts.data.value.length === 0
    : loans.data.value.length === 0,
);

function reload() {
  current.value.load().catch(notifyError);
}

function onPage(p: number) {
  current.value.goToPage(p);
  reload();
}

watch(tab, (side) => {
  if (side === "received") return; // l'onglet « Reçus » charge lui-même
  loans.data.value = [];
  current.value.goToPage(1);
  reload();
});
onMounted(() => {
  if (tab.value !== "received") reload();
});
</script>
