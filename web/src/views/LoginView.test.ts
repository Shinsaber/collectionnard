import { flushPromises, mount } from "@vue/test-utils";
import { createPinia, setActivePinia } from "pinia";
import ElementPlus from "element-plus";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { nextTick } from "vue";
import { createMemoryHistory, createRouter } from "vue-router";
import { useLang } from "@/composables/useLang";
import LoginView from "./LoginView.vue";

const { loginMock } = vi.hoisted(() => ({ loginMock: vi.fn() }));
vi.mock("@/api/client", () => ({
  authApi: { login: loginMock, logout: vi.fn(), refresh: vi.fn(), me: vi.fn() },
  // Code d'erreur de l'API : porté ici par l'erreur simulée.
  extractApiError: (e: { code?: string }) => ({ code: e.code, message: "" }),
}));

function mountLogin() {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: "/", component: { template: "<div>home</div>" } },
      { path: "/login", component: LoginView },
    ],
  });
  const wrapper = mount(LoginView, {
    global: { plugins: [ElementPlus, router] },
  });
  return { wrapper, router };
}

describe("LoginView", () => {
  beforeEach(async () => {
    setActivePinia(createPinia());
    loginMock.mockReset();
    localStorage.clear();
    // jsdom annonce en-US : on fixe la langue de l'interface (textes attendus en français).
    useLang().setLang("fr");
    await nextTick();
  });

  it("connecte l'utilisateur et redirige vers l'accueil", async () => {
    loginMock.mockResolvedValue({
      token: "access",
      refresh_token: "refresh",
      expires_in: 900,
      user: { id: "u1", username: "admin", email: "a@b", role: "admin" },
    });
    const { wrapper, router } = mountLogin();
    const push = vi.spyOn(router, "push");

    await wrapper.find('input[autocomplete="username"]').setValue("admin");
    await wrapper.find('input[type="password"]').setValue("secret");
    await wrapper.find("form").trigger("submit");
    await flushPromises();

    expect(loginMock).toHaveBeenCalledWith({
      username: "admin",
      password: "secret",
    });
    expect(push).toHaveBeenCalledWith("/");
    expect(localStorage.getItem("token")).toBe("access");
    expect(localStorage.getItem("refresh_token")).toBe("refresh");
  });

  it("affiche une erreur quand les identifiants sont refusés", async () => {
    loginMock.mockRejectedValue(new Error("401"));
    const { wrapper } = mountLogin();

    await wrapper.find('input[autocomplete="username"]').setValue("ghost");
    await wrapper.find('input[type="password"]').setValue("x");
    await wrapper.find("form").trigger("submit");
    await flushPromises();

    expect(wrapper.text()).toContain("Identifiants invalides");
    expect(localStorage.getItem("token")).toBeNull();
  });

  it("signale un compte désactivé", async () => {
    loginMock.mockRejectedValue(
      Object.assign(new Error("401"), { code: "account_disabled" }),
    );
    const { wrapper } = mountLogin();
    await wrapper.find('input[autocomplete="username"]').setValue("parti");
    await wrapper.find('input[type="password"]').setValue("x");
    await wrapper.find("form").trigger("submit");
    await flushPromises();
    expect(wrapper.text()).toContain("Ce compte est désactivé");
  });
});
