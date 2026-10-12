<template>
  <ElDialog
    :model-value="visible"
    :title="t('admin.external.edit.title', { name: title })"
    width="36rem"
    @update:model-value="emit('update:visible', $event)"
  >
    <p class="text-sm text-gray-500 dark:text-gray-400 mb-3">
      {{ t("admin.external.edit.help") }}
    </p>
    <DynamicForm
      v-if="schema"
      ref="formRef"
      v-model="attributes"
      :schema="schema"
    />
    <template #footer>
      <ElButton v-if="corrected" text @click="restore">{{
        t("admin.external.edit.restore")
      }}</ElButton>
      <ElButton text @click="emit('update:visible', false)">{{
        t("common.cancel")
      }}</ElButton>
      <ElButton type="primary" @click="save">
        <i class="pi pi-check mr-2" aria-hidden="true"></i>
        {{ t("admin.external.edit.apply") }}
      </ElButton>
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
// Correction d'un nœud de l'aperçu avant la simulation (ex. tome sans numéro) : formulaire
// du type (DynamicForm), rien n'est écrit ici — la correction part avec la simulation.
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useLang } from "@/composables/useLang";
import { useMetaTypes } from "@/stores/metaTypes";
import { itemLabel } from "@/lib/schema";
import DynamicForm from "@/components/common/DynamicForm.vue";
import type { ExternalNode } from "@/api/types";

const props = defineProps<{
  visible: boolean;
  node: ExternalNode | null;
  /** Attributs actuels du nœud (correction précédente ou proposition des sources). */
  current: Record<string, unknown>;
  /** Le nœud porte déjà une correction. */
  corrected: boolean;
}>();
const emit = defineEmits<{
  "update:visible": [open: boolean];
  save: [key: string, attributes: Record<string, unknown>];
  restore: [key: string];
}>();

const { t } = useI18n();
const { lang } = useLang();
const metaTypes = useMetaTypes();
const formRef = ref<InstanceType<typeof DynamicForm> | null>(null);
const attributes = ref<Record<string, unknown>>({});

const itemType = computed(() =>
  metaTypes.itemTypes.find((it) => it.id === props.node?.item_type_id),
);
const schema = computed(
  () =>
    itemType.value?.attributes_schema as Record<string, unknown> | undefined,
);
const title = computed(() =>
  props.node
    ? itemLabel(
        {
          id: props.node.key,
          attributes: props.current,
          item_type: itemType.value,
        },
        lang.value,
      )
    : "",
);

watch(
  () => props.visible,
  (open) => {
    // Copie profonde : le formulaire ne doit pas modifier l'aperçu tant qu'on n'applique pas.
    if (open) attributes.value = JSON.parse(JSON.stringify(props.current));
  },
);

function save() {
  if (!props.node || (formRef.value && !formRef.value.validate())) return;
  emit("save", props.node.key, attributes.value);
  emit("update:visible", false);
}

function restore() {
  if (!props.node) return;
  emit("restore", props.node.key);
  emit("update:visible", false);
}
</script>
