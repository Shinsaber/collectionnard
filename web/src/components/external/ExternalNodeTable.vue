<template>
  <ElTable :data="nodes" size="small" row-key="key">
    <ElTableColumn width="50" align="center">
      <template #default="{ row }">
        <ElCheckbox
          :model-value="selected.has((row as ExternalNode).key)"
          :aria-label="
            t('admin.external.importIt', { name: label(row as ExternalNode) })
          "
          @change="(v) => emit('toggle', (row as ExternalNode).key, !!v)"
        />
      </template>
    </ElTableColumn>
    <ElTableColumn :label="t('admin.external.item')" min-width="220">
      <template #default="{ row }">
        <span class="font-medium">{{ label(row as ExternalNode) }}</span>
        <span
          v-if="relationText(row as ExternalNode)"
          class="ml-2 text-xs text-gray-500 dark:text-gray-400"
          >{{ relationText(row as ExternalNode) }}</span
        >
      </template>
    </ElTableColumn>
    <ElTableColumn :label="t('admin.external.type')" width="120">
      <template #default="{ row }">
        {{ typeName((row as ExternalNode).item_type_id) }}
      </template>
    </ElTableColumn>
    <ElTableColumn :label="t('admin.external.status')" width="190">
      <template #default="{ row }">
        <ElTag size="small" :type="STATUS_TAG[(row as ExternalNode).status]">{{
          t(`admin.external.statuses.${(row as ExternalNode).status}`)
        }}</ElTag>
        <ElTag
          v-if="(row as ExternalNode).note"
          size="small"
          type="warning"
          class="ml-1"
          >{{ t(`admin.external.notes.${(row as ExternalNode).note}`) }}</ElTag
        >
        <ElTag
          v-if="(row as ExternalNode).key in overrides"
          size="small"
          type="primary"
          class="ml-1"
          >{{ t("admin.external.edit.corrected") }}</ElTag
        >
      </template>
    </ElTableColumn>
    <ElTableColumn width="50" align="center">
      <template #default="{ row }">
        <ElButton
          size="small"
          text
          :aria-label="
            t('admin.external.edit.button', {
              name: label(row as ExternalNode),
            })
          "
          data-test="edit-node"
          @click="emit('edit', row as ExternalNode)"
        >
          <i class="pi pi-pencil" aria-hidden="true"></i>
        </ElButton>
      </template>
    </ElTableColumn>
    <ElTableColumn width="50" align="center">
      <template #default="{ row }">
        <a
          :href="(row as ExternalNode).source_url"
          target="_blank"
          rel="noopener noreferrer"
          :aria-label="t('admin.external.openSource')"
          ><i class="pi pi-external-link" aria-hidden="true"></i
        ></a>
      </template>
    </ElTableColumn>
  </ElTable>
</template>

<script setup lang="ts">
// Lignes d'un aperçu d'import externe : case à cocher, libellé (x-label du type), attributs
// de la relation au parent (rôles d'une personne), type, état face au catalogue, source.
import { useI18n } from "vue-i18n";
import { useLang } from "@/composables/useLang";
import { useMetaTypes } from "@/stores/metaTypes";
import { fieldValue, itemLabel, orderedKeys, typeTitle } from "@/lib/schema";
import type { Schema } from "@/lib/schema";
import {
  effectiveAttributes,
  parentRelation,
  type Overrides,
} from "@/lib/externalPlan";
import type {
  ExternalNode,
  ExternalNodeStatus,
  ExternalPlan,
} from "@/api/types";

const props = defineProps<{
  plan: ExternalPlan;
  nodes: ExternalNode[];
  selected: Set<string>;
  /** Corrections faites dans l'aperçu (libellé recalculé, étiquette « corrigé »). */
  overrides: Overrides;
  /** Affiche les attributs de la relation au parent (ex. rôles sur l'œuvre). */
  showRelation?: boolean;
}>();
const emit = defineEmits<{
  toggle: [key: string, on: boolean];
  edit: [node: ExternalNode];
}>();

const STATUS_TAG: Record<ExternalNodeStatus, "success" | "warning" | "info"> = {
  new: "success",
  update: "warning",
  unchanged: "info",
};

const { t } = useI18n();
const { lang } = useLang();
const metaTypes = useMetaTypes();

function itemType(id: string) {
  return metaTypes.itemTypes.find((it) => it.id === id);
}

function typeName(id: string): string {
  return typeTitle(itemType(id), lang.value);
}

function label(node: ExternalNode): string {
  return itemLabel(
    {
      id: node.key,
      attributes: effectiveAttributes(node, props.overrides),
      item_type: itemType(node.item_type_id),
    },
    lang.value,
  );
}

function relationText(node: ExternalNode): string {
  if (!props.showRelation) return "";
  const rel = parentRelation(props.plan, node);
  if (!rel) return "";
  const rt = metaTypes.relationTypes.find((r) => r.id === rel.relation_type_id);
  const schema = rt?.attributes_schema as Schema | undefined;
  return orderedKeys(schema, rel.attributes)
    .map((k) => fieldValue(schema, k, rel.attributes[k], lang.value))
    .filter((s) => s !== "")
    .join(" · ");
}
</script>
