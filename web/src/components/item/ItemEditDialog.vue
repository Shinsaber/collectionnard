<template>
  <ElDialog
    :model-value="visible"
    :title="
      propose
        ? item
          ? t('proposals.suggest')
          : t('proposals.proposeItem')
        : item
          ? t('items.editDialog.editTitle')
          : t('items.editDialog.createTitle')
    "
    width="34rem"
    @update:model-value="$emit('update:visible', $event)"
  >
    <div class="flex flex-col gap-4">
      <!-- Proposition (membre, P4-3b) : un titre, relu par un contributeur -->
      <div v-if="propose" class="flex flex-col gap-1">
        <label for="proposal-title" class="text-sm font-medium">{{
          t("proposals.proposalTitle")
        }}</label>
        <ElInput
          id="proposal-title"
          v-model="proposalTitle"
          maxlength="200"
          :placeholder="defaultTitle"
        />
      </div>
      <!-- Type selector (create mode only) -->
      <div v-if="!item" class="flex flex-col gap-1">
        <label for="item-type" class="text-sm font-medium">{{
          t("common.type")
        }}</label>
        <ElSelect
          id="item-type"
          v-model="form.item_type_id"
          :placeholder="t('items.editDialog.chooseType')"
          @change="onTypeChange"
        >
          <ElOption
            v-for="type in itemTypes"
            :key="type.id"
            :label="typeTitle(type, lang)"
            :value="type.id"
          />
        </ElSelect>
      </div>
      <div v-else class="text-sm text-muted">
        {{ t("items.editDialog.typeLabel") }}
        <strong>{{ typeTitle(item.item_type, lang) }}</strong>
      </div>

      <DynamicForm
        v-if="currentSchema"
        ref="dynamicFormRef"
        v-model="form.attributes"
        :schema="currentSchema"
      />

      <ElAlert
        v-if="duplicates.length"
        type="warning"
        :closable="false"
        show-icon
        :title="t('proposals.duplicates.title')"
        data-test="duplicates"
      >
        <p>{{ t("proposals.duplicates.help") }}</p>
        <ul class="mt-1 list-disc pl-5">
          <li v-for="d in duplicates" :key="d.item.id">
            <RouterLink
              :to="`/items/${d.item.id}`"
              target="_blank"
              class="font-semibold text-link"
              >{{ itemLabel(d.item, lang) }}</RouterLink
            >
            <span class="font-mono text-xs">
              · {{ d.field }} {{ d.value }}</span
            >
          </li>
        </ul>
      </ElAlert>

      <div class="flex flex-col gap-1">
        <label for="item-tags" class="text-sm font-medium">{{
          t("items.editDialog.tags")
        }}</label>
        <ElInput
          id="item-tags"
          v-model="form.tagsText"
          :placeholder="t('items.editDialog.tagsPlaceholder')"
        />
      </div>
    </div>
    <template #footer>
      <ElButton text @click="$emit('update:visible', false)">{{
        t("common.cancel")
      }}</ElButton>
      <ElButton
        type="primary"
        :loading="saving"
        :disabled="!!item && !dirty"
        data-test="item-save"
        @click="save"
      >
        <i class="pi pi-check mr-2" aria-hidden="true"></i>
        {{
          duplicates.length
            ? t("proposals.duplicates.confirm")
            : propose
              ? t("proposals.propose")
              : item
                ? t("common.save")
                : t("common.create")
        }}
      </ElButton>
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
import { reactive, computed, watch, ref } from "vue";
import { useI18n } from "vue-i18n";
import { itemApi, proposalApi, extractApiError } from "@/api/client";
import { useLang } from "@/composables/useLang";
import { useDirty } from "@/composables/useDirty";
import { useNotify } from "@/composables/useNotify";
import { itemLabel, typeTitle } from "@/lib/schema";
import { itemPayload } from "@/lib/proposals";
import { identifierPrefill } from "@/lib/barcode";
import DynamicForm from "@/components/common/DynamicForm.vue";
import type { Duplicate, Item, ItemType } from "@/api/types";

const props = defineProps<{
  visible: boolean;
  item?: Item | null;
  itemTypes: ItemType[];
  /** Membre (P4-3b) : envoie une proposition au lieu d'écrire dans le catalogue. */
  propose?: boolean;
  /** Code-barres scanné (M3) : préremplit, au choix du type, ses identifiants qui l'acceptent. */
  prefillCode?: string;
}>();

