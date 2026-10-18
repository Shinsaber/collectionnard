<template>
  <section
    v-if="quota && !quota.unlimited"
    class="flex flex-col gap-2 rounded-2xl bg-surface p-4 shadow-card"
    data-test="proposal-quota"
  >
    <h2 class="font-bold">{{ t("proposals.quota.title") }}</h2>
    <div class="flex flex-col gap-1">
      <span class="text-sm">{{
        t("proposals.quota.waiting", {
          waiting: quota.waiting,
          limit: quota.limit,
        })
      }}</span>
      <ElProgress
        :percentage="percent(quota.waiting, quota.limit)"
        :show-text="false"
        :stroke-width="8"
        :color="full ? 'var(--c-danger)' : 'var(--c-primary)'"
        :aria-label="
          t('proposals.quota.waiting', {
            waiting: quota.waiting,
            limit: quota.limit,
          })
        "
      />
      <span class="text-xs text-muted">{{
        t("proposals.quota.rule", {
          base: quota.base,
          accepted: quota.accepted,
          max: quota.max,
        })
      }}</span>
    </div>
    <span class="text-xs text-muted">{{
      t("proposals.quota.today", {
        today: quota.today,
        limit: quota.daily_limit,
      })
    }}</span>
    <p v-if="full" class="text-sm text-danger" role="status">
      {{ t("proposals.quota.full") }}
    </p>
  </section>
</template>

<script setup lang="ts">
// Quota de propositions d'un membre (P4-3b) : en attente sur la limite (qui grandit avec les
// propositions acceptées) et créations des 24 dernières heures ; masqué sans quota (modérateurs).
import { computed, onMounted, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { proposalApi } from "@/api/client";
import { percent } from "@/lib/stats";
import type { ProposalQuota } from "@/api/types";

/** Change après un envoi ou un retrait : le quota est relu. */
const props = defineProps<{ version?: number }>();
const { t } = useI18n();
const quota = ref<ProposalQuota | null>(null);
const full = computed(
  () =>
    !!quota.value &&
    (quota.value.waiting >= quota.value.limit ||
      quota.value.today >= quota.value.daily_limit),
);

async function load() {
  // Le quota n'est qu'une indication : une erreur ici ne gêne pas la page.
  quota.value = await proposalApi.quota().catch(() => null);
}

onMounted(load);
watch(() => props.version, load);
</script>
