<template>
  <div v-if="loading" class="flex justify-center py-12">
    <ProgressSpinner />
  </div>

  <div v-else-if="item">
    <div class="flex items-center gap-3 mb-6">
      <Button icon="pi pi-arrow-left" text @click="router.back()" />
      <div>
        <h1 class="text-2xl font-bold text-gray-800">{{ itemLabel }}</h1>
        <div class="flex items-center gap-2 mt-1">
          <Tag :value="item.item_type?.name ?? '?'" severity="info" />
          <Tag
            v-for="t in item.tags ?? []"
            :key="t.id"
            :value="t.name"
            severity="secondary"
            class="text-xs"
          />
        </div>
      </div>
      <div class="flex-1"></div>
      <Button
        v-if="auth.isAdmin"
        icon="pi pi-pencil"
        label="Modifier"
        outlined
        @click="openEdit"
        class="mr-2"
      />
      <Button
        icon="pi pi-plus"
        label="Ajouter à une collection"
        outlined
        @click="openAddToCollection"
        class="mr-2"
      />
      <Button
        icon="pi pi-sitemap"
        label="Graphe"
        outlined
        @click="router.push(`/items/${item.id}/graph`)"
      />
    </div>

    <!-- Attributes -->
    <div class="bg-white rounded-lg shadow p-6 mb-6">
      <h2 class="text-lg font-semibold text-gray-700 mb-4">Attributs</h2>
      <div class="flex gap-6">
        <!-- Default image (portrait) -->
        <div v-if="defaultImageUrl" class="flex-shrink-0">
          <img
            :src="defaultImageUrl"
            class="w-48 h-auto rounded-lg shadow object-cover"
            :style="{ aspectRatio: defaultImageRatio }"
            alt="Image par défaut"
          />
        </div>

        <!-- Attribute grid -->
        <div class="flex-1 grid grid-cols-1 md:grid-cols-2 gap-3">
          <template v-for="key in orderedAttributeKeys" :key="key">
            <div v-if="!isImageGallery(key)" class="flex flex-col">
              <span class="text-xs text-gray-400 uppercase">{{
                attrLabel(key)
              }}</span>
              <span class="text-gray-800">{{
                displayValue(item.attributes[key])
              }}</span>
            </div>
          </template>
        </div>
      </div>
    </div>

    <!-- Images gallery with tabs -->
    <div v-if="imageGalleryField" class="bg-white rounded-lg shadow p-6 mb-6">
      <h2 class="text-lg font-semibold text-gray-700 mb-4">
        {{ attrLabel(imageGalleryField.key) }}
      </h2>
      <Tabs :value="imageGalleryField.categories[0]?.name ?? ''">
        <TabList>
          <Tab
            v-for="cat in imageGalleryField.categories"
            :key="cat.name"
            :value="cat.name"
          >
            <span class="capitalize">{{ cat.name }}</span>
            <span class="ml-1 text-xs text-gray-400"
              >({{
                getAttrGalleryCategory(
                  item.attributes[imageGalleryField.key],
                  cat.name,
                ).length
              }})</span
            >
          </Tab>
        </TabList>
        <TabPanels>
          <TabPanel
            v-for="cat in imageGalleryField.categories"
            :key="cat.name"
            :value="cat.name"
          >
            <div class="flex flex-wrap gap-3 pt-3">
              <div
                v-for="(url, imgIdx) in getAttrGalleryCategory(
                  item.attributes[imageGalleryField.key],
                  cat.name,
                )"
                :key="imgIdx"
                class="relative"
              >
                <img
                  :src="url"
                  class="w-32 object-cover rounded-lg border"
                  :style="{ aspectRatio: cat.ratio || '1/1' }"
                  :class="{
                    'ring-2 ring-blue-500':
                      getAttrGalleryDefault(
                        item.attributes[imageGalleryField.key],
                      ) === url,
                  }"
                />
                <span
                  v-if="
                    getAttrGalleryDefault(
                      item.attributes[imageGalleryField.key],
                    ) === url
                  "
                  class="absolute top-1 left-1 bg-blue-500 text-white text-[10px] px-1.5 py-0.5 rounded"
                  >défaut</span
                >
              </div>
              <div
                v-if="
                  getAttrGalleryCategory(
                    item.attributes[imageGalleryField.key],
                    cat.name,
                  ).length === 0
                "
                class="text-sm text-gray-400 italic py-4"
              >
                Aucune image dans cette catégorie.
              </div>
            </div>
          </TabPanel>
        </TabPanels>
      </Tabs>
    </div>

    <!-- Relations -->
    <div class="bg-white rounded-lg shadow p-6">
      <div class="flex items-center justify-between mb-4">
        <h2 class="text-lg font-semibold text-gray-700">Relations</h2>
        <Button
          icon="pi pi-plus"
          label="Créer une relation"
          outlined
          size="small"
          @click="openCreateRelation"
        />
      </div>

      <div v-if="relations.length === 0" class="text-sm text-gray-400 italic">
        Aucune relation trouvée.
      </div>

      <div v-else class="flex flex-col gap-5">
        <div v-for="group in groupedRelations" :key="group.typeId">
          <!-- Group header -->
          <!-- 
          <div
            class="flex items-center gap-2 pb-2 border-b border-gray-200 mb-1"
          >
            <Tag :value="group.typeName" class="text-xs" />
            <span class="text-xs text-gray-400"
              >({{ group.relations.length }})</span
            >
          </div> -->

          <!-- Relation rows via recursive component -->
          <RelationTree
            :relations="group.relations"
            :context-item-id="item!.id"
            :show-delete="true"
            @delete="confirmDeleteRelation"
            @edit="openEditRelation"
          />
        </div>
      </div>
    </div>

    <!-- Collections contenant cet item -->
    <div
      v-if="itemCollections.length > 0"
      class="bg-white rounded-lg shadow p-6 mt-6"
    >
      <h2 class="text-lg font-semibold text-gray-700 mb-4">
        Dans vos collections
      </h2>
      <div class="flex flex-col gap-2">
        <div
          v-for="ic in itemCollections"
          :key="ic.collection.id"
          class="flex items-center gap-3 p-3 bg-gray-50 rounded-lg"
        >
          <Tag
            :value="ic.collection.collection_type?.name ?? '?'"
            severity="info"
          />
          <router-link
            :to="`/collections/${ic.collection.id}`"
            class="text-indigo-600 hover:underline font-medium"
          >
            {{ ic.collection.name }}
          </router-link>
          <div class="flex gap-2 flex-wrap text-xs ml-auto">
            <span
              v-for="(val, key) in ic.attributes"
              :key="String(key)"
              class="bg-gray-200 rounded px-2 py-0.5"
            >
              {{ ctxLabel(ic.collection, String(key)) }}: {{ val }}
            </span>
          </div>
        </div>
      </div>
    </div>

    <!-- Add to Collection Dialog -->
    <Dialog
      v-model:visible="addToCollectionVisible"
      header="Ajouter à une collection"
      modal
      :style="{ width: '500px' }"
    >
      <div class="flex flex-col gap-4">
        <div
          v-if="availableCollections.length === 0"
          class="text-sm text-gray-400 italic"
        >
          Aucune collection disponible (toutes contiennent déjà cet item, ou
          vous n'avez pas de collection).
        </div>
        <template v-else>
          <div class="flex flex-col gap-1">
            <label class="text-sm font-medium text-gray-700">Collection</label>
            <Select
              v-model="selectedCollectionId"
              :options="availableCollections"
              optionLabel="name"
              optionValue="id"
              placeholder="Choisir une collection…"
            />
          </div>

          <div
            v-if="
              selectedCollectionSchema &&
              Object.keys(selectedCollectionSchema.properties ?? {}).length > 0
            "
          >
            <label class="text-sm font-medium text-gray-700 mb-2 block"
              >Attributs contextuels</label
            >
            <DynamicForm
              :schema="selectedCollectionSchema"
              :modelValue="addToCollectionAttrs"
              @update:modelValue="addToCollectionAttrs = $event"
            />
          </div>
        </template>
      </div>

      <template #footer>
        <Button label="Annuler" text @click="addToCollectionVisible = false" />
        <Button
          label="Ajouter"
          icon="pi pi-plus"
          :disabled="!selectedCollectionId"
          :loading="addToCollectionLoading"
          @click="submitAddToCollection"
        />
      </template>
    </Dialog>

    <!-- Create Relation Dialog -->
    <Dialog
      v-model:visible="createRelationVisible"
      header="Créer une relation"
      modal
      :style="{ width: '550px' }"
    >
      <div class="flex flex-col gap-4">
        <div class="flex flex-col gap-1">
          <label class="text-sm font-medium text-gray-700"
            >Type de relation</label
          >
          <Select
            v-model="newRelationTypeId"
            :options="availableRelationTypes"
            optionLabel="name"
            optionValue="id"
            placeholder="Choisir un type…"
            @change="onRelationTypeChange"
          />
        </div>

        <div v-if="newRelationTypeId" class="flex flex-col gap-1">
          <label class="text-sm font-medium text-gray-700">Direction</label>
          <Select
            v-model="newRelationDirection"
            :options="directionOptions"
            optionLabel="label"
            optionValue="value"
          />
        </div>

        <div v-if="newRelationTypeId" class="flex flex-col gap-1">
          <label class="text-sm font-medium text-gray-700">Item lié</label>
          <Select
            v-model="newRelationTargetItemId"
            :options="searchedItems"
            optionLabel="_label"
            optionValue="id"
            filter
            :filterPlaceholder="'Rechercher un item…'"
            @filter="onItemFilter"
            :loading="searchingItems"
            placeholder="Sélectionner un item…"
          />
        </div>

        <div
          v-if="
            newRelationSchema &&
            Object.keys(newRelationSchema.properties ?? {}).length > 0
          "
        >
          <label class="text-sm font-medium text-gray-700 mb-2 block"
            >Attributs</label
          >
          <DynamicForm
            :schema="newRelationSchema"
            :modelValue="newRelationAttrs"
            @update:modelValue="newRelationAttrs = $event"
          />
        </div>
      </div>

      <template #footer>
        <Button label="Annuler" text @click="createRelationVisible = false" />
        <Button
          label="Créer"
          icon="pi pi-plus"
          :disabled="!newRelationTypeId || !newRelationTargetItemId"
          :loading="createRelationLoading"
          @click="submitCreateRelation"
        />
      </template>
    </Dialog>

    <!-- Edit Relation Dialog -->
    <Dialog
      v-model:visible="editRelationVisible"
      header="Modifier la relation"
      modal
      :style="{ width: '500px' }"
    >
      <div v-if="editingRelation" class="flex flex-col gap-4">
        <div class="flex items-center gap-2 text-sm text-gray-500">
          <Tag
            :value="editingRelation.relation_type?.name ?? '?'"
            class="text-xs"
          />
          <span>{{ editRelationLinkedLabel }}</span>
        </div>

        <div
          v-if="
            editRelationSchema &&
            Object.keys(editRelationSchema.properties ?? {}).length > 0
          "
        >
          <label class="text-sm font-medium text-gray-700 mb-2 block"
            >Attributs</label
          >
          <DynamicForm
            :schema="editRelationSchema"
            :modelValue="editRelationAttrs"
            @update:modelValue="editRelationAttrs = $event"
          />
        </div>
        <div v-else class="text-sm text-gray-400 italic">
          Ce type de relation n'a pas d'attributs modifiables.
        </div>
      </div>

      <template #footer>
        <Button label="Annuler" text @click="editRelationVisible = false" />
        <Button
          label="Enregistrer"
          icon="pi pi-check"
          :loading="editRelationLoading"
          @click="submitEditRelation"
        />
      </template>
    </Dialog>

    <!-- Edit Dialog -->
    <ItemEditDialog
      v-model:visible="editDialogVisible"
      :item="item"
      :item-types="[]"
      @saved="loadItem(item!.id)"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed, watch } from "vue";
