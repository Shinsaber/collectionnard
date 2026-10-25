<template>
  <!--
    Page « Scanner » (lot M3) : la caméra reste ouverte (mode continu), chaque code lu interroge le
    catalogue ; les résultats s'empilent, le plus récent en premier, avec « dans N de mes collections »,
    l'ajout à une collection et, pour un code inconnu, les sources externes ou la création.
  -->
  <div class="max-w-3xl">
    <PageHeader
      :title="t('scan.title')"
      :meta="entries.length ? t('scan.count', entries.length) : undefined"
    >
      <template #subtitle>
        <p class="max-w-2xl text-[15px] text-muted">{{ t("scan.subtitle") }}</p>
      </template>
      <template #actions>
        <ElButton v-if="entries.length" text @click="entries = []">
          {{ t("scan.clear") }}
        </ElButton>
        <ElButton type="primary" size="large" @click="scannerOpen = true">
          <i class="pi pi-barcode mr-2" aria-hidden="true"></i>
          {{ t("scan.start") }}
        </ElButton>
      </template>
    </PageHeader>

    <p
      v-if="!entries.length"
      class="rounded-2xl bg-surface px-6 py-10 text-center text-muted shadow-card"
    >
      {{ t("scan.empty") }}
    </p>

    <ul v-else class="flex flex-col gap-3" data-test="scan-entries">
      <li
        v-for="entry in entries"
        :key="entry.code"
        class="rounded-2xl bg-surface p-4 shadow-card"
      >
        <div class="flex items-center justify-between gap-3">
          <span class="font-mono text-sm text-ink">{{ entry.code }}</span>
          <ElTag :type="tagType(entry.status)" size="small">{{
            statusLabel(entry.status)
          }}</ElTag>
        </div>

        <div
          v-for="hit in entry.items"
          :key="hit.item.id"
          class="mt-3 flex items-center gap-3"
        >
          <ItemCover
            :item="hit.item"
            :label="itemLabel(hit.item, lang)"
            :width="44"
          />
          <div class="min-w-0 flex-1">
            <RouterLink
              :to="`/items/${hit.item.id}`"
              class="font-semibold text-link"
              >{{ itemLabel(hit.item, lang) }}</RouterLink
            >
            <div class="text-xs text-muted">
              {{ typeTitle(hit.item.item_type, lang) }}
              <template v-if="hit.collections">
                · {{ membershipLabel(hit.collections.length) }}
              </template>
            </div>
          </div>
          <ElButton
            size="small"
            :aria-label="t('scan.add')"
            :title="t('scan.add')"
            @click="addTarget = hit"
          >
            <i class="pi pi-plus" aria-hidden="true"></i>
          </ElButton>
        </div>

        <div
          v-if="entry.status === 'missing'"
          class="mt-3 flex flex-wrap gap-2"
        >
          <ElButton size="small" @click="toExternal(entry.code)">
            <i class="pi pi-cloud-download mr-2" aria-hidden="true"></i>
            {{ t("scan.external") }}
          </ElButton>
          <ElButton size="small" @click="createFrom(entry.code)">
            <i class="pi pi-plus mr-2" aria-hidden="true"></i>
            {{ auth.canEditCatalog ? t("scan.create") : t("scan.propose") }}
          </ElButton>
        </div>
      </li>
    </ul>

    <!-- Scanner en continu : le dernier résultat s'affiche sous l'aperçu, sans quitter la caméra. -->
    <BarcodeScannerDialog
      v-model:visible="scannerOpen"
      continuous
      @detected="onDetected"
    >
      <div
        v-if="latest"
        class="mt-3 rounded-lg bg-soft px-3 py-2 text-sm"
        data-test="scan-live"
      >
        <template v-if="latest.status === 'found'">
          <span
            v-for="hit in latest.items"
            :key="hit.item.id"
            class="block font-semibold text-ink"
          >
            {{ itemLabel(hit.item, lang) }}
            <span v-if="hit.collections" class="font-normal text-muted">
              · {{ membershipLabel(hit.collections.length) }}</span
            >
          </span>
        </template>
        <span v-else class="text-muted">{{ statusLabel(latest.status) }}</span>
      </div>
    </BarcodeScannerDialog>

    <AddToCollectionDialog
      v-if="addTarget"
      :visible="!!addTarget"
      :item="addTarget.item"
      :exclude-ids="(addTarget.collections ?? []).map((m) => m.collection.id)"
      @update:visible="(v) => !v && (addTarget = null)"
      @added="refreshMemberships(addTarget!)"
    />
    <ItemEditDialog
      v-model:visible="createVisible"
      :item="null"
      :item-types="metaTypes.itemTypes"
      :propose="!auth.canEditCatalog"
      :prefill-code="createCode"
      @saved="onCreated"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { itemApi } from "@/api/client";
