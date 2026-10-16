<template>
  <!-- Item de la copie hors ligne : libellé, type, et chaque collection avec ses attributs. -->
  <li class="flex gap-3 px-3.5 py-3">
    <slot name="cover" />
    <div class="flex min-w-0 flex-1 flex-col gap-1.5">
      <div class="flex min-w-0 flex-col">
        <RouterLink
          v-if="linkable"
          :to="`/items/${row.item.id}`"
          class="truncate text-[15px] font-semibold text-ink hover:text-link"
          >{{ row.label }}</RouterLink
        >
        <span v-else class="truncate text-[15px] font-semibold">{{
          row.label
        }}</span>
        <span class="font-mono text-[11px] uppercase text-muted">{{
          row.type
        }}</span>
      </div>
      <div
        v-for="e in row.entries"
        :key="e.collection.id"
        class="flex flex-wrap items-center gap-x-2 gap-y-1 text-xs"
      >
        <span
          class="inline-flex items-center gap-1 rounded-full bg-tint px-2 py-0.5 font-semibold text-tint-ink"
        >
          <NavIcon name="collections" :size="12" />
          {{ e.collection.name }}
        </span>
        <span
          v-for="key in keysOf(e)"
          :key="key"
          class="rounded-full bg-soft px-2 py-0.5"
        >
          {{ fieldLabel(schemaOf(e), key, lang) }} :
          {{ fieldValue(schemaOf(e), key, e.entry.attributes[key], lang) }}
        </span>
      </div>
    </div>
  </li>
</template>

<script setup lang="ts">
import { useLang } from "@/composables/useLang";
import { fieldLabel, fieldValue, orderedKeys, type Schema } from "@/lib/schema";
import NavIcon from "@/components/layout/NavIcon.vue";
import type { OfflineRow } from "@/lib/offlineSearch";

defineProps<{
  row: OfflineRow;
  /** Lien vers la fiche quand le serveur répond. */
  linkable: boolean;
}>();
const { lang } = useLang();

type Entry = OfflineRow["entries"][number];

function schemaOf(e: Entry): Schema | undefined {
  return e.collectionType?.attributes_schema as Schema | undefined;
}

/** Attributs contextuels renseignés, dans l'ordre du schéma du type de collection. */
function keysOf(e: Entry): string[] {
  return orderedKeys(schemaOf(e), e.entry.attributes).filter(
    (k) => k in e.entry.attributes,
  );
}
</script>
