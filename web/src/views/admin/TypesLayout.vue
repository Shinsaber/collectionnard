<template>
  <div>
    <ElTabs
      :model-value="route.path"
      data-test="types-tabs"
      @tab-change="(path) => router.push(String(path))"
    >
      <ElTabPane
        v-for="tab in tabs"
        :key="tab.path"
        :name="tab.path"
        :label="tab.label"
      />
    </ElTabs>
    <RouterView />
  </div>
</template>

<script setup lang="ts">
// Gestion des méta-types : un seul lien dans le menu, un onglet par sorte de type. Les onglets suivent
// l'URL (actualiser, revenir en arrière, liens existants) ; l'édition d'un type reste sur sa page.
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute, useRouter } from "vue-router";

const { t } = useI18n();
const route = useRoute();
const router = useRouter();

const tabs = computed(() => [
  { path: "/admin/item-types", label: t("admin.types.items") },
  { path: "/admin/relation-types", label: t("admin.types.relations") },
  { path: "/admin/collection-types", label: t("admin.types.collections") },
  { path: "/admin/review-types", label: t("admin.types.reviews") },
]);
</script>
