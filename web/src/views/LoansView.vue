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
        :label="t(`loans.tabs.${side}`)"
        :name="side"
      />
    </ElTabs>
    <ElCheckbox v-model="all" data-test="loans-all" @change="reload">{{
      t("loans.status.all")
    }}</ElCheckbox>
    <AppSpinner v-if="list.loading.value && list.data.value.length === 0" />
    <p
      v-else-if="list.data.value.length === 0"
      class="rounded-2xl bg-surface p-5 text-muted shadow-card"
      data-test="loans-none"
    >
      {{ t(`loans.none.${tab}`) }}
    </p>
    <ul
      v-else
      class="flex flex-col gap-2 rounded-2xl bg-surface p-3 shadow-card"
    >
      <LoanCard
        v-for="l in list.data.value"
        :key="l.id"
        :loan="l"
        :borrower-side="tab === 'borrowed'"
        @changed="reload"
      />
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
// Prêts (P7-3) : ce que j'ai prêté (actions du prêteur) et ce que mes amis m'ont prêté (lecture). L'onglet suit
// `?tab=borrowed`, cible des notifications d'emprunt.
import { computed, onMounted, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute, useRouter } from "vue-router";
import { loanApi } from "@/api/client";
import { useNotify } from "@/composables/useNotify";
import { usePagination } from "@/composables/usePagination";
import AppSpinner from "@/components/common/AppSpinner.vue";
import PageHeader from "@/components/common/PageHeader.vue";
import LoanCard from "@/components/loans/LoanCard.vue";

const SIDES = ["lent", "borrowed"] as const;
type Side = (typeof SIDES)[number];

const { t } = useI18n();
const { notifyError } = useNotify();
const route = useRoute();
const router = useRouter();

const tab = computed<Side>({
  get: () => (route.query.tab === "borrowed" ? "borrowed" : "lent"),
  set: (v) => {
    router.replace({ query: v === "lent" ? {} : { tab: v } });
  },
});
/** Sans cette case : les prêts en cours seulement. */
const all = ref(false);

const list = usePagination((params: Record<string, unknown>) => {
  const p = { ...params, ...(all.value ? {} : { status: "active" as const }) };
  return tab.value === "borrowed" ? loanApi.borrowings(p) : loanApi.mine(p);
});

function reload() {
  list.load().catch(notifyError);
}

function onPage(p: number) {
  list.goToPage(p);
  reload();
}

watch(tab, () => {
  list.goToPage(1);
  reload();
});
onMounted(reload);
</script>
