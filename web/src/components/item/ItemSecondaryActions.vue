<template>
  <div class="grid auto-cols-fr grid-flow-col gap-2 sm:flex sm:flex-wrap">
    <button type="button" :class="tile" @click="emit('edit')">
      <i class="pi pi-pencil text-sm" aria-hidden="true"></i>
      {{ auth.canEditCatalog ? t("common.edit") : t("proposals.suggest") }}
    </button>
    <RouterLink
      :to="{ path: '/items', query: { related_to: itemId } }"
      :class="tile"
    >
      <i class="pi pi-list text-sm" aria-hidden="true"></i>
      {{ t("items.search.relatedItems") }}
    </RouterLink>
    <RouterLink :to="`/items/${itemId}/graph`" :class="tile">
      <NavIcon name="graph" :size="16" />
      {{ t("items.detail.graph") }}
    </RouterLink>
    <ItemReportButton
      v-if="!auth.isAdmin"
      :class="tile"
      :item-id="itemId"
      :item-label="itemLabel"
    />
  </div>
</template>

<script setup lang="ts">
// Actions secondaires d'une fiche : modifier (ou suggérer une correction pour un membre, P4-3b),
// items liés, graphe, signaler (P4-3c, tout le monde sauf l'admin, qui corrige ou supprime lui-même).
import { useI18n } from "vue-i18n";
import { useAuthStore } from "@/stores/auth";
import NavIcon from "@/components/layout/NavIcon.vue";
import ItemReportButton from "@/components/proposals/ItemReportButton.vue";

defineProps<{ itemId: string; itemLabel: string }>();
const emit = defineEmits<{ edit: [] }>();
const { t } = useI18n();
const auth = useAuthStore();

/** Tuile icône + texte sur mobile, bouton doux en ligne au-delà. */
const tile =
  "flex h-14 flex-col items-center justify-center gap-1 rounded-xl bg-soft px-2 text-xs font-semibold text-ink sm:h-10 sm:flex-row sm:gap-2 sm:rounded-[10px] sm:px-4 sm:text-sm";
</script>
