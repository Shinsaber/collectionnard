<template>
  <!-- Item lié présent dans mes collections : nom de la collection, ou leur nombre ;
       au clic, chaque collection avec les attributs contextuels de l'item (état, prix…). -->
  <ElPopover
    trigger="click"
    placement="bottom"
    :width="280"
    @before-enter="loadTypes"
  >
    <template #reference>
      <button
        type="button"
        class="inline-flex max-w-44 shrink-0 items-center gap-1 rounded-full bg-tint px-2 py-0.5 text-xs font-semibold text-tint-ink"
        :aria-label="
          t('items.relations.inCollectionsLabel', {
            names: collections.map((c) => c.name).join(', '),
          })
        "
      >
        <NavIcon name="collections" :size="12" />
        <span class="truncate">{{
          collections.length === 1
            ? collections[0]!.name
            : t("items.relations.inCollections", collections.length)
        }}</span>
      </button>
    </template>
    <p class="mb-1.5 font-mono text-[11px] uppercase tracking-wider text-muted">
      {{ t("items.relations.inCollectionsTitle") }}
    </p>
    <ul class="flex flex-col gap-2">
      <li v-for="c in collections" :key="c.id" class="flex flex-col">
        <RouterLink
          :to="`/collections/${c.id}`"
          class="truncate rounded-md px-1.5 py-1 text-sm font-semibold text-link hover:bg-soft"
          >{{ c.name }}</RouterLink
        >
        <dl v-if="keysOf(c).length" class="flex flex-col gap-0.5 px-1.5">
          <div
            v-for="key in keysOf(c)"
            :key="key"
            class="flex justify-between gap-3 text-xs"
          >
            <dt class="text-muted">{{ fieldLabel(schemaOf(c), key, lang) }}</dt>
            <dd class="text-right font-medium">
              {{ fieldValue(schemaOf(c), key, c.attributes[key], lang) }}
            </dd>
          </div>
        </dl>
      </li>
    </ul>
  </ElPopover>
</template>

<script setup lang="ts">
import { useI18n } from "vue-i18n";
import { useLang } from "@/composables/useLang";
import { useNotify } from "@/composables/useNotify";
import { useMetaTypes } from "@/stores/metaTypes";
import { fieldLabel, fieldValue, orderedKeys, type Schema } from "@/lib/schema";
import NavIcon from "@/components/layout/NavIcon.vue";
import type { RelatedCollection } from "@/api/types";

defineProps<{ collections: RelatedCollection[] }>();
const { t } = useI18n();
const { lang } = useLang();
const { notifyError } = useNotify();
const metaTypes = useMetaTypes();

/** Schémas des types de collection (cache partagé), chargés à la première ouverture. */
function loadTypes() {
  metaTypes.loadCollectionTypes().catch((e) => notifyError(e));
}

function schemaOf(c: RelatedCollection): Schema | undefined {
  return metaTypes.collectionTypes.find((ct) => ct.id === c.collection_type_id)
    ?.attributes_schema as Schema | undefined;
}

/** Attributs renseignés, dans l'ordre du schéma du type de collection. */
function keysOf(c: RelatedCollection): string[] {
  return orderedKeys(schemaOf(c), c.attributes).filter(
    (k) => k in c.attributes,
  );
}
</script>
