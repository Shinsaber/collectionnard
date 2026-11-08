import { flushPromises, mount } from "@vue/test-utils";
import ElementPlus from "element-plus";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { createMemoryHistory, createRouter } from "vue-router";
import type { Loan } from "@/api/types";
import { today } from "@/lib/loans";
import LoanCard from "./LoanCard.vue";

const api = vi.hoisted(() => ({ update: vi.fn(), remove: vi.fn() }));
vi.mock("@/api/client", () => ({
  loanApi: api,
  extractApiError: () => ({ message: "" }),
}));

const loan: Loan = {
  id: "p1",
  lot_id: "l1",
  collection_id: "c1",
  quantity: 2,
  borrower: { username: "cam", display_name: "Camille" },
  borrower_name: "Camille",
  lent_at: "2026-11-01",
  due_at: "2026-11-05",
  overdue: true,
  note: "le coffret",
  owner: { username: "user" },
  created_at: "",
  updated_at: "",
};

function mountCard(props: { loan: Loan; borrowerSide?: boolean }) {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: "/:any(.*)*", component: { template: "<div />" } }],
  });
  return mount(LoanCard, { props, global: { plugins: [ElementPlus, router] } });
}

describe("LoanCard", () => {
  beforeEach(() => {
    api.update.mockReset().mockResolvedValue({});
  });

  it("côté prêteur : l'emprunteur, le retard, et « Rendu aujourd'hui » qui garde le reste du prêt", async () => {
    const w = mountCard({ loan });
    expect(w.text()).toContain("Prêté à Camille");
    expect(w.find('[data-test="loan-overdue"]').exists()).toBe(true);
    await w.find('[data-test="loan-return"]').trigger("click");
    await flushPromises();
    expect(api.update).toHaveBeenCalledWith("p1", {
      quantity: 2,
      lent_at: "2026-11-01",
      due_at: "2026-11-05",
      returned_at: today(),
      note: "le coffret",
    });
    expect(w.emitted("changed")).toBeTruthy();
  });

  it("côté emprunteur : le prêteur nommé, sans action", () => {
    const w = mountCard({ loan, borrowerSide: true });
    expect(w.text()).toContain("Prêté par user");
    expect(w.find('[data-test="loan-return"]').exists()).toBe(false);
    expect(w.text()).not.toContain("le coffret");
  });
});
