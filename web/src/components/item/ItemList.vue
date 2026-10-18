<template>
  <div>
    <!-- Bureau / tablette : tableau posé sur une fiche (ligne cliquable, navigation clavier). -->
    <div
      class="hidden overflow-hidden rounded-2xl bg-surface shadow-card md:block"
    >
      <ElTable
        v-loading="loading"
        :data="items"
        row-key="id"
        highlight-current-row
        class="cursor-pointer"
        :empty-text="emptyText"
        @row-click="(row: Item) => router.push(`/items/${row.id}`)"
      >
        <ElTableColumn width="76" class-name="cover-cell">
          <template #default="{ row }">
            <ItemCover :item="row as Item" :label="labelOf(row as Item)" />
          </template>
        </ElTableColumn>
        <ElTableColumn :label="t('items.list.colLabel')" min-width="260">
          <template #default="{ row }">
            <div class="flex flex-col gap-0.5 py-1">
              <RouterLink
                :to="`/items/${(row as Item).id}`"
                class="text-[15px] font-bold text-ink hover:text-link"
                @click.stop
                >{{ labelOf(row as Item) }}</RouterLink
              >
              <span class="font-mono text-xs uppercase text-muted">{{
                typeTitle((row as Item).item_type, lang) || "?"
              }}</span>
            </div>
          </template>
        </ElTableColumn>
        <ElTableColumn :label="t('items.list.colTags')" min-width="160">
          <template #default="{ row }">
            <span class="text-[13px] text-muted">{{
              tagNames(row as Item)
            }}</span>
          </template>
        </ElTableColumn>
        <ElTableColumn v-if="canEdit" width="184" align="right">
          <template #default="{ row }">
            <div class="flex justify-end gap-1">
              <ElButton
                text
                :title="t('items.list.graph')"
                :aria-label="t('items.list.viewGraph')"
                @click.stop="router.push(`/items/${(row as Item).id}/graph`)"
              >
                <NavIcon name="graph" :size="16" />
              </ElButton>
              <ElButton
                text
                :aria-label="t('common.edit')"
                @click.stop="emit('edit', row as Item)"
              >
                <i class="pi pi-pencil" aria-hidden="true"></i>
              </ElButton>
              <ElButton
                v-if="canDelete"
                text
                type="danger"
                :aria-label="t('common.delete')"
                @click.stop="emit('remove', row as Item)"
              >
                <i class="pi pi-trash" aria-hidden="true"></i>
              </ElButton>
            </div>
          </template>
        </ElTableColumn>
      </ElTable>
    </div>

    <!-- Mobile : cartes (jaquette, type, libellé, tags) ; toute la carte est un lien. -->
    <ul v-loading="loading" class="flex flex-col gap-2.5 md:hidden">
      <li v-for="item in items" :key="item.id">
        <RouterLink
          :to="`/items/${item.id}`"
          class="flex gap-3.5 rounded-2xl bg-surface p-2.5 text-ink shadow-card"
        >
          <ItemCover :item="item" :label="labelOf(item)" :width="56" />
          <div class="flex min-w-0 flex-1 flex-col justify-center gap-1">
            <span
              class="font-mono text-[11px] uppercase tracking-wide text-muted"
              >{{ typeTitle(item.item_type, lang) || "?" }}</span
            >
            <span class="text-base font-bold leading-tight">{{
              labelOf(item)
            }}</span>
            <span v-if="item.tags?.length" class="text-[13px] text-muted">{{
              tagNames(item)
            }}</span>
          </div>
        </RouterLink>
      </li>
      <li
        v-if="!loading && items.length === 0"
        class="rounded-2xl bg-surface p-6 text-center text-sm text-muted shadow-card"
      >
        {{ emptyText }}
      </li>
    </ul>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { useLang } from "@/composables/useLang";
import { itemLabel, typeTitle } from "@/lib/schema";
import ItemCover from "@/components/item/ItemCover.vue";
import NavIcon from "@/components/layout/NavIcon.vue";
import type { Item } from "@/api/types";

// Résultats du catalogue : tableau sur grand écran, cartes sur mobile (rendu CSS, pas de JS).
defineProps<{
  items: Item[];
  loading: boolean;
  /** Actions d'édition (graphe, modifier) dans le tableau : admins et contributeurs. */
  canEdit: boolean;
  /** Suppression : admins seulement (un contributeur signale). */
  canDelete?: boolean;
  emptyText: string;
}>();

const emit = defineEmits<{ edit: [item: Item]; remove: [item: Item] }>();

const { t } = useI18n();
const { lang } = useLang();
const router = useRouter();

function labelOf(item: Item): string {
  return itemLabel(item, lang.value);
}

function tagNames(item: Item): string {
  return (item.tags ?? []).map((tag) => tag.name).join(", ");
}
</script>
