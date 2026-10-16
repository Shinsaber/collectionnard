<template>
  <!-- Item lié présent dans mes collections : nom de la collection, ou leur nombre ; détail au clic. -->
  <ElPopover trigger="click" placement="bottom" :width="240">
    <template #reference>
      <button
        type="button"
        class="inline-flex max-w-44 shrink-0 items-center gap-1 rounded-full bg-tint px-2 py-0.5 text-xs font-semibold text-tint-ink"
        :aria-label="
          t('items.relations.inCollectionsLabel', {
            names: collections.map((c) => c.name).join(', '),
          })
        "
      >
        <NavIcon name="collections" :size="12" />
        <span class="truncate">{{
          collections.length === 1
            ? collections[0]!.name
            : t("items.relations.inCollections", collections.length)
        }}</span>
      </button>
    </template>
    <p class="mb-1.5 font-mono text-[11px] uppercase tracking-wider text-muted">
      {{ t("items.relations.inCollectionsTitle") }}
    </p>
    <ul class="flex flex-col">
      <li v-for="c in collections" :key="c.id">
        <RouterLink
          :to="`/collections/${c.id}`"
          class="block truncate rounded-md px-1.5 py-1 text-sm font-semibold text-link hover:bg-soft"
          >{{ c.name }}</RouterLink
        >
      </li>
    </ul>
  </ElPopover>
</template>

<script setup lang="ts">
import { useI18n } from "vue-i18n";
import NavIcon from "@/components/layout/NavIcon.vue";
import type { CollectionRef } from "@/api/types";

defineProps<{ collections: CollectionRef[] }>();
const { t } = useI18n();
</script>
