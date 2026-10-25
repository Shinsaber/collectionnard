<template>
  <ul
    v-if="view.kind === 'links'"
    class="flex flex-col gap-1.5"
    data-test="attribute-links"
  >
    <li v-for="l in view.links" :key="l.url">
      <a
        :href="l.url"
        target="_blank"
        rel="noopener noreferrer"
        class="font-semibold break-words text-link underline-offset-2 hover:underline"
        >{{ linkText(l) }}</a
      >
    </li>
  </ul>
  <dl
    v-else-if="view.kind === 'object'"
    class="grid grid-cols-[max-content_minmax(0,1fr)] gap-x-3.5 gap-y-1 text-sm"
  >
    <template v-for="r in view.rows" :key="r.key">
      <dt class="text-muted">{{ r.label }}</dt>
      <dd
        class="min-w-0 [overflow-wrap:anywhere]"
        :class="{ 'font-mono': r.mono }"
      >
        <AttributeValue :view="r.view" />
      </dd>
    </template>
  </dl>
  <ul v-else-if="view.kind === 'list'" class="flex flex-col gap-2">
    <li
      v-for="(v, i) in view.items"
      :key="i"
      class="border-l-2 border-dash pl-3"
    >
      <AttributeValue :view="v" />
    </li>
  </ul>
  <span v-else class="whitespace-pre-line [overflow-wrap:anywhere]">{{
    view.text
  }}</span>
</template>

<script setup lang="ts">
// Valeur d'un attribut (bloc « Informations ») : texte, liens par leur libellé, objet en
// sous-liste clé : valeur (récursif), liste. Tout passe à la ligne, rien ne déborde.
import { linkText } from "@/lib/schema";
import type { AttrView } from "@/lib/attributeView";

defineProps<{ view: AttrView }>();
</script>
