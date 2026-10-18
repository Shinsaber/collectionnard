<template>
  <PublicCollectionPage
    :key="id"
    :load-collection="loadCollection"
    :load-items="loadItems"
    :load-stats="loadStats"
  />
</template>

<script setup lang="ts">
// Page publique d'une collection (/c/:id) : visible selon sa visibilité (P4-2).
import { publicApi } from "@/api/client";
import PublicCollectionPage from "@/components/sharing/PublicCollectionPage.vue";

const props = defineProps<{ id: string }>();
const loadCollection = () => publicApi.collection(props.id);
const loadItems = (params: Record<string, unknown>) =>
  publicApi.items(props.id, params);
const loadStats = () => publicApi.stats(props.id);
</script>
