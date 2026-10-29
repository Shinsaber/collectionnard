<template>
  <!-- Ajout d'un item à une collection : recherche, choix, puis attributs contextuels. -->
  <ElDialog
    v-model="visible"
    :title="t('collections.detail.addItem')"
    width="34rem"
    @open="reset"
  >
    <div class="flex flex-col gap-4">
      <!-- Type de collection limité : la recherche ne porte que sur les types permis. -->
      <div v-if="allowedTypes.length" class="flex flex-col gap-1">
        <label :for="typeId" class="text-sm font-medium">{{
          t("common.type")
        }}</label>
        <ElSelect :id="typeId" v-model="typeFilter" @change="searchItems">
          <ElOption
            v-for="it in allowedTypes"
            :key="it.id"
            :label="typeTitle(it, lang)"
            :value="it.id"
          />
        </ElSelect>
      </div>
      <div class="flex flex-col gap-1">
        <label :for="searchId" class="text-sm font-medium">{{
          t("collections.detail.searchItem")
        }}</label>
        <div class="flex gap-2">
          <ElInput
            :id="searchId"
            v-model="searchText"
            :placeholder="t('collections.detail.searchPlaceholder')"
            clearable
            @keyup.enter="searchItems"
          />
          <ElButton :aria-label="t('common.search')" @click="searchItems">
            <i class="pi pi-search" aria-hidden="true"></i>
          </ElButton>
          <ScanButton @detected="onScanned" />
        </div>
      </div>

      <ElTable
        v-if="searchResults.length"
        :data="searchResults"
        highlight-current-row
        row-key="id"
        size="small"
        class="cursor-pointer"
        @current-change="selectedItem = $event"
      >
        <ElTableColumn :label="t('common.type')" width="110">
          <template #default="{ row }">{{
            typeTitle((row as Item).item_type, lang) || "?"
          }}</template>
        </ElTableColumn>
        <ElTableColumn :label="t('collections.detail.label')">
          <template #default="{ row }">{{
            itemLabel(row as Item, lang)
          }}</template>
        </ElTableColumn>
      </ElTable>

      <div v-if="selectedItem && schema">
        <h4 class="mb-2 text-sm font-medium">
          {{ t("collections.detail.contextAttributes") }}
        </h4>
        <DynamicForm ref="formRef" v-model="attributes" :schema="schema" />
      </div>
    </div>
    <template #footer>
      <ElButton text @click="visible = false">{{
        t("common.cancel")
      }}</ElButton>
      <ElButton
        type="primary"
        :disabled="!selectedItem"
        :loading="saving"
        @click="addItem"
      >
        <i class="pi pi-plus mr-2" aria-hidden="true"></i>
        {{ t("common.add") }}
      </ElButton>
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
import { computed, ref, useId } from "vue";
import { useI18n } from "vue-i18n";
import { collectionApi, itemApi } from "@/api/client";
import { useLang } from "@/composables/useLang";
import { useNotify } from "@/composables/useNotify";
import { useMetaTypes } from "@/stores/metaTypes";
import { itemLabel, typeTitle, type Schema } from "@/lib/schema";
import DynamicForm from "@/components/common/DynamicForm.vue";
import ScanButton from "@/components/common/ScanButton.vue";
import type { Item } from "@/api/types";

const props = defineProps<{
  collectionId: string;
  /** Schéma du CollectionType : attributs contextuels saisis à l'ajout. */
  schema: Schema | undefined;
  /** Types d'item permis par le CollectionType (vide = tous). */
  allowedTypeIds?: string[];
}>();
const emit = defineEmits<{ added: [] }>();
const visible = defineModel<boolean>("visible", { required: true });

const { t } = useI18n();
const { lang } = useLang();
const { notifyError, notifySuccess } = useNotify();
const searchId = useId();
const typeId = useId();
const metaTypes = useMetaTypes();

// Types permis, dans l'ordre d'affichage (celui de la liste des types).
const allowedTypes = computed(() =>
  metaTypes.itemTypes.filter((it) => props.allowedTypeIds?.includes(it.id)),
);
const typeFilter = ref<string>();

const searchText = ref("");
const searchResults = ref<Item[]>([]);
const selectedItem = ref<Item | null>(null);
const attributes = ref<Record<string, unknown>>({});
const formRef = ref<InstanceType<typeof DynamicForm> | null>(null);
const saving = ref(false);

async function reset() {
  searchText.value = "";
  searchResults.value = [];
  selectedItem.value = null;
  attributes.value = {};
  typeFilter.value = undefined;
  if (!props.allowedTypeIds?.length) return;
  await metaTypes.loadItemTypes().catch((e) => notifyError(e));
  typeFilter.value = allowedTypes.value[0]?.id;
}

async function searchItems() {
  try {
    const res = await itemApi.list({
      search: searchText.value,
      per_page: 10,
      ...(typeFilter.value ? { item_type_id: typeFilter.value } : {}),
    });
    searchResults.value = res.data;
  } catch (e) {
    notifyError(e, t("collections.detail.searchFailed"));
  }
}

/** Code-barres lu (lot M2) : cherché comme du texte, les identifiants sont indexés. */
function onScanned(code: string) {
  searchText.value = code;
  void searchItems();
}

async function addItem() {
  if (!selectedItem.value) return;
  if (formRef.value && !formRef.value.validate()) return;
  saving.value = true;
  try {
    await collectionApi.addItem(props.collectionId, {
      item_id: selectedItem.value.id,
      attributes: attributes.value,
    });
    notifySuccess(t("collections.detail.itemAdded"));
    visible.value = false;
    emit("added");
  } catch (e) {
    notifyError(e);
  } finally {
    saving.value = false;
  }
}
</script>
