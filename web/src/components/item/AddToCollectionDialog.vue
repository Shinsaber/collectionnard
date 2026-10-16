<template>
  <ElDialog
    :model-value="visible"
    :title="t('items.addToCollection.title')"
    width="34rem"
    @update:model-value="$emit('update:visible', $event)"
  >
    <div class="flex flex-col gap-5">
      <p
        v-if="availableCollections.length === 0"
        class="text-sm italic text-muted"
      >
        {{ t("items.addToCollection.empty") }}
      </p>
      <template v-else>
        <!-- Choix de la collection : cartes radio (cible tactile pleine largeur). -->
        <fieldset class="flex flex-col gap-2">
          <legend
            class="mb-2 font-mono text-[11px] uppercase tracking-wider text-muted"
          >
            {{ t("items.addToCollection.collection") }}
          </legend>
          <label
            v-for="c in availableCollections"
            :key="c.id"
            class="flex min-h-14 cursor-pointer items-center gap-3 rounded-[14px] bg-surface px-3.5 transition-shadow"
            :class="
              selectedCollectionId === c.id
                ? 'shadow-[0_0_0_2px_var(--c-primary)]'
                : 'shadow-card'
            "
          >
            <input
              v-model="selectedCollectionId"
              type="radio"
              name="add-to-collection"
              :value="c.id"
              class="size-5 accent-primary"
            />
            <span class="flex min-w-0 flex-col">
              <span class="truncate text-[15px] font-bold">{{ c.name }}</span>
              <span class="text-xs text-muted">{{
                typeTitle(c.collection_type, lang) || "?"
              }}</span>
            </span>
          </label>
        </fieldset>

        <div
          v-if="
            selectedSchema &&
            Object.keys(selectedSchema.properties ?? {}).length > 0
          "
          class="flex flex-col gap-3"
        >
          <span
            class="font-mono text-[11px] uppercase tracking-wider text-muted"
            >{{ t("items.addToCollection.contextAttributes") }}</span
          >
          <DynamicForm
            ref="formRef"
            :schema="selectedSchema"
            :model-value="attrs"
            @update:model-value="attrs = $event"
          />
        </div>
      </template>
    </div>

    <template #footer>
      <ElButton text @click="$emit('update:visible', false)">{{
        t("common.cancel")
      }}</ElButton>
      <ElButton
        type="primary"
        :disabled="!selectedCollectionId"
        :loading="saving"
        @click="submit"
      >
        <i class="pi pi-plus mr-2" aria-hidden="true"></i>
        {{ t("common.add") }}
      </ElButton>
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { collectionApi } from "@/api/client";
import { useNotify } from "@/composables/useNotify";
import { useLang } from "@/composables/useLang";
import { typeTitle } from "@/lib/schema";
import DynamicForm from "@/components/common/DynamicForm.vue";
import type { Collection, Item } from "@/api/types";

const props = defineProps<{
  visible: boolean;
  item: Item;
  /** Collections contenant déjà l'item (exclues de la liste). */
  excludeIds: string[];
}>();

const emit = defineEmits<{
  "update:visible": [value: boolean];
  added: [];
}>();

const { t } = useI18n();
const { notifyError, notifySuccess } = useNotify();
const { lang } = useLang();

const collections = ref<Collection[]>([]);
const selectedCollectionId = ref<string | null>(null);
const attrs = ref<Record<string, unknown>>({});
const saving = ref(false);
const formRef = ref<InstanceType<typeof DynamicForm> | null>(null);

const availableCollections = computed(() => {
  const exclude = new Set(props.excludeIds);
  return collections.value.filter((c) => !exclude.has(c.id));
});

const selectedSchema = computed(() => {
  const col = collections.value.find(
    (c) => c.id === selectedCollectionId.value,
  );
  return (
    (col?.collection_type?.attributes_schema as Record<string, unknown>) ?? null
  );
});

watch(
  () => props.visible,
  async (open) => {
    if (!open) return;
    selectedCollectionId.value = null;
    attrs.value = {};
    try {
      collections.value = (await collectionApi.list({ per_page: 100 })).data;
    } catch (e) {
      collections.value = [];
      notifyError(e, t("items.addToCollection.collectionsUnavailable"));
    }
  },
);

async function submit() {
  if (!selectedCollectionId.value) return;
  if (formRef.value && !formRef.value.validate()) return;
  saving.value = true;
  try {
    await collectionApi.addItem(selectedCollectionId.value, {
      item_id: props.item.id,
      attributes: attrs.value,
    });
    notifySuccess(t("items.addToCollection.added"));
    emit("update:visible", false);
    emit("added");
  } catch (e) {
    notifyError(e);
  } finally {
    saving.value = false;
  }
}
</script>
