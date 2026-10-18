import { flushPromises, mount } from "@vue/test-utils";
import { createPinia, setActivePinia } from "pinia";
import ElementPlus from "element-plus";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { nextTick } from "vue";
import { createMemoryHistory, createRouter } from "vue-router";
import { useLang } from "@/composables/useLang";
import RegisterView from "./RegisterView.vue";

const api = vi.hoisted(() => ({
  checkInvitation: vi.fn(),
  register: vi.fn(),
  me: vi.fn(),
}));
vi.mock("@/api/client", () => ({
  authApi: api,
  meApi: {},
  extractApiError: (e: { message?: string }) => ({ message: e.message ?? "" }),
}));

async function mountAt(path: string) {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: "/", component: { template: "<div>accueil</div>" } },
      { path: "/register", component: RegisterView },
    ],
  });
  await router.push(path);
  const wrapper = mount(RegisterView, {
    global: { plugins: [ElementPlus, router] },
  });
  await flushPromises();
  return { wrapper, router };
}

describe("RegisterView", () => {
  beforeEach(async () => {
    setActivePinia(createPinia());
    Object.values(api).forEach((m) => m.mockReset());
    localStorage.clear();
    useLang().setLang("fr");
    await nextTick();
  });

  it("refuse une invitation qui n'est plus valable", async () => {
    api.checkInvitation.mockResolvedValue({ valid: false });
    const { wrapper } = await mountAt("/register?token=vieux");
    expect(api.checkInvitation).toHaveBeenCalledWith("vieux");
    expect(wrapper.find('[data-test="invalid-invitation"]').exists()).toBe(
      true,
    );
    expect(wrapper.find("form").exists()).toBe(false);
  });

  it("sans jeton, n'interroge pas le serveur", async () => {
    const { wrapper } = await mountAt("/register");
    expect(api.checkInvitation).not.toHaveBeenCalled();
    expect(wrapper.find('[data-test="invalid-invitation"]').exists()).toBe(
      true,
    );
  });

  it("crée le compte une fois le mot de passe confirmé, puis connecte", async () => {
    api.checkInvitation.mockResolvedValue({ valid: true });
    api.register.mockResolvedValue({
      token: "access",
      refresh_token: "refresh",
      expires_in: 900,
      user: {
        id: "u1",
        username: "camille",
        email: "c@example.org",
        role: "user",
      },
    });
    api.me.mockResolvedValue({
      id: "u1",
      username: "camille",
      email: "c@example.org",
      role: "user",
    });
    const { wrapper, router } = await mountAt("/register?token=tok");
    const submit = () => wrapper.find('button[type="submit"]');

    await wrapper.find("#reg-username").setValue(" camille ");
    await wrapper.find("#reg-email").setValue("c@example.org");
    const [first, second] = wrapper.findAll('input[type="password"]');
    await first!.setValue("longenough");
    await second!.setValue("different!");
    expect(wrapper.text()).toContain("Les deux saisies diffèrent.");
    expect(submit().attributes("disabled")).toBeDefined();

    await second!.setValue("longenough");
    expect(submit().attributes("disabled")).toBeUndefined();
    await wrapper.find("form").trigger("submit");
    await flushPromises();
    expect(api.register).toHaveBeenCalledWith({
      token: "tok",
      username: "camille",
      email: "c@example.org",
      password: "longenough",
    });
    expect(localStorage.getItem("refresh_token")).toBe("refresh");
    expect(router.currentRoute.value.path).toBe("/");
  });
});
