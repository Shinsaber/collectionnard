import { flushPromises, mount } from "@vue/test-utils";
import ElementPlus from "element-plus";
import { beforeEach, describe, expect, it, vi } from "vitest";
import ProposalQuotaCard from "./ProposalQuotaCard.vue";

const quota = vi.hoisted(() => vi.fn());
vi.mock("@/api/client", () => ({ proposalApi: { quota } }));

const base = {
  unlimited: false,
  accepted: 1,
  base: 5,
  max: 50,
  daily_limit: 20,
  today: 2,
};

async function mountCard() {
  const w = mount(ProposalQuotaCard, { global: { plugins: [ElementPlus] } });
  await flushPromises();
  return w;
}

describe("ProposalQuotaCard", () => {
  beforeEach(() => quota.mockReset());

  it("montre les propositions en attente sur la limite et la règle", async () => {
    quota.mockResolvedValue({ ...base, waiting: 2, limit: 6 });
    const w = await mountCard();
    expect(w.text()).toContain("2 sur 6 propositions en attente de relecture");
    expect(w.text()).toContain(
      "+1 par proposition acceptée (1 pour l'instant), jusqu'à 50",
    );
    expect(w.text()).toContain("2 sur 20 créées ces dernières 24 h");
    expect(w.text()).not.toContain("Quota atteint");
  });

  it("prévient quand le quota est atteint ; rien pour un modérateur", async () => {
    quota.mockResolvedValue({ ...base, waiting: 6, limit: 6 });
    expect((await mountCard()).text()).toContain("Quota atteint");
    quota.mockResolvedValue({ ...base, unlimited: true, waiting: 0, limit: 6 });
    expect(
      (await mountCard()).find('[data-test="proposal-quota"]').exists(),
    ).toBe(false);
  });
});
