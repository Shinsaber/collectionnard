<template>
  <AppSpinner v-if="loading" />

  <div v-else-if="item">
    <div class="flex items-center gap-3 mb-6">
      <ElButton text aria-label="Retour" @click="router.back()">
        <i class="pi pi-arrow-left" aria-hidden="true"></i>
      </ElButton>
      <div>
        <h1 class="text-2xl font-bold text-gray-800 dark:text-gray-100">
          {{ title }}
        </h1>
        <div class="flex items-center gap-2 mt-1">
          <ElTag type="info" size="small">{{
            item.item_type?.name ?? "?"
          }}</ElTag>
          <ElTag
            v-for="t in item.tags ?? []"
            :key="t.id"
            type="info"
            effect="plain"
            size="small"
            >{{ t.name }}</ElTag
          >
        </div>
      </div>
      <div class="flex-1"></div>
      <ElButton v-if="auth.isAdmin" plain @click="editDialogVisible = true">
        <i class="pi pi-pencil mr-2" aria-hidden="true"></i> Modifier
      </ElButton>
      <ElButton plain @click="addToCollectionVisible = true">
        <i class="pi pi-plus mr-2" aria-hidden="true"></i> Ajouter à une
        collection
      </ElButton>
      <ElButton plain @click="router.push(`/items//graph`)">
        <i class="pi pi-sitemap mr-2" aria-hidden="true"></i> Graphe
      </ElButton>
    </div>

    <!-- Attributes -->
    <div class="bg-white dark:bg-gray-800 rounded-lg shadow-sm p-6 mb-6">
      <h2 class="text-lg font-semibold text-gray-700 dark:text-gray-200 mb-4">
        Attributs
      </h2>
      <div class="flex gap-6">
        <div v-if="defaultImageUrl" class="shrink-0">
          <img
            :src="defaultImageUrl"
            class="w-48 h-auto rounded-lg shadow-sm object-cover"
            :style="{ aspectRatio: defaultImageRatio }"
            :alt="`Image par défaut : ${title}`"
          />
        </div>

        <dl class="flex-1 grid grid-cols-1 md:grid-cols-2 gap-3">
          <template v-for="key in orderedAttributeKeys" :key="key">
            <div v-if="!isImageGallery(schema, key)" class="flex flex-col">
              <dt class="text-xs text-gray-400 dark:text-gray-500 uppercase">
                {{ fieldLabel(schema, key, lang) }}
              </dt>
              <dd class="text-gray-800 dark:text-gray-200">
                {{ displayValue(item.attributes[key], lang) }}
              </dd>
            </div>
          </template>
        </dl>
      </div>
    </div>

    <ItemGallery :attributes="item.attributes" :schema="schema" />

    <!-- Relations -->
    <div class="bg-white dark:bg-gray-800 rounded-lg shadow-sm p-6">
      <div class="flex items-center justify-between mb-4">
        <h2 class="text-lg font-semibold text-gray-700 dark:text-gray-200">
          Relations
        </h2>
        <ElButton
          v-if="auth.isAdmin"
          plain
          size="small"
          @click="createRelationVisible = true"
        >
          <i class="pi pi-plus mr-2" aria-hidden="true"></i> Créer une relation
        </ElButton>
      </div>

      <div
        v-if="relations.length === 0"
        class="text-sm text-gray-400 dark:text-gray-500 italic"
      >
        Aucune relation trouvée.
      </div>

      <div v-else class="flex flex-col gap-5">
        <RelationTree
          v-for="group in groupedRelations"
          :key="group.typeId"
          :relations="group.relations"
          :context-item-id="item.id"
          :show-delete="auth.isAdmin"
          @delete="confirmDeleteRelation"
          @edit="openEditRelation"
        />
      </div>
    </div>

    <!-- Collections contenant cet item -->
    <div
      v-if="itemCollections.length > 0"
      class="bg-white dark:bg-gray-800 rounded-lg shadow-sm p-6 mt-6"
    >
      <h2 class="text-lg font-semibold text-gray-700 dark:text-gray-200 mb-4">
        Dans vos collections
      </h2>
      <div class="flex flex-col gap-2">
        <div
          v-for="ic in itemCollections"
          :key="ic.collection.id"
          class="flex items-center gap-3 p-3 bg-gray-50 dark:bg-gray-700 rounded-lg"
        >
          <ElTag type="info" size="small">{{
            ic.collection.collection_type?.name ?? "?"
          }}</ElTag>
          <router-link
            :to="`/collections/${ic.collection.id}`"
            class="text-indigo-600 dark:text-indigo-400 hover:underline font-medium"
          >
            {{ ic.collection.name }}
          </router-link>
          <div class="flex gap-2 flex-wrap text-xs ml-auto">
            <span
              v-for="(val, key) in ic.attributes"
              :key="String(key)"
              class="bg-gray-200 dark:bg-gray-600 dark:text-gray-300 rounded-sm px-2 py-0.5"
            >
              {{ ctxLabel(ic.collection, String(key)) }}:
              {{ displayValue(val, lang) }}
            </span>
          </div>
        </div>
      </div>
    </div>

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
      @saved="loadItem(item!.id)"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";
