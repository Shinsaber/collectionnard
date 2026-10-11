import { defineStore } from "pinia";
import { ref, computed } from "vue";
import { authApi } from "@/api/client";
import {
  clearTokens,
  getAccessToken,
  getRefreshToken,
  setTokens,
} from "@/api/tokens";
import type { User, LoginRequest } from "@/api/types";

export const useAuthStore = defineStore("auth", () => {
  const user = ref<User | null>(null);
  const token = ref<string | null>(getAccessToken());

  // Un refresh token suffit à considérer la session ouverte : le client HTTP
  // renouvelle l'access token à la première requête si nécessaire.
  const isAuthenticated = computed(() => !!token.value || !!getRefreshToken());
  const isAdmin = computed(() => user.value?.role === "admin");

  async function login(credentials: LoginRequest) {
    const res = await authApi.login(credentials);
    setTokens(res.token, res.refresh_token);
    token.value = res.token;
    user.value = res.user;
  }

  async function fetchMe() {
    if (!isAuthenticated.value) return;
    try {
      user.value = await authApi.me();
      token.value = getAccessToken();
    } catch {
      clearSession();
    }
  }

  function clearSession() {
    clearTokens();
    token.value = null;
    user.value = null;
  }

  async function logout() {
    const refreshToken = getRefreshToken();
    clearSession();
    if (refreshToken) {
      try {
        await authApi.logout(refreshToken);
      } catch {
        /* révocation côté serveur impossible (réseau) : la session locale est déjà purgée */
      }
    }
  }

  return { user, token, isAuthenticated, isAdmin, login, fetchMe, logout };
});
