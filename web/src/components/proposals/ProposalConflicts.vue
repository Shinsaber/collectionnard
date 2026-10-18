<template>
  <section
    class="flex flex-col gap-3 rounded-2xl bg-surface p-5 shadow-card"
    data-test="proposal-conflicts"
  >
    <div class="flex flex-col gap-1">
      <h2 class="text-lg font-extrabold tracking-tight">
        {{ t("proposals.conflicts.title") }}
      </h2>
      <p class="text-sm text-muted">{{ t("proposals.conflicts.help") }}</p>
    </div>
    <div v-for="c in conflicts" :key="c.id" class="flex flex-col gap-2">
      <RouterLink
        v-if="c.kind === 'item'"
        :to="`/items/${c.id}`"
        class="font-bold text-link"
        >{{ labelOf(c) }}</RouterLink
      >
      <span v-if="!c.current" class="text-sm text-danger">{{
        t("proposals.conflicts.deleted")
      }}</span>
      <div
        v-for="field in conflictFields(c)"
        :key="field"
        class="grid grid-cols-1 gap-2 sm:grid-cols-[10rem_1fr_1fr] sm:items-center"
      >
        <span class="text-sm text-muted">{{
          fieldLabel(schemaOf(c), field, lang, "pretty")
        }}</span>
        <ElRadioGroup
          :model-value="choice(c.id, field)"
          class="contents!"
          @update:model-value="set(c.id, field, $event as Side)"
        >
          <ElRadio
            value="current"
            border
            class="mr-0! h-auto! py-1.5!"
            :data-test="`pick-current-${field}`"
          >
            <span class="text-xs text-muted">{{
              t("proposals.conflicts.current")
            }}</span>
            {{ show(c, field, c.current?.[field]) }}
          </ElRadio>
          <ElRadio
            value="proposed"
            border
            class="mr-0! h-auto! py-1.5!"
            :data-test="`pick-proposed-${field}`"
          >
            <span class="text-xs text-muted">{{
              t("proposals.conflicts.proposed")
            }}</span>
            {{ show(c, field, c.proposed?.[field]) }}
          </ElRadio>
        </ElRadioGroup>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
// Conflits à trois volets d'une proposition (P4-3b) : pour chaque champ changé des deux côtés depuis la
// soumission, le modérateur garde la valeur actuelle ou la proposée (fusion : lib/proposals.ts).
import { useI18n } from "vue-i18n";
import { useLang } from "@/composables/useLang";
import { useMetaTypes } from "@/stores/metaTypes";
import { fieldLabel, fieldValue, itemLabel, type Schema } from "@/lib/schema";
import { conflictFields, type Side } from "@/lib/proposals";
import type { ProposalConflict, ProposalPayload } from "@/api/types";

const props = defineProps<{
  conflicts: ProposalConflict[];
  payload: ProposalPayload;
  choices: Record<string, Record<string, Side>>;
}>();
const emit = defineEmits<{
  "update:choices": [value: Record<string, Record<string, Side>>];
}>();
const { t } = useI18n();
const { lang } = useLang();
const metaTypes = useMetaTypes();

function typeIdOf(c: ProposalConflict): string | undefined {
  if (c.kind === "item")
    return props.payload.items?.find((it) => it.id === c.id)?.item_type_id;
  return undefined;
}
function schemaOf(c: ProposalConflict): Schema | undefined {
  if (c.kind === "relation") {
    const rel = props.payload.relations?.find((r) => r.id === c.id);
    return metaTypes.relationTypes.find((rt) => rt.id === rel?.relation_type_id)
      ?.attributes_schema as Schema | undefined;
  }
  return metaTypes.itemTypes.find((it) => it.id === typeIdOf(c))
    ?.attributes_schema as Schema | undefined;
}
function labelOf(c: ProposalConflict): string {
  return itemLabel(
    {
      id: c.id,
      attributes: c.current ?? c.proposed ?? {},
      item_type: metaTypes.itemTypes.find((it) => it.id === typeIdOf(c)),
    },
    lang.value,
  );
}
function show(c: ProposalConflict, key: string, v: unknown): string {
  if (v === undefined || v === null) return "—";
  return fieldValue(schemaOf(c), key, v, lang.value) || "—";
}
function choice(id: string, field: string): Side {
  return props.choices[id]?.[field] ?? "proposed";
}
function set(id: string, field: string, side: Side) {
  emit("update:choices", {
    ...props.choices,
    [id]: { ...(props.choices[id] ?? {}), [field]: side },
  });
}
</script>
