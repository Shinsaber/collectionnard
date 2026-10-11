<template>
  <ElDialog
    :model-value="visible"
    :title="title"
    width="34rem"
    @update:model-value="$emit('update:visible', $event)"
  >
    <div v-if="node" class="flex flex-col gap-4">
      <div class="flex items-center gap-2">
        <ElTag type="info" size="small">{{ node.type }}</ElTag>
        <span
          v-if="node.depth === 0"
          class="text-xs text-gray-400 dark:text-gray-500"
          >(racine)</span
        >
      </div>

      <div v-if="image" class="flex justify-center">
        <img
          :src="image"
          :alt="title"
          class="max-h-48 rounded-lg shadow-sm object-contain"
        />
      </div>

      <dl class="grid grid-cols-1 md:grid-cols-2 gap-2 text-sm">
        <div v-for="attr in attributes" :key="attr.key" class="flex flex-col">
          <dt class="text-gray-400 dark:text-gray-500 text-xs">
            {{ attr.label }}
          </dt>
          <dd class="dark:text-gray-200">{{ attr.text }}</dd>
        </div>
      </dl>
    </div>

    <template #footer>
      <ElButton text @click="$emit('update:visible', false)">Fermer</ElButton>
      <ElButton type="primary" @click="$emit('open', node!.id)">
        <i class="pi pi-external-link mr-2" aria-hidden="true"></i> Voir la
        fiche
      </ElButton>
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useLang } from "@/composables/useLang";
import {
  defaultImage,
  displayValue,
  fieldLabel,
  isImageGallery,
  orderedKeys,
  type Schema,
} from "@/lib/schema";
import type { GraphNode } from "@/api/types";

// Détail d'un nœud du graphe : image par défaut et attributs (hors galeries),
// libellés via les x-labels du schéma de l'ItemType.
const props = defineProps<{
  visible: boolean;
  node: GraphNode | null;
  schema: Schema | undefined;
  title: string;
}>();

defineEmits<{
  "update:visible": [value: boolean];
  open: [itemId: string];
}>();

const { lang } = useLang();

const image = computed(() =>
  props.node ? defaultImage(props.schema, props.node.attributes) : "",
);

const attributes = computed(() => {
  if (!props.node) return [];
  const attrs = props.node.attributes;
  return orderedKeys(props.schema, attrs)
    .filter((k) => k in attrs && !isImageGallery(props.schema, k))
    .map((k) => {
      const value = attrs[k];
      let text = displayValue(value, lang.value);
      if (text === "" && value != null && typeof value === "object") {
        text = JSON.stringify(value);
      }
      return {
        key: k,
        label: fieldLabel(props.schema, k, lang.value),
        text: text || "—",
      };
    });
});
</script>
