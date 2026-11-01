import { flushPromises, mount } from "@vue/test-utils";
import ElementPlus, { ElDropdown } from "element-plus";
import { createPinia, setActivePinia } from "pinia";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { createMemoryHistory, createRouter } from "vue-router";
import type { Item, ItemRelation } from "@/api/types";
import { i18n } from "@/i18n";
import RelationTree from "./RelationTree.vue";

// Audit Q-005 : le composant des relations (menu « ⋮ », sous-relations) n'avait aucun test.
const api = vi.hoisted(() => ({
  relations: vi.fn(),
  relatedCollections: vi.fn(),
}));
vi.mock("@/api/client", () => ({
  itemApi: api,
  extractApiError: () => ({ message: "" }),
}));

const ME = "e0000000-0000-0000-0000-000000000001";

/** Réponse paginée de `GET /items/{id}/relations` (une page). */
function page<T>(data: T[]) {
  return { data, total: data.length, page: 1, per_page: 100, total_pages: 1 };
}

function relationType(name: string, fr: string) {
  return {
    id: `rt-${name}`,
    name,
    attributes_schema: {
      type: "object",
      "x-title": { fr },
      "x-labels": { fr: { order: "Ordre", role: "Rôle" } },
      properties: { order: { type: "integer" }, role: { type: "string" } },
    },
  } as unknown as ItemRelation["relation_type"];
}

function item(id: string, title: string, type = "Episode", fr = "Épisode") {
  return {
    id,
    item_type_id: `b-${type}`,
    attributes: { title },
    tags: [],
    item_type: {
      id: `b-${type}`,
      name: type,
      attributes_schema: { "x-label": "{title}", "x-title": { fr } },
    },
  } as unknown as Item;
}

function rel(
  id: string,
  type: ItemRelation["relation_type"],
  source: Item,
  target: Item,
  attributes: Record<string, unknown>,
): ItemRelation {
  return {
    id,
    relation_type_id: type!.id,
    relation_type: type,
    source_item_id: source.id,
    source_item: source,
    target_item_id: target.id,
    target_item: target,
    attributes,
    created_at: "2026-10-18T00:00:00Z",
    updated_at: "2026-10-18T00:00:00Z",
  } as ItemRelation;
}

const me = item(ME, "Breaking Bad", "Series", "Série");
const contains = relationType("contains_episode", "Contient l'épisode");
const credited = relationType("credited", "Crédité");
const pilot = rel("r-pilot", contains, me, item("ep-2", "Pilote"), {
  order: 3,
});
const first = rel("r-first", contains, me, item("ep-1", "Cat's in the Bag"), {
  order: 1,
});
const actor = rel(
  "r-actor",
  credited,
  item("p-1", "Bryan Cranston", "Person", "Personne"),
  me,
  { role: "actor" },
);

function mountTree(
  props: Partial<InstanceType<typeof RelationTree>["$props"]> = {},
) {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: "/:any(.*)*", component: { template: "<div />" } }],
  });
  return mount(RelationTree, {
    props: { relations: [pilot, actor, first], contextItemId: ME, ...props },
    global: { plugins: [ElementPlus, router], stubs: { ItemCover: true } },
  });
}

describe("RelationTree", () => {
  beforeEach(() => {
    setActivePinia(createPinia()); // cache des méta-types lu par le composant (vide ici)
    api.relations.mockReset().mockResolvedValue(page([]));
    api.relatedCollections.mockReset().mockResolvedValue([]);
  });

  it("groupe les relations par type, les trie par ordre et affiche sens, libellé et attributs", async () => {
    const w = mountTree();
    await flushPromises();
    const groups = w.findAll('[data-test="relation-group"]');
    expect(groups.map((g) => g.find("h3").text())).toEqual([
      "Contient l'épisode · 2",
      "Crédité · 1",
    ]);
    // Dans le premier groupe : l'ordre 1 avant l'ordre 3, lien vers la fiche de l'item lié.
    const rows = groups[0]!.findAll("li");
    expect(rows.map((r) => r.text())).toEqual([
      expect.stringContaining("#1"),
      expect.stringContaining("#3"),
    ]);
    expect(rows[0]!.find("a[href='/items/ep-1']").text()).toBe(
      "Cat's in the Bag",
    );
    expect(rows[0]!.text()).toContain("Épisode");
    // Relation entrante : la personne, et son attribut hors `order` en pastille libellée.
    const incoming = groups[1]!.find("li");
    expect(incoming.find("a[href='/items/p-1']").text()).toBe("Bryan Cranston");
    expect(incoming.text()).toContain("Rôle : actor");
    expect(api.relatedCollections).toHaveBeenCalledWith(ME);
  });

  it("émet modifier / supprimer depuis le menu « ⋮ », absent en lecture seule", async () => {
    const w = mountTree();
    await flushPromises();
    const menus = w.findAllComponents(ElDropdown);
    expect(menus).toHaveLength(3);
    menus[0]!.vm.$emit("command", "edit");
    menus[0]!.vm.$emit("command", "delete");
    expect(w.emitted("edit")?.[0]?.[0]).toMatchObject({ id: "r-first" });
    expect(w.emitted("delete")?.[0]?.[0]).toMatchObject({ id: "r-first" });

    const readonly = mountTree({ showDelete: false });
    await flushPromises();
    expect(readonly.findAllComponents(ElDropdown)).toHaveLength(0);
    expect(readonly.find('[data-test="relation-more"]').exists()).toBe(false);
  });

  it("déplie les sous-relations de l'item lié à la demande", async () => {
    api.relations.mockResolvedValue(
      page([
        rel(
          "r-sub",
          credited,
          item("ep-1", "Cat's in the Bag"),
          item("p-2", "Anna Gunn", "Person", "Personne"),
          {
            role: "actor",
          },
        ),
      ]),
    );
    const w = mountTree();
    await flushPromises();
    const toggle = w.find(
      '[data-test="relation-group"] li button[aria-expanded="false"]',
    );
    await toggle.trigger("click");
    await flushPromises();
    expect(api.relations).toHaveBeenCalledWith("ep-1");
    expect(
      w
        .find('[data-test="relation-group"] li button[aria-expanded="true"]')
        .exists(),
    ).toBe(true);
    // Sous-arbre imbriqué : la relation de l'épisode, sans menu (lecture seule).
    const nested = w.findAllComponents(RelationTree);
    expect(nested.length).toBeGreaterThanOrEqual(1);
    expect(w.text()).toContain("Anna Gunn");

    // Une relation sans sous-relation sortante : message dédié.
    api.relations.mockResolvedValue(page([]));
    const other = mountTree({ relations: [pilot] });
    await flushPromises();
    await other.find('li button[aria-expanded="false"]').trigger("click");
    await flushPromises();
    expect(api.relations).toHaveBeenLastCalledWith("ep-2");
    expect(other.text()).toContain(
      i18n.global.t("items.relations.noSubRelations"),
    );
  });
});
