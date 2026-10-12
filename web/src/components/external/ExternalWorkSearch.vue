<template>
  <section
    class="p-4 rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800"
  >
    <h2 class="text-lg font-semibold text-gray-800 dark:text-gray-100 mb-3">
      {{ t("admin.external.work.title") }}
    </h2>
    <form class="flex gap-2 mb-3" @submit.prevent="search">
      <label :for="inputId" class="sr-only">{{
        t("admin.external.work.title")
      }}</label>
      <ElInput
        :id="inputId"
        v-model="query"
        :placeholder="t('admin.external.work.placeholder')"
        clearable
      />
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

    <div
      v-if="modelValue"
      class="flex items-center gap-2 mb-3 text-sm"
      data-test="chosen-work"
    >
      <span class="font-medium text-gray-800 dark:text-gray-100">{{
        t("admin.external.work.chosen", { name: modelValue.label })
      }}</span>
      <ElButton
        size="small"
        text
        :aria-label="t('admin.external.work.clear')"
        @click="emit('update:modelValue', null)"
      >
        <i class="pi pi-times" aria-hidden="true"></i>
      </ElButton>
    </div>

    <ElTable
      v-if="hits"
      :data="hits"
      size="small"
      highlight-current-row
      row-key="wikidata_id"
      :empty-text="t('admin.external.work.none')"
      data-test="work-hits"
      @current-change="pick"
    >
      <ElTableColumn :label="t('admin.external.work.label')" min-width="180">
        <template #default="{ row }">
          <span class="font-medium">{{ (row as ExternalWorkHit).label }}</span>
          <ElTag
            v-if="(row as ExternalWorkHit).existing_item_id"
            size="small"
            type="info"
            class="ml-2"
            >{{ t("admin.external.inCatalog") }}</ElTag
          >
        </template>
      </ElTableColumn>
      <ElTableColumn
        prop="description"
        :label="t('admin.external.work.description')"
        min-width="200"
      />
      <ElTableColumn
        prop="start_year"
        :label="t('admin.external.work.year')"
        width="80"
      />
      <ElTableColumn :label="t('admin.external.work.type')" width="100">
        <template #default="{ row }">
          {{ typeName((row as ExternalWorkHit).item_type_id) }}
        </template>
      </ElTableColumn>
      <ElTableColumn width="50" align="center">
        <template #default="{ row }">
          <a
            :href="(row as ExternalWorkHit).source_url"
            target="_blank"
            rel="noopener noreferrer"
            :aria-label="t('admin.external.openSource')"
            @click.stop
            ><i class="pi pi-external-link" aria-hidden="true"></i
          ></a>
        </template>
      </ElTableColumn>
    </ElTable>
  </section>
</template>

<script setup lang="ts">
// Recherche d'une œuvre sur Wikidata ; la ligne cliquée devient l'œuvre choisie.
import { ref, useId } from "vue";
import { useI18n } from "vue-i18n";
import { externalApi } from "@/api/client";
import { useLang } from "@/composables/useLang";
import { useNotify } from "@/composables/useNotify";
import { useMetaTypes } from "@/stores/metaTypes";
import { typeTitle } from "@/lib/schema";
import type { ExternalWorkHit } from "@/api/types";

defineProps<{ modelValue: ExternalWorkHit | null }>();
const emit = defineEmits<{
  "update:modelValue": [hit: ExternalWorkHit | null];
}>();

const { t } = useI18n();
const { lang } = useLang();
const { notifyError } = useNotify();
const metaTypes = useMetaTypes();
const inputId = useId();

const query = ref("");
const hits = ref<ExternalWorkHit[] | null>(null);
const loading = ref(false);

function typeName(id: string): string {
  return typeTitle(
    metaTypes.itemTypes.find((it) => it.id === id),
    lang.value,
  );
}

async function search() {
  const q = query.value.trim();
  if (q.length < 2) return;
  loading.value = true;
  try {
    hits.value = await externalApi.searchWorks(q);
  } catch (e) {
    notifyError(e, t("admin.external.searchFailed"));
  } finally {
    loading.value = false;
  }
}

/** Une nouvelle liste réinitialise la ligne courante (null) : seule une ligne choisie compte. */
function pick(row: ExternalWorkHit | null) {
  if (row) emit("update:modelValue", row);
}
</script>
