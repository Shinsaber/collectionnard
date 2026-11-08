<template>
  <!-- Modifier une collection (le type ne change pas) : nom, description, types d'item acceptés, tri par défaut. -->
  <ElDialog
    v-model="visible"
    :title="t('collections.detail.editTitle')"
    width="30rem"
  >
    <div class="flex flex-col gap-4">
      <div class="flex flex-col gap-1">
        <label for="col-name" class="text-sm font-medium">{{
          t("common.name")
        }}</label>
        <ElInput
          id="col-name"
          v-model="form.name"
          :placeholder="t('collections.list.namePlaceholder')"
        />
      </div>
      <div class="flex flex-col gap-1">
        <label for="col-desc" class="text-sm font-medium">{{
          t("common.description")
        }}</label>
        <ElInput
          id="col-desc"
          v-model="form.description"
          type="textarea"
          :rows="2"
        />
      </div>
      <CollectionSettingsFields
        v-model:item-type-ids="form.item_type_ids"
        v-model:default-sort="form.default_sort"
        :collection-type="collection.collection_type"
      />
    </div>
    <template #footer>
      <ElButton text @click="visible = false">{{
        t("common.cancel")
      }}</ElButton>
      <ElButton
        type="primary"
        :loading="saving"
        :disabled="!dirty || !form.name.trim()"
        data-test="collection-save"
        @click="save"
      >
        <i class="pi pi-check mr-2" aria-hidden="true"></i>
        {{ t("common.save") }}
      </ElButton>
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
import { ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { collectionApi } from "@/api/client";
import { useDirty } from "@/composables/useDirty";
import { useNotify } from "@/composables/useNotify";
import CollectionSettingsFields from "./CollectionSettingsFields.vue";
import type { Collection } from "@/api/types";

const props = defineProps<{ collection: Collection }>();
const emit = defineEmits<{ saved: [collection: Collection] }>();
const visible = defineModel<boolean>("visible", { required: true });

const { t } = useI18n();
const { notifyError, notifySuccess } = useNotify();
const form = ref({
  name: "",
  description: "",
  item_type_ids: [] as string[],
  default_sort: "",
});
const saving = ref(false);
// « Enregistrer » reste grisé tant que rien n'a changé.
const { dirty, markClean } = useDirty(() => form.value);

function reset() {
  form.value = {
    name: props.collection.name,
    description: props.collection.description ?? "",
    item_type_ids: [...(props.collection.item_type_ids ?? [])],
    default_sort: props.collection.default_sort ?? "",
  };
  markClean();
}

// Rempli à l'ouverture, avant le premier rendu du dialogue.
watch(visible, (open) => open && reset());

async function save() {
  saving.value = true;
  try {
    const updated = await collectionApi.update(props.collection.id, {
      name: form.value.name.trim(),
      description: form.value.description,
      item_type_ids: form.value.item_type_ids,
      default_sort: form.value.default_sort,
    });
    notifySuccess(t("collections.detail.updated"));
    visible.value = false;
    emit("saved", updated);
  } catch (e) {
    notifyError(e);
  } finally {
    saving.value = false;
  }
}
</script>
