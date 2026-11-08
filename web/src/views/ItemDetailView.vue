<template>
  <AppSpinner v-if="loading" />
  <div v-else-if="item" class="flex flex-col gap-7">
    <nav
      :aria-label="t('items.detail.breadcrumb')"
      class="-mb-2 flex flex-wrap items-center gap-2 text-sm text-muted"
    >
      <RouterLink to="/items" class="font-semibold text-link">{{
        t("items.list.title")
      }}</RouterLink>
      <span aria-hidden="true">›</span>
      <RouterLink
        :to="{ path: '/items', query: { item_type_id: item.item_type_id } }"
        class="font-semibold text-link"
        >{{ typeTitle(item.item_type, lang) || "?" }}</RouterLink
      >
      <span aria-hidden="true">›</span>
      <span aria-current="page" class="truncate">{{ title }}</span>
    </nav>

    <!-- En-tête : couverture, type et collections, titre, appartenance, tags, actions -->
    <section
      class="flex flex-wrap gap-5 rounded-2xl bg-surface p-4.5 shadow-card sm:gap-8 sm:p-7"
    >
      <!-- La couverture s'agrandit (visionneuse sur toutes les images de la fiche). -->
      <button
        v-if="defaultImageUrl"
        type="button"
        class="shrink-0 self-start"
        :aria-label="t('common.enlarge', { name: title })"
        data-test="cover-enlarge"
        @click="openCover"
      >
        <ItemCover
          :item="item"
          :label="title"
          :width="coverWidth"
          :ratio="defaultImageRatio"
          :alt="t('items.detail.defaultImageAlt', { title })"
        />
      </button>
      <ItemCover
        v-else
        :item="item"
        :label="title"
        :width="coverWidth"
        :ratio="defaultImageRatio"
        :alt="t('items.detail.defaultImageAlt', { title })"
      />
      <ImageLightbox
        v-model:visible="coverViewerOpen"
        :urls="allImageUrls"
        :index="Math.max(0, allImageUrls.indexOf(defaultImageUrl))"
      />
      <div class="flex min-w-0 flex-[1_1_300px] flex-col gap-3.5">
        <div class="flex flex-wrap items-center justify-between gap-3">
          <span class="font-mono text-xs uppercase tracking-wider text-muted">{{
            typeTitle(item.item_type, lang) || "?"
          }}</span>
          <button
            type="button"
            class="rounded-full bg-tint px-3.5 py-1.5 text-sm font-semibold text-tint-ink"
            data-test="collections-pill"
            @click="openTab('collections')"
          >
            {{
              itemCollections.length
                ? t(
                    "items.detail.inCollections",
                    collectionCount(itemCollections),
                  )
                : t("items.detail.inNoCollection")
            }}
          </button>
        </div>
        <h1
          class="text-[28px] font-extrabold leading-none tracking-tight sm:text-[44px]"
        >
          {{ title }}
        </h1>
        <ReviewHeaderSummary
          v-if="reviewSummary?.enabled"
          :summary="reviewSummary"
          :mine="myReview"
          @open="openTab('reviews')"
        />
        <!-- Appartenance : relations de composition entrantes (x-completeness) -->
        <p
          v-if="partOf.length"
          class="text-[15px] text-muted"
          data-test="part-of"
        >
          {{ t("items.detail.partOf") }}
          <template v-for="(p, i) in partOf" :key="p.id">
            <template v-if="i > 0"> · </template>
            <RouterLink
              :to="`/items/${p.id}`"
              class="font-semibold text-link"
              >{{ p.label }}</RouterLink
            >
            <span v-if="p.order != null" class="font-mono text-xs">
              (n° {{ p.order }})</span
            >
          </template>
        </p>
        <div v-if="item.tags?.length" class="flex flex-wrap gap-2">
          <span
            v-for="tag in item.tags"
            :key="tag.id"
            class="rounded-full bg-soft px-2.5 py-0.5 text-[13px]"
            >{{ tag.name }}</span
          >
        </div>
        <div class="mt-1 flex flex-wrap items-center gap-2.5">
          <ElButton
            type="primary"
            size="large"
            class="max-sm:flex-1"
            @click="addToCollectionVisible = true"
          >
            <i class="pi pi-plus mr-2" aria-hidden="true"></i>
            {{ t("items.detail.addToCollection") }}…
          </ElButton>
          <ItemMoreMenu
            :item-id="item.id"
            :item-label="title"
            @edit="editDialogVisible = true"
          />
        </div>
      </div>
    </section>

    <ItemAttributes :schema="schema" :attributes="item.attributes" />

    <!-- Onglets : relations, collections, images, historique -->
    <section ref="tabsRef">
      <ElTabs v-model="tab">
        <ElTabPane
          name="relations"
          :label="`${t('items.detail.relations')} · ${relationsTotal}`"
        >
          <div class="flex flex-col gap-3.5">
            <ElButton
              v-if="auth.canEditCatalog"
              class="self-end"
              @click="createRelationVisible = true"
            >
              <i class="pi pi-plus mr-2" aria-hidden="true"></i>
              {{ t("items.detail.createRelation") }}
            </ElButton>
            <p
              v-if="relations.length === 0"
              class="rounded-2xl bg-surface p-5 text-sm text-muted shadow-card"
            >
              {{ t("items.detail.noRelations") }}
            </p>
            <RelationTree
              v-else
              :relations="relations"
              :context-item-id="item.id"
              :show-delete="auth.canEditCatalog"
              @delete="confirmDeleteRelation"
              @edit="openEditRelation"
            />
            <!-- Au-delà d'une page (nœud très lié : personne, éditeur), pagination serveur. -->
            <ElPagination
              v-if="relationsTotal > RELATIONS_PER_PAGE"
              class="justify-end"
              layout="prev, pager, next"
              :total="relationsTotal"
              :page-size="RELATIONS_PER_PAGE"
              :current-page="relationsPage"
              @current-change="onRelationsPage"
            />
          </div>
        </ElTabPane>
        <ElTabPane
          name="collections"
          :label="`${t('items.detail.inYourCollections')} · ${collectionCount(itemCollections)}`"
        >
          <ItemCollectionsCard
            bare
            :memberships="itemCollections"
            @add="addToCollectionVisible = true"
          />
        </ElTabPane>
        <ElTabPane
          v-if="imageCount > 0"
          name="images"
          :label="`${t('items.detail.images')} · ${imageCount}`"
        >
          <ItemGallery :attributes="item.attributes" :schema="schema" bare />
        </ElTabPane>
        <ElTabPane
          v-if="reviewSummary?.enabled"
          name="reviews"
          :label="`${t('reviews.tab')} · ${reviewSummary.count}`"
          lazy
        >
          <ReviewSection
            :item-id="item.id"
            :summary="reviewSummary"
            @changed="loadReviewSummary(item!.id)"
          />
        </ElTabPane>
        <ElTabPane
          name="articles"
          :label="`${t('articles.tab')} · ${articleCount}`"
          lazy
        >
          <ArticleSection
            :item-id="item.id"
            @count="(n) => (articleCount = n)"
          />
        </ElTabPane>
        <ElTabPane name="history" :label="t('history.title')" lazy>
          <ItemHistoryCard
            :item="item"
            :version="historyVersion"
            bare
            @reverted="(i) => (item = i)"
          />
        </ElTabPane>
      </ElTabs>
    </section>

    <AddToCollectionDialog
      v-model:visible="addToCollectionVisible"
      :item="item"
      :memberships="itemCollections"
      @added="loadCollections(item.id)"
    />
    <RelationCreateDialog
      v-model:visible="createRelationVisible"
      :item="item"
      :relations="relations"
      @created="reloadRelations"
    />
    <RelationEditDialog
      v-model:visible="editRelationVisible"
      :relation="editingRelation"
      :item="item"
      @updated="reloadRelations"
    />
    <ItemEditDialog
      v-model:visible="editDialogVisible"
      :item="item"
      :item-types="[]"
      :propose="!auth.canEditCatalog"
      @saved="loadItem(item!.id)"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useConfirmDialog } from "@/composables/useConfirmDialog";
