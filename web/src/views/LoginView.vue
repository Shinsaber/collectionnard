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

      <form @submit.prevent="handleLogin" class="flex flex-col gap-4">
        <div class="flex flex-col gap-1">
          <label
            for="username"
            class="text-sm font-medium text-gray-700 dark:text-gray-300"
            >Utilisateur</label
          >
          <InputText
            id="username"
            v-model="username"
            placeholder="Nom d'utilisateur"
            required
            autofocus
          />
        </div>

        <div class="flex flex-col gap-1">
          <label
            for="password"
            class="text-sm font-medium text-gray-700 dark:text-gray-300"
            >Mot de passe</label
          >
          <Password
            id="password"
            v-model="password"
            :feedback="false"
            toggle-mask
            required
            class="w-full"
            input-class="w-full"
          />
        </div>

        <Message v-if="error" severity="error" :closable="false">{{
          error
        }}</Message>

        <Button
          type="submit"
          label="Se connecter"
          icon="pi pi-sign-in"
          :loading="loading"
          class="w-full"
        />
      </form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from "vue";
import { useRouter } from "vue-router";
import InputText from "primevue/inputtext";
import Password from "primevue/password";
import Button from "primevue/button";
import Message from "primevue/message";
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
