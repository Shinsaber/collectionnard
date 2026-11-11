<template>
  <!-- Ajout (P7-1) : nombre d'exemplaires, et, si l'item est déjà dans la collection, lot qui les reçoit. -->
  <div class="flex flex-col gap-3" data-test="lot-picker">
    <fieldset v-if="lots.length" class="flex flex-col gap-2">
      <legend
        class="mb-2 font-mono text-[11px] uppercase tracking-wider text-muted"
      >
        {{ t("collections.lots.alreadyIn", lots.length) }}
      </legend>
      <label
        v-for="l in lots"
        :key="l.lot_id"
        class="flex min-h-12 cursor-pointer items-center gap-3 rounded-[14px] bg-surface px-3.5 py-2 transition-shadow"
        :class="
          target === l.lot_id
            ? 'shadow-[0_0_0_2px_var(--c-primary)]'
            : 'shadow-card'
        "
        data-test="lot-option"
      >
        <input
          v-model="target"
          type="radio"
          :name="name"
          :value="l.lot_id"
          class="size-5 accent-primary"
        />
        <span class="flex min-w-0 flex-col">
          <span class="text-[15px] font-bold">{{
            t("collections.lots.addTo", { n: l.quantity })
          }}</span>
          <span class="truncate text-xs text-muted">{{
            lotSummary(schema, l.attributes, lang) ||
            t("collections.lots.noAttributes")
          }}</span>
        </span>
      </label>
      <label
        class="flex min-h-12 cursor-pointer items-center gap-3 rounded-[14px] bg-surface px-3.5 transition-shadow"
        :class="
          target === 'new'
            ? 'shadow-[0_0_0_2px_var(--c-primary)]'
            : 'shadow-card'
        "
        data-test="lot-option-new"
      >
        <input
          v-model="target"
          type="radio"
          :name="name"
          value="new"
          class="size-5 accent-primary"
        />
        <span class="text-[15px] font-bold">{{
          t("collections.lots.newLot")
        }}</span>
      </label>
    </fieldset>
    <div class="flex items-center gap-3">
      <label :for="`${name}-qty`" class="text-sm font-medium">{{
        t("collections.lots.copiesToAdd")
      }}</label>
      <ElInputNumber
        :id="`${name}-qty`"
        v-model="quantity"
        :min="1"
        :max="MAX_QUANTITY"
        size="small"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { useId } from "vue";
import { useI18n } from "vue-i18n";
import { useLang } from "@/composables/useLang";
import { lotSummary, type LotTarget } from "@/lib/lots";
import type { Schema } from "@/lib/schema";
import { MAX_QUANTITY, type ItemCollectionMembership } from "@/api/types";

defineProps<{
  /** Lots de l'item dans la collection choisie. */
  lots: ItemCollectionMembership[];
  schema: Schema | undefined;
}>();
const target = defineModel<LotTarget>("target", { required: true });
const quantity = defineModel<number>("quantity", { required: true });

const { t } = useI18n();
const { lang } = useLang();
const name = useId();
</script>
