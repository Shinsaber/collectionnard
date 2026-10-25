<template>
  <section class="flex flex-col gap-3.5" data-test="item-history">
    <div class="flex flex-col gap-1">
      <h2
        v-if="!bare"
        class="text-xl font-extrabold tracking-tight sm:text-[22px]"
      >
        {{ t("history.title") }}
      </h2>
      <p
        v-if="history?.created || latest"
        class="text-sm text-muted"
        data-test="attribution"
      >
        <template v-if="history?.created">{{
          t("history.created", {
            author: authorName(history.created),
            date: date(history.created.created_at),
          })
        }}</template>
        <template v-if="latest && latest.version > 1">
          ·
          {{
            t("history.updated", {
              author: authorName(latest),
              date: date(latest.created_at),
            })
          }}</template
        >
      </p>
    </div>

    <p
      v-if="history && history.entries.length === 0"
      class="rounded-2xl bg-surface p-5 text-sm text-muted shadow-card"
    >
      {{ t("history.none") }}
    </p>
    <ol
      v-else-if="history"
      class="flex flex-col overflow-hidden rounded-2xl bg-surface shadow-card"
    >
      <li
        v-for="e in shown"
        :key="`${e.kind}-${e.entity_id}-${e.version}`"
        class="flex flex-col gap-1.5 border-b border-line px-4 py-3 last:border-b-0"
        data-test="history-entry"
      >
        <div class="flex flex-wrap items-center gap-2 text-sm">
          <span class="font-mono text-xs text-muted">{{
            dateTime(e.created_at)
          }}</span>
          <span class="font-semibold">{{ authorName(e) }}</span>
          <span
            class="rounded-full bg-soft px-2 py-0.5 font-mono text-[11px] uppercase"
            >{{ e.kind === "item" ? t("history.item") : relationTitle(e) }} ·
            {{ t(`history.kinds.${changeKind(e)}`) }}</span
          >
          <span class="text-xs text-muted">{{ originText(e) }}</span>
          <ElButton
            v-if="canRevert(e)"
            text
            size="small"
            class="ml-auto"
            :data-test="`revert-${e.version}`"
            @click="revert(e)"
          >
            <i class="pi pi-replay mr-1" aria-hidden="true"></i>
            {{ t("history.revert") }}
          </ElButton>
        </div>
        <RouterLink
          v-if="e.relation"
          :to="`/items/${otherEnd(e)}`"
          class="text-sm text-link"
          >{{ t("history.otherItem") }}</RouterLink
        >
        <dl
          v-if="changedFields(e.before, e.after).length"
          class="grid grid-cols-[auto_1fr] gap-x-3 gap-y-0.5 text-sm"
        >
          <template v-for="c in changedFields(e.before, e.after)" :key="c.key">
            <dt class="text-muted">{{ label(e, c.key) }}</dt>
            <dd class="min-w-0 break-words">
              <del class="text-muted">{{ show(e, c.key, c.before) }}</del>
              → <ins class="no-underline">{{ show(e, c.key, c.after) }}</ins>
            </dd>
          </template>
        </dl>
      </li>
    </ol>
    <ElButton
      v-if="history && history.entries.length > LIMIT"
      text
      class="self-start"
      @click="all = !all"
    >
      {{
        all
          ? t("history.showLess")
          : t("history.showAll", { n: history.entries.length })
      }}
    </ElButton>
    <p v-if="history?.truncated" class="text-xs text-muted">
      {{ t("history.truncated") }}
    </p>
  </section>
</template>

