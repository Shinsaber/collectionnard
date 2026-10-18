<template>
  <div>
    <!-- Bureau / tablette : une colonne par attribut contextuel du type de collection. -->
    <div
      class="hidden overflow-hidden rounded-2xl bg-surface shadow-card md:block"
    >
      <ElTable
        v-loading="loading"
        :data="items"
        row-key="id"
        :empty-text="t('collections.detail.empty')"
      >
        <ElTableColumn width="68" class-name="cover-cell">
          <template #default="{ row }">
            <ItemCover
              v-if="(row as CollectionItem).item"
              :item="(row as CollectionItem).item!"
              :label="labelOf(row as CollectionItem)"
              :width="36"
            />
          </template>
        </ElTableColumn>
        <ElTableColumn :label="t('collections.detail.item')" min-width="240">
          <template #default="{ row }">
            <div class="flex flex-col gap-0.5 py-1">
              <RouterLink
                v-if="linkItems"
                :to="`/items/${(row as CollectionItem).item_id}`"
                class="text-[15px] font-bold text-ink hover:text-link"
                >{{ labelOf(row as CollectionItem) }}</RouterLink
              >
              <span v-else class="text-[15px] font-bold text-ink">{{
                labelOf(row as CollectionItem)
              }}</span>
              <span class="font-mono text-xs uppercase text-muted">{{
                typeTitle((row as CollectionItem).item?.item_type, lang) || "?"
              }}</span>
            </div>
          </template>
        </ElTableColumn>
        <ElTableColumn
          v-for="key in attrKeys"
          :key="key"
          :label="fieldLabel(schema, key, lang)"
          min-width="130"
        >
          <template #default="{ row }">
            <span class="text-sm">{{
              valueOf(row as CollectionItem, key)
            }}</span>
          </template>
        </ElTableColumn>
        <ElTableColumn v-if="!readonly" width="104" align="right">
          <template #default="{ row }">
            <div class="flex justify-end gap-1">
              <ElButton
                text
                circle
                :aria-label="t('collections.detail.editAttributes')"
                @click="emit('edit', row as CollectionItem)"
              >
                <i class="pi pi-pencil" aria-hidden="true"></i>
              </ElButton>
              <ElButton
                text
                circle
                type="danger"
                :aria-label="t('collections.detail.removeFromCollection')"
                @click="emit('remove', row as CollectionItem)"
              >
                <i class="pi pi-trash" aria-hidden="true"></i>
              </ElButton>
            </div>
          </template>
        </ElTableColumn>
      </ElTable>
    </div>

    <!-- Mobile : cartes ; attributs renseignés en ligne, actions dans un menu. -->
    <ul v-loading="loading" class="flex flex-col gap-2.5 md:hidden">
      <li
        v-for="ci in items"
        :key="ci.id"
        class="flex gap-3 rounded-2xl bg-surface p-2.5 shadow-card"
      >
        <ItemCover
          v-if="ci.item"
          :item="ci.item"
          :label="labelOf(ci)"
          :width="48"
        />
        <div class="flex min-w-0 flex-1 flex-col justify-center gap-1.5">
          <RouterLink
            v-if="linkItems"
            :to="`/items/${ci.item_id}`"
            class="text-[15px] font-bold leading-tight text-ink"
            >{{ labelOf(ci) }}</RouterLink
          >
          <span v-else class="text-[15px] font-bold leading-tight text-ink">{{
            labelOf(ci)
          }}</span>
          <dl
            v-if="filledKeys(ci).length"
            class="flex flex-wrap gap-x-3 gap-y-1 text-[13px]"
          >
            <div v-for="key in filledKeys(ci)" :key="key" class="flex gap-1">
              <dt class="text-muted">{{ fieldLabel(schema, key, lang) }} :</dt>
              <dd class="font-medium">{{ valueOf(ci, key) }}</dd>
            </div>
          </dl>
        </div>
        <ElDropdown
          v-if="!readonly"
          trigger="click"
          class="self-center"
          @command="(cmd: 'edit' | 'remove') => onCommand(cmd, ci)"
        >
          <button
            type="button"
            class="flex size-11 items-center justify-center rounded-[10px] text-muted"
            :aria-label="
              t('collections.detail.itemActions', { label: labelOf(ci) })
            "
          >
            <i class="pi pi-ellipsis-v" aria-hidden="true"></i>
          </button>
          <template #dropdown>
            <ElDropdownMenu>
              <ElDropdownItem command="edit">
                <i class="pi pi-pencil mr-2" aria-hidden="true"></i>
                {{ t("collections.detail.editAttributes") }}
              </ElDropdownItem>
              <ElDropdownItem command="remove" class="text-danger!">
                <i class="pi pi-trash mr-2" aria-hidden="true"></i>
                {{ t("collections.detail.removeFromCollection") }}
              </ElDropdownItem>
            </ElDropdownMenu>
          </template>
        </ElDropdown>
      </li>
      <li
        v-if="!loading && items.length === 0"
        class="rounded-2xl bg-surface p-6 text-center text-sm text-muted shadow-card"
      >
        {{ t("collections.detail.empty") }}
      </li>
    </ul>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useLang } from "@/composables/useLang";
import {
  fieldLabel,
  fieldValue,
  isImageGallery,
  itemLabel,
  orderedKeys,
  typeTitle,
  type Schema,
} from "@/lib/schema";
import ItemCover from "@/components/item/ItemCover.vue";
import type { CollectionItem } from "@/api/types";

// Items d'une collection : colonnes / attributs tirés du schéma du CollectionType.
const props = withDefaults(
  defineProps<{
    items: CollectionItem[];
    schema: Schema | undefined;
    loading: boolean;
    /** Lecture seule (page publique, lien de partage) : ni modification ni retrait. */
    readonly?: boolean;
    /** Lien vers la fiche de l'item (réservée aux comptes connectés). */
    linkItems?: boolean;
  }>(),
  { readonly: false, linkItems: true },
);

const emit = defineEmits<{
  edit: [ci: CollectionItem];
  remove: [ci: CollectionItem];
}>();

const { t } = useI18n();
const { lang } = useLang();

/** Attributs contextuels du type de collection, dans l'ordre du schéma (hors galeries). */
const attrKeys = computed(() =>
  orderedKeys(props.schema, {}).filter((k) => !isImageGallery(props.schema, k)),
);

function labelOf(ci: CollectionItem): string {
  return itemLabel(ci.item, lang.value);
}

function valueOf(ci: CollectionItem, key: string): string {
  if (!(key in (ci.attributes ?? {}))) return "";
  return fieldValue(props.schema, key, ci.attributes[key], lang.value);
}

function onCommand(cmd: "edit" | "remove", ci: CollectionItem) {
  if (cmd === "edit") emit("edit", ci);
  else emit("remove", ci);
}

function filledKeys(ci: CollectionItem): string[] {
  return attrKeys.value.filter((k) => valueOf(ci, k) !== "");
}
</script>