import { useRouter } from "vue-router";
import Button from "primevue/button";
import Tag from "primevue/tag";
import ProgressSpinner from "primevue/progressspinner";
import Dialog from "primevue/dialog";
import Select from "primevue/select";
import Tabs from "primevue/tabs";
import TabList from "primevue/tablist";
import Tab from "primevue/tab";
import TabPanels from "primevue/tabpanels";
import TabPanel from "primevue/tabpanel";
import { useConfirm } from "primevue/useconfirm";
import { useToast } from "primevue/usetoast";
import {
  itemApi,
  collectionApi,
  relationApi,
  relationTypeApi,
} from "@/api/client";
import { extractApiError } from "@/api/client";
import { useAuthStore } from "@/stores/auth";
import ItemEditDialog from "@/components/item/ItemEditDialog.vue";
import RelationTree from "@/components/item/RelationTree.vue";
import DynamicForm from "@/components/common/DynamicForm.vue";
import type {
  Item,
  ItemRelation,
  ItemCollectionMembership,
  Collection,
  RelationType,
} from "@/api/types";

const props = defineProps<{ id: string }>();
const router = useRouter();
const auth = useAuthStore();
const confirm = useConfirm();
const toast = useToast();

const item = ref<Item | null>(null);
const relations = ref<ItemRelation[]>([]);
const loading = ref(true);
const itemCollections = ref<ItemCollectionMembership[]>([]);

