<template>
  <div class="flex flex-col">
    <template v-for="(rel, idx) in sortedRelations" :key="rel.id">
      <!-- Relation type separator (shown once per group) -->
      <div
        v-if="isFirstOfType(idx)"
        class="flex items-center gap-2 mt-2 first:mt-0 mb-0.5 px-2"
      >
        <Tag :value="rel.relation_type?.name ?? '?'" class="text-[10px]" />
        <span class="text-xs text-gray-400 dark:text-gray-500"
          >({{ typeCount(rel.relation_type_id) }})</span
        >
        <div class="flex-1 border-b border-gray-100 dark:border-gray-700"></div>
      </div>

      <div class="flex flex-col">
        <div
          class="flex items-center gap-3 px-2 py-2 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700 group/rel transition-colors"
        >
          <!-- Direction icon (SVG) -->
          <svg
            class="w-4 h-4 shrink-0"
            :class="isOutgoing(rel) ? 'text-emerald-500' : 'text-blue-500'"
            viewBox="0 0 16 16"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
          >
            <template v-if="isOutgoing(rel)">
              <line x1="2" y1="8" x2="14" y2="8" />
              <polyline points="10,4 14,8 10,12" />
            </template>
            <template v-else>
              <line x1="2" y1="8" x2="14" y2="8" />
              <polyline points="6,4 2,8 6,12" />
            </template>
          </svg>

          <!-- Linked item type -->
          <Tag
            :value="linkedTypeName(rel)"
            severity="secondary"
            class="text-[11px]"
          />

          <!-- Order badge -->
          <span
            v-if="orderVal(rel) != null"
            class="bg-gray-200 dark:bg-gray-600 text-gray-600 dark:text-gray-300 text-[11px] font-semibold font-mono rounded px-1.5 py-0.5 shrink-0"
            title="Ordre"
          >
            #{{ orderVal(rel) }}
          </span>

          <!-- Linked item label (clickable) -->
          <router-link
            :to="`/items/${linkedId(rel)}`"
            class="text-indigo-600 dark:text-indigo-400 hover:underline font-medium text-sm"
          >
            {{ linkedLabel(rel) }}
          </router-link>

          <!-- Attributes pills -->
          <div class="flex gap-1.5 flex-wrap items-center ml-auto">
            <span
              v-for="(val, key) in displayAttrs(rel)"
              :key="String(key)"
              class="bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-300 text-[11px] rounded-full px-2 py-0.5"
            >
              {{ attrLabel(rel, String(key)) }}: {{ displayValue(val) }}
            </span>
          </div>

          <!-- Actions -->
          <div class="flex gap-0.5 shrink-0">
            <Button
              v-tooltip="'Sous-relations'"
              :icon="
                expanded.has(rel.id) ? 'pi pi-chevron-up' : 'pi pi-chevron-down'
              "
              text
              size="small"
              rounded
              @click.stop="toggle(rel)"
            />
            <Button
              v-if="showDelete"
              v-tooltip="'Modifier'"
              icon="pi pi-pencil"
              text
              size="small"
              rounded
              aria-label="Modifier la relation"
              class="opacity-0 group-hover/rel:opacity-100 transition-opacity"
              @click.stop="$emit('edit', rel)"
            />
            <Button
              v-if="showDelete"
              v-tooltip="'Supprimer'"
              icon="pi pi-trash"
              text
              size="small"
              rounded
              severity="danger"
              aria-label="Supprimer la relation"
              class="opacity-0 group-hover/rel:opacity-100 transition-opacity"
              @click.stop="$emit('delete', rel)"
            />
          </div>
        </div>

        <!-- Sub-relations (recursive) -->
        <div
          v-if="expanded.has(rel.id)"
          class="ml-6 pl-3 border-l-2 border-gray-200 dark:border-gray-600"
        >
          <div
            v-if="loading.has(rel.id)"
            class="py-2 text-xs text-gray-400 dark:text-gray-500"
          >
            <i class="pi pi-spin pi-spinner mr-1"></i> Chargement…
          </div>
          <div
            v-else-if="!children[rel.id]?.length"
            class="py-2 text-xs text-gray-400 dark:text-gray-500 italic"
          >
            Aucune sous-relation sortante.
          </div>
          <RelationTree
            v-else
            :relations="children[rel.id] ?? []"
            :context-item-id="linkedId(rel)"
            :show-delete="false"
            :max-depth="maxDepth - 1"
          />
        </div>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from "vue";