<script setup lang="ts">
// Historique d'un item (P4-3a) : ses versions et celles de ses relations, auteur (selon la
// visibilité actuelle de son profil, calculée par le serveur), origine, différences champ par
// champ ; retour à une version pour les admins et contributeurs.
import { computed, onMounted, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { itemApi } from "@/api/client";
import { useLang } from "@/composables/useLang";
import { useNotify } from "@/composables/useNotify";
import { useConfirmDialog } from "@/composables/useConfirmDialog";
import { useAuthStore } from "@/stores/auth";
import { useMetaTypes } from "@/stores/metaTypes";
import { fieldLabel, fieldValue, typeTitle, type Schema } from "@/lib/schema";
import { changeKind, changedFields, latestItemVersion } from "@/lib/history";
import type { HistoryEntry, Item, ItemHistory } from "@/api/types";

const LIMIT = 10;

const props = defineProps<{
  item: Item;
  /** Change après une modification de la fiche : l'historique est relu. */
  version?: number;
  /** Sans titre (onglet « Historique » de la fiche). */
  bare?: boolean;
}>();
const emit = defineEmits<{ reverted: [item: Item] }>();
const { t } = useI18n();
const { lang } = useLang();
const { notifyError, notifySuccess } = useNotify();
const { confirm } = useConfirmDialog();
const auth = useAuthStore();
const metaTypes = useMetaTypes();

const history = ref<ItemHistory | null>(null);
const all = ref(false);
const shown = computed(() =>
  all.value ? history.value!.entries : history.value!.entries.slice(0, LIMIT),
);
const latest = computed(() =>
  history.value?.entries.find((e) => e.kind === "item"),
);
const latestVersion = computed(() =>
  latestItemVersion(history.value?.entries ?? []),
);

function date(iso: string): string {
  return new Date(iso).toLocaleDateString(lang.value, { dateStyle: "medium" });
}
function dateTime(iso: string): string {
  return new Date(iso).toLocaleString(lang.value, {
    dateStyle: "short",
    timeStyle: "short",
  });
}

function authorName(e: HistoryEntry): string {
  if (e.author.kind === "user" && e.author.user)
    return e.author.user.display_name || e.author.user.username;
  return e.author.kind === "system" ? t("history.system") : t("history.member");
}

function originText(e: HistoryEntry): string {
  const base = t(`history.origins.${e.origin}`, { ref: e.origin_ref ?? "" });
  return e.origin === "external" && e.origin_ref
    ? `${base} (${e.origin_ref})`
    : base;
}

function schemaOf(e: HistoryEntry): Schema | undefined {
  if (e.kind === "item")
    return props.item.item_type?.attributes_schema as Schema | undefined;
  return metaTypes.relationTypes.find(
    (rt) => rt.id === e.relation?.relation_type_id,
  )?.attributes_schema as Schema | undefined;
}

function relationTitle(e: HistoryEntry): string {
  const rt = metaTypes.relationTypes.find(
    (r) => r.id === e.relation?.relation_type_id,
  );
  return typeTitle(rt, lang.value) || t("history.relation");
}

function otherEnd(e: HistoryEntry): string {
  const r = e.relation!;
  return r.source_item_id === props.item.id
    ? r.target_item_id
    : r.source_item_id;
}

function label(e: HistoryEntry, key: string): string {
  return fieldLabel(schemaOf(e), key, lang.value, "pretty");
}

function show(e: HistoryEntry, key: string, value: unknown): string {
  if (value === undefined || value === null) return t("history.empty");
  return fieldValue(schemaOf(e), key, value, lang.value) || t("history.empty");
}

/** Une version antérieure de la fiche, encore existante (pas une suppression). */
function canRevert(e: HistoryEntry): boolean {
  return (
    auth.canEditCatalog &&
    e.kind === "item" &&
    e.after !== null &&
    e.version !== latestVersion.value
  );
}

async function load() {
  try {
    const [h] = await Promise.all([
      itemApi.history(props.item.id),
      metaTypes.loadRelationTypes(),
    ]);
    history.value = h;
  } catch (e) {
    notifyError(e);
  }
}

async function revert(e: HistoryEntry) {
  const ok = await confirm({
    message: t("history.revertConfirm", { n: e.version }),
  });
  if (!ok) return;
  try {
    const item = await itemApi.revert(props.item.id, e.version);
    notifySuccess(t("history.reverted", { n: e.version }));
    emit("reverted", item);
    await load();
  } catch (err) {
    notifyError(err);
  }
}

onMounted(load);
watch(() => [props.item.id, props.version], load);
</script>
