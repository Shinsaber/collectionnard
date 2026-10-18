import { flushPromises, mount } from "@vue/test-utils";
import { createPinia, setActivePinia } from "pinia";
import ElementPlus from "element-plus";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { nextTick } from "vue";
import { useLang } from "@/composables/useLang";
import type { Item } from "@/api/types";
import fixtures from "../../tests/fixtures/seed-schemas.json";
import ItemDetailView from "./ItemDetailView.vue";

const api = vi.hoisted(() => ({
  get: vi.fn(),
  relations: vi.fn(),
  collections: vi.fn(),
}));
vi.mock("@/api/client", () => ({
  itemApi: api,
  relationApi: { delete: vi.fn() },
  extractApiError: () => ({ message: "" }),
}));

const publisherType = fixtures.find((f) => f.name === "Publisher")!;

function item(attributes: Record<string, unknown>): Item {
  return {
    id: "e0000000-0000-0000-0000-000000000501",
    item_type_id: publisherType.id,
    item_type: {
      id: publisherType.id,
      name: "Publisher",
      attributes_schema: publisherType.schema as Record<string, unknown>,
      display_order: 140,
      created_at: "2026-10-17T00:00:00Z",
      updated_at: "2026-10-17T00:00:00Z",
    },
    attributes,
    tags: [{ id: "t1", name: "japon" }] as Item["tags"],
    created_at: "2026-10-17T00:00:00Z",
    updated_at: "2026-10-17T00:00:00Z",
  };
}

// Sous-composants chargés à part (collections, relations, galerie, dialogues) : remplacés.
const stubs = {
  RouterLink: { template: "<a><slot /></a>" },
  ItemCover: true,
  ItemCollectionsCard: true,
  RelationTree: true,
  ItemGallery: true,
  AddToCollectionDialog: true,
  RelationCreateDialog: true,
  RelationEditDialog: true,
  ItemEditDialog: true,
  NavIcon: true,
};

async function mountDetail(attributes: Record<string, unknown>) {
  api.get.mockResolvedValue(item(attributes));
  const w = mount(ItemDetailView, {
    props: { id: "e0000000-0000-0000-0000-000000000501" },
    global: { plugins: [ElementPlus], stubs },
  });
  await flushPromises();
  return w;
}

/** Libellé → valeur affichée de chaque attribut de la fiche. */
function rows(
  w: Awaited<ReturnType<typeof mountDetail>>,
): Record<string, string> {
  const out: Record<string, string> = {};
  const dl = w.find('[data-test="item-attributes"] > dl');
  const dts = dl.findAll(":scope > dt");
  const dds = dl.findAll(":scope > dd");
  dts.forEach((dt, i) => (out[dt.text()] = dds[i]!.text()));
  return out;
}

describe("ItemDetailView", () => {
  beforeEach(async () => {
    setActivePinia(createPinia());
    Object.values(api).forEach((m) => m.mockReset());
    api.relations.mockResolvedValue([]);
    api.collections.mockResolvedValue([]);
    useLang().setLang("fr");
    await nextTick();
  });

  it("affiche le titre, le type, les étiquettes et les attributs dans l'ordre du schéma", async () => {
    const w = await mountDetail({
      name: "Nintendo",
      country: "JP",
      description: {
        fr: "entreprise japonaise de jeu vidéo",
        en: "Japanese company",
      },
    });
    expect(api.get).toHaveBeenCalledWith(
      "e0000000-0000-0000-0000-000000000501",
    );
    expect(w.find("h1").text()).toBe("Nintendo");
    expect(w.text()).toContain("japon");
    // Attributs présents seulement, libellés x-labels, texte multilingue dans la langue courante.
    expect(rows(w)).toEqual({
      Nom: "Nintendo",
      "Pays (code)": "JP",
      Description: "entreprise japonaise de jeu vidéo",
    });
  });

  it("rend les liens cliquables (libellé, sinon nom du site) et écarte les adresses non http(s)", async () => {
    const w = await mountDetail({
      name: "Nintendo",
      website: "https://www.nintendo.com/fr-fr/",
      links: [
        {
          label: "Wikipédia (fr)",
          url: "https://fr.wikipedia.org/wiki/Nintendo",
        },
        { url: "https://www.nintendo.co.jp/" },
        { label: "piège", url: "javascript:alert(1)" },
      ],
    });
    const anchors = w
      .findAll('[data-test="attribute-links"] a')
      .map((a) => [
        a.text(),
        a.attributes("href"),
        a.attributes("target"),
        a.attributes("rel"),
      ]);
    expect(anchors).toEqual([
      // Champ texte au format URL (site web) : un lien lui aussi.
      [
        "www.nintendo.com",
        "https://www.nintendo.com/fr-fr/",
        "_blank",
        "noopener noreferrer",
      ],
      [
        "Wikipédia (fr)",
        "https://fr.wikipedia.org/wiki/Nintendo",
        "_blank",
        "noopener noreferrer",
      ],
      [
        "www.nintendo.co.jp",
        "https://www.nintendo.co.jp/",
        "_blank",
        "noopener noreferrer",
      ],
    ]);
    expect(w.html()).not.toContain("javascript:");
  });

  it("signale un item introuvable sans afficher de fiche", async () => {
    api.get.mockRejectedValue(new Error("404"));
    const w = mount(ItemDetailView, {
      props: { id: "e0000000-0000-0000-0000-000000000999" },
      global: { plugins: [ElementPlus], stubs },
    });
    await flushPromises();
    expect(w.find("h1").exists()).toBe(false);
  });
});

describe("ItemDetailView — en-tête A2", () => {
  beforeEach(async () => {
    setActivePinia(createPinia());
    Object.values(api).forEach((m) => m.mockReset());
    api.collections.mockResolvedValue([
      { collection: { id: "c1", name: "Mes mangas" }, attributes: {} },
    ]);
    useLang().setLang("fr");
    await nextTick();
  });

  it("déduit l'appartenance des relations x-completeness et ouvre l'onglet des collections", async () => {
    api.relations.mockResolvedValue([
      {
        id: "r1",
        relation_type_id: "rt",
        relation_type: {
          id: "rt",
          name: "contains_volume",
          attributes_schema: { type: "object", "x-completeness": true },
        },
        source_item_id: "ed1",
        source_item: {
          id: "ed1",
          item_type_id: "x",
          attributes: { name: "Glénat" },
        },
        target_item_id: "e0000000-0000-0000-0000-000000000501",
        attributes: { order: 3 },
      },
      {
        id: "r2",
        relation_type_id: "other",
        relation_type: {
          id: "other",
          name: "published_by",
          attributes_schema: { type: "object" },
        },
        source_item_id: "p1",
        source_item: {
          id: "p1",
          item_type_id: "x",
          attributes: { name: "Autre" },
        },
        target_item_id: "e0000000-0000-0000-0000-000000000501",
        attributes: {},
      },
    ]);
    const w = await mountDetail({ name: "Tome 3" });
    const partOf = w.find('[data-test="part-of"]');
    expect(partOf.text()).toContain("Fait partie de");
    expect(partOf.text()).toContain("Glénat");
    expect(partOf.text()).toContain("(n° 3)");
    expect(partOf.text()).not.toContain("Autre");

    const pill = w.find('[data-test="collections-pill"]');
    expect(pill.text()).toBe("Dans 1 de mes collections");
    await pill.trigger("click");
    await flushPromises();
    expect(w.find(".el-tabs__item.is-active").text()).toContain(
      "Dans vos collections",
    );
  });
});
