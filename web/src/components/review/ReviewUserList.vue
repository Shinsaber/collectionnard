<template>
  <AppSpinner v-if="loading" />
  <p v-else-if="reviews.length === 0" class="text-sm text-muted">
    {{ emptyText }}
  </p>
  <div v-else class="flex flex-col gap-3" data-test="review-user-list">
    <ReviewCard
      v-for="r in reviews"
      :key="r.id"
      :review="r"
      data-test="review-user-item"
    >
      <template #heading>
        <component
          :is="auth.isAuthenticated ? RouterLink : 'span'"
          :to="auth.isAuthenticated ? `/items/${r.item_id}` : undefined"
          class="font-bold text-ink"
          data-test="review-item"
        >
          {{ r.item ? itemLabel(r.item, lang) : "?" }}
        </component>
        <span
          v-if="showVisibility"
          class="rounded-full bg-tint px-2.5 py-0.5 text-xs font-semibold text-tint-ink"
          >{{ t(`reviews.visibility.${r.visibility}`) }}</span
        >
      </template>
    </ReviewCard>
    <ElPagination
      class="justify-end"
      hide-on-single-page
      layout="prev, pager, next"
      :total="total"
      :page-size="PER_PAGE"
      :current-page="page"
      @current-change="onPage"
    />
  </div>
</template>

<script setup lang="ts">
// Avis d'une personne, avec leur item : « Mes avis » (profil) ou avis signés d'un membre (page publique).
import { onMounted, ref } from "vue";
import { RouterLink } from "vue-router";
import { useI18n } from "vue-i18n";
import { useAuthStore } from "@/stores/auth";
import { useLang } from "@/composables/useLang";
import { useNotify } from "@/composables/useNotify";
import { itemLabel } from "@/lib/schema";
import AppSpinner from "@/components/common/AppSpinner.vue";
import ReviewCard from "./ReviewCard.vue";
import type { PaginatedResponse, PaginationParams, Review } from "@/api/types";

const PER_PAGE = 10;

const props = defineProps<{
  fetch: (params: PaginationParams) => Promise<PaginatedResponse<Review>>;
  emptyText: string;
  /** Sa propre liste : rappelle la visibilité de chaque avis. */
  showVisibility?: boolean;
}>();

const { t } = useI18n();
const { lang } = useLang();
const auth = useAuthStore();
const { notifyError } = useNotify();

const reviews = ref<Review[]>([]);
const total = ref(0);
const page = ref(1);
const loading = ref(true);

async function load() {
  loading.value = true;
  try {
    const res = await props.fetch({ page: page.value, per_page: PER_PAGE });
    reviews.value = res.data;
    total.value = res.total;
  } catch (e) {
    notifyError(e);
  } finally {
    loading.value = false;
  }
}

function onPage(p: number) {
  page.value = p;
  load();
}

onMounted(load);
</script>
