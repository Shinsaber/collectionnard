<template>
  <!-- Relations d'un item, groupées par type ; sous-relations dépliables (récursif). -->
  <div class="flex flex-col" :class="nested ? 'gap-2' : 'gap-5'">
    <div
      v-for="group in groups"
      :key="group.typeId"
      class="flex flex-col gap-2"
    >
      <div class="flex items-center gap-2 px-1">
        <h3
          class="font-mono text-[11px] font-medium uppercase tracking-wider text-muted"
        >
          {{ group.title }} · {{ group.relations.length }}
        </h3>
        <RouterLink
          :to="{
            path: '/items',
            query: {
              related_to: contextItemId,
              relation_type_id: group.typeId,
            },
          }"
          class="flex size-7 items-center justify-center rounded-md text-muted hover:bg-soft hover:text-link"
          :title="t('items.search.relatedItems')"
          :aria-label="t('items.search.relatedItems')"
          ><i class="pi pi-list text-xs" aria-hidden="true"></i
        ></RouterLink>
      </div>

      <ul
        :class="
          nested ? '' : 'overflow-hidden rounded-2xl bg-surface shadow-card'
        "
      >
        <li
          v-for="rel in group.relations"
          :key="rel.id"
          class="border-b border-line last:border-b-0"
        >
          <div
            class="group/rel flex flex-wrap items-center gap-x-3 gap-y-1.5 px-3.5 py-2.5 sm:flex-nowrap"
          >
            <!-- Sens du lien : sortant → / entrant ← -->
            <svg
              class="size-4 shrink-0 text-muted"
              viewBox="0 0 16 16"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
              aria-hidden="true"
            >
              <line x1="2" y1="8" x2="14" y2="8" />
              <polyline v-if="isOutgoing(rel)" points="10,4 14,8 10,12" />
              <polyline v-else points="6,4 2,8 6,12" />
            </svg>

            <span
              v-if="orderVal(rel) != null"
              class="shrink-0 font-mono text-xs font-medium text-link"
              :title="t('items.relations.order')"
              >#{{ orderVal(rel) }}</span
            >

            <div class="flex min-w-0 flex-1 flex-col">
              <RouterLink
                :to="`/items/${linkedId(rel)}`"
                class="truncate text-[15px] font-semibold text-ink hover:text-link"
                >{{ linkedLabel(rel) }}</RouterLink
              >
              <span class="font-mono text-[11px] uppercase text-muted">{{
                linkedTypeName(rel)
              }}</span>
            </div>

            <div
              v-if="Object.keys(displayAttrs(rel)).length"
              class="order-last flex w-full flex-wrap gap-1.5 pl-7 sm:order-none sm:w-auto sm:justify-end sm:pl-0"
            >
              <span
                v-for="(val, key) in displayAttrs(rel)"
                :key="String(key)"
                class="rounded-full bg-soft px-2 py-0.5 text-xs"
              >
                {{ attrLabel(rel, String(key)) }} :
                {{ attrValue(rel, String(key), val) }}
              </span>
            </div>

            <div class="flex shrink-0 gap-0.5">
              <ElButton
                text
                circle
                :title="t('items.relations.subRelations')"
                :aria-label="
                  expanded.has(rel.id)
                    ? t('items.relations.collapse')
                    : t('items.relations.expand')
                "
                :aria-expanded="expanded.has(rel.id)"
                @click.stop="toggle(rel)"
              >
                <i
                  :class="
                    expanded.has(rel.id)
                      ? 'pi pi-chevron-up'
                      : 'pi pi-chevron-down'
                  "
                  aria-hidden="true"
                ></i>
              </ElButton>
              <ElButton
                v-if="showDelete"
                text
                circle
                :title="t('common.edit')"
                :aria-label="t('items.relations.editRelation')"
                class="transition-opacity lg:opacity-0 lg:group-hover/rel:opacity-100 lg:focus:opacity-100"
                @click.stop="$emit('edit', rel)"
              >
                <i class="pi pi-pencil" aria-hidden="true"></i>
              </ElButton>
              <ElButton
                v-if="showDelete"
                text
                circle
                type="danger"
                :title="t('common.delete')"
                :aria-label="t('items.relations.deleteRelation')"
                class="transition-opacity lg:opacity-0 lg:group-hover/rel:opacity-100 lg:focus:opacity-100"
                @click.stop="$emit('delete', rel)"
              >
                <i class="pi pi-trash" aria-hidden="true"></i>
              </ElButton>
            </div>
          </div>

          <!-- Sous-relations (récursif) -->
          <div
            v-if="expanded.has(rel.id)"
            class="mb-2.5 ml-5 border-l-2 border-line pl-3 sm:ml-9"
          >
            <p v-if="loading.has(rel.id)" class="py-2 text-xs text-muted">
              <i class="pi pi-spin pi-spinner mr-1" aria-hidden="true"></i>
              {{ t("common.loading") }}
            </p>
            <p
              v-else-if="!children[rel.id]?.length"
              class="py-2 text-xs italic text-muted"
            >
              {{ t("items.relations.noSubRelations") }}
            </p>
            <RelationTree
              v-else
              nested
              :relations="children[rel.id] ?? []"
              :context-item-id="linkedId(rel)"
              :show-delete="false"
              :max-depth="maxDepth - 1"
            />
          </div>
        </li>
      </ul>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from "vue";
