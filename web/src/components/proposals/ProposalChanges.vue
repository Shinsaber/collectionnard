<template>
  <section class="flex flex-col gap-3" data-test="proposal-changes">
    <h2 class="text-lg font-extrabold tracking-tight">
      {{ t("proposals.changes") }}
    </h2>
    <ul class="flex flex-col gap-2.5">
      <li
        v-for="(it, idx) in payload.items ?? []"
        :key="it.id"
        class="flex flex-col gap-2 rounded-2xl bg-surface p-4 shadow-card"
        data-test="proposal-item"
      >
        <div class="flex flex-wrap items-center gap-2">
          <span
            class="rounded-full bg-soft px-2 py-0.5 font-mono text-[11px] uppercase"
            >{{
              base?.items?.[it.id]
                ? t("proposals.updated")
                : t("proposals.created")
            }}
            · {{ typeTitle(typeOf(it.item_type_id), lang) }}</span
          >
          <RouterLink
            v-if="base?.items?.[it.id]"
            :to="`/items/${it.id}`"
            class="font-bold text-link"
            >{{ label(it) }}</RouterLink
          >
          <span v-else class="font-bold">{{ label(it) }}</span>
          <ElButton
            v-if="editable"
            text
            size="small"
            class="ml-auto"
            :data-test="`edit-entry-${idx}`"
            @click="edit(idx)"
          >
            <i class="pi pi-pencil mr-1" aria-hidden="true"></i>
            {{ t("proposals.editEntry") }}
          </ElButton>
        </div>
        <dl
          v-if="changes(it).length"
          class="grid grid-cols-[auto_1fr] gap-x-3 gap-y-0.5 text-sm"
        >
          <template v-for="c in changes(it)" :key="c.key">
            <dt class="text-muted">
              {{ fieldLabel(schemaOf(it), c.key, lang, "pretty") }}
            </dt>
            <dd class="min-w-0 break-words">
              <template v-if="base?.items?.[it.id]">
                <del class="text-muted">{{ show(it, c.key, c.before) }}</del>
                →
              </template>
              <ins class="no-underline">{{ show(it, c.key, c.after) }}</ins>
            </dd>
          </template>
        </dl>
        <p v-else class="text-sm text-muted">{{ t("proposals.noChange") }}</p>
      </li>
    </ul>
    <p
      v-if="payload.relations?.length"
      class="rounded-2xl bg-surface p-4 text-sm text-muted shadow-card"
    >
      {{ t("proposals.relations", { n: payload.relations.length }) }}
    </p>

    <ElDialog v-model="dialog" :title="t('proposals.editEntry')" width="34rem">
      <DynamicForm
        v-if="editing !== null && editSchema"
        ref="formRef"
        v-model="draft"
        :schema="editSchema"
      />
      <template #footer>
        <ElButton text @click="dialog = false">{{
          t("common.cancel")
        }}</ElButton>
        <ElButton type="primary" @click="apply">{{
          t("common.save")
        }}</ElButton>
      </template>
    </ElDialog>
  </section>
</template>

<script setup lang="ts">
// Changements d'une proposition `catalog` : items créés ou modifiés (différences avec l'instantané de
// départ), relations ; correction d'un item avant envoi (auteur) ou avant acceptation (modérateur).
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useLang } from "@/composables/useLang";
import { useMetaTypes } from "@/stores/metaTypes";
import {
  fieldLabel,
  fieldValue,
  itemLabel,
  typeTitle,
  type Schema,
} from "@/lib/schema";
import { changedFields } from "@/lib/history";
import DynamicForm from "@/components/common/DynamicForm.vue";
import type { PayloadItem, Proposal, ProposalPayload } from "@/api/types";

const props = defineProps<{
  payload: ProposalPayload;
  base?: Proposal["base"];
  editable?: boolean;
}>();
const emit = defineEmits<{ "update:payload": [value: ProposalPayload] }>();
const { t } = useI18n();
const { lang } = useLang();
const metaTypes = useMetaTypes();

function typeOf(id: string) {
  return metaTypes.itemTypes.find((it) => it.id === id);
}
function schemaOf(it: PayloadItem): Schema | undefined {
  return typeOf(it.item_type_id)?.attributes_schema as Schema | undefined;
}
function label(it: PayloadItem): string {
  return itemLabel({ ...it, item_type: typeOf(it.item_type_id) }, lang.value);
}
function changes(it: PayloadItem) {
  return changedFields(props.base?.items?.[it.id] ?? null, it.attributes);
}
function show(it: PayloadItem, key: string, v: unknown): string {
  if (v === undefined || v === null) return "—";
  return fieldValue(schemaOf(it), key, v, lang.value) || "—";
}

const dialog = ref(false);
const editing = ref<number | null>(null);
const draft = ref<Record<string, unknown>>({});
const formRef = ref<InstanceType<typeof DynamicForm> | null>(null);
const editSchema = computed(() => {
  const it =
    editing.value === null ? null : props.payload.items?.[editing.value];
  return it ? schemaOf(it) : undefined;
});

function edit(idx: number) {
  editing.value = idx;
  draft.value = { ...(props.payload.items?.[idx]?.attributes ?? {}) };
  dialog.value = true;
}

function apply() {
  if (formRef.value && !formRef.value.validate()) return;
  const items = [...(props.payload.items ?? [])];
  items[editing.value!] = {
    ...items[editing.value!]!,
    attributes: draft.value,
  };
  emit("update:payload", { ...props.payload, items });
  dialog.value = false;
}
</script>