import AppSpinner from "@/components/common/AppSpinner.vue";
import { articleApi, itemApi, relationApi, reviewApi } from "@/api/client";
import { useAuthStore } from "@/stores/auth";
import { useLang } from "@/composables/useLang";
import { useNotify } from "@/composables/useNotify";
import { MOBILE_QUERY, useMediaQuery } from "@/composables/useMediaQuery";
import {
  RELATIONS_PER_PAGE,
  useItemRelations,
} from "@/composables/useItemRelations";
import { useItemImages } from "@/composables/useItemImages";
import { partOf as partOfRelations } from "@/lib/relations";
import { itemLabel, typeTitle, type Schema } from "@/lib/schema";
import ImageLightbox from "@/components/common/ImageLightbox.vue";
import ItemCover from "@/components/item/ItemCover.vue";
import ItemCollectionsCard from "@/components/item/ItemCollectionsCard.vue";
import ItemEditDialog from "@/components/item/ItemEditDialog.vue";
import ItemGallery from "@/components/item/ItemGallery.vue";
import ItemHistoryCard from "@/components/item/ItemHistoryCard.vue";
import ItemAttributes from "@/components/item/ItemAttributes.vue";
import ItemMoreMenu from "@/components/item/ItemMoreMenu.vue";
import AddToCollectionDialog from "@/components/item/AddToCollectionDialog.vue";
import { collectionCount } from "@/lib/lots";
import RelationCreateDialog from "@/components/item/RelationCreateDialog.vue";
import RelationEditDialog from "@/components/item/RelationEditDialog.vue";
import RelationTree from "@/components/item/RelationTree.vue";
import ReviewSection from "@/components/review/ReviewSection.vue";
import ArticleSection from "@/components/review/ArticleSection.vue";
import ReviewHeaderSummary from "@/components/review/ReviewHeaderSummary.vue";
import type {
  Item,
  ItemCollectionMembership,
  ItemRelation,
  Review,
  ReviewSummary,
} from "@/api/types";

