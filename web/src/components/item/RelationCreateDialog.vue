<template>
  <ElDialog
    :model-value="visible"
    title="Créer une relation"
    width="34rem"
    @update:model-value="$emit('update:visible', $event)"
  >
    <div class="flex flex-col gap-4">
      <div class="flex flex-col gap-1">
        <label
          for="relation-type-select"
          class="text-sm font-medium text-gray-700 dark:text-gray-300"
          >Type de relation</label
        >
        <ElSelect
          id="relation-type-select"
          v-model="relationTypeId"
          placeholder="Choisir un type…"
          @change="onRelationTypeChange"
        >
          <ElOption
            v-for="rt in availableRelationTypes"
            :key="rt.id"
            :label="rt.name"
            :value="rt.id"
          />
        </ElSelect>
      </div>

      <div v-if="relationTypeId" class="flex flex-col gap-1">
        <label
          for="relation-direction-select"
          class="text-sm font-medium text-gray-700 dark:text-gray-300"
          >Direction</label
        >
        <ElSelect
          id="relation-direction-select"
          v-model="direction"
          @change="loadCandidates('')"
        >
          <ElOption
            v-for="o in directionOptions"
            :key="o.value"
            :label="o.label"
            :value="o.value"
          />
        </ElSelect>
      </div>

      <div v-if="relationTypeId" class="flex flex-col gap-1">
        <label
          for="relation-item-select"
          class="text-sm font-medium text-gray-700 dark:text-gray-300"
          >Item lié</label
        >
        <ElSelect
          id="relation-item-select"
          v-model="targetItemId"
          filterable
          remote
          :remote-method="loadCandidates"
          :loading="searching"
          placeholder="Sélectionner un item… (tapez pour rechercher)"
        >
          <ElOption
            v-for="c in candidates"
            :key="c.id"
            :label="c._label"
            :value="c.id"
          >
            <div class="flex items-center gap-2">
              <ElTag size="small" type="info">{{
                c.item_type?.name ?? "?"
              }}</ElTag>
              <span>{{ c._label }}</span>
            </div>
          </ElOption>
        </ElSelect>
      </div>

      <div v-if="hasFields">
        <span
          class="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2 block"
          >Attributs</span
        >
        <DynamicForm
          ref="formRef"
          :schema="schema!"
          :model-value="attrs"
          @update:model-value="attrs = $event"
        />
      </div>
    </div>

    <template #footer>
      <ElButton text @click="$emit('update:visible', false)">Annuler</ElButton>
      <ElButton
        type="primary"
        :disabled="!relationTypeId || !targetItemId"
        :loading="saving"
        @click="submit"
      >
        <i class="pi pi-plus mr-2" aria-hidden="true"></i> Créer
      </ElButton>
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { itemApi, relationApi } from "@/api/client";
import { useLang } from "@/composables/useLang";
import { useNotify } from "@/composables/useNotify";
import { useMetaTypes } from "@/stores/metaTypes";
import { itemLabel } from "@/lib/schema";
import DynamicForm from "@/components/common/DynamicForm.vue";
import type { Item, ItemRelation } from "@/api/types";

type Direction = "source" | "target";

const props = defineProps<{
  visible: boolean;
  item: Item;
  /** Relations existantes de l'item (pour exclure les items déjà liés). */
  relations: ItemRelation[];
}>();

const emit = defineEmits<{
  "update:visible": [value: boolean];
  created: [];
}>();

const { lang } = useLang();
const { notifyError, notifySuccess } = useNotify();
const metaTypes = useMetaTypes();

const relationTypeId = ref<string | null>(null);
const direction = ref<Direction>("source");
const targetItemId = ref<string | null>(null);
const attrs = ref<Record<string, unknown>>({});
const candidates = ref<(Item & { _label: string })[]>([]);
const searching = ref(false);
const saving = ref(false);
const formRef = ref<InstanceType<typeof DynamicForm> | null>(null);

const availableRelationTypes = computed(() => {
  const typeId = props.item.item_type_id;
  return metaTypes.relationTypes.filter(
    (rt) =>
      rt.source_item_type_ids?.includes(typeId) ||
      rt.target_item_type_ids?.includes(typeId),
  );
});

const relationType = computed(() =>
  metaTypes.relationTypes.find((r) => r.id === relationTypeId.value),
);

const directionOptions = computed(() => {
  const rt = relationType.value;
  if (!rt) return [];
  const typeId = props.item.item_type_id;
  const opts: { label: string; value: Direction }[] = [];
  if (rt.source_item_type_ids?.includes(typeId))
    opts.push({ label: "→ sortant (cet item est la source)", value: "source" });
  if (rt.target_item_type_ids?.includes(typeId))
    opts.push({ label: "← entrant (cet item est la cible)", value: "target" });
  return opts;
});

const schema = computed(
  () =>
    (relationType.value?.attributes_schema as Record<string, unknown>) ?? null,
);
const hasFields = computed(
  () => !!schema.value && Object.keys(schema.value.properties ?? {}).length > 0,
);

watch(
  () => props.visible,
  async (open) => {
    if (!open) return;
    relationTypeId.value = null;
    targetItemId.value = null;
    direction.value = "source";
    attrs.value = {};
    candidates.value = [];
    try {
      await metaTypes.loadRelationTypes();
    } catch (e) {
      notifyError(e, "Types de relation indisponibles");
    }
  },
);

function onRelationTypeChange() {
  targetItemId.value = null;
  attrs.value = {};
  candidates.value = [];
  const first = directionOptions.value[0];
  direction.value =
    directionOptions.value.length === 1 && first ? first.value : "source";
  loadCandidates("");
}

/** Items de « l'autre côté » de la relation, hors item courant et items déjà liés. */
async function loadCandidates(search: string) {
  const rt = relationType.value;
  if (!rt) return;
  const linkedTypeIds =
    direction.value === "source"
      ? rt.target_item_type_ids
      : rt.source_item_type_ids;
  if (!linkedTypeIds?.length) return;
  searching.value = true;
  try {
    const responses = await Promise.all(
      linkedTypeIds.map((tid) =>
        itemApi.list({
          item_type_id: tid,
          search: search || undefined,
          per_page: 50,
        }),
      ),
    );
    const alreadyLinked = new Set(
      props.relations
        .filter((r) => r.relation_type_id === rt.id)
        .map((r) =>
          direction.value === "source" ? r.target_item_id : r.source_item_id,
        ),
    );
    candidates.value = responses
      .flatMap((resp) => resp.data ?? [])
      .filter((i) => i.id !== props.item.id && !alreadyLinked.has(i.id))
      .map((i) => ({ ...i, _label: itemLabel(i, lang.value) }));
  } catch (e) {
    notifyError(e, "Recherche impossible");
  } finally {
    searching.value = false;
  }
}

async function submit() {
  if (!relationTypeId.value || !targetItemId.value) return;
  if (formRef.value && !formRef.value.validate()) return;
  saving.value = true;
  try {
    const outgoing = direction.value === "source";
    await relationApi.create({
      relation_type_id: relationTypeId.value,
      source_item_id: outgoing ? props.item.id : targetItemId.value,
      target_item_id: outgoing ? targetItemId.value : props.item.id,
      attributes: attrs.value,
    });
    notifySuccess("Relation créée");
    emit("update:visible", false);
    emit("created");
  } catch (e) {
    notifyError(e);
  } finally {
    saving.value = false;
  }
}
</script>
