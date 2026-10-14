import { ref, type Ref } from "vue";
import type { PaginationParams, PaginatedResponse } from "@/api/types";

export function usePagination<T>(
  fetchFn: (
    params: PaginationParams & Record<string, unknown>,
  ) => Promise<PaginatedResponse<T>>,
) {
  const data: Ref<T[]> = ref([]);
  const total = ref(0);
  const page = ref(1);
  const perPage = ref(20);
  const totalPages = ref(0);
  const loading = ref(false);

  async function load(extra: Record<string, unknown> = {}) {
    loading.value = true;
    try {
      const res = await fetchFn({
        page: page.value,
        per_page: perPage.value,
        ...extra,
      });
      data.value = res.data ?? [];
      total.value = res.total ?? 0;
      totalPages.value = res.total_pages ?? 0;
    } finally {
      loading.value = false;
    }
  }

  /** Événement de pagination 0-based (ancien DataTable PrimeVue). */
  function onPage(event: { page: number; rows: number }) {
    page.value = event.page + 1;
    perPage.value = event.rows;
  }

  /** Page 1-based (ElPagination `@current-change`). */
  function goToPage(p: number) {
    page.value = p;
  }

  return {
    data,
    total,
    page,
    perPage,
    totalPages,
    loading,
    load,
    onPage,
    goToPage,
  };
}
