<template>
  <AuthShell :subtitle="t('auth.tagline')">
    <form class="flex flex-col gap-4" @submit.prevent="handleLogin">
      <div class="flex flex-col gap-1.5">
        <label
          for="username"
          class="font-mono text-[11px] uppercase tracking-wider text-muted"
          >{{ t("auth.username") }}</label
        >
        <ElInput
          id="username"
          v-model="username"
          size="large"
          :placeholder="t('auth.usernamePlaceholder')"
          autocomplete="username"
          autofocus
        />
      </div>

      <div class="flex flex-col gap-1.5">
        <label
          for="password"
          class="font-mono text-[11px] uppercase tracking-wider text-muted"
          >{{ t("auth.password") }}</label
        >
        <ElInput
          id="password"
          v-model="password"
          size="large"
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

      <ElButton
        type="primary"
        size="large"
        native-type="submit"
        class="mt-1"
        :loading="loading"
      >
        {{ t("auth.login") }}
      </ElButton>
    </form>
  </AuthShell>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { useAuthStore } from "@/stores/auth";
import { extractApiError } from "@/api/client";
import AuthShell from "@/components/layout/AuthShell.vue";

const auth = useAuthStore();
const router = useRouter();
const { t } = useI18n();

const username = ref("");
const password = ref("");
const loading = ref(false);
// Clé de catalogue plutôt que texte : le message suit un changement de langue.
const errorKey = ref("");
const error = computed(() => (errorKey.value ? t(errorKey.value) : ""));

async function handleLogin() {
  errorKey.value = "";
  loading.value = true;
  try {
    await auth.login({ username: username.value, password: password.value });
    router.push("/");
  } catch (e) {
    errorKey.value =
      extractApiError(e).code === "account_disabled"
        ? "errors.api.account_disabled"
        : "auth.invalidCredentials";
  } finally {
    loading.value = false;
  }
}
</script>
