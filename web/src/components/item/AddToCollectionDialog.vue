<template>
  <Dialog
    :visible="visible"
    header="Ajouter à une collection"
    modal
    :style="{ width: '500px' }"
    @update:visible="$emit('update:visible', $event)"
  >
    <div class="flex flex-col gap-4">
      <div
        v-if="availableCollections.length === 0"
        class="text-sm text-gray-400 dark:text-gray-500 italic"
      >
        Aucune collection disponible (toutes contiennent déjà cet item, ou vous
        n'avez pas de collection).
      </div>
      <template v-else>
        <div class="flex flex-col gap-1">
          <label
            for="add-to-collection-select"
            class="text-sm font-medium text-gray-700 dark:text-gray-300"
            >Collection</label
          >
          <Select
            v-model="selectedCollectionId"
            input-id="add-to-collection-select"
            :options="availableCollections"
            option-label="name"
            option-value="id"
            placeholder="Choisir une collection…"
          />
        </div>

        <div
          v-if="
            selectedSchema &&
            Object.keys(selectedSchema.properties ?? {}).length > 0
          "
        >
          <span
            class="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2 block"
            >Attributs contextuels</span
          >
          <DynamicForm
            ref="formRef"
            :schema="selectedSchema"
            :model-value="attrs"
            @update:model-value="attrs = $event"
          />
        </div>
      </template>
    </div>

    <template #footer>
      <Button label="Annuler" text @click="$emit('update:visible', false)" />
      <Button
        label="Ajouter"
        icon="pi pi-plus"
        :disabled="!selectedCollectionId"
        :loading="saving"
        @click="submit"
      />
    </template>
  </Dialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from "vue";
import Dialog from "primevue/dialog";
import Button from "primevue/button";
import Select from "primevue/select";
import { collectionApi } from "@/api/client";
import { useNotify } from "@/composables/useNotify";
import DynamicForm from "@/components/common/DynamicForm.vue";
import type { Collection, Item } from "@/api/types";

const props = defineProps<{
  visible: boolean;
  item: Item;
  /** Collections contenant déjà l'item (exclues de la liste). */
  excludeIds: string[];
}>();

const emit = defineEmits<{
  "update:visible": [value: boolean];
  added: [];
}>();

const { notifyError, notifySuccess } = useNotify();

const collections = ref<Collection[]>([]);
const selectedCollectionId = ref<string | null>(null);
const attrs = ref<Record<string, unknown>>({});
const saving = ref(false);
const formRef = ref<InstanceType<typeof DynamicForm> | null>(null);

const availableCollections = computed(() => {
  const exclude = new Set(props.excludeIds);
  return collections.value.filter((c) => !exclude.has(c.id));
});

const selectedSchema = computed(() => {
  const col = collections.value.find(
    (c) => c.id === selectedCollectionId.value,
  );
  return (
    (col?.collection_type?.attributes_schema as Record<string, unknown>) ?? null
  );
});

watch(
  () => props.visible,
  async (open) => {
    if (!open) return;
    selectedCollectionId.value = null;
    attrs.value = {};
    try {
      collections.value = (await collectionApi.list({ per_page: 100 })).data;
    } catch (e) {
      collections.value = [];
      notifyError(e, "Collections indisponibles");
    }
  },
);

async function submit() {
  if (!selectedCollectionId.value) return;
  if (formRef.value && !formRef.value.validate()) return;
  saving.value = true;
  try {
    await collectionApi.addItem(selectedCollectionId.value, {
      item_id: props.item.id,
      attributes: attrs.value,
    });
    notifySuccess("Item ajouté à la collection");
    emit("update:visible", false);
    emit("added");
  } catch (e) {
    notifyError(e);
  } finally {
    saving.value = false;
  }
}
</script>
