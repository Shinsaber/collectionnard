<template>
  <div
    class="min-h-screen flex items-center justify-center bg-gray-100 dark:bg-gray-900"
  >
    <div
      class="bg-white dark:bg-gray-800 rounded-xl shadow-lg p-8 w-full max-w-sm"
    >
      <h1
        class="text-2xl font-bold text-center text-indigo-600 dark:text-indigo-400 mb-6"
      >
        📦 Collectionnard
      </h1>

      <form class="flex flex-col gap-4" @submit.prevent="handleLogin">
        <div class="flex flex-col gap-1">
          <label
            for="username"
            class="text-sm font-medium text-gray-700 dark:text-gray-300"
            >Utilisateur</label
          >
          <ElInput
            id="username"
            v-model="username"
            placeholder="Nom d'utilisateur"
            autocomplete="username"
            autofocus
          />
        </div>

        <div class="flex flex-col gap-1">
          <label
            for="password"
            class="text-sm font-medium text-gray-700 dark:text-gray-300"
            >Mot de passe</label
          >
          <ElInput
            id="password"
            v-model="password"
            type="password"
            show-password
            autocomplete="current-password"
          />
        </div>

        <ElAlert
          v-if="error"
          type="error"
          :closable="false"
          show-icon
          :title="error"
        />

        <ElButton type="primary" native-type="submit" :loading="loading">
          <i class="pi pi-sign-in mr-2" aria-hidden="true"></i> Se connecter
        </ElButton>
      </form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from "vue";
import { useRouter } from "vue-router";
import { useAuthStore } from "@/stores/auth";

const auth = useAuthStore();
const router = useRouter();

const username = ref("");
const password = ref("");
const loading = ref(false);
const error = ref("");

async function handleLogin() {
  error.value = "";
  loading.value = true;
  try {
    await auth.login({ username: username.value, password: password.value });
    router.push("/");
  } catch {
    error.value = "Identifiants invalides";
  } finally {
    loading.value = false;
  }
}
</script>