const emit = defineEmits<{
  "update:visible": [value: boolean];
  saved: [];
}>();

const { t } = useI18n();
const { lang } = useLang();
const { notifyError, notifySuccess } = useNotify();
const saving = ref(false);
const dynamicFormRef = ref<InstanceType<typeof DynamicForm> | null>(null);

const proposalTitle = ref("");
/** Doublons trouvés au premier clic : un second clic enregistre quand même. */
const duplicates = ref<Duplicate[]>([]);
const checkedFor = ref("");
const defaultTitle = computed(() => {
  const label = itemLabel(
    {
      id: props.item?.id ?? "",
      attributes: form.attributes,
      item_type:
        props.item?.item_type ??
        props.itemTypes.find((it) => it.id === form.item_type_id),
    },
    lang.value,
  );
  return props.item
    ? t("proposals.correctionTitle", { label })
    : t("proposals.newTitle", { label });
});

const form = reactive({
  item_type_id: "",
  attributes: {} as Record<string, unknown>,
  tagsText: "",
});

function tagList(text: string): string[] {
  return text
    .split(",")
    .map((tag) => tag.trim())
    .filter(Boolean);
}

// En modification, « Enregistrer » reste grisé tant que rien n'a changé.
const { dirty, markClean } = useDirty(() => ({
  attributes: form.attributes,
  tags: tagList(form.tagsText),
}));

const currentSchema = computed(() => {
  if (props.item?.item_type?.attributes_schema) {
    return props.item.item_type.attributes_schema;
  }
  const found = props.itemTypes.find((it) => it.id === form.item_type_id);
  return found?.attributes_schema ?? null;
});

/** Nouveau type choisi : attributs vides, sauf les identifiants préremplis par un code scanné. */
function onTypeChange() {
  form.attributes = props.prefillCode
    ? identifierPrefill(currentSchema.value, props.prefillCode)
    : {};
}

// Reset form when dialog opens or item changes
watch(
  () => props.visible,
  (open) => {
    if (!open) return;
    if (props.item) {
      form.item_type_id = props.item.item_type_id;
      form.attributes = { ...props.item.attributes };
      form.tagsText = (props.item.tags ?? []).map((tag) => tag.name).join(", ");
    } else {
      form.item_type_id = "";
      form.attributes = {};
      form.tagsText = "";
    }
    proposalTitle.value = "";
    duplicates.value = [];
    checkedFor.value = "";
    markClean();
  },
);

/** Vérifie les identifiants (x-identifier) une fois par contenu ; vrai s'il faut s'arrêter. */
async function hasDuplicates(): Promise<boolean> {
  const key = JSON.stringify([form.item_type_id, form.attributes]);
  if (checkedFor.value === key) return false;
  checkedFor.value = key;
  duplicates.value = await itemApi.duplicates(
    form.item_type_id,
    form.attributes,
    props.item?.id,
  );
  return duplicates.value.length > 0;
}

async function save() {
  if (dynamicFormRef.value && !dynamicFormRef.value.validate()) return;
  saving.value = true;
  try {
    const tags = tagList(form.tagsText);
    if (await hasDuplicates()) return;

    if (props.propose) {
      await proposalApi.create({
        kind: "catalog",
        title: proposalTitle.value.trim() || defaultTitle.value,
        payload: itemPayload({
          id: props.item?.id,
          item_type_id: form.item_type_id,
          attributes: form.attributes,
          tags,
        }),
        submit: true,
      });
      notifySuccess(t("proposals.sent"));
    } else if (props.item) {
      await itemApi.update(props.item.id, {
        attributes: form.attributes,
        tags,
      });
      notifySuccess(t("items.editDialog.updated"));
    } else {
      await itemApi.create({
        item_type_id: form.item_type_id,
        attributes: form.attributes,
        tags,
      });
      notifySuccess(t("items.editDialog.created"));
    }
    emit("update:visible", false);
    emit("saved");
  } catch (e: unknown) {
    const apiErr = extractApiError(e);
    if (apiErr.fieldErrors && dynamicFormRef.value) {
      dynamicFormRef.value.setErrors(apiErr.fieldErrors);
    }
    notifyError(e);
  } finally {
    saving.value = false;
  }
}
</script>
