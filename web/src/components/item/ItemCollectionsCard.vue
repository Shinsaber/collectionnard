<template>
  <!-- Fiche d'un item : collections de l'utilisateur qui le contiennent, avec leurs attributs contextuels. -->
  <section class="flex flex-col gap-3">
    <h2 v-if="!bare" class="text-lg font-extrabold tracking-tight">
      {{ t("items.detail.inYourCollections") }}
    </h2>
    <article
      v-for="m in memberships"
      :key="m.collection.id"
      class="flex flex-col gap-3 rounded-2xl bg-surface p-4.5 shadow-card"
    >
      <div class="flex min-w-0 flex-col">
        <RouterLink
          :to="`/collections/${m.collection.id}`"
          class="truncate text-base font-bold text-ink hover:text-link"
          >{{ m.collection.name }}</RouterLink
        >
        <span class="text-[13px] text-muted">{{
          typeTitle(m.collection.collection_type, lang) || "?"
        }}</span>
      </div>
      <dl v-if="keysOf(m).length" class="flex flex-col gap-2">
        <div
          v-for="key in keysOf(m)"
          :key="key"
          class="flex justify-between gap-3 text-sm"
        >
          <dt class="text-muted">{{ fieldLabel(schemaOf(m), key, lang) }}</dt>
          <dd class="text-right font-medium">
            {{ fieldValue(schemaOf(m), key, m.attributes[key], lang) }}
          </dd>
        </div>
      </dl>
    </article>
    <button
      type="button"
      class="flex min-h-13 items-center gap-3 rounded-2xl border-[1.5px] border-dashed border-dash px-4.5 py-3 text-left text-sm text-muted hover:border-primary hover:text-link"
      @click="emit('add')"
    >
      <i class="pi pi-plus text-xs" aria-hidden="true"></i>
      {{
        memberships.length
          ? t("items.detail.addToCollection")
          : t("items.detail.notInCollection")
      }}
    </button>
  </section>
</template>

<script setup lang="ts">
import { useI18n } from "vue-i18n";
import { useLang } from "@/composables/useLang";
import { fieldLabel, fieldValue, orderedKeys, typeTitle } from "@/lib/schema";
import type { Schema } from "@/lib/schema";
import type { ItemCollectionMembership } from "@/api/types";

/** `bare` : sans titre (dans l'onglet « Dans mes collections » de la fiche). */
defineProps<{ memberships: ItemCollectionMembership[]; bare?: boolean }>();
const emit = defineEmits<{ add: [] }>();

const { t } = useI18n();
const { lang } = useLang();

function schemaOf(m: ItemCollectionMembership): Schema | undefined {
  return m.collection.collection_type?.attributes_schema as Schema | undefined;
}

/** Attributs renseignés, dans l'ordre du schéma du type de collection. */
function keysOf(m: ItemCollectionMembership): string[] {
  return orderedKeys(schemaOf(m), m.attributes).filter(
    (k) => k in m.attributes,
  );
}
</script>
