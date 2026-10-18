<template>
  <AppSpinner v-if="loading" />
  <p
    v-else-if="!collection"
    class="rounded-2xl bg-surface p-6 text-muted shadow-card"
    data-test="public-not-found"
  >
    {{ t("sharing.public.notFound") }}
  </p>
  <div v-else class="flex flex-col gap-4">
    <ElAlert
      v-if="shared"
      type="info"
      :closable="false"
      show-icon
      :title="t('sharing.public.shared')"
    />
    <PageHeader
      :title="collection.name"
      :meta="t('collections.itemCount', collection.item_count)"
    >
      <template #subtitle>
        <div class="flex flex-wrap items-center gap-2">
          <span
            class="rounded-full bg-tint px-2.5 py-0.5 font-mono text-[11px] uppercase text-tint-ink"
            >{{ typeTitle(collection.collection_type, lang) || "?" }}</span
          >
          <RouterLink
            :to="`/u/${collection.owner.username}`"
            class="flex items-center gap-1.5 text-sm font-semibold text-link"
            data-test="public-owner"
          >
            <UserAvatar :user="collection.owner" :size="22" />
            {{
              t("sharing.public.by", {
                name:
                  collection.owner.display_name || collection.owner.username,
              })
            }}
          </RouterLink>
          <p v-if="collection.description" class="text-[15px] text-muted">
            {{ collection.description }}
          </p>
        </div>
      </template>
    </PageHeader>

    <ElTabs v-model="tab">
      <ElTabPane :label="t('stats.itemsTab')" name="items">
        <CollectionItemList
          :items="items.data.value"
          :schema="schema"
          :loading="items.loading.value"
          readonly
          :link-items="auth.isAuthenticated"
        />
        <ElPagination
          class="mt-4 justify-end"
          hide-on-single-page
          layout="prev, pager, next"
          :total="items.total.value"
          :page-size="items.perPage.value"
          :current-page="items.page.value"
          @current-change="onPage"
        />
      </ElTabPane>
      <ElTabPane :label="t('stats.tab')" name="stats" lazy>
        <CollectionStatsPanel
          :load="loadStats"
          :schema="schema"
          :link-items="auth.isAuthenticated"
        />
      </ElTabPane>
    </ElTabs>
  </div>
</template>

<script setup lang="ts">
// Collection lue par un autre que son propriétaire (P4-2), en lecture seule : page publique
// ou lien de partage selon les fonctions de chargement. Les champs cachés sont déjà retirés
// par le serveur (schéma et attributs) : rien n'est filtré ici.
import { computed, defineAsyncComponent, onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useLang } from "@/composables/useLang";
import { useNotify } from "@/composables/useNotify";
import { usePagination } from "@/composables/usePagination";
import { useAuthStore } from "@/stores/auth";
import { typeTitle, type Schema } from "@/lib/schema";
import AppSpinner from "@/components/common/AppSpinner.vue";
import PageHeader from "@/components/common/PageHeader.vue";
import CollectionItemList from "@/components/collection/CollectionItemList.vue";
import UserAvatar from "@/components/account/UserAvatar.vue";
import type {
  CollectionItem,
  CollectionStats,
  PaginatedResponse,
  PublicCollection,
} from "@/api/types";

// ECharts n'est téléchargé qu'à l'ouverture de l'onglet Statistiques.
const CollectionStatsPanel = defineAsyncComponent(
  () => import("@/components/stats/CollectionStatsPanel.vue"),
);

const props = defineProps<{
  loadCollection: () => Promise<PublicCollection>;
  loadItems: (
    params: Record<string, unknown>,
  ) => Promise<PaginatedResponse<CollectionItem>>;
  loadStats: () => Promise<CollectionStats>;
  /** Ouverte par un lien de partage (bandeau d'information). */
  shared?: boolean;
}>();
const { t } = useI18n();
const { lang } = useLang();
const { notifyError } = useNotify();
const auth = useAuthStore();

const collection = ref<PublicCollection | null>(null);
const loading = ref(true);
const tab = ref<"items" | "stats">("items");
const items = usePagination((params: Record<string, unknown>) =>
  props.loadItems(params),
);
const schema = computed(
  () =>
    collection.value?.collection_type?.attributes_schema as Schema | undefined,
);

function isNotFound(e: unknown): boolean {
  const status = (e as { response?: { status?: number } })?.response?.status;
  return status === 404 || status === 400;
}

onMounted(async () => {
  try {
    collection.value = await props.loadCollection();
    await items.load();
  } catch (e) {
    collection.value = null;
    if (!isNotFound(e)) notifyError(e);
  } finally {
    loading.value = false;
  }
});

function onPage(p: number) {
  items.goToPage(p);
  items.load().catch((e) => notifyError(e));
}
</script>
