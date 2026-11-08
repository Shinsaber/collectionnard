<template>
  <!-- Un prêt (P7-3). Côté prêteur : changer le retour prévu, marquer rendu, supprimer ; côté emprunteur : lecture. -->
  <li
    class="flex flex-col gap-2 rounded-xl bg-soft px-3 py-2.5 sm:flex-row sm:items-center"
    data-test="loan"
  >
    <div class="flex min-w-0 flex-1 items-center gap-3">
      <ItemCover
        v-if="loan.item"
        :item="loan.item"
        :label="itemLabel(loan.item, lang)"
        :width="40"
      />
      <div class="flex min-w-0 flex-1 flex-col gap-0.5">
        <RouterLink
          v-if="loan.item"
          :to="`/items/${loan.item.id}`"
          class="truncate font-bold text-ink hover:text-link"
          >{{ itemLabel(loan.item, lang) }}</RouterLink
        >
        <span class="text-sm">
          <template v-if="borrowerSide">{{
            t("loans.by", { name: ownerName })
          }}</template>
          <template v-else>{{
            t("loans.to", { name: borrowerName(loan) })
          }}</template>
          <span v-if="loan.quantity > 1" class="font-mono text-xs text-muted">
            · ×{{ loan.quantity }}</span
          >
          <span
            v-if="loan.collection_name"
            class="font-mono text-[11px] uppercase text-muted"
          >
            · {{ loan.collection_name }}</span
          >
        </span>
        <span class="font-mono text-[11px] text-muted">
          {{ t("loans.lentOn", { date: formatDay(loan.lent_at, lang) }) }}
          <template v-if="loan.returned_at">
            ·
            {{
              t("loans.returnedOn", { date: formatDay(loan.returned_at, lang) })
            }}
          </template>
          <template v-else-if="loan.due_at">
            · {{ t("loans.dueOn", { date: formatDay(loan.due_at, lang) }) }}
          </template>
        </span>
        <span
          v-if="loan.overdue"
          class="self-start rounded-full bg-danger/10 px-2 py-0.5 font-mono text-[11px] text-danger"
          data-test="loan-overdue"
          >{{ t("loans.overdue") }}</span
        >
        <p v-if="loan.note && !borrowerSide" class="text-xs text-muted">
          {{ loan.note }}
        </p>
      </div>
    </div>
    <div
      v-if="!borrowerSide"
      class="flex flex-wrap items-center gap-2 sm:justify-end"
    >
      <template v-if="!loan.returned_at">
        <ElInput
          :model-value="loan.due_at ?? ''"
          type="date"
          size="small"
          class="w-40!"
          :min="loan.lent_at"
          :aria-label="t('loans.dueAt')"
          data-test="loan-due-edit"
          @change="(v: string) => save({ due_at: v || undefined })"
        />
        <ElButton
          size="small"
          type="primary"
          data-test="loan-return"
          @click="save({ returned_at: today() }, 'loans.returnedDone')"
          >{{ t("loans.markReturned") }}</ElButton
        >
      </template>
      <ElButton
        text
        size="small"
        :aria-label="t('loans.delete')"
        data-test="loan-delete"
        @click="remove"
      >
        <i class="pi pi-trash" aria-hidden="true"></i>
      </ElButton>
    </div>
  </li>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { loanApi } from "@/api/client";
import { useConfirmDialog } from "@/composables/useConfirmDialog";
import { useLang } from "@/composables/useLang";
import { useNotify } from "@/composables/useNotify";
import { borrowerName, formatDay, today } from "@/lib/loans";
import { itemLabel } from "@/lib/schema";
import ItemCover from "@/components/item/ItemCover.vue";
import type { Loan, LoanUpdate } from "@/api/types";

const props = defineProps<{
  loan: Loan;
  /** Vu par l'emprunteur : le prêteur nommé, aucune action. */
  borrowerSide?: boolean;
}>();
const emit = defineEmits<{ changed: [] }>();

const { t } = useI18n();
const { lang } = useLang();
const { notifyError, notifySuccess } = useNotify();
const { confirm } = useConfirmDialog();

const ownerName = computed(
  () => props.loan.owner?.display_name || props.loan.owner?.username || "",
);

/** Remplace les champs modifiables du prêt (PUT), en gardant ceux qu'on ne touche pas. */
async function save(change: Partial<LoanUpdate>, done = "loans.dueSaved") {
  const l = props.loan;
  try {
    await loanApi.update(l.id, {
      quantity: l.quantity,
      lent_at: l.lent_at,
      due_at: l.due_at,
      returned_at: l.returned_at,
      note: l.note,
      ...change,
    });
    notifySuccess(t(done));
    emit("changed");
  } catch (e) {
    notifyError(e);
  }
}

async function remove() {
  const ok = await confirm({
    message: t("loans.deleteConfirm"),
    confirmText: t("loans.delete"),
    danger: true,
  });
  if (!ok) return;
  try {
    await loanApi.remove(props.loan.id);
    notifySuccess(t("loans.deleted"));
    emit("changed");
  } catch (e) {
    notifyError(e);
  }
}
</script>
