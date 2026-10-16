<template>
  <!-- Contenu du panneau de filtres du catalogue (colonne de droite sur grand écran, feuille sinon). -->
  <div class="flex flex-col gap-5">
    <div class="flex flex-col gap-1.5">
      <label
        :for="`${uid}-type`"
        class="font-mono text-[11px] uppercase tracking-wider text-muted"
        >{{ t("common.type") }}</label
      >
      <ElSelect
        :id="`${uid}-type`"
        :model-value="itemTypeId ?? ''"
        :placeholder="t('items.list.allTypes')"
        clearable
        filterable
        @update:model-value="emit('update:type', $event || undefined)"
      >
        <ElOption
          v-for="type in itemTypes"
          :key="type.id"
          :label="typeTitle(type, lang)"
          :value="type.id"
        />
      </ElSelect>
    </div>

    <div class="flex flex-col gap-1.5">
      <label
        :for="`${uid}-tag`"
        class="font-mono text-[11px] uppercase tracking-wider text-muted"
        >{{ t("items.list.tag") }}</label
      >
      <ElInput
        :id="`${uid}-tag`"
        v-model="tagDraft"
        :placeholder="t('items.list.tagPlaceholder')"
        clearable
        @change="applyTag"
        @clear="applyTag"
      />
    </div>

    <div class="flex flex-col gap-1.5">
      <span class="font-mono text-[11px] uppercase tracking-wider text-muted">{{
        t("items.search.filters")
      }}</span>
      <ItemSearchFilters
        v-if="schema"
        :schema="schema"
        :model-value="attrs"
        @update:model-value="emit('update:attrs', $event)"
      />
      <p v-else class="text-sm text-muted">
        {{ t("items.search.chooseTypeFirst") }}
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, useId, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useLang } from "@/composables/useLang";
import { typeTitle, type Schema } from "@/lib/schema";
import ItemSearchFilters from "@/components/item/ItemSearchFilters.vue";
import type { ItemType } from "@/api/types";

const props = defineProps<{
  itemTypes: ItemType[];
  itemTypeId?: string;
  tag?: string;
  attrs: Record<string, string[]>;
  /** Schéma du type choisi : les filtres d'attributs en dépendent. */
  schema?: Schema;
}>();

const emit = defineEmits<{
  "update:type": [id: string | undefined];
  "update:tag": [tag: string | undefined];
  "update:attrs": [attrs: Record<string, string[]>];
}>();

const { t } = useI18n();
const { lang } = useLang();
const uid = useId();

// Le tag s'applique à la validation (Entrée / sortie du champ), pas à chaque frappe.
const tagDraft = ref(props.tag ?? "");
watch(
  () => props.tag,
  (v) => {
    tagDraft.value = v ?? "";
  },
);

function applyTag() {
  const next = tagDraft.value.trim() || undefined;
  if (next !== props.tag) emit("update:tag", next);
}
</script>
