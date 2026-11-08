<template>
  <!-- Scinder un lot (P7-1) : des exemplaires en sortent pour former un nouveau lot (une carte abîmée). -->
  <ElDialog
    v-model="visible"
    :title="t('collections.lots.splitTitle')"
    width="34rem"
    @open="reset"
  >
    <div v-if="lot" class="flex flex-col gap-4">
      <p class="text-sm text-muted">
        {{ t("collections.lots.splitHelp", { n: lot.quantity }) }}
      </p>
      <div class="flex items-center gap-3">
        <label :for="qtyId" class="text-sm font-medium">{{
          t("collections.lots.copiesToMove")
        }}</label>
        <ElInputNumber
          :id="qtyId"
          v-model="quantity"
          :min="1"
          :max="Math.max(1, lot.quantity - 1)"
          size="small"
          data-test="split-quantity"
        />
      </div>
      <div v-if="schema">
        <h4 class="mb-2 text-sm font-medium">
          {{ t("collections.lots.newLotAttributes") }}
        </h4>
        <DynamicForm ref="formRef" v-model="attributes" :schema="schema" />
      </div>
    </div>
    <template #footer>
      <ElButton text @click="visible = false">{{
        t("common.cancel")
      }}</ElButton>
      <ElButton
        type="primary"
        :loading="saving"
        data-test="split-submit"
        @click="split"
      >
        <i class="pi pi-clone mr-2" aria-hidden="true"></i>
        {{ t("collections.lots.split") }}
      </ElButton>
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
import { ref, useId } from "vue";
import { useI18n } from "vue-i18n";
import { collectionApi } from "@/api/client";
import { useNotify } from "@/composables/useNotify";
import DynamicForm from "@/components/common/DynamicForm.vue";
import type { Schema } from "@/lib/schema";
import type { CollectionItem } from "@/api/types";

const props = defineProps<{
  collectionId: string;
  schema: Schema | undefined;
  /** Lot d'au moins deux exemplaires. */
  lot: CollectionItem | null;
}>();
const emit = defineEmits<{ done: [] }>();
const visible = defineModel<boolean>("visible", { required: true });

const { t } = useI18n();
const { notifyError, notifySuccess } = useNotify();
const qtyId = useId();
const quantity = ref(1);
// Attributs du nouveau lot : ceux du lot d'origine, à corriger (l'état de la carte abîmée).
const attributes = ref<Record<string, unknown>>({});
const formRef = ref<InstanceType<typeof DynamicForm> | null>(null);
const saving = ref(false);

function reset() {
  quantity.value = 1;
  attributes.value = JSON.parse(JSON.stringify(props.lot?.attributes ?? {}));
}

async function split() {
  if (!props.lot) return;
  if (formRef.value && !formRef.value.validate()) return;
  saving.value = true;
  try {
    await collectionApi.splitLot(props.collectionId, props.lot.id, {
      quantity: quantity.value,
      attributes: attributes.value,
    });
    notifySuccess(t("collections.lots.splitDone"));
    visible.value = false;
    emit("done");
  } catch (e) {
    notifyError(e);
  } finally {
    saving.value = false;
  }
}
</script>
