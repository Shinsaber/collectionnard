<template>
  <!-- Modifier un lot (P7-1) : quantité et attributs contextuels. -->
  <ElDialog
    v-model="visible"
    :title="t('collections.detail.editAttributes')"
    width="34rem"
    @open="reset"
  >
    <div class="flex flex-col gap-4">
      <div class="flex items-center gap-3">
        <label :for="qtyId" class="text-sm font-medium">{{
          t("collections.lots.quantity")
        }}</label>
        <ElInputNumber
          :id="qtyId"
          v-model="form.quantity"
          :min="1"
          :max="MAX_QUANTITY"
          size="small"
          data-test="lot-quantity"
        />
      </div>
      <DynamicForm
        v-if="schema"
        ref="formRef"
        v-model="form.attributes"
        :schema="schema"
      />
    </div>
    <template #footer>
      <ElButton text @click="visible = false">{{
        t("common.cancel")
      }}</ElButton>
      <ElButton
        type="primary"
        :loading="saving"
        :disabled="!dirty"
        data-test="lot-save"
        @click="save"
      >
        <i class="pi pi-check mr-2" aria-hidden="true"></i>
        {{ t("common.save") }}
      </ElButton>
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
import { ref, useId } from "vue";
import { useI18n } from "vue-i18n";
import { collectionApi } from "@/api/client";
import { useDirty } from "@/composables/useDirty";
import { useNotify } from "@/composables/useNotify";
import DynamicForm from "@/components/common/DynamicForm.vue";
import type { Schema } from "@/lib/schema";
import { MAX_QUANTITY, type CollectionItem } from "@/api/types";

const props = defineProps<{
  collectionId: string;
  schema: Schema | undefined;
  lot: CollectionItem | null;
}>();
const emit = defineEmits<{ saved: [] }>();
const visible = defineModel<boolean>("visible", { required: true });

const { t } = useI18n();
const { notifyError, notifySuccess } = useNotify();
const qtyId = useId();
const form = ref({ quantity: 1, attributes: {} as Record<string, unknown> });
const formRef = ref<InstanceType<typeof DynamicForm> | null>(null);
const saving = ref(false);
// « Enregistrer » reste grisé tant que rien n'a changé.
const { dirty, markClean } = useDirty(() => form.value);

function reset() {
  form.value = {
    quantity: props.lot?.quantity ?? 1,
    attributes: { ...(props.lot?.attributes ?? {}) },
  };
  markClean();
}

async function save() {
  if (!props.lot) return;
  if (formRef.value && !formRef.value.validate()) return;
  saving.value = true;
  try {
    await collectionApi.updateLot(props.collectionId, props.lot.id, {
      attributes: form.value.attributes,
      quantity: form.value.quantity,
    });
    notifySuccess(t("collections.detail.attributesUpdated"));
    visible.value = false;
    emit("saved");
  } catch (e) {
    notifyError(e);
  } finally {
    saving.value = false;
  }
}
</script>
