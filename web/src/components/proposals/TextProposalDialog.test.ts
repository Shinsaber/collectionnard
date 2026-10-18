import { flushPromises, mount } from "@vue/test-utils";
import ElementPlus from "element-plus";
import { beforeEach, describe, expect, it, vi } from "vitest";
import TextProposalDialog from "./TextProposalDialog.vue";

const create = vi.hoisted(() => vi.fn());
vi.mock("@/api/client", () => ({
  proposalApi: { create },
  extractApiError: () => ({ message: "" }),
}));

describe("TextProposalDialog", () => {
  beforeEach(() => {
    document.body.innerHTML = "";
    create.mockReset().mockResolvedValue({ id: "p1" });
  });

  it("envoie un signalement visant l'item, titre par défaut", async () => {
    const w = mount(TextProposalDialog, {
      props: {
        visible: false,
        kind: "report",
        itemId: "i1",
        itemLabel: "Tome 3",
      },
      global: { plugins: [ElementPlus] },
      attachTo: document.body,
    });
    await w.setProps({ visible: true });
    await flushPromises();
    const area = document.querySelector<HTMLTextAreaElement>(
      'textarea[data-test="text-proposal-body"]',
    )!;
    area.value = "Mauvais numéro de tome";
    area.dispatchEvent(new Event("input"));
    await flushPromises();
    document
      .querySelector<HTMLElement>('[data-test="text-proposal-send"]')!
      .click();
    await flushPromises();
    expect(create).toHaveBeenCalledWith({
      kind: "report",
      title: "Signalement : Tome 3",
      description: "Mauvais numéro de tome",
      item_id: "i1",
      submit: true,
    });
    expect(w.emitted("sent")).toBeTruthy();
  });
});
