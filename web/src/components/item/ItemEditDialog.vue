<template>
  <ElDialog
    :model-value="visible"
    :title="
      item ? t('items.editDialog.editTitle') : t('items.editDialog.createTitle')
    "
    width="34rem"
    @update:model-value="$emit('update:visible', $event)"
  >
    <div class="flex flex-col gap-4">
      <!-- Type selector (create mode only) -->
      <div v-if="!item" class="flex flex-col gap-1">
        <label for="item-type" class="text-sm font-medium">{{
          t("common.type")
        }}</label>
        <ElSelect
          id="item-type"
          v-model="form.item_type_id"
          :placeholder="t('items.editDialog.chooseType')"
          @change="form.attributes = {}"
        >
          <ElOption
            v-for="type in itemTypes"
            :key="type.id"
            :label="typeTitle(type, lang)"
            :value="type.id"
          />
        </ElSelect>
      </div>
      <div v-else class="text-sm text-gray-500 dark:text-gray-400">
        {{ t("items.editDialog.typeLabel") }}
        <strong>{{ typeTitle(item.item_type, lang) }}</strong>
      </div>

      <DynamicForm
        v-if="currentSchema"
        ref="dynamicFormRef"
        v-model="form.attributes"
        :schema="currentSchema"
      />

      <div class="flex flex-col gap-1">
        <label for="item-tags" class="text-sm font-medium">{{
          t("items.editDialog.tags")
        }}</label>
        <ElInput
          id="item-tags"
          v-model="form.tagsText"
          :placeholder="t('items.editDialog.tagsPlaceholder')"
        />
      </div>
    </div>
    <template #footer>
      <ElButton text @click="$emit('update:visible', false)">{{
        t("common.cancel")
      }}</ElButton>
      <ElButton type="primary" :loading="saving" @click="save">
        <i class="pi pi-check mr-2" aria-hidden="true"></i>
        {{ item ? t("common.save") : t("common.create") }}
      </ElButton>
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
import { reactive, computed, watch, ref } from "vue";
import { useI18n } from "vue-i18n";
import { itemApi, extractApiError } from "@/api/client";
import { useLang } from "@/composables/useLang";
import { useNotify } from "@/composables/useNotify";
import { typeTitle } from "@/lib/schema";
import DynamicForm from "@/components/common/DynamicForm.vue";
import type { Item, ItemType } from "@/api/types";

const props = defineProps<{
  visible: boolean;
  item?: Item | null;
  itemTypes: ItemType[];
}>();

const emit = defineEmits<{
  "update:visible": [value: boolean];
  saved: [];
}>();

const { t } = useI18n();
const { lang } = useLang();
const { notifyError, notifySuccess } = useNotify();
const saving = ref(false);
const dynamicFormRef = ref<InstanceType<typeof DynamicForm> | null>(null);

const form = reactive({
  item_type_id: "",
  attributes: {} as Record<string, unknown>,
  tagsText: "",
});

const currentSchema = computed(() => {
  if (props.item?.item_type?.attributes_schema) {
    return props.item.item_type.attributes_schema;
  }
  const found = props.itemTypes.find((it) => it.id === form.item_type_id);
  return found?.attributes_schema ?? null;
});

// Reset form when dialog opens or item changes
watch(
  () => props.visible,
  (open) => {
    if (!open) return;
    if (props.item) {
      form.item_type_id = props.item.item_type_id;
      form.attributes = { ...props.item.attributes };
      form.tagsText = (props.item.tags ?? []).map((tag) => tag.name).join(", ");
    } else {
      form.item_type_id = "";
      form.attributes = {};
      form.tagsText = "";
    }
  },
);

async function save() {
  if (dynamicFormRef.value && !dynamicFormRef.value.validate()) return;
  saving.value = true;
  try {
    const tags = form.tagsText
      .split(",")
      .map((tag) => tag.trim())
      .filter(Boolean);

    if (props.item) {
      await itemApi.update(props.item.id, {
        attributes: form.attributes,
        tags,
      });
      notifySuccess(t("items.editDialog.updated"));
    } else {
      await itemApi.create({
        item_type_id: form.item_type_id,
        attributes: form.attributes,
        tags,
      });
      notifySuccess(t("items.editDialog.created"));
    }
    emit("update:visible", false);
    emit("saved");
  } catch (e: unknown) {
    const apiErr = extractApiError(e);
    if (apiErr.fieldErrors && dynamicFormRef.value) {
      dynamicFormRef.value.setErrors(apiErr.fieldErrors);
    }
    notifyError(e);
  } finally {
    saving.value = false;
  }
}
</script>
