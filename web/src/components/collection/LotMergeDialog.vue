<template>
  <!-- Fusionner un lot dans un autre lot du même item (P7-1) ; le lot fusionné perd ses attributs. -->
  <ElDialog
    v-model="visible"
    :title="t('collections.lots.mergeTitle')"
    width="34rem"
    @open="load"
  >
    <AppSpinner v-if="loading" />
    <p v-else-if="others.length === 0" class="text-sm italic text-muted">
      {{ t("collections.lots.noOtherLot") }}
    </p>
    <fieldset v-else class="flex flex-col gap-2">
      <legend class="mb-2 text-sm text-muted">
        {{ t("collections.lots.mergeHelp", lot?.quantity ?? 0) }}
      </legend>
      <label
        v-for="l in others"
        :key="l.lot_id"
        class="flex min-h-12 cursor-pointer items-center gap-3 rounded-[14px] bg-surface px-3.5 py-2 transition-shadow"
        :class="
          into === l.lot_id
            ? 'shadow-[0_0_0_2px_var(--c-primary)]'
            : 'shadow-card'
        "
        data-test="merge-option"
      >
        <input
          v-model="into"
          type="radio"
          :name="name"
          :value="l.lot_id"
          class="size-5 accent-primary"
        />
        <span class="flex min-w-0 flex-col">
          <span class="text-[15px] font-bold">{{
            t("collections.lots.copies", l.quantity)
          }}</span>
          <span class="truncate text-xs text-muted">{{
            lotSummary(schema, l.attributes, lang) ||
            t("collections.lots.noAttributes")
          }}</span>
        </span>
      </label>
    </fieldset>
    <template #footer>
      <ElButton text @click="visible = false">{{
        t("common.cancel")
      }}</ElButton>
      <ElButton
        type="primary"
        :loading="saving"
        :disabled="!into"
        data-test="merge-submit"
        @click="merge"
      >
        <i class="pi pi-sitemap mr-2" aria-hidden="true"></i>
        {{ t("collections.lots.merge") }}
      </ElButton>
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
import { computed, ref, useId } from "vue";
import { useI18n } from "vue-i18n";
import { collectionApi, itemApi } from "@/api/client";
import { useLang } from "@/composables/useLang";
import { useNotify } from "@/composables/useNotify";
import { lotSummary, lotsIn } from "@/lib/lots";
import AppSpinner from "@/components/common/AppSpinner.vue";
import type { Schema } from "@/lib/schema";
import type { CollectionItem, ItemCollectionMembership } from "@/api/types";

const props = defineProps<{
  collectionId: string;
  schema: Schema | undefined;
  lot: CollectionItem | null;
}>();
const emit = defineEmits<{ done: [] }>();
const visible = defineModel<boolean>("visible", { required: true });

const { t } = useI18n();
const { lang } = useLang();
const { notifyError, notifySuccess } = useNotify();
const name = useId();
const lots = ref<ItemCollectionMembership[]>([]);
const loading = ref(false);
const into = ref("");
const saving = ref(false);

/** Autres lots du même item dans la collection (ils peuvent être sur une autre page de la liste). */
const others = computed(() =>
  lots.value.filter((l) => l.lot_id !== props.lot?.id),
);

async function load() {
  into.value = "";
  lots.value = [];
  if (!props.lot) return;
  loading.value = true;
  try {
    lots.value = lotsIn(
      await itemApi.collections(props.lot.item_id),
      props.collectionId,
    );
  } catch (e) {
    notifyError(e);
  } finally {
    loading.value = false;
  }
}

async function merge() {
  if (!props.lot || !into.value) return;
  saving.value = true;
  try {
    await collectionApi.mergeLot(props.collectionId, props.lot.id, into.value);
    notifySuccess(t("collections.lots.mergeDone"));
    visible.value = false;
    emit("done");
  } catch (e) {
    notifyError(e);
  } finally {
    saving.value = false;
  }
}
</script>
