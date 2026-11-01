<template>
  <!-- Filtres actifs du catalogue (puces) et interrupteur du panneau de filtres (ItemsView). -->
  <div class="mb-4 flex min-h-10 flex-wrap items-center justify-between gap-3">
    <div class="flex flex-wrap items-center gap-2">
      <FilterChip
        v-if="selectedType"
        :label="
          t('items.search.typeChip', { type: typeTitle(selectedType, lang) })
        "
        @remove="emit('remove-type')"
      />
      <FilterChip
        v-if="tag"
        :label="t('items.search.tagChip', { tag })"
        @remove="emit('remove-tag')"
      />
      <FilterChip
        v-if="attrCount > 0"
        :label="t('items.search.attrChip', attrCount)"
        @remove="emit('remove-attrs')"
      />
      <RelatedToTag
        v-if="relatedTo"
        :item-id="relatedTo"
        :relation-type-id="relationTypeId"
        @close="emit('remove-related')"
      />
    </div>
    <button
      type="button"
      class="ml-auto flex h-10 items-center gap-2.5 rounded-full bg-surface pr-2 pl-3.5 text-sm font-semibold text-ink shadow-control"
      v-bind="toggleAttrs"
      @click="emit('toggle')"
    >
      <i class="pi pi-sliders-h text-sm" aria-hidden="true"></i>
      {{ t("items.search.filters") }}
      <span
        v-if="activeCount > 0"
        class="rounded-full bg-accent px-1.5 font-mono text-[11px] text-accent-ink"
        >{{ activeCount }}</span
      >
      <span
        v-if="isWide"
        aria-hidden="true"
        class="relative block h-5.5 w-9.5 rounded-full transition-colors"
        :class="panelOpen ? 'bg-primary' : 'bg-dash'"
      >
        <span
          class="absolute top-0.75 size-4 rounded-full bg-surface shadow-control transition-[left]"
          :class="panelOpen ? 'left-4.75' : 'left-0.75'"
        ></span>
      </span>
      <span v-else class="w-1.5"></span>
    </button>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useLang } from "@/composables/useLang";
import { typeTitle } from "@/lib/schema";
import FilterChip from "@/components/common/FilterChip.vue";
import RelatedToTag from "./RelatedToTag.vue";
import type { ItemType } from "@/api/types";

const props = defineProps<{
  selectedType?: ItemType;
  tag?: string;
  /** Filtres d'attributs actifs. */
  attrCount: number;
  relatedTo?: string;
  relationTypeId?: string;
  /** Pastille de l'interrupteur : type, tag, lien et attributs. */
  activeCount: number;
  /** Grand écran : panneau latéral (interrupteur) ; sinon feuille (bouton d'ouverture). */
  isWide: boolean;
  panelOpen: boolean;
  sheetOpen: boolean;
}>();
const emit = defineEmits<{
  "remove-type": [];
  "remove-tag": [];
  "remove-attrs": [];
  "remove-related": [];
  toggle: [];
}>();
const { t } = useI18n();
const { lang } = useLang();

/** Interrupteur sur grand écran (panneau), ouverture d'une feuille sinon. */
const toggleAttrs = computed(() =>
  props.isWide
    ? {
        role: "switch",
        "aria-checked": props.panelOpen,
        "aria-controls": "panneau-filtres",
      }
    : {
        "aria-haspopup": "dialog" as const,
        "aria-expanded": props.sheetOpen,
        "aria-label": t("items.search.showFilters"),
      },
);
</script>
