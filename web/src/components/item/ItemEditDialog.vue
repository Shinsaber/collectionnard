<template>
  <Dialog
    :visible="visible"
    @update:visible="$emit('update:visible', $event)"
    :header="item ? 'Modifier l\'item' : 'Nouvel item'"
    modal
    class="w-full max-w-lg"
  >
    <div class="flex flex-col gap-4">
      <!-- Type selector (create mode only) -->
      <div class="flex flex-col gap-1" v-if="!item">
        <label class="text-sm font-medium">Type</label>
        <Select
          v-model="form.item_type_id"
          :options="itemTypes"
          optionLabel="name"
          optionValue="id"
          placeholder="Choisir un type..."
          @change="form.attributes = {}"
        />
      </div>
      <div v-else class="text-sm text-gray-500 dark:text-gray-400">
        Type : <strong>{{ item.item_type?.name }}</strong>
      </div>

      <DynamicForm
        ref="dynamicFormRef"
        v-if="currentSchema"
        :schema="currentSchema"
        v-model="form.attributes"
      />

      <div class="flex flex-col gap-1">
        <label class="text-sm font-medium"
          >Tags (séparés par des virgules)</label
        >
        <InputText v-model="form.tagsText" placeholder="drame, thriller, ..." />
      </div>
    </div>
    <template #footer>
      <Button label="Annuler" text @click="$emit('update:visible', false)" />
      <Button
        :label="item ? 'Enregistrer' : 'Créer'"
        icon="pi pi-check"
        :loading="saving"
        @click="save"
      />
    </template>
  </Dialog>
</template>

<script setup lang="ts">
import { reactive, computed, watch, ref } from "vue";
import Dialog from "primevue/dialog";
import Button from "primevue/button";
import InputText from "primevue/inputtext";
import Select from "primevue/select";
import { useToast } from "primevue/usetoast";
import { itemApi, extractApiError } from "@/api/client";
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

const toast = useToast();
const saving = ref(false);
const dynamicFormRef = ref<InstanceType<typeof DynamicForm> | null>(null);

const form = reactive({
  item_type_id: "",
  attributes: {} as Record<string, unknown>,
  tagsText: "",
});

const currentSchema = computed(() => {
  const typeId = props.item ? props.item.item_type_id : form.item_type_id;
  if (props.item?.item_type?.attributes_schema && props.item) {
    return props.item.item_type.attributes_schema;
  }
  const t = props.itemTypes.find((it) => it.id === typeId);
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
      toast.add({ severity: "success", summary: "Item modifié", life: 3000 });
    } else {
      await itemApi.create({
        item_type_id: form.item_type_id,
        attributes: form.attributes,
        tags,
      });
      toast.add({ severity: "success", summary: "Item créé", life: 3000 });
    }
    emit("update:visible", false);
    emit("saved");
  } catch (e: unknown) {
    const apiErr = extractApiError(e);
    if (apiErr.fieldErrors && dynamicFormRef.value) {
      dynamicFormRef.value.setErrors(apiErr.fieldErrors);
    }
    toast.add({
      severity: "error",
      summary: "Erreur",
      detail: apiErr.message,
      life: 5000,
    });
  } finally {
    saving.value = false;
  }
}
</script>
