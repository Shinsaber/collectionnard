<template>
  <ElDialog
    :model-value="visible"
    :title="item ? 'Modifier l\'item' : 'Nouvel item'"
    width="34rem"
    @update:model-value="$emit('update:visible', $event)"
  >
    <div class="flex flex-col gap-4">
      <!-- Type selector (create mode only) -->
      <div v-if="!item" class="flex flex-col gap-1">
        <label for="item-type" class="text-sm font-medium">Type</label>
        <ElSelect
          id="item-type"
          v-model="form.item_type_id"
          placeholder="Choisir un type..."
          @change="form.attributes = {}"
        >
          <ElOption
            v-for="t in itemTypes"
            :key="t.id"
            :label="t.name"
            :value="t.id"
          />
        </ElSelect>
      </div>
      <div v-else class="text-sm text-gray-500 dark:text-gray-400">
        Type : <strong>{{ item.item_type?.name }}</strong>
      </div>

      <DynamicForm
        v-if="currentSchema"
        ref="dynamicFormRef"
        v-model="form.attributes"
        :schema="currentSchema"
      />

      <div class="flex flex-col gap-1">
        <label for="item-tags" class="text-sm font-medium"
          >Tags (séparés par des virgules)</label
        >
        <ElInput
          id="item-tags"
          v-model="form.tagsText"
          placeholder="drama, thriller, ..."
        />
      </div>
    </div>
    <template #footer>
      <ElButton text @click="$emit('update:visible', false)">Annuler</ElButton>
      <ElButton type="primary" :loading="saving" @click="save">
        <i class="pi pi-check mr-2" aria-hidden="true"></i>
        {{ item ? "Enregistrer" : "Créer" }}
      </ElButton>
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
import { reactive, computed, watch, ref } from "vue";
import { itemApi, extractApiError } from "@/api/client";
import { useNotify } from "@/composables/useNotify";
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
  const t = props.itemTypes.find((it) => it.id === form.item_type_id);
  return t?.attributes_schema ?? null;
});

// Reset form when dialog opens or item changes
watch(
  () => props.visible,
  (open) => {
    if (!open) return;
    if (props.item) {
      form.item_type_id = props.item.item_type_id;
      form.attributes = { ...props.item.attributes };
      form.tagsText = (props.item.tags ?? []).map((t) => t.name).join(", ");
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
      .map((t) => t.trim())
      .filter(Boolean);

    if (props.item) {
      await itemApi.update(props.item.id, {
        attributes: form.attributes,
        tags,
      });
      notifySuccess("Item modifié");
    } else {
      await itemApi.create({
        item_type_id: form.item_type_id,
        attributes: form.attributes,
        tags,
      });
      notifySuccess("Item créé");
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