const itemLabel = computed(() => {
  if (!item.value) return "";
  const attrs = item.value.attributes;
  for (const key of ["titre", "nom", "name", "title", "label"]) {
    const val = attrs[key];
    if (val == null) continue;
    if (typeof val === "object" && !Array.isArray(val)) {
      const obj = val as Record<string, unknown>;
      if (typeof obj["fr"] === "string") return obj["fr"];
      if (typeof obj["en"] === "string") return obj["en"];
    }
    return String(val);
  }
  return item.value.id.slice(0, 8);
});

function linkedItemLabel(rel: ItemRelation): string {
  const linked =
    rel.source_item_id === item.value?.id ? rel.target_item : rel.source_item;
  if (!linked) return "?";
  for (const key of ["titre", "nom", "name", "title", "label"]) {
    const val = linked.attributes?.[key];
    if (val == null) continue;
    if (typeof val === "object" && !Array.isArray(val)) {
      const obj = val as Record<string, unknown>;
      if (typeof obj["fr"] === "string") return obj["fr"];
      if (typeof obj["en"] === "string") return obj["en"];
    }
    return String(val);
  }
  return linked.id.slice(0, 8);
}

function attrLabel(key: string): string {
  const schema = item.value?.item_type?.attributes_schema as
    | Record<string, unknown>
    | undefined;
  const labels = (
    schema?.["x-labels"] as Record<string, Record<string, string>> | undefined
  )?.["fr"];
  return labels?.[key] ?? key;
}

