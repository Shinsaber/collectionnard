<template>
  <div>
    <!-- Bureau / tablette : tableau ; un clic sur la ligne ouvre l'éditeur. -->
    <div
      class="hidden overflow-hidden rounded-2xl bg-surface shadow-card md:block"
    >
      <ElTable
        v-loading="loading"
        :data="rows"
        row-key="id"
        highlight-current-row
        class="cursor-pointer"
        @row-click="(row: AdminTypeRow) => router.push(`${basePath}/${row.id}`)"
      >
        <ElTableColumn
          v-if="hasOrder"
          :label="t('admin.itemTypes.order')"
          width="90"
          align="right"
        >
          <template #default="{ row }">
            <span class="font-mono text-[13px] text-muted">{{
              (row as AdminTypeRow).order
            }}</span>
          </template>
        </ElTableColumn>
        <ElTableColumn :label="t('common.type')" min-width="190">
          <template #default="{ row }">
            <div class="flex flex-col gap-0.5 py-1">
              <RouterLink
                :to="`${basePath}/${(row as AdminTypeRow).id}`"
                class="text-[15px] font-bold text-ink hover:text-link"
                @click.stop
                >{{ (row as AdminTypeRow).title }}</RouterLink
              >
              <span class="font-mono text-xs text-muted">{{
                (row as AdminTypeRow).name
              }}</span>
            </div>
          </template>
        </ElTableColumn>
        <ElTableColumn :label="t('common.description')" min-width="220">
          <template #default="{ row }">
            <span class="text-sm">{{ (row as AdminTypeRow).description }}</span>
          </template>
        </ElTableColumn>
        <ElTableColumn :label="summaryLabel" min-width="260">
          <template #default="{ row }">
            <span class="text-[13px] text-muted">
              <b class="font-semibold text-ink">{{
                (row as AdminTypeRow).countLabel
              }}</b>
              <template v-if="(row as AdminTypeRow).summary">
                : {{ (row as AdminTypeRow).summary }}</template
              >
            </span>
          </template>
        </ElTableColumn>
        <ElTableColumn width="104" align="right">
          <template #default="{ row }">
            <div class="flex justify-end gap-1">
              <ElButton
                text
                circle
                :aria-label="`${t('common.edit')} : ${(row as AdminTypeRow).title}`"
                @click.stop="
                  router.push(`${basePath}/${(row as AdminTypeRow).id}`)
                "
              >
                <i class="pi pi-pencil" aria-hidden="true"></i>
              </ElButton>
              <ElButton
                text
                circle
                type="danger"
                :aria-label="`${t('common.delete')} : ${(row as AdminTypeRow).title}`"
                @click.stop="emit('remove', row as AdminTypeRow)"
              >
                <i class="pi pi-trash" aria-hidden="true"></i>
              </ElButton>
            </div>
          </template>
        </ElTableColumn>
      </ElTable>
    </div>

    <!-- Mobile : cartes-liens vers l'éditeur -->
    <ul v-loading="loading" class="flex flex-col gap-2.5 md:hidden">
      <li v-for="row in rows" :key="row.id">
        <RouterLink
          :to="`${basePath}/${row.id}`"
          class="flex min-h-16 items-center gap-3 rounded-2xl bg-surface px-3.5 py-3 text-ink shadow-card"
        >
          <div class="flex min-w-0 flex-1 flex-col gap-0.5">
            <div class="flex items-baseline gap-2">
              <span
                v-if="row.order !== undefined"
                class="font-mono text-xs text-muted"
                :title="t('admin.itemTypes.displayOrder')"
                >#{{ row.order }}</span
              >
              <span class="text-base font-bold">{{ row.title }}</span>
              <span class="truncate font-mono text-xs text-muted">{{
                row.name
              }}</span>
            </div>
            <span class="truncate text-[13px] text-muted"
              >{{ row.countLabel
              }}<template v-if="row.summary">
                : {{ row.summary }}</template
              ></span
            >
          </div>
          <i class="pi pi-angle-right text-muted" aria-hidden="true"></i>
        </RouterLink>
      </li>
    </ul>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";

/** Ligne de la liste d'un méta-type, déjà mise en forme par la vue. */
export interface AdminTypeRow {
  id: string;
  /** Nom affiché (x-title), et identifiant technique. */
  title: string;
  name: string;
  description: string;
  /** « 10 champs » ; `summary` détaille (libellés, types source → cible…). */
  countLabel: string;
  summary: string;
  /** Ordre d'affichage (types d'item seulement). */
  order?: number;
}

const props = defineProps<{
  rows: AdminTypeRow[];
  loading: boolean;
  /** Liste des types : l'éditeur est `${basePath}/:id`. */
  basePath: string;
  summaryLabel: string;
}>();
const emit = defineEmits<{ remove: [row: AdminTypeRow] }>();

const { t } = useI18n();
const router = useRouter();
// Colonne « Ordre » pour les seules sortes qui en ont un (types d'item).
const hasOrder = computed(() => props.rows.some((r) => r.order !== undefined));
</script>
