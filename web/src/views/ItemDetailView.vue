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

    <div class="flex flex-wrap items-start gap-6">
      <!-- Fiche : affiche, titre, tags, attributs, actions -->
      <section
        class="flex min-w-0 flex-[999_1_560px] flex-wrap gap-5 rounded-2xl bg-surface p-4.5 shadow-card sm:gap-7 sm:p-6"
      >
        <ItemCover
          :item="item"
          :label="title"
          :width="coverWidth"
          :ratio="defaultImageRatio"
          :alt="t('items.detail.defaultImageAlt', { title })"
        />
        <div class="flex min-w-0 flex-[1_1_300px] flex-col gap-3.5">
          <span class="font-mono text-xs uppercase tracking-wider text-muted">{{
            typeTitle(item.item_type, lang) || "?"
          }}</span>
          <h1
            class="text-[28px] font-extrabold leading-none tracking-tight sm:text-[44px]"
          >
            {{ title }}
          </h1>
          <div v-if="item.tags?.length" class="flex flex-wrap gap-2">
            <span
              v-for="tag in item.tags"
              :key="tag.id"
              class="rounded-full bg-soft px-2.5 py-0.5 text-[13px]"
              >{{ tag.name }}</span
            >
          </div>

          <dl
            v-if="attributeRows.length"
            class="mt-1 grid grid-cols-2 gap-x-5 gap-y-3.5 sm:grid-cols-3"
          >
            <div
              v-for="row in attributeRows"
              :key="row.key"
              class="flex min-w-0 flex-col gap-0.5"
              :class="{ 'col-span-full': row.long }"
            >
              <dt
                class="font-mono text-[11px] uppercase tracking-wider text-muted"
              >
                {{ row.label }}
              </dt>
              <dd
                class="break-words"
                :class="row.long ? 'text-[15px] leading-relaxed' : 'text-base'"
              >
                <ul
                  v-if="row.links.length"
                  class="flex flex-wrap gap-x-3 gap-y-1"
                  data-test="attribute-links"
                >
                  <li v-for="l in row.links" :key="l.url">
                    <a
                      :href="l.url"
                      target="_blank"
                      rel="noopener noreferrer"
                      class="text-link underline-offset-2 hover:underline"
                      >{{ linkText(l) }}</a
                    >
                  </li>
                </ul>
                <template v-else>{{ row.value }}</template>
              </dd>
            </div>
          </dl>

          <!-- Actions : une principale ; les secondaires en grille d'icônes sur mobile. -->
          <div class="mt-1.5 flex flex-col gap-2 sm:flex-row sm:flex-wrap">
            <ElButton
              type="primary"
              size="large"
              @click="addToCollectionVisible = true"
            >
              <i class="pi pi-plus mr-2" aria-hidden="true"></i>
              {{ t("items.detail.addToCollection") }}
            </ElButton>
            <div
              class="grid auto-cols-fr grid-flow-col gap-2 sm:flex sm:flex-wrap"
            >
              <button
                type="button"
                :class="secondaryClass"
                @click="editDialogVisible = true"
              >
                <i class="pi pi-pencil text-sm" aria-hidden="true"></i>
                {{
                  auth.canEditCatalog
                    ? t("common.edit")
                    : t("proposals.suggest")
                }}
              </button>
              <RouterLink
                :to="{ path: '/items', query: { related_to: item.id } }"
                :class="secondaryClass"
              >
                <i class="pi pi-list text-sm" aria-hidden="true"></i>
                {{ t("items.search.relatedItems") }}
              </RouterLink>
              <RouterLink
                :to="`/items/${item.id}/graph`"
                :class="secondaryClass"
              >
                <NavIcon name="graph" :size="16" />
                {{ t("items.detail.graph") }}
              </RouterLink>
            </div>
          </div>
        </div>
      </section>

      <ItemCollectionsCard
        class="min-w-0 flex-[1_1_300px]"
        :memberships="itemCollections"
        @add="addToCollectionVisible = true"
      />
    </div>

    <!-- Relations -->
    <section class="flex flex-col gap-3.5">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <h2 class="text-xl font-extrabold tracking-tight sm:text-[22px]">
          {{ t("items.detail.relations") }}
        </h2>
        <ElButton
          v-if="auth.canEditCatalog"
          @click="createRelationVisible = true"
        >
          <i class="pi pi-plus mr-2" aria-hidden="true"></i>
          {{ t("items.detail.createRelation") }}
        </ElButton>
      </div>
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
    </section>

    <ItemGallery :attributes="item.attributes" :schema="schema" />

    <ItemHistoryCard
      :item="item"
      :version="historyVersion"
      @reverted="(i) => (item = i)"
    />

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
import { computed, onMounted, ref, watch } from "vue";
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
  fieldLabel,
  fieldValue,
  imageCategories,
  isImageGallery,
  isLinks,
  itemLabel,
  linksOf,
  linkText,
  safeURL,
  orderedKeys,
  typeTitle,
  type Schema,
} from "@/lib/schema";
import ItemCover from "@/components/item/ItemCover.vue";
import ItemCollectionsCard from "@/components/item/ItemCollectionsCard.vue";
import ItemEditDialog from "@/components/item/ItemEditDialog.vue";
import ItemGallery from "@/components/item/ItemGallery.vue";
import ItemHistoryCard from "@/components/item/ItemHistoryCard.vue";
import AddToCollectionDialog from "@/components/item/AddToCollectionDialog.vue";
import RelationCreateDialog from "@/components/item/RelationCreateDialog.vue";
import RelationEditDialog from "@/components/item/RelationEditDialog.vue";
import RelationTree from "@/components/item/RelationTree.vue";
import NavIcon from "@/components/layout/NavIcon.vue";
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

/** Boutons secondaires : tuile icône + texte sur mobile, bouton doux en ligne au-delà. */
const secondaryClass =
  "flex h-14 flex-col items-center justify-center gap-1 rounded-xl bg-soft px-2 text-xs font-semibold text-ink sm:h-10 sm:flex-row sm:gap-2 sm:rounded-[10px] sm:px-4 sm:text-sm";

const coverWidth = computed(() => (isMobile.value ? 108 : 200));

const schema = computed(
  () => item.value?.item_type?.attributes_schema as Schema | undefined,
);
const title = computed(() =>
  item.value ? itemLabel(item.value, lang.value) : "",
);

/** Valeur au-delà de laquelle un attribut (synopsis…) prend toute la largeur. */
const LONG_VALUE = 60;

// Liens cliquables : un champ « liens », ou un texte au format URL (site web d'une société).
function rowLinks(key: string, value: unknown) {
  if (isLinks(schema.value, key)) return linksOf(value);
  const prop = (
    schema.value?.properties as Record<string, Schema> | undefined
  )?.[key];
  const url = prop?.format === "uri" ? safeURL(value) : "";
  return url ? [{ url }] : [];
}

// Seules les clés présentes dans les données sont affichées, dans l'ordre du schéma ;
// la galerie a sa propre section.
const attributeRows = computed(() => {
  const it = item.value;
  if (!it) return [];
  return orderedKeys(schema.value, it.attributes)
    .filter((k) => k in it.attributes && !isImageGallery(schema.value, k))
    .map((key) => {
      const value = fieldValue(
        schema.value,
        key,
        it.attributes[key],
        lang.value,
      );
      return {
        key,
        label: fieldLabel(schema.value, key, lang.value),
        value,
        long: value.length > LONG_VALUE,
        links: rowLinks(key, it.attributes[key]),
      };
    });
});

const defaultImageUrl = computed(() =>
  item.value ? defaultImage(schema.value, item.value.attributes) : "",
);

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
