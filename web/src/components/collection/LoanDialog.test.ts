import { flushPromises, mount } from "@vue/test-utils";
import ElementPlus from "element-plus";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { CollectionItem } from "@/api/types";
import LoanDialog from "./LoanDialog.vue";

const api = vi.hoisted(() => ({
  friends: vi.fn(),
  create: vi.fn(),
}));
vi.mock("@/api/client", () => ({
  friendApi: { list: api.friends },
  loanApi: { create: api.create },
  extractApiError: () => ({ message: "" }),
}));

const lot = {
  id: "l1",
  quantity: 3,
  loans: [{ quantity: 1, borrower_name: "Odile" }],
} as unknown as CollectionItem;

async function open() {
  const w = mount(LoanDialog, {
    props: { collectionId: "c1", lot, label: "Tome 1", visible: false },
    global: { plugins: [ElementPlus] },
    attachTo: document.body,
  });
  await w.setProps({ visible: true });
  await flushPromises();
  return w;
}

const submit = () =>
  (
    document.querySelector('[data-test="loan-submit"]') as HTMLButtonElement
  ).click();

describe("LoanDialog", () => {
  beforeEach(() => {
    api.friends.mockReset();
    api.create.mockReset().mockResolvedValue({});
    document.body.innerHTML = "";
  });

  it("sans ami, prête à un nom, dans la limite des exemplaires disponibles", async () => {
    api.friends.mockResolvedValue({ data: [], total: 0 });
    const w = await open();
    const input = document.querySelector(
      '[data-test="loan-name"] input, input[data-test="loan-name"]',
    ) as HTMLInputElement;
    input.value = "  Tante Odile ";
    input.dispatchEvent(new Event("input"));
    await flushPromises();
    expect(document.body.textContent).toContain("2 disponibles");
    submit();
    await flushPromises();
    expect(api.create).toHaveBeenCalledWith(
      "c1",
      "l1",
      expect.objectContaining({ quantity: 1, borrower_name: "Tante Odile" }),
    );
    expect(w.emitted("done")).toBeTruthy();
    w.unmount();
  });

  it("avec des amis, propose d'abord un ami", async () => {
    api.friends.mockResolvedValue({
      data: [{ user: { username: "cam", display_name: "Camille" } }],
      total: 1,
    });
    const w = await open();
    expect(document.querySelector('[data-test="loan-friend"]')).not.toBeNull();
    expect(
      (document.querySelector('[data-test="loan-submit"]') as HTMLButtonElement)
        .disabled,
    ).toBe(true);
    w.unmount();
  });
});