import { useRouter } from "vue-router";
import { useConfirmDialog } from "@/composables/useConfirmDialog";
import AppSpinner from "@/components/common/AppSpinner.vue";
import { itemApi, relationApi } from "@/api/client";
import { useAuthStore } from "@/stores/auth";
import { useLang } from "@/composables/useLang";
import { useNotify } from "@/composables/useNotify";
import {
  defaultImage,
  displayValue,
  fieldLabel,
  imageCategories,
  isImageGallery,
  itemLabel,
  orderedKeys,
  type Schema,
} from "@/lib/schema";
import ItemEditDialog from "@/components/item/ItemEditDialog.vue";
import ItemGallery from "@/components/item/ItemGallery.vue";
import AddToCollectionDialog from "@/components/item/AddToCollectionDialog.vue";
import RelationCreateDialog from "@/components/item/RelationCreateDialog.vue";
import RelationEditDialog from "@/components/item/RelationEditDialog.vue";
import RelationTree from "@/components/item/RelationTree.vue";
import type { Item, ItemCollectionMembership, ItemRelation } from "@/api/types";

const props = defineProps<{ id: string }>();
const router = useRouter();
const auth = useAuthStore();
const { confirm } = useConfirmDialog();
const { lang } = useLang();
const { notifyError, notifySuccess } = useNotify();

const item = ref<Item | null>(null);
const relations = ref<ItemRelation[]>([]);
const itemCollections = ref<ItemCollectionMembership[]>([]);
const loading = ref(true);

const editDialogVisible = ref(false);
const addToCollectionVisible = ref(false);
const createRelationVisible = ref(false);
const editRelationVisible = ref(false);
const editingRelation = ref<ItemRelation | null>(null);

const schema = computed(
  () => item.value?.item_type?.attributes_schema as Schema | undefined,
);
const title = computed(() =>
  item.value ? itemLabel(item.value, lang.value) : "",
);

// Seules les clés présentes dans les données sont affichées, dans l'ordre du schéma.
const orderedAttributeKeys = computed(() =>
  item.value
    ? orderedKeys(schema.value, item.value.attributes).filter(
        (k) => k in item.value!.attributes,
      )
    : [],
);

const defaultImageUrl = computed(() =>
  item.value ? defaultImage(schema.value, item.value.attributes) : "",
);

// Ratio de la catégorie qui contient l'image par défaut (sinon portrait 2/3).
const defaultImageRatio = computed(() => {
  const url = defaultImageUrl.value;
  if (!item.value || !url) return "2/3";
  for (const key of Object.keys(item.value.attributes)) {
    if (!isImageGallery(schema.value, key)) continue;
    const data = item.value.attributes[key] as Record<string, unknown>;
    for (const cat of imageCategories(schema.value, key)) {
      if (((data[cat.name] as string[]) ?? []).includes(url)) return cat.ratio;
    }
  }
  return "2/3";
});

const groupedRelations = computed(() => {
  const groups = new Map<
    string,
    { typeId: string; relations: ItemRelation[] }
  >();
  for (const rel of relations.value) {
    const g = groups.get(rel.relation_type_id) ?? {
      typeId: rel.relation_type_id,
      relations: [],
    };
    g.relations.push(rel);
    groups.set(rel.relation_type_id, g);
  }
  return [...groups.values()];
});

function ctxLabel(
  collection: ItemCollectionMembership["collection"],
  key: string,
): string {
  return fieldLabel(
    collection.collection_type?.attributes_schema as Schema | undefined,
    key,
    lang.value,
  );
}

function openEditRelation(rel: ItemRelation) {
  editingRelation.value = rel;
  editRelationVisible.value = true;
}

async function confirmDeleteRelation(rel: ItemRelation) {
  const linked =
    rel.source_item_id === item.value?.id ? rel.target_item : rel.source_item;
  const ok = await confirm({
    message: `Supprimer la relation avec « ${itemLabel(linked, lang.value)} » ?`,
    confirmText: "Supprimer",
    danger: true,
  });
  if (!ok) return;
  try {
    await relationApi.delete(rel.id);
    notifySuccess("Relation supprimée");
    relations.value = relations.value.filter((r) => r.id !== rel.id);
  } catch (e) {
    notifyError(e);
  }
}

async function reloadRelations() {
  if (!item.value) return;
  try {
    relations.value = (await itemApi.relations(item.value.id)) ?? [];
  } catch (e) {
    notifyError(e, "Relations indisponibles");
  }
}

async function loadCollections(itemId: string) {
  try {
    itemCollections.value = await itemApi.collections(itemId);
  } catch (e) {
    itemCollections.value = [];
    notifyError(e, "Collections indisponibles");
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
    loadCollections(id);
  } catch (e) {
    notifyError(e, "Item introuvable");
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