function displayValue(value: unknown): string {
  if (value != null && typeof value === "object" && !Array.isArray(value)) {
    const obj = value as Record<string, unknown>;
    // i18n object: show current lang, fallback to first available
    if (typeof obj["fr"] === "string") return obj["fr"];
    if (typeof obj["en"] === "string") return obj["en"];
  }
  return String(value ?? "");
}

function isImageGallery(key: string): boolean {
  const schema = item.value?.item_type?.attributes_schema as
    | Record<string, unknown>
    | undefined;
  const props = schema?.properties as
    | Record<string, Record<string, unknown>>
    | undefined;
  return props?.[key]?.["x-widget"] === "image-gallery";
}

function getSchemaImageCategories(
  key: string,
): { name: string; ratio: string }[] {
  const schema = item.value?.item_type?.attributes_schema as
    | Record<string, unknown>
    | undefined;
  const props = schema?.properties as
    | Record<string, Record<string, unknown>>
    | undefined;
  const raw = props?.[key]?.["x-image-categories"];
  if (raw && typeof raw === "object" && !Array.isArray(raw)) {
    return Object.entries(raw as Record<string, { ratio?: string }>).map(
      ([name, v]) => ({ name, ratio: v?.ratio ?? "1/1" }),
    );
  }
  if (Array.isArray(raw)) {
    return (raw as string[]).map((name) => ({ name, ratio: "1/1" }));
  }
  return [];
}

