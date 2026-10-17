import { defineStore } from "pinia";
import { ref, computed } from "vue";
import { authApi, meApi } from "@/api/client";
import {
  clearTokens,
  getAccessToken,
  getRefreshToken,
  setTokens,
} from "@/api/tokens";
import type {
  LoginRequest,
  LoginResponse,
  PasswordChange,
  RegisterRequest,
  User,
} from "@/api/types";
import { clearOfflineData } from "@/lib/offlineDb";
import { useOfflineStore } from "@/stores/offline";
import axios from "axios";

export const useAuthStore = defineStore("auth", () => {
  const user = ref<User | null>(null);
  const token = ref<string | null>(getAccessToken());

  // Un refresh token suffit à considérer la session ouverte : le client HTTP
  // renouvelle l'access token à la première requête si nécessaire.
  const isAuthenticated = computed(() => !!token.value || !!getRefreshToken());
  const isAdmin = computed(() => user.value?.role === "admin");

  // Session ouverte (connexion, inscription, nouveau mot de passe) : jetons et profil.
  function openSession(res: LoginResponse) {
    setTokens(res.token, res.refresh_token);
    token.value = res.token;
    user.value = res.user;
  }

  async function login(credentials: LoginRequest) {
    openSession(await authApi.login(credentials));
  }

  /** Compte créé depuis une invitation, puis connecté (P4-0). */
  async function register(data: RegisterRequest) {
    openSession(await authApi.register(data));
    await fetchMe(); // can_invite n'est donné que par /auth/me
  }

  /** Toutes les sessions sont révoquées : la courante reçoit de nouveaux jetons. */
  async function changePassword(data: PasswordChange) {
    const canInvite = user.value?.can_invite;
    openSession(await meApi.changePassword(data));
    if (user.value) user.value.can_invite = canInvite;
  }

  async function fetchMe() {
    if (!isAuthenticated.value) return;
    try {
      user.value = await authApi.me();
      token.value = getAccessToken();
    } catch (e) {
      // Pas de réseau : la session reste ouverte, l'utilisateur vient de la copie hors ligne
      // (lot E5c). Seul un refus du serveur (session expirée, compte supprimé) déconnecte.
      if (axios.isAxiosError(e) && !e.response) {
        const offline = useOfflineStore();
        offline.reachable = false;
        await offline.load();
        user.value = offline.snapshot?.user ?? null;
        return;
      }
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
    // La copie hors ligne (lot E5) appartient à l'utilisateur : effacée de l'appareil.
    await clearOfflineData();
    if (refreshToken) {
      try {
        await authApi.logout(refreshToken);
      } catch {
        /* révocation côté serveur impossible (réseau) : la session locale est déjà purgée */
      }
    }
  }

  return {
    user,
    token,
    isAuthenticated,
    isAdmin,
    login,
    register,
    changePassword,
    fetchMe,
    logout,
  };
});
