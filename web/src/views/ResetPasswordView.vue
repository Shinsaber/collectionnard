<template>
  <AuthShell :subtitle="t('account.reset.subtitle')">
    <div v-if="done" class="flex flex-col gap-4" data-test="reset-done">
      <ElAlert
        type="success"
        :closable="false"
        show-icon
        :title="t('account.reset.done')"
      />
      <RouterLink to="/login">
        <ElButton type="primary" size="large" class="w-full">{{
          t("auth.login")
        }}</ElButton>
      </RouterLink>
    </div>
    <form v-else class="flex flex-col gap-4" @submit.prevent="submit">
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
        :disabled="!password || !token"
      >
        {{ t("account.reset.submit") }}
      </ElButton>
    </form>
  </AuthShell>
</template>

<script setup lang="ts">
// Nouveau mot de passe depuis un lien de réinitialisation donné par un admin (P4-0).
import { ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute } from "vue-router";
import { authApi, extractApiError } from "@/api/client";
import AuthShell from "@/components/layout/AuthShell.vue";
import PasswordFields from "@/components/account/PasswordFields.vue";

const { t } = useI18n();
const token = String(useRoute().query.token ?? "");
const password = ref("");
const loading = ref(false);
const error = ref("");
const done = ref(false);

async function submit() {
  error.value = "";
  loading.value = true;
  try {
    await authApi.resetPassword(token, password.value);
    done.value = true;
  } catch (e) {
    error.value = extractApiError(e).message;
  } finally {
    loading.value = false;
  }
}
</script>
