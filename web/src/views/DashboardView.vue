<template>
  <div>
    <PageHeader
      :eyebrow="today"
      :title="t('dashboard.greeting', { name: auth.user?.username ?? '' })"
    >
      <template #actions>
        <ElInput
          v-model="search"
          size="large"
          class="w-full! sm:w-80!"
          :placeholder="t('dashboard.searchPlaceholder')"
          :aria-label="t('common.search')"
          @keyup.enter="goSearch"
        >
          <template #prefix>
            <i class="pi pi-search" aria-hidden="true"></i>
          </template>
        </ElInput>
      </template>
    </PageHeader>

    <AppSpinner v-if="loading" />

    <template v-else>
      <!-- Chiffres clés -->
      <div
        v-if="stats"
        class="mb-7 grid grid-cols-2 gap-3 lg:grid-cols-4 lg:gap-4"
      >
        <div
          v-for="s in statTiles"
          :key="s.label"
          class="flex flex-col gap-1.5 rounded-2xl bg-surface px-4 py-3.5 shadow-card lg:px-5 lg:py-4.5"
        >
          <span
            class="font-mono text-[11px] uppercase tracking-wider text-muted"
            >{{ s.label }}</span
          >
          <span
            class="text-[28px] font-extrabold leading-none tracking-tight lg:text-4xl"
            >{{ s.value.toLocaleString(lang) }}</span
          >
        </div>
      </div>

      <div class="flex flex-wrap items-start gap-6">
        <!-- Ajoutés récemment -->
        <section class="flex min-w-0 flex-[3_1_480px] flex-col gap-3">
          <div class="flex items-baseline justify-between gap-3">
            <h2 class="text-lg font-extrabold tracking-tight lg:text-xl">
              {{ t("dashboard.recent") }}
            </h2>
            <RouterLink to="/items" class="text-sm font-semibold text-link"
              >{{ t("dashboard.seeCatalog") }} →</RouterLink
            >
          </div>
          <p
            v-if="recent.length === 0"
            class="rounded-2xl bg-surface p-5 text-sm text-muted shadow-card"
          >
            {{ t("dashboard.noItems") }}
          </p>
          <!-- Mobile : rangée de jaquettes qui défile ; plus large : liste. -->
          <ul class="-mr-4 flex gap-3 overflow-x-auto pr-4 pb-2.5 md:hidden">
            <li v-for="item in recent" :key="item.id" class="w-24 shrink-0">
              <RouterLink
                :to="`/items/${item.id}`"
                class="flex flex-col gap-1.5 text-ink"
              >
                <ItemCover :item="item" :label="labelOf(item)" :width="96" />
                <span
                  class="line-clamp-2 text-[13px] font-bold leading-tight"
                  >{{ labelOf(item) }}</span
                >
              </RouterLink>
            </li>
          </ul>
          <ul class="hidden flex-col gap-2.5 md:flex">
            <li v-for="item in recent" :key="item.id">
              <RouterLink
                :to="`/items/${item.id}`"
                class="flex items-center gap-3.5 rounded-2xl bg-surface py-2.5 pr-4 pl-2.5 text-ink shadow-card"
              >
                <ItemCover :item="item" :label="labelOf(item)" />
                <div class="flex min-w-0 flex-1 flex-col gap-0.5">
                  <span
                    class="font-mono text-[11px] uppercase tracking-wide text-muted"
                    >{{ typeTitle(item.item_type, lang) || "?" }}</span
                  >
                  <span class="truncate text-base font-bold">{{
                    labelOf(item)
                  }}</span>
                </div>
                <span class="shrink-0 text-[13px] text-muted">{{
                  shortDate(item.created_at)
                }}</span>
              </RouterLink>
            </li>
          </ul>
        </section>

        <section class="flex min-w-0 flex-[2_1_320px] flex-col gap-3">
          <!-- Mes collections -->
          <div class="flex items-baseline justify-between gap-3">
            <h2 class="text-lg font-extrabold tracking-tight lg:text-xl">
              {{ t("dashboard.myCollections") }}
            </h2>
            <RouterLink
              to="/collections"
              class="text-sm font-semibold text-link"
              >{{ t("dashboard.seeCollections") }} →</RouterLink
            >
          </div>
          <p
            v-if="collections.length === 0"
            class="rounded-2xl bg-surface p-5 text-sm text-muted shadow-card"
          >
            {{ t("dashboard.noCollections") }}
          </p>
          <ul class="flex flex-col gap-2.5">
            <li v-for="c in collections" :key="c.id">
              <RouterLink
                :to="`/collections/${c.id}`"
                class="flex items-center gap-3.5 rounded-2xl bg-surface px-4 py-3 text-ink shadow-card"
              >
                <span
                  class="flex size-10 shrink-0 items-center justify-center rounded-[10px] bg-tint text-tint-ink"
                >
                  <NavIcon name="collections" />
                </span>
                <div class="flex min-w-0 flex-1 flex-col">
                  <span class="truncate text-base font-bold">{{ c.name }}</span>
                  <span class="text-[13px] text-muted">{{
                    typeTitle(c.collection_type, lang)
                  }}</span>
                </div>
                <span class="font-mono text-[13px]">{{ c.item_count }}</span>
              </RouterLink>
            </li>
          </ul>

          <!-- Items par type -->
          <div
            v-if="byType.length"
            class="mt-2 flex flex-col gap-3 rounded-2xl bg-surface px-5 py-4.5 shadow-card"
          >
            <h2
              class="font-mono text-[11px] font-medium uppercase tracking-wider text-muted"
            >
              {{ t("dashboard.itemsByType") }}
            </h2>
            <RouterLink
              v-for="b in byType"
              :key="b.id"
              :to="{ path: '/items', query: { item_type_id: b.id } }"
              class="-mx-2 flex flex-col gap-1 rounded-lg px-2 py-1 transition-colors hover:bg-canvas"
              :aria-label="
                t('dashboard.seeType', { type: b.label, n: b.count })
              "
            >
              <div class="flex justify-between gap-3 text-sm">
                <span>{{ b.label }}</span>
                <span class="font-mono text-[13px]">{{
                  b.count.toLocaleString(lang)
                }}</span>
              </div>
              <div class="h-1.5 rounded-full bg-soft" aria-hidden="true">
                <div
                  class="h-1.5 rounded-full bg-primary"
                  :style="{ width: `${b.pct}%` }"
                ></div>
              </div>
            </RouterLink>
          </div>
        </section>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { collectionApi, itemApi, statsApi } from "@/api/client";
