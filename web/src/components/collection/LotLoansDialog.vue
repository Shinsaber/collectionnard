<template>
  <!-- Prêts d'un lot (P7-3) : en cours, puis l'historique des prêts rendus. -->
  <ElDialog
    v-model="visible"
    :title="t('loans.loansTitle', { label })"
    width="40rem"
    @open="open"
    @closed="changed && emit('changed')"
  >
    <AppSpinner v-if="loading" />
    <p v-else-if="loans.length === 0" class="text-sm text-muted">
      {{ t("loans.noneLot") }}
    </p>
    <div v-else class="flex flex-col gap-4">
      <section v-for="g in groups" :key="g.key" class="flex flex-col gap-2">
        <h3 class="text-sm font-bold">{{ g.title }}</h3>
        <ul class="flex flex-col gap-2">
          <LoanCard
            v-for="l in g.list"
            :key="l.id"
            :loan="l"
            @changed="onChanged"
          />
        </ul>
      </section>
    </div>
    <template #footer>
      <ElButton @click="visible = false">{{ t("common.close") }}</ElButton>
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { loanApi } from "@/api/client";
import { useNotify } from "@/composables/useNotify";
import AppSpinner from "@/components/common/AppSpinner.vue";
import LoanCard from "@/components/loans/LoanCard.vue";
import type { CollectionItem, Loan } from "@/api/types";

const props = defineProps<{
  collectionId: string;
  lot: CollectionItem | null;
  label: string;
}>();
/** À la fermeture, si un prêt a changé : la liste de la collection se recharge (drapeaux). */
const emit = defineEmits<{ changed: [] }>();
const visible = defineModel<boolean>("visible", { required: true });

const { t } = useI18n();
const { notifyError } = useNotify();
const loans = ref<Loan[]>([]);
const loading = ref(false);
const changed = ref(false);

const groups = computed(() =>
  [
    {
      key: "active",
      title: t("loans.active"),
      list: loans.value.filter((l) => !l.returned_at),
    },
    {
      key: "history",
      title: t("loans.history"),
      list: loans.value.filter((l) => l.returned_at),
    },
  ].filter((g) => g.list.length > 0),
);

function open() {
  changed.value = false;
  load();
}

async function load() {
  if (!props.lot) return;
  loading.value = true;
  try {
    loans.value = await loanApi.listLot(props.collectionId, props.lot.id);
  } catch (e) {
    notifyError(e);
  } finally {
    loading.value = false;
  }
}

function onChanged() {
  changed.value = true;
  load();
}
</script>
