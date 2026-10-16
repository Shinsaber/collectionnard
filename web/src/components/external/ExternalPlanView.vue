<template>
  <div data-test="external-plan">
    <p class="text-xs text-muted mb-3">
      {{ t("admin.external.attribution", { date: retrievedAt }) }}
    </p>

    <section v-if="groups.main.length > 0" class="mb-6">
      <h3 class="font-semibold text-ink mb-2">
        {{ t("admin.external.groups.main") }}
      </h3>
      <ExternalNodeTable
        :plan="plan"
        :nodes="groups.main"
        :selected="selectedSet"
        :overrides="overrides"
        show-relation
        @toggle="toggle"
        @edit="openEdit"
      />
    </section>

    <section
      v-for="g in groups.editions"
      :key="g.edition.key"
      class="mb-6"
      data-test="plan-edition"
    >
      <h3 class="font-semibold text-ink mb-2">
        {{ heading(g.edition) }}
      </h3>
      <ExternalNodeTable
        :plan="plan"
        :nodes="[g.edition, ...g.publishers]"
        :selected="selectedSet"
        :overrides="overrides"
        @toggle="toggle"
        @edit="openEdit"
      />
      <div class="flex items-center gap-2 mt-3 mb-2">
        <h4 class="text-sm font-semibold text-ink">
          {{ t("admin.external.groups.volumes") }} ({{ g.volumes.length }})
        </h4>
        <ElButton size="small" text @click="setAll(g.volumes, true)">{{
          t("admin.external.selectAll")
        }}</ElButton>
        <ElButton size="small" text @click="setAll(g.volumes, false)">{{
          t("admin.external.selectNone")
        }}</ElButton>
      </div>
      <ExternalNodeTable
        :plan="plan"
        :nodes="g.volumes"
        :selected="selectedSet"
        :overrides="overrides"
        @toggle="toggle"
        @edit="openEdit"
      />
    </section>

    <p class="text-sm text-ink" data-test="plan-count">
      {{
        t("admin.external.selectedCount", {
          n: modelValue.length,
          total: plan.nodes.length,
          written: writtenCount(plan, modelValue, overrides),
        })
      }}
    </p>

    <ExternalNodeEditDialog
      v-model:visible="editOpen"
      :node="editing"
      :current="editing ? effectiveAttributes(editing, overrides) : {}"
      :corrected="!!editing && editing.key in overrides"
      @save="saveOverride"
      @restore="restoreOverride"
    />
  </div>
</template>

<script setup lang="ts">
// Aperçu d'un import externe : œuvre, univers et auteurs, puis chaque édition avec ses
// éditeurs et ses tomes ; v-model = clés des nœuds sélectionnés, v-model:overrides = corrections
// faites avant la simulation (un nœud corrigé est coché).
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useLang } from "@/composables/useLang";
import { useMetaTypes } from "@/stores/metaTypes";
import { itemLabel, typeTitle } from "@/lib/schema";
import {
  effectiveAttributes,
  groupPlan,
  writtenCount,
  type Overrides,
} from "@/lib/externalPlan";
import ExternalNodeEditDialog from "./ExternalNodeEditDialog.vue";
import ExternalNodeTable from "./ExternalNodeTable.vue";
import type { ExternalNode, ExternalPlan } from "@/api/types";

const props = defineProps<{
  plan: ExternalPlan;
  modelValue: string[];
  overrides: Overrides;
}>();
const emit = defineEmits<{
  "update:modelValue": [keys: string[]];
  "update:overrides": [overrides: Overrides];
}>();

const { t } = useI18n();
const { lang } = useLang();
const metaTypes = useMetaTypes();

const groups = computed(() => groupPlan(props.plan));
const selectedSet = computed(() => new Set(props.modelValue));
const retrievedAt = computed(() =>
  new Date(`${props.plan.retrieved_at}T00:00:00`).toLocaleDateString(
    lang.value,
  ),
);

/** « Édition : Dragon Ball » (nom du type, puis libellé de l'édition). */
function heading(edition: ExternalNode): string {
  const type = metaTypes.itemTypes.find((it) => it.id === edition.item_type_id);
  const label = itemLabel(
    { id: edition.key, attributes: edition.attributes, item_type: type },
    lang.value,
  );
  return t("admin.external.editionHeading", {
    type: typeTitle(type, lang.value),
    name: label,
  });
}

function toggle(key: string, on: boolean) {
  const rest = props.modelValue.filter((k) => k !== key);
  emit("update:modelValue", on ? [...rest, key] : rest);
}

// ── Corrections ─────────────────────────────────────────────────────
const editing = ref<ExternalNode | null>(null);
const editOpen = ref(false);

function openEdit(node: ExternalNode) {
  editing.value = node;
  editOpen.value = true;
}

function saveOverride(key: string, attributes: Record<string, unknown>) {
  emit("update:overrides", { ...props.overrides, [key]: attributes });
  if (!props.modelValue.includes(key)) toggle(key, true);
}

function restoreOverride(key: string) {
  const rest = { ...props.overrides };
  delete rest[key];
  emit("update:overrides", rest);
}

function setAll(nodes: ExternalNode[], on: boolean) {
  const keys = new Set(nodes.map((n) => n.key));
  const rest = props.modelValue.filter((k) => !keys.has(k));
  emit("update:modelValue", on ? [...rest, ...keys] : rest);
}
</script>