import { useAuthStore } from "@/stores/auth";
import { useLang } from "@/composables/useLang";
import { useNotify } from "@/composables/useNotify";
import { useMetaTypes } from "@/stores/metaTypes";
import { itemLabel, typeTitle } from "@/lib/schema";
import AppSpinner from "@/components/common/AppSpinner.vue";
import PageHeader from "@/components/common/PageHeader.vue";
import ItemCover from "@/components/item/ItemCover.vue";
import NavIcon from "@/components/layout/NavIcon.vue";
import type { Collection, Item, Stats } from "@/api/types";

const { t } = useI18n();
const { lang } = useLang();
const { notifyError } = useNotify();
const auth = useAuthStore();
const router = useRouter();
const metaTypes = useMetaTypes();

const stats = ref<Stats | null>(null);
const recent = ref<Item[]>([]);
const collections = ref<Collection[]>([]);
const loading = ref(true);
const search = ref("");

const today = computed(() =>
  new Date().toLocaleDateString(lang.value, {
    weekday: "long",
    day: "numeric",
    month: "long",
  }),
);

const statTiles = computed(() => {
  const s = stats.value;
  if (!s) return [];
  return [
    { label: t("dashboard.totalItems"), value: s.total_items },
    { label: t("dashboard.totalRelations"), value: s.total_relations },
    { label: t("dashboard.totalCollections"), value: s.total_collections },
    { label: t("dashboard.itemTypes"), value: s.items_by_type?.length ?? 0 },
  ];
});

/** Répartition par type, dans l'ordre d'affichage des types (API) ; barres relatives au maximum. */
const byType = computed(() => {
  const rows = stats.value?.items_by_type ?? [];
  const max = Math.max(1, ...rows.map((r) => r.count));
  return rows.map((r) => ({
    id: r.type_id,
    label: typeName(r.type_id, r.type_name),
    count: r.count,
    pct: Math.max(2, Math.round((r.count / max) * 100)),
  }));
});

/** Nom affiché d'un ItemType des stats ; repli sur le nom brut tant que les types manquent. */
function typeName(id: string, name: string): string {
  const type = metaTypes.itemTypes.find((it) => it.id === id);
  return type ? typeTitle(type, lang.value) : name;
}

function labelOf(item: Item): string {
  return itemLabel(item, lang.value);
}

function shortDate(iso: string): string {
  return new Date(iso).toLocaleDateString(lang.value, {
    day: "numeric",
    month: "short",
  });
}

function goSearch() {
  const q = search.value.trim();
  router.push({ path: "/items", query: q ? { search: q } : {} });
}

onMounted(async () => {
  // Méta-types en parallèle : tant qu'ils manquent, les noms bruts s'affichent.
  metaTypes.loadItemTypes().catch((e) => notifyError(e));
  const [s, r, c] = await Promise.allSettled([
    statsApi.get(),
    itemApi.list({ per_page: 5, sort: "-created_at" }),
    collectionApi.list({ per_page: 5 }),
  ]);
  if (s.status === "fulfilled") stats.value = s.value;
  else notifyError(s.reason, t("dashboard.statsUnavailable"));
  if (r.status === "fulfilled") recent.value = r.value.data ?? [];
  else notifyError(r.reason);
  if (c.status === "fulfilled") collections.value = c.value.data ?? [];
  else notifyError(c.reason);
  loading.value = false;
});
</script>
