<template>
  <ElDialog
    :model-value="visible"
    title="Modifier la relation"
    width="32rem"
    @update:model-value="$emit('update:visible', $event)"
  >
    <div v-if="relation" class="flex flex-col gap-4">
      <div
        class="flex items-center gap-2 text-sm text-gray-500 dark:text-gray-400"
      >
        <ElTag size="small" type="info">{{
          relation.relation_type?.name ?? "?"
        }}</ElTag>
        <span>{{ linkedLabel }}</span>
      </div>

      <div v-if="hasFields">
        <span
          class="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2 block"
          >Attributs</span
        >
        <DynamicForm
          ref="formRef"
          :schema="schema!"
          :model-value="attrs"
          @update:model-value="attrs = $event"
        />
      </div>
      <div v-else class="text-sm text-gray-400 dark:text-gray-500 italic">
        Ce type de relation n'a pas d'attributs modifiables.
      </div>
    </div>

    <template #footer>
      <ElButton text @click="$emit('update:visible', false)">Annuler</ElButton>
      <ElButton type="primary" :loading="saving" @click="submit">
        <i class="pi pi-check mr-2" aria-hidden="true"></i> Enregistrer
      </ElButton>
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { relationApi } from "@/api/client";
import { useLang } from "@/composables/useLang";
import { useNotify } from "@/composables/useNotify";
import { itemLabel } from "@/lib/schema";
import DynamicForm from "@/components/common/DynamicForm.vue";
import type { Item, ItemRelation } from "@/api/types";

const props = defineProps<{
  visible: boolean;
  relation: ItemRelation | null;
  /** Item depuis lequel la relation est vue (pour nommer « l'autre » item). */
  item: Item;
}>();

const emit = defineEmits<{
  "update:visible": [value: boolean];
  updated: [];
}>();

const { lang } = useLang();
const { notifyError, notifySuccess } = useNotify();

const attrs = ref<Record<string, unknown>>({});
const saving = ref(false);
const formRef = ref<InstanceType<typeof DynamicForm> | null>(null);

const schema = computed(
  () =>
    (props.relation?.relation_type?.attributes_schema as Record<
      string,
      unknown
    >) ?? null,
);
const hasFields = computed(
  () => !!schema.value && Object.keys(schema.value.properties ?? {}).length > 0,
);
const linkedLabel = computed(() => {
  if (!props.relation) return "";
  const linked =
    props.relation.source_item_id === props.item.id
      ? props.relation.target_item
      : props.relation.source_item;
  return itemLabel(linked, lang.value);
});

watch(
  () => props.visible,
  (open) => {
    if (open) attrs.value = { ...(props.relation?.attributes ?? {}) };
  },
);

async function submit() {
  if (!props.relation) return;
  if (formRef.value && !formRef.value.validate()) return;
  saving.value = true;
  try {
    await relationApi.update(props.relation.id, { attributes: attrs.value });
    notifySuccess("Relation modifiée");
    emit("update:visible", false);
    emit("updated");
  } catch (e) {
    notifyError(e);
  } finally {
    saving.value = false;
  }
}
</script>