function getAttrGalleryCategory(value: unknown, cat: string): string[] {
  if (!value || typeof value !== "object") return [];
  return ((value as Record<string, unknown>)[cat] as string[]) ?? [];
}

function getAttrGalleryDefault(value: unknown): string {
  if (!value || typeof value !== "object") return "";
  return ((value as Record<string, unknown>).default_image as string) ?? "";
}

const defaultImageUrl = computed(() => {
  if (!item.value) return "";
  for (const [key, value] of Object.entries(item.value.attributes)) {
    if (isImageGallery(key)) {
      const url = getAttrGalleryDefault(value);
      if (url) return url;
    }
  }
  return "";
});

const defaultImageRatio = computed(() => {
  if (!item.value || !defaultImageUrl.value) return "2/3";
  for (const [key, value] of Object.entries(item.value.attributes)) {
    if (isImageGallery(key)) {
      const url = getAttrGalleryDefault(value);
      if (!url) continue;
      const cats = getSchemaImageCategories(key);
      const data = value as Record<string, unknown>;
      for (const cat of cats) {
        const urls = (data[cat.name] as string[]) ?? [];
        if (urls.includes(url)) return cat.ratio;
      }
    }
  }
  return "2/3";
});

const imageGalleryField = computed<{
  key: string;
  categories: { name: string; ratio: string }[];
} | null>(() => {
  if (!item.value) return null;
  for (const key of Object.keys(item.value.attributes)) {
    if (isImageGallery(key)) {
      return {
        key,
        categories: getSchemaImageCategories(key),
      };
    }
  }
  return null;
});

const orderedAttributeKeys = computed<string[]>(() => {
  if (!item.value) return [];
  const schema = item.value.item_type?.attributes_schema as
    | Record<string, unknown>
    | undefined;
  const xOrder = (schema?.["x-order"] as string[]) ?? [];
  const allKeys = Object.keys(item.value.attributes);
  if (xOrder.length > 0) {
    return [
      ...xOrder.filter((k) => k in item.value!.attributes),
      ...allKeys.filter((k) => !xOrder.includes(k)),
    ];
  }
  return allKeys;
});

// ── Grouped relations ───────────────────────────────────────────────
const groupedRelations = computed(() => {
  if (!item.value) return [];
  const groups = new Map<
    string,
    { typeId: string; typeName: string; relations: ItemRelation[] }
  >();
  for (const rel of relations.value) {
    const typeId = rel.relation_type_id;
    if (!groups.has(typeId)) {
      groups.set(typeId, {
        typeId,
        typeName: rel.relation_type?.name ?? "?",
        relations: [],
      });
    }
    groups.get(typeId)!.relations.push(rel);
  }
  return Array.from(groups.values());
});

// Edit
const editDialogVisible = ref(false);

function openEdit() {
  editDialogVisible.value = true;
}

// Add to collection
const addToCollectionVisible = ref(false);
const addToCollectionLoading = ref(false);
const allCollections = ref<Collection[]>([]);
const selectedCollectionId = ref<string | null>(null);
const addToCollectionAttrs = ref<Record<string, unknown>>({});

const availableCollections = computed(() => {
  const alreadyIn = new Set(
    itemCollections.value.map((ic) => ic.collection.id),
  );
  return allCollections.value.filter((c) => !alreadyIn.has(c.id));
});

const selectedCollectionSchema = computed(() => {
  if (!selectedCollectionId.value) return null;
  const col = allCollections.value.find(
    (c) => c.id === selectedCollectionId.value,
  );
  return (
    (col?.collection_type?.attributes_schema as Record<string, unknown>) ?? null
  );
});

async function openAddToCollection() {
  selectedCollectionId.value = null;
  addToCollectionAttrs.value = {};
  try {
    const resp = await collectionApi.list({ per_page: 100 });
    allCollections.value = resp.data;
  } catch {
    allCollections.value = [];
  }
  addToCollectionVisible.value = true;
}

async function submitAddToCollection() {
  if (!selectedCollectionId.value || !item.value) return;
  addToCollectionLoading.value = true;
  try {
    await collectionApi.addItem(selectedCollectionId.value, {
      item_id: item.value.id,
      attributes: addToCollectionAttrs.value,
    });
    addToCollectionVisible.value = false;
    loadCollections(item.value.id);
  } finally {
    addToCollectionLoading.value = false;
  }
}

