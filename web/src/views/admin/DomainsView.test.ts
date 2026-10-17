import { flushPromises, mount } from "@vue/test-utils";
import { createPinia, setActivePinia } from "pinia";
import ElementPlus from "element-plus";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { nextTick } from "vue";
import { useLang } from "@/composables/useLang";
import type { DomainInfo, DomainInstallReport } from "@/api/types";
import DomainsView from "./DomainsView.vue";

const api = vi.hoisted(() => ({ list: vi.fn(), install: vi.fn() }));
vi.mock("@/api/client", () => ({
  domainApi: api,
  extractApiError: () => ({ message: "" }),
}));

const info = (id: string, title: string, requires: string[] = []) =>
  ({
    id,
    title: { fr: title, en: title },
    description: { fr: "", en: "" },
    package_version: 1,
    requires,
    status: "not_installed",
    modified_locally: false,
    item_types: 2,
    relation_types: 0,
    collection_types: 0,
  }) satisfies DomainInfo;

const report = (
  domain: string,
  names: string[],
  required?: DomainInstallReport[],
): DomainInstallReport => ({
  dry_run: true,
  applied: false,
  domain,
  to_version: 1,
  entries: names.map((name, i) => ({
    entity: "item_types",
    id: `b0000000-0000-0000-0000-00000000001${i}`,
    name,
    outcome: "created",
  })),
  errors: [],
  required,
});

describe("DomainsView", () => {
  beforeEach(async () => {
    setActivePinia(createPinia());
    api.list.mockReset();
    api.install.mockReset();
    useLang().setLang("fr");
    await nextTick();
  });

  it("affiche les paquets requis et leur rapport dans la simulation", async () => {
    api.list.mockResolvedValue([
      info("books", "Livres", ["people"]),
      info("people", "Personnes et éditeurs"),
    ]);
    api.install.mockResolvedValue(
      report("books", ["Manga"], [report("people", ["Person", "Publisher"])]),
    );
    const w = mount(DomainsView, { global: { plugins: [ElementPlus] } });
    await flushPromises();
    expect(w.find('[data-test="requires"]').text()).toBe(
      "Requiert : Personnes et éditeurs (installé avec)",
    );
    await w
      .findAll("button")
      .find((b) => b.text() === "Simuler")!
      .trigger("click");
    await flushPromises();
    expect(api.install).toHaveBeenCalledWith("books", true);
    const req = w.find('[data-test="required-report"]');
    expect(req.text()).toContain(
      "Paquet requis, installé d'abord : Personnes et éditeurs",
    );
    expect(req.text()).toContain("Publisher");
    expect(w.text()).toContain("Manga");
  });
});
