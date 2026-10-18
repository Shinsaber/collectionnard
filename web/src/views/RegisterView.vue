<template>
  <AuthShell :subtitle="t('account.register.subtitle')">
    <AppSpinner v-if="checking" />
    <ElAlert
      v-else-if="!valid"
      type="error"
      :closable="false"
      show-icon
      :title="t('account.register.invalid')"
      data-test="invalid-invitation"
    />
    <form
      v-else
      class="flex flex-col gap-4"
      data-test="register-form"
      @submit.prevent="submit"
    >
      <div class="flex flex-col gap-1.5">
        <label for="reg-username" :class="labelClass">{{
          t("auth.username")
        }}</label>
        <ElInput
          id="reg-username"
          v-model="username"
          size="large"
          autocomplete="username"
          autofocus
        />
        <small class="text-xs text-muted">{{
          t("account.register.usernameHelp")
        }}</small>
      </div>
      <div class="flex flex-col gap-1.5">
        <label for="reg-email" :class="labelClass">{{
          t("account.email")
        }}</label>
        <ElInput
          id="reg-email"
          v-model="email"
          size="large"
          type="email"
          autocomplete="email"
        />
      </div>
      <PasswordFields v-model="password" />
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
        :disabled="!password"
      >
        {{ t("account.register.submit") }}
      </ElButton>
    </form>
  </AuthShell>
</template>

<script setup lang="ts">
// Création d'un compte depuis un lien d'invitation (/register?token=…), puis connexion (P4-0).
import { onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute, useRouter } from "vue-router";
import { authApi, extractApiError } from "@/api/client";
import { useAuthStore } from "@/stores/auth";
import AppSpinner from "@/components/common/AppSpinner.vue";
import AuthShell from "@/components/layout/AuthShell.vue";
import PasswordFields from "@/components/account/PasswordFields.vue";

const { t } = useI18n();
const route = useRoute();
const router = useRouter();
const auth = useAuthStore();
const labelClass = "font-mono text-[11px] uppercase tracking-wider text-muted";

const token = String(route.query.token ?? "");
const checking = ref(true);
const valid = ref(false);
const username = ref("");
const email = ref("");
/** Mot de passe confirmé (PasswordFields ne l'émet que s'il est valide). */
const password = ref("");
const loading = ref(false);
const error = ref("");

onMounted(async () => {
  try {
    valid.value = !!token && (await authApi.checkInvitation(token)).valid;
  } catch (e) {
    error.value = extractApiError(e).message;
  } finally {
    checking.value = false;
  }
});

async function submit() {
  error.value = "";
  loading.value = true;
  try {
    await auth.register({
      token,
      username: username.value.trim(),
      email: email.value.trim(),
      password: password.value,
    });
    router.push("/");
  } catch (e) {
    error.value = extractApiError(e).message;
  } finally {
    loading.value = false;
  }
}
</script>