function ctxLabel(
  collection: ItemCollectionMembership["collection"],
  key: string,
): string {
  const schema = collection.collection_type?.attributes_schema as
    | Record<string, unknown>
    | undefined;
  const labels = (
    schema?.["x-labels"] as Record<string, Record<string, string>> | undefined
  )?.["fr"];
  return labels?.[key] ?? key;
}

// ── Delete relation ──────────────────────────────────────────────────
function confirmDeleteRelation(rel: ItemRelation) {
  const label = linkedItemLabel(rel);
  confirm.require({
    message: `Supprimer la relation avec « ${label} » ?`,
    header: "Confirmation",
    icon: "pi pi-exclamation-triangle",
    acceptLabel: "Supprimer",
    rejectLabel: "Annuler",
    acceptClass: "p-button-danger",
    accept: async () => {
      try {
        await relationApi.delete(rel.id);
        toast.add({
          severity: "success",
          summary: "Relation supprimée",
          life: 3000,
        });
        relations.value = relations.value.filter((r) => r.id !== rel.id);
      } catch (e) {
        const err = extractApiError(e);
        toast.add({
          severity: "error",
          summary: "Erreur",
          detail: err.message,
          life: 5000,
        });
      }
    },
  });
}

// ── Edit relation ───────────────────────────────────────────────────
const editRelationVisible = ref(false);
const editRelationLoading = ref(false);
const editingRelation = ref<ItemRelation | null>(null);
const editRelationAttrs = ref<Record<string, unknown>>({});

const editRelationSchema = computed(() => {
  if (!editingRelation.value) return null;
  return (
    (editingRelation.value.relation_type?.attributes_schema as Record<
      string,
      unknown
    >) ?? null
  );
});

const editRelationLinkedLabel = computed(() => {
  if (!editingRelation.value || !item.value) return "";
  return linkedItemLabel(editingRelation.value);
});

function openEditRelation(rel: ItemRelation) {
  editingRelation.value = rel;
  editRelationAttrs.value = { ...(rel.attributes ?? {}) };
  editRelationVisible.value = true;
}

async function submitEditRelation() {
  if (!editingRelation.value || !item.value) return;
  editRelationLoading.value = true;
  try {
    await relationApi.update(editingRelation.value.id, {
      attributes: editRelationAttrs.value,
    });
    toast.add({
      severity: "success",
      summary: "Relation modifiée",
      life: 3000,
    });
    editRelationVisible.value = false;
    relations.value = (await itemApi.relations(item.value.id)) ?? [];
  } catch (e) {
    const err = extractApiError(e);
    toast.add({
      severity: "error",
      summary: "Erreur",
      detail: err.message,
      life: 5000,
    });
  } finally {
    editRelationLoading.value = false;
  }
}

// ── Create relation ─────────────────────────────────────────────────
const createRelationVisible = ref(false);
const createRelationLoading = ref(false);
const allRelationTypes = ref<RelationType[]>([]);
const newRelationTypeId = ref<string | null>(null);
const newRelationDirection = ref<"source" | "target">("source");
const newRelationTargetItemId = ref<string | null>(null);
const newRelationAttrs = ref<Record<string, unknown>>({});
const searchedItems = ref<(Item & { _label: string })[]>([]);
const searchingItems = ref(false);

const availableRelationTypes = computed(() => {
  if (!item.value) return [];
  const typeId = item.value.item_type_id;
  return allRelationTypes.value.filter(
    (rt) =>
      rt.source_item_type_ids?.includes(typeId) ||
      rt.target_item_type_ids?.includes(typeId),
  );
});

const directionOptions = computed(() => {
  if (!item.value || !newRelationTypeId.value) return [];
  const rt = allRelationTypes.value.find(
    (r) => r.id === newRelationTypeId.value,
  );
  if (!rt) return [];
  const typeId = item.value.item_type_id;
  const opts: { label: string; value: "source" | "target" }[] = [];
  if (rt.source_item_type_ids?.includes(typeId))
    opts.push({ label: "→ sortant (cet item est la source)", value: "source" });
  if (rt.target_item_type_ids?.includes(typeId))
    opts.push({ label: "← entrant (cet item est la cible)", value: "target" });
  return opts;
});

