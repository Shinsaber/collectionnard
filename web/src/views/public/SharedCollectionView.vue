<template>
  <PublicCollectionPage
    :key="token"
    shared
    :load-collection="loadCollection"
    :load-items="loadItems"
    :load-stats="loadStats"
  />
</template>

<script setup lang="ts">
// Collection ouverte par un lien de partage (/share?token=…), lecture seule (P4-2).
import { computed } from "vue";
import { useRoute } from "vue-router";
import { sharedApi } from "@/api/client";
import PublicCollectionPage from "@/components/sharing/PublicCollectionPage.vue";

const route = useRoute();
const token = computed(() =>
  typeof route.query.token === "string" ? route.query.token : "",
);
const loadCollection = () => sharedApi.collection(token.value);
const loadItems = (params: Record<string, unknown>) =>
  sharedApi.items(token.value, params);
const loadStats = () => sharedApi.stats(token.value);
</script>
