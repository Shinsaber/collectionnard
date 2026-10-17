import { flushPromises, mount } from "@vue/test-utils";
import { createPinia, setActivePinia } from "pinia";
import ElementPlus from "element-plus";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { createMemoryHistory, createRouter } from "vue-router";
import type { CollectionStats } from "@/api/types";
import type { Schema } from "@/lib/schema";
import CollectionStatsPanel from "./CollectionStatsPanel.vue";

const api = vi.hoisted(() => ({
  collection: vi.fn(),
  itemTypes: vi.fn(),
}));
vi.mock("@/api/client", () => ({
  statsApi: { collection: api.collection },
  itemTypeApi: { list: api.itemTypes },
  relationTypeApi: { list: vi.fn() },
  collectionTypeApi: { list: vi.fn() },
  extractApiError: () => ({ message: "" }),
}));

const schema: Schema = {
  type: "object",
  "x-labels": { fr: { purchase_price: "Prix d'achat", condition: "État" } },
  properties: {
    purchase_price: { type: "number" },
    condition: { type: "string", enum: ["new", "good"] },
  },
};

function sample(over: Partial<CollectionStats> = {}): CollectionStats {
  return {
    collection_id: "c1",
    total_items: 3,
    items_by_type: [{ type_id: "t-vol", type_name: "Volume", count: 3 }],
    numbers: [
      {
        field: "purchase_price",
        count: 2,
        sum: 10,
        avg: 5,
        min: 3.1,
        max: 6.9,
      },
    ],
    distributions: [
      {
        field: "condition",
        values: [
          { value: "new", count: 1 },
          { value: "good", count: 1 },
        ],
        missing: 1,
      },
    ],
    timelines: [{ field: "acquisition_date", months: [] }],
    added: [{ month: "2026-10", count: 3 }],
    completeness: [
      {
        item: {
          id: "e1",
          item_type_id: "t-ed",
          attributes: { title: "Dragon Ball" },
        },
        owned: 30,
        total: 42,
      },
    ],
    completeness_truncated: false,
    ...over,
  };
}

async function mountPanel() {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: "/:any(.*)*", component: { template: "<div />" } }],
  });
  const w = mount(CollectionStatsPanel, {
    props: { collectionId: "c1", schema },
    global: {
      plugins: [ElementPlus, router],
      stubs: {
        StatsChart: {
          props: ["title", "labels"],
          template:
            "<div class='chart'>{{ title }}: {{ labels.join('|') }}</div>",
        },
      },
    },
  });
  await flushPromises();
  return w;
}

describe("CollectionStatsPanel", () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    api.collection.mockReset();
    api.itemTypes.mockResolvedValue({ data: [], total: 0 });
  });

  it("affiche les chiffres, les répartitions et la complétude d'après le schéma", async () => {
    api.collection.mockResolvedValue(sample());
    const w = await mountPanel();
    expect(api.collection).toHaveBeenCalledWith("c1");

    const tile = w.find('[data-test="number-stat"]');
    expect(tile.text()).toContain("Prix d'achat");
    expect(tile.text()).toContain("10");
    expect(tile.text()).toContain("2 valeurs");

    const charts = w.findAll(".chart").map((c) => c.text());
    expect(charts).toContain("État: new|good|Non renseigné");
    // Une seule sorte d'item : pas de graphique par type ; date jamais saisie : pas de courbe.
    expect(charts.some((c) => c.startsWith("Par type"))).toBe(false);
    expect(charts.some((c) => c.startsWith("acquisition"))).toBe(false);
    expect(charts.some((c) => c.startsWith("Ajouts par mois"))).toBe(true);

    const row = w.find('[data-test="completeness-row"]');
    expect(row.text()).toContain("Dragon Ball");
    expect(row.text()).toContain("30 sur 42");
    expect(row.find("a").attributes("href")).toBe("/items/e1");
  });

  it("signale une collection vide", async () => {
    api.collection.mockResolvedValue(
      sample({
        total_items: 0,
        numbers: [],
        distributions: [],
        completeness: [],
      }),
    );
    const w = await mountPanel();
    expect(w.find('[data-test="stats-empty"]').exists()).toBe(true);
    expect(w.find('[data-test="completeness"]').exists()).toBe(false);
  });
});