import { useAuthStore } from "@/stores/auth";
import { useMetaTypes } from "@/stores/metaTypes";
import { useLang } from "@/composables/useLang";
import { useNotify } from "@/composables/useNotify";
import { itemLabel, typeTitle } from "@/lib/schema";
import PageHeader from "@/components/common/PageHeader.vue";
import BarcodeScannerDialog from "@/components/common/BarcodeScannerDialog.vue";
import ItemCover from "@/components/item/ItemCover.vue";
import AddToCollectionDialog from "@/components/item/AddToCollectionDialog.vue";
import ItemEditDialog from "@/components/item/ItemEditDialog.vue";
import type { Item, ItemCollectionMembership } from "@/api/types";

interface Hit {
  item: Item;
  /** Mes collections contenant l'item ; null tant que non chargées. */
  collections: ItemCollectionMembership[] | null;
}

type Status = "loading" | "found" | "missing" | "error";

interface Entry {
  code: string;
  format: string;
  status: Status;
  items: Hit[];
}

const { t } = useI18n();
const { lang } = useLang();
const { notifyError } = useNotify();
const router = useRouter();
const auth = useAuthStore();
const metaTypes = useMetaTypes();

const entries = ref<Entry[]>([]);
const latest = computed(() => entries.value[0]);
const scannerOpen = ref(false);
const addTarget = ref<Hit | null>(null);
const createVisible = ref(false);
const createCode = ref("");

onMounted(() => {
  metaTypes.loadItemTypes().catch((e: unknown) => notifyError(e));
});

/** Un code déjà lu remonte en tête sans nouvelle requête ; un nouveau est cherché. */
function onDetected(code: string, format: string) {
  const idx = entries.value.findIndex((e) => e.code === code);
  if (idx >= 0) {
    const [existing] = entries.value.splice(idx, 1);
    entries.value.unshift(existing!);
    return;
  }
  entries.value.unshift({ code, format, status: "loading", items: [] });
  // Relu depuis le tableau réactif : c'est le proxy qu'il faut muter, pas l'objet brut.
  void lookup(entries.value[0]!);
}

/** Les identifiants (chiffres seuls) sont indexés par la recherche plein texte. */
async function lookup(entry: Entry) {
  entry.status = "loading";
  try {
    const res = await itemApi.list({ search: entry.code, per_page: 10 });
    entry.items = res.data.map((item) => ({ item, collections: null }));
    entry.status = entry.items.length ? "found" : "missing";
  } catch (e) {
    entry.status = "error";
    notifyError(e, t("scan.failed"));
    return;
  }
  await Promise.all(entry.items.map((hit) => refreshMemberships(hit)));
}

async function refreshMemberships(hit: Hit) {
  try {
    hit.collections = await itemApi.collections(hit.item.id);
  } catch (e) {
    notifyError(e);
  }
}

function toExternal(code: string) {
  router.push({ name: "external-import", query: { code } });
}

function createFrom(code: string) {
  createCode.value = code;
  createVisible.value = true;
}

/** Item créé (ou proposé) : le code est recherché à nouveau pour refléter le catalogue. */
function onCreated() {
  const entry = entries.value.find((e) => e.code === createCode.value);
  if (entry) void lookup(entry);
}

function membershipLabel(n: number): string {
  return n ? t("items.detail.inCollections", n) : t("scan.notInCollections");
}

function statusLabel(status: Status): string {
  switch (status) {
    case "loading":
      return t("scan.searching");
    case "found":
      return t("scan.found");
    case "missing":
      return t("scan.missing");
    default:
      return t("scan.failed");
  }
}

function tagType(status: Status): "info" | "success" | "warning" | "danger" {
  switch (status) {
    case "loading":
      return "info";
    case "found":
      return "success";
    case "missing":
      return "warning";
    default:
      return "danger";
  }
}
</script>
