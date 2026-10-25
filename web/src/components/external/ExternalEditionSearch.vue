<template>
  <section class="p-4 rounded-lg border border-line bg-surface">
    <h2 class="text-lg font-semibold text-ink mb-3">
      {{ t(`admin.external.connectors.${connector}.editionTitle`) }}
    </h2>
    <form class="flex flex-wrap gap-2 mb-3" @submit.prevent="search">
      <ElRadioGroup v-model="mode" size="small" @change="hits = null">
        <ElRadioButton value="title">{{
          t("admin.external.edition.byTitle")
        }}</ElRadioButton>
        <ElRadioButton value="code">{{
          t(`admin.external.connectors.${connector}.byCode`)
        }}</ElRadioButton>
      </ElRadioGroup>
      <label :for="inputId" class="sr-only">{{
        t(`admin.external.connectors.${connector}.editionTitle`)
      }}</label>
      <ElInput
        :id="inputId"
        v-model="query"
        class="flex-1 min-w-48"
        :placeholder="
          mode === 'title'
            ? t(`admin.external.connectors.${connector}.editionPlaceholder`)
            : t(`admin.external.connectors.${connector}.codePlaceholder`)
        "
        clearable
      />
      <!-- Code-barres (lot M2) : passe en mode code et lance la recherche. -->
      <ScanButton @detected="onScanned" />
      <ElButton
        native-type="submit"
        type="primary"
        :loading="loading"
        :disabled="query.trim().length < 2"
      >
        <i v-if="!loading" class="pi pi-search mr-2" aria-hidden="true"></i>
        {{ t("admin.external.search") }}
      </ElButton>
    </form>

    <div v-if="modelValue.length > 0" class="mb-3" data-test="chosen-editions">
      <p class="text-sm font-medium text-ink mb-1">
        {{ t("admin.external.edition.chosen") }}
      </p>
      <div class="flex flex-wrap gap-2">
        <ElTag
          v-for="e in modelValue"
          :key="e.id"
          closable
          :aria-label="t('admin.external.edition.remove', { name: e.title })"
          @close="toggle(e, false)"
          >{{ editionLabel(e) }}</ElTag
        >
      </div>
    </div>

    <!-- Romans : les traductions sont nombreuses (L'Étranger : 55 éditions) ; masquées par défaut. -->
    <div v-if="hasTranslations" class="mb-2 flex items-center gap-2 text-sm">
      <ElSwitch
        v-model="showTranslations"
        :aria-label="
          t('admin.external.edition.translations', { n: translationCount })
        "
      />
      {{ t("admin.external.edition.translations", { n: translationCount }) }}
    </div>
    <ElTable
      v-if="hits"
      :data="shownHits"
      size="small"
      row-key="id"
      :empty-text="t('admin.external.edition.none')"
      data-test="edition-hits"
    >
      <ElTableColumn width="50" align="center">
        <template #default="{ row }">
          <ElCheckbox
            :model-value="isChosen(row as ExternalEditionHit)"
            :disabled="
              !isChosen(row as ExternalEditionHit) && modelValue.length >= MAX
            "
            :aria-label="
              t('admin.external.edition.choose', {
                name: (row as ExternalEditionHit).title,
              })
            "
            @change="(v) => toggle(row as ExternalEditionHit, !!v)"
          />
        </template>
      </ElTableColumn>
      <ElTableColumn :label="t('admin.external.edition.label')" min-width="200">
        <template #default="{ row }">
          <span class="font-medium">{{
            (row as ExternalEditionHit).title
          }}</span>
          <ElTag
            v-if="(row as ExternalEditionHit).existing_item_id"
            size="small"
            type="info"
            class="ml-2"
            >{{ t("admin.external.inCatalog") }}</ElTag
          >
          <div
            v-if="(row as ExternalEditionHit).matched_volume"
            class="text-xs text-muted"
          >
            {{ matchedLabel(row as ExternalEditionHit) }}
          </div>
          <div
            v-if="details(row as ExternalEditionHit)"
            class="text-xs text-muted"
          >
            {{ details(row as ExternalEditionHit) }}
          </div>
        </template>
      </ElTableColumn>
      <ElTableColumn
        :label="t(`admin.external.connectors.${connector}.publishers`)"
        min-width="140"
      >
        <template #default="{ row }">
          {{ (row as ExternalEditionHit).publishers.join(", ") }}
        </template>
      </ElTableColumn>
      <ElTableColumn
        prop="start_year"
        :label="t('admin.external.edition.year')"
        width="80"
      />
      <ElTableColumn width="50" align="center">
        <template #default="{ row }">
          <a
            :href="(row as ExternalEditionHit).source_url"
            target="_blank"
            rel="noopener noreferrer"
            :aria-label="t('admin.external.openSource')"
            ><i class="pi pi-external-link" aria-hidden="true"></i
          ></a>
        </template>
      </ElTableColumn>
    </ElTable>
    <p v-if="modelValue.length >= MAX" class="text-xs text-muted mt-2">
      {{ t("admin.external.edition.max") }}
    </p>
  </section>
</template>

<script setup lang="ts">
// Recherche d'éditions (BnF : titre ou ISBN d'un tome ; MusicBrainz : titre ou code-barres) ; plusieurs
// éditions peuvent être cochées (10 au plus, limite du serveur). `suggestion` (titre de
// l'œuvre choisie) préremplit la recherche par titre et la lance.
import { computed, ref, useId, watch } from "vue";
import { useI18n } from "vue-i18n";
import { externalApi } from "@/api/client";
import { useNotify } from "@/composables/useNotify";
import type { ExternalEditionHit } from "@/api/types";
import ScanButton from "@/components/common/ScanButton.vue";

const MAX = 10;

const props = defineProps<{
  modelValue: ExternalEditionHit[];
  suggestion?: string;
  /** Œuvre choisie (QID, MBID) : ses éditions sont listées si le connecteur le permet (`byWork`). */
  work?: string;
  byWork?: boolean;
  /** Connecteur (`bd-manga`, `novels`, `music`…). */
  /** Code de recherche d'une édition : ISBN (livres) ou code-barres (musique). */
  code?: "isbn" | "barcode";
  connector: string;
}>();
const emit = defineEmits<{
  "update:modelValue": [editions: ExternalEditionHit[]];
}>();

const { t } = useI18n();
const { notifyError } = useNotify();
const inputId = useId();

const mode = ref<"title" | "code">("title");
const query = ref("");
const hits = ref<ExternalEditionHit[] | null>(null);
const loading = ref(false);

watch(
  () => props.suggestion,
  (s) => {
    if (props.byWork && props.work) return; // éditions de l'œuvre (ci-dessous)
    if (s && mode.value === "title") {
      query.value = s;
      void search();
    }
  },
);

// Œuvre choisie : toutes ses éditions (roman : notice d'œuvre BnF ; album : parutions MusicBrainz), sans saisie.
watch(
  () => [props.work, props.byWork] as const,
  async ([work, byWork]) => {
    if (!work || !byWork) return;
    if (props.suggestion) query.value = props.suggestion;
    loading.value = true;
    try {
      hits.value = await externalApi.searchEditions(props.connector, { work });
    } catch (e) {
      notifyError(e, t("admin.external.searchFailed"));
    } finally {
      loading.value = false;
    }
  },
  { immediate: true },
);

// Éditions dans une autre langue que le français (langue inconnue : gardée).
const showTranslations = ref(false);
const isTranslation = (e: ExternalEditionHit) =>
  !!e.language && e.language !== "fr";
const translationCount = computed(
  () => (hits.value ?? []).filter(isTranslation).length,
);
const hasTranslations = computed(() => translationCount.value > 0);
const shownHits = computed(() =>
  showTranslations.value
    ? (hits.value ?? [])
    : (hits.value ?? []).filter((e) => !isTranslation(e)),
);

/** Détails d'une édition : langue, collection et ISBN (romans) ; artiste, supports, pays, référence et code-barres (musique). */
function details(e: ExternalEditionHit): string {
  return [
    e.artist,
    e.format,
    e.country,
    e.language?.toUpperCase(),
    e.collection,
    e.catalog_number,
    e.isbn ?? e.barcode,
  ]
    .filter(Boolean)
    .join(" · ");
}

async function search() {
  const q = query.value.trim();
  if (q.length < 2) return;
  loading.value = true;
  try {
    hits.value = await externalApi.searchEditions(
      props.connector,
      mode.value === "title"
        ? { q }
        : props.code === "barcode"
          ? { barcode: q }
          : { isbn: q },
    );
  } catch (e) {
    notifyError(e, t("admin.external.searchFailed"));
  } finally {
    loading.value = false;
  }
}

function onScanned(code: string) {
  mode.value = "code";
  hits.value = null;
  query.value = code;
  void search();
}

function isChosen(e: ExternalEditionHit): boolean {
  return props.modelValue.some((c) => c.id === e.id);
}

function toggle(e: ExternalEditionHit, on: boolean) {
  const rest = props.modelValue.filter((c) => c.id !== e.id);
  emit("update:modelValue", on ? [...rest, e] : rest);
}

function editionLabel(e: ExternalEditionHit): string {
  const parts = [e.title, e.publishers.join(", "), e.start_year];
  return parts.filter((p) => p !== undefined && p !== "").join(" · ");
}

function matchedLabel(e: ExternalEditionHit): string {
  const v = e.matched_volume!;
  return v.number !== undefined
    ? t("admin.external.edition.matched", { number: v.number, title: v.title })
    : t("admin.external.edition.matchedNoNumber", { title: v.title });
}
</script>