const props = defineProps<{ id: string }>();
const { t } = useI18n();
const auth = useAuthStore();
const { confirm } = useConfirmDialog();
const { lang } = useLang();
const { notifyError, notifySuccess } = useNotify();
const isMobile = useMediaQuery(MOBILE_QUERY);

const item = ref<Item | null>(null);
// Incrémenté après chaque écriture (fiche, relations) : l'historique est relu.
const historyVersion = ref(0);
// Relations paginées par l'API (audit Q-001) : total pour l'onglet, page courante.
const {
  relations,
  total: relationsTotal,
  page: relationsPage,
  load: fetchRelations,
  remove: forgetRelation,
} = useItemRelations();
const itemCollections = ref<ItemCollectionMembership[]>([]);
const reviewSummary = ref<ReviewSummary | null>(null);
const myReview = ref<Review | null>(null);
const articleCount = ref(0);

/** Nombre d'articles de l'onglet « Actualités » (publiés, et ceux du lecteur). */
async function loadArticleCount(itemId: string) {
  try {
    articleCount.value = (await articleApi.list(itemId, { per_page: 1 })).total;
  } catch {
    articleCount.value = 0;
  }
}

// Notes du premier bloc. Sans avis ouverts, la fiche reste utilisable : un échec ne se signale pas ici.
async function loadReviewSummary(itemId: string) {
  try {
    reviewSummary.value = await reviewApi.summary(itemId);
    myReview.value = reviewSummary.value.enabled
      ? await reviewApi.mine(itemId)
      : null;
  } catch {
    reviewSummary.value = null;
    myReview.value = null;
  }
}
const loading = ref(true);

const editDialogVisible = ref(false);
const addToCollectionVisible = ref(false);
const createRelationVisible = ref(false);
const editRelationVisible = ref(false);
const editingRelation = ref<ItemRelation | null>(null);

const coverWidth = computed(() => (isMobile.value ? 108 : 200));

const schema = computed(
  () => item.value?.item_type?.attributes_schema as Schema | undefined,
);
const title = computed(() =>
  item.value ? itemLabel(item.value, lang.value) : "",
);

type Tab =
  "relations" | "collections" | "images" | "reviews" | "articles" | "history";
const tab = ref<Tab>("relations");
const tabsRef = ref<HTMLElement | null>(null);

/** Ouvre un onglet et le fait défiler à l'écran (pastille des collections). */
async function openTab(name: Tab) {
  tab.value = name;
  await nextTick();
  tabsRef.value?.scrollIntoView?.({ behavior: "smooth", block: "start" });
}

/** Appartenance : relations entrantes `x-completeness` (lib/relations). */
const partOf = computed(() =>
  partOfRelations(relations.value, item.value?.id, lang.value),
);

// Couverture, visionneuse et compteur de l'onglet « Images » (composables/useItemImages).
const {
  imageCount,
  defaultImageUrl,
  allImageUrls,
  defaultImageRatio,
  coverViewerOpen,
  openCover,
} = useItemImages(item, schema);

function openEditRelation(rel: ItemRelation) {
  editingRelation.value = rel;
  editRelationVisible.value = true;
}

async function confirmDeleteRelation(rel: ItemRelation) {
  const linked =
    rel.source_item_id === item.value?.id ? rel.target_item : rel.source_item;
  const ok = await confirm({
    message: t("items.detail.confirmDeleteRelation", {
      label: itemLabel(linked, lang.value),
    }),
    confirmText: t("common.delete"),
    danger: true,
  });
  if (!ok) return;
  try {
    await relationApi.delete(rel.id);
    notifySuccess(t("items.detail.relationDeleted"));
    forgetRelation(rel.id);
    historyVersion.value++;
  } catch (e) {
    notifyError(e);
  }
}

async function onRelationsPage(page: number) {
  if (!item.value) return;
  try {
    await fetchRelations(item.value.id, page);
  } catch (e) {
    notifyError(e, t("items.detail.relationsUnavailable"));
  }
}

async function reloadRelations() {
  if (!item.value) return;
  try {
    await fetchRelations(item.value.id, relationsPage.value);
    historyVersion.value++;
  } catch (e) {
    notifyError(e, t("items.detail.relationsUnavailable"));
  }
}

async function loadCollections(itemId: string) {
  try {
    itemCollections.value = await itemApi.collections(itemId);
  } catch (e) {
    itemCollections.value = [];
    notifyError(e, t("items.detail.collectionsUnavailable"));
  }
}

async function loadItem(id: string) {
  loading.value = true;
  try {
    const [i] = await Promise.all([itemApi.get(id), fetchRelations(id, 1)]);
    item.value = i;
    historyVersion.value++;
    loadCollections(id);
    loadReviewSummary(id);
    loadArticleCount(id);
  } catch (e) {
    notifyError(e, t("items.detail.notFound"));
  } finally {
    loading.value = false;
  }
}

onMounted(() => loadItem(props.id));
watch(
  () => props.id,
  (newId) => loadItem(newId),
);
</script>
