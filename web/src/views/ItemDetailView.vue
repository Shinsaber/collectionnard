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
                ? t("items.detail.inCollections", itemCollections.length)
                : t("items.detail.inNoCollection")
            }}
          </button>
        </div>
        <h1
          class="text-[28px] font-extrabold leading-none tracking-tight sm:text-[44px]"
        >
          {{ title }}
        </h1>
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
          :label="`${t('items.detail.relations')} · ${relations.length}`"
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
          </div>
        </ElTabPane>
        <ElTabPane
          name="collections"
          :label="`${t('items.detail.inYourCollections')} · ${itemCollections.length}`"
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
      :exclude-ids="itemCollections.map((ic) => ic.collection.id)"
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
import { itemApi, relationApi } from "@/api/client";
import { useAuthStore } from "@/stores/auth";
import { useLang } from "@/composables/useLang";
import { useNotify } from "@/composables/useNotify";
import { MOBILE_QUERY, useMediaQuery } from "@/composables/useMediaQuery";
import {
  defaultImage,
  galleryUrls,
  imageCategories,
  isCompleteness,
  isImageGallery,
  itemLabel,
  typeTitle,
  type Schema,
} from "@/lib/schema";
import ImageLightbox from "@/components/common/ImageLightbox.vue";
import ItemCover from "@/components/item/ItemCover.vue";
import ItemCollectionsCard from "@/components/item/ItemCollectionsCard.vue";
import ItemEditDialog from "@/components/item/ItemEditDialog.vue";
import ItemGallery from "@/components/item/ItemGallery.vue";
import ItemHistoryCard from "@/components/item/ItemHistoryCard.vue";
import ItemAttributes from "@/components/item/ItemAttributes.vue";
import ItemMoreMenu from "@/components/item/ItemMoreMenu.vue";
import AddToCollectionDialog from "@/components/item/AddToCollectionDialog.vue";
import RelationCreateDialog from "@/components/item/RelationCreateDialog.vue";
import RelationEditDialog from "@/components/item/RelationEditDialog.vue";
import RelationTree from "@/components/item/RelationTree.vue";
import type { Item, ItemCollectionMembership, ItemRelation } from "@/api/types";

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
const relations = ref<ItemRelation[]>([]);
const itemCollections = ref<ItemCollectionMembership[]>([]);
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

type Tab = "relations" | "collections" | "images" | "history";
const tab = ref<Tab>("relations");
const tabsRef = ref<HTMLElement | null>(null);

/** Ouvre un onglet et le fait défiler à l'écran (pastille des collections). */
async function openTab(name: Tab) {
  tab.value = name;
  await nextTick();
  tabsRef.value?.scrollIntoView?.({ behavior: "smooth", block: "start" });
}

/**
 * Appartenance : sources des relations entrantes marquées `x-completeness` (tome d'une
 * édition, épisode d'une saison…), avec le numéro d'ordre de la relation s'il existe.
 */
const partOf = computed(() =>
  relations.value
    .filter(
      (r) =>
        r.target_item_id === item.value?.id &&
        isCompleteness(r.relation_type?.attributes_schema),
    )
    .map((r) => ({
      id: r.source_item_id,
      label: itemLabel(r.source_item, lang.value),
      order: (r.attributes?.order as number | undefined) ?? null,
    })),
);

/** Images de la galerie (toutes catégories) : l'onglet n'apparaît que s'il y en a. */
const imageCount = computed(() => {
  const it = item.value;
  if (!it) return 0;
  let n = 0;
  for (const key of Object.keys(it.attributes)) {
    if (!isImageGallery(schema.value, key)) continue;
    for (const cat of imageCategories(schema.value, key))
      n +=
        (
          (it.attributes[key] as Record<string, unknown>)?.[cat.name] as
            unknown[] | undefined
        )?.length ?? 0;
  }
  return n;
});

const defaultImageUrl = computed(() =>
  item.value ? defaultImage(schema.value, item.value.attributes) : "",
);

/** Toutes les images de la fiche, pour la visionneuse ouverte depuis la couverture. */
const allImageUrls = computed(() =>
  item.value ? galleryUrls(schema.value, item.value.attributes) : [],
);
const coverViewerOpen = ref(false);
function openCover() {
  coverViewerOpen.value = true;
}

// Ratio de la catégorie qui contient l'image par défaut (sinon portrait 2/3).
const defaultImageRatio = computed(() => {
  const url = defaultImageUrl.value;
  if (!item.value || !url) return "2 / 3";
  for (const key of Object.keys(item.value.attributes)) {
    if (!isImageGallery(schema.value, key)) continue;
    const data = item.value.attributes[key] as Record<string, unknown>;
    for (const cat of imageCategories(schema.value, key)) {
      if (((data[cat.name] as string[]) ?? []).includes(url)) return cat.ratio;
    }
  }
  return "2 / 3";
});

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
    relations.value = relations.value.filter((r) => r.id !== rel.id);
    historyVersion.value++;
  } catch (e) {
    notifyError(e);
  }
}

async function reloadRelations() {
  if (!item.value) return;
  try {
    relations.value = (await itemApi.relations(item.value.id)) ?? [];
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
    const [i, rels] = await Promise.all([
      itemApi.get(id),
      itemApi.relations(id),
    ]);
    item.value = i;
    relations.value = rels ?? [];
    historyVersion.value++;
    loadCollections(id);
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