import { useI18n } from "vue-i18n";
import { itemApi } from "@/api/client";
import { useLang } from "@/composables/useLang";
import { useNotify } from "@/composables/useNotify";
import {
  fieldLabel,
  fieldValue,
  itemLabel as renderItemLabel,
  orderedKeys,
  typeTitle,
  type Schema,
} from "@/lib/schema";
import type { Item, ItemRelation } from "@/api/types";

const props = withDefaults(
  defineProps<{
    relations: ItemRelation[];
    contextItemId: string;
    showDelete?: boolean;
    maxDepth?: number;
    /** Sous-relations : lignes sans fiche ni ombre, à l'intérieur de la relation parente. */
    nested?: boolean;
  }>(),
  { showDelete: true, maxDepth: 5, nested: false },
);

defineEmits<{
  delete: [rel: ItemRelation];
  edit: [rel: ItemRelation];
}>();

const { t } = useI18n();
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
  return typeTitle(linked?.item_type, lang.value) || "?";
}

function linkedLabel(rel: ItemRelation): string {
  const linked = isOutgoing(rel) ? rel.target_item : rel.source_item;
  if (!linked) return "?";
  return computeLabel(linked);
}

function computeLabel(i: Item): string {
  return renderItemLabel(i, lang.value);
}

function relSchema(rel: ItemRelation): Schema | undefined {
  return rel.relation_type?.attributes_schema as Schema | undefined;
}

// Convention : l'attribut de relation `order` (entier) sert au tri et s'affiche en badge.
const ORDER_KEY = "order";

function orderVal(rel: ItemRelation): unknown {
  return rel.attributes?.[ORDER_KEY] ?? null;
}

function displayAttrs(rel: ItemRelation): Record<string, unknown> {
  const result: Record<string, unknown> = {};
  for (const k of orderedKeys(relSchema(rel), rel.attributes)) {
    if (k === ORDER_KEY || !(k in (rel.attributes ?? {}))) continue;
    result[k] = rel.attributes[k];
  }
  return result;
}

function attrLabel(rel: ItemRelation, key: string): string {
  return fieldLabel(relSchema(rel), key, lang.value);
}

function attrValue(rel: ItemRelation, key: string, value: unknown): string {
  return fieldValue(relSchema(rel), key, value, lang.value);
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

/** Un groupe par type de relation, dans l'ordre du tri (relations déjà ordonnées dans chacun). */
const groups = computed(() => {
  const byType = new Map<
    string,
    { typeId: string; title: string; relations: ItemRelation[] }
  >();
  for (const rel of sortedRelations.value) {
    let g = byType.get(rel.relation_type_id);
    if (!g) {
      g = {
        typeId: rel.relation_type_id,
        title: typeTitle(rel.relation_type, lang.value) || "?",
        relations: [],
      };
      byType.set(rel.relation_type_id, g);
    }
    g.relations.push(rel);
  }
  return [...byType.values()];
});

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
    notifyError(e, t("items.relations.subRelationsUnavailable"));
  } finally {
    loading.value.delete(rel.id);
    loading.value = new Set(loading.value);
  }
}
</script>
