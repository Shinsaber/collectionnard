<template>
  <section
    v-if="rows.length"
    class="flex flex-col gap-3 rounded-2xl bg-surface px-4.5 py-4 shadow-card sm:px-7 sm:py-6"
    data-test="item-attributes"
  >
    <h2 class="text-lg font-extrabold tracking-tight">
      {{ t("items.detail.information") }}
    </h2>
    <dl
      class="grid grid-cols-1 text-[15px] sm:grid-cols-[minmax(8rem,12rem)_minmax(0,1fr)] sm:gap-x-5"
    >
      <template v-for="r in rows" :key="r.key">
        <dt
          class="pt-2.5 text-[13px] text-muted sm:border-b sm:border-line sm:py-2.5 sm:text-[15px]"
        >
          {{ r.label }}
        </dt>
        <dd
          class="min-w-0 border-b border-line pb-2.5 [overflow-wrap:anywhere] sm:py-2.5"
          :class="{ 'font-mono text-sm': r.mono }"
        >
          <AttributeValue :view="r.view" />
        </dd>
      </template>
    </dl>
  </section>
</template>

<script setup lang="ts">
// Bloc « Informations » d'une fiche : un seul bloc, tous les attributs dans l'ordre du schéma
// (lib/attributeView.ts) ; aucun champ métier codé en dur.
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useLang } from "@/composables/useLang";
import { attributeRows } from "@/lib/attributeView";
import type { Schema } from "@/lib/schema";
import AttributeValue from "./AttributeValue.vue";

const props = defineProps<{
  schema: Schema | undefined;
  attributes: Record<string, unknown>;
}>();
const { t } = useI18n();
const { lang } = useLang();

const rows = computed(() =>
  attributeRows(props.schema, props.attributes, lang.value),
);
</script>
