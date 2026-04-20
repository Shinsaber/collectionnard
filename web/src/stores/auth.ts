import { defineStore } from "pinia";
import { ref, computed } from "vue";
import { authApi } from "@/api/client";
import type { User, LoginRequest } from "@/api/types";

export const useAuthStore = defineStore("auth", () => {
  const user = ref<User | null>(null);
  const token = ref<string | null>(localStorage.getItem("token"));

  const isAuthenticated = computed(() => !!token.value);
  const isAdmin = computed(() => user.value?.role === "admin");

  async function login(credentials: LoginRequest) {
    const res = await authApi.login(credentials);
    token.value = res.token;
    user.value = res.user;
    localStorage.setItem("token", res.token);
  }

  async function fetchMe() {
    if (!token.value) return;
    try {
      user.value = await authApi.me();
    } catch {
      logout();
    }
  }

  function logout() {
    token.value = null;
    user.value = null;
    localStorage.removeItem("token");
  }

  return { user, token, isAuthenticated, isAdmin, login, fetchMe, logout };
});