const newRelationSchema = computed(() => {
  if (!newRelationTypeId.value) return null;
  const rt = allRelationTypes.value.find(
    (r) => r.id === newRelationTypeId.value,
  );
  return (rt?.attributes_schema as Record<string, unknown>) ?? null;
});

function onRelationTypeChange() {
  newRelationTargetItemId.value = null;
  newRelationAttrs.value = {};
  searchedItems.value = [];
  // default direction
  const firstDir = directionOptions.value[0];
  if (directionOptions.value.length === 1 && firstDir) {
    newRelationDirection.value = firstDir.value;
  } else {
    newRelationDirection.value = "source";
  }
  // load candidate items
  loadCandidateItems("");
}

async function loadCandidateItems(search: string) {
  if (!item.value || !newRelationTypeId.value) return;
  const rt = allRelationTypes.value.find(
    (r) => r.id === newRelationTypeId.value,
  );
  if (!rt) return;
  // determine which item types to search for (the "other side")
  const linkedTypeIds =
    newRelationDirection.value === "source"
      ? rt.target_item_type_ids
      : rt.source_item_type_ids;
  searchingItems.value = true;
  try {
    // load items for each linked type
    const results: Item[] = [];
    for (const tid of linkedTypeIds ?? []) {
      const resp = await itemApi.list({
        item_type_id: tid,
        search: search || undefined,
        per_page: 50,
      });
      results.push(...resp.data);
    }
    // exclude current item
    searchedItems.value = results
      .filter((i) => i.id !== item.value!.id)
      .map((i) => ({ ...i, _label: computeItemLabel(i) }));
  } finally {
    searchingItems.value = false;
  }
}

function onItemFilter(event: { value: string }) {
  loadCandidateItems(event.value);
}

function computeItemLabel(i: Item): string {
  const attrs = i.attributes;
  for (const key of ["titre", "nom", "name", "title", "label"]) {
    const val = attrs[key];
    if (val == null) continue;
    if (typeof val === "object" && !Array.isArray(val)) {
      const obj = val as Record<string, unknown>;
      if (typeof obj["fr"] === "string") return obj["fr"];
      if (typeof obj["en"] === "string") return obj["en"];
    }
    return String(val);
  }
  return `${i.item_type?.name ?? "Item"} – ${i.id.slice(0, 8)}`;
}

async function openCreateRelation() {
  newRelationTypeId.value = null;
  newRelationTargetItemId.value = null;
  newRelationDirection.value = "source";
  newRelationAttrs.value = {};
  searchedItems.value = [];
  try {
    const resp = await relationTypeApi.list({ per_page: 200 });
    allRelationTypes.value = resp.data;
  } catch {
    allRelationTypes.value = [];
  }
  createRelationVisible.value = true;
}

async function submitCreateRelation() {
  if (!item.value || !newRelationTypeId.value || !newRelationTargetItemId.value)
    return;
  createRelationLoading.value = true;
  try {
    const sourceId =
      newRelationDirection.value === "source"
        ? item.value.id
        : newRelationTargetItemId.value;
    const targetId =
      newRelationDirection.value === "source"
        ? newRelationTargetItemId.value
        : item.value.id;
    await relationApi.create({
      relation_type_id: newRelationTypeId.value,
      source_item_id: sourceId,
      target_item_id: targetId,
      attributes: newRelationAttrs.value,
    });
    toast.add({ severity: "success", summary: "Relation créée", life: 3000 });
    createRelationVisible.value = false;
    // reload relations
    relations.value = (await itemApi.relations(item.value.id)) ?? [];
  } catch (e) {
    const err = extractApiError(e);
    toast.add({
      severity: "error",
      summary: "Erreur",
      detail: err.message,
      life: 5000,
    });
  } finally {
    createRelationLoading.value = false;
  }
}

async function loadCollections(itemId: string) {
  try {
    itemCollections.value = await itemApi.collections(itemId);
  } catch {
    itemCollections.value = [];
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
