import { describe, expect, it, vi } from "vitest";
import { usePagination } from "./usePagination";
import type { PaginatedResponse } from "@/api/types";

function response<T>(data: T[], total: number): PaginatedResponse<T> {
  return {
    data,
    total,
    page: 1,
    per_page: 20,
    total_pages: Math.ceil(total / 20),
  };
}

describe("usePagination", () => {
  it("charge les données avec page / per_page et les paramètres supplémentaires", async () => {
    const fetchFn = vi.fn().mockResolvedValue(response(["a", "b"], 42));
    const p = usePagination<string>(fetchFn);

    await p.load({ search: "bre" });

    expect(fetchFn).toHaveBeenCalledWith({
      page: 1,
      per_page: 20,
      search: "bre",
    });
    expect(p.data.value).toEqual(["a", "b"]);
    expect(p.total.value).toBe(42);
    expect(p.totalPages.value).toBe(3);
    expect(p.loading.value).toBe(false);
  });

  it("onPage convertit l'index 0-based de PrimeVue en page 1-based", async () => {
    const fetchFn = vi.fn().mockResolvedValue(response([], 0));
    const p = usePagination<string>(fetchFn);

    p.onPage({ page: 2, rows: 50 });
    await p.load();

    expect(p.page.value).toBe(3);
    expect(p.perPage.value).toBe(50);
    expect(fetchFn).toHaveBeenLastCalledWith({ page: 3, per_page: 50 });
  });

  it("remet loading à false même en cas d'erreur", async () => {
    const fetchFn = vi.fn().mockRejectedValue(new Error("réseau"));
    const p = usePagination<string>(fetchFn);

    await expect(p.load()).rejects.toThrow("réseau");
    expect(p.loading.value).toBe(false);
  });
});