import Tag from "primevue/tag";
import Button from "primevue/button";
import { itemApi } from "@/api/client";
import { useLang } from "@/composables/useLang";
import { useNotify } from "@/composables/useNotify";
import {
  displayValue as displayAttr,
  fieldLabel,
  itemLabel as renderItemLabel,
  orderedKeys,
} from "@/lib/schema";
import type { Item, ItemRelation } from "@/api/types";

const props = withDefaults(
  defineProps<{
    relations: ItemRelation[];
    contextItemId: string;
    showDelete?: boolean;
    maxDepth?: number;
  }>(),
  { showDelete: true, maxDepth: 5 },
);

defineEmits<{
  delete: [rel: ItemRelation];
  edit: [rel: ItemRelation];
}>();

const { lang } = useLang();
const { notifyError } = useNotify();
const expanded = ref(new Set<string>());
const children = ref<Record<string, ItemRelation[]>>({});
const loading = ref(new Set<string>());

function isOutgoing(rel: ItemRelation): boolean {
  return rel.source_item_id === props.contextItemId;
}

function linkedId(rel: ItemRelation): string {
  return isOutgoing(rel) ? rel.target_item_id : rel.source_item_id;
}

function linkedTypeName(rel: ItemRelation): string {
  const linked = isOutgoing(rel) ? rel.target_item : rel.source_item;
  return linked?.item_type?.name ?? "?";
}

function linkedLabel(rel: ItemRelation): string {
  const linked = isOutgoing(rel) ? rel.target_item : rel.source_item;
  if (!linked) return "?";
  return computeLabel(linked);
}

function computeLabel(i: Item): string {
  return renderItemLabel(i, lang.value);
}

function displayValue(value: unknown): string {
  return displayAttr(value, lang.value);
}

// Convention : l'attribut de relation `order` (entier) sert au tri et s'affiche en badge.
const ORDER_KEY = "order";

function orderVal(rel: ItemRelation): unknown {
  return rel.attributes?.[ORDER_KEY] ?? null;
}

function displayAttrs(rel: ItemRelation): Record<string, unknown> {
  const schema = rel.relation_type?.attributes_schema as
    Record<string, unknown> | undefined;
  const result: Record<string, unknown> = {};
  for (const k of orderedKeys(schema, rel.attributes)) {
    if (k === ORDER_KEY || !(k in (rel.attributes ?? {}))) continue;
    result[k] = rel.attributes[k];
  }
  return result;
}

function attrLabel(rel: ItemRelation, key: string): string {
  const schema = rel.relation_type?.attributes_schema as
    Record<string, unknown> | undefined;
  return fieldLabel(schema, key, lang.value);
}

const sortedRelations = computed(() => {
  return [...props.relations].sort((a, b) => {
    // 1. Group by relation type name
    const rtA = a.relation_type?.name ?? "";
    const rtB = b.relation_type?.name ?? "";
    const rtCmp = rtA.localeCompare(rtB);
    if (rtCmp !== 0) return rtCmp;
    // 2. Then by order attribute
    const oa = orderVal(a);
    const ob = orderVal(b);
    if (oa != null && ob != null) return Number(oa) - Number(ob);
    if (oa != null) return -1;
    if (ob != null) return 1;
    return a.created_at.localeCompare(b.created_at);
  });
});

function isFirstOfType(idx: number): boolean {
  if (idx === 0) return true;
  const sorted = sortedRelations.value;
  return sorted[idx]?.relation_type_id !== sorted[idx - 1]?.relation_type_id;
}

function typeCount(relationTypeId: string): number {
  return sortedRelations.value.filter(
    (r) => r.relation_type_id === relationTypeId,
  ).length;
}

async function toggle(rel: ItemRelation) {
  if (expanded.value.has(rel.id)) {
    expanded.value.delete(rel.id);
    expanded.value = new Set(expanded.value);
    return;
  }
  if (props.maxDepth <= 0) return;
  expanded.value.add(rel.id);
  expanded.value = new Set(expanded.value);
  if (children.value[rel.id]) return;

  const lid = linkedId(rel);
  loading.value.add(rel.id);
  loading.value = new Set(loading.value);
  try {
    const rels = await itemApi.relations(lid);
    children.value[rel.id] = (rels ?? []).filter(
      (r) =>
        r.source_item_id === lid && r.target_item_id !== props.contextItemId,
    );
  } catch (e) {
    children.value[rel.id] = [];
    notifyError(e, "Sous-relations indisponibles");
  } finally {
    loading.value.delete(rel.id);
    loading.value = new Set(loading.value);
  }
}
</script>
