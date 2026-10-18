<template>
  <div class="flex max-w-3xl flex-col gap-6">
    <PageHeader :title="t('account.profile.title')">
      <template #subtitle>
        <p class="text-[15px] text-muted">
          {{
            t("account.profile.subtitle", {
              username: auth.user?.username ?? "",
              email: auth.user?.email ?? "",
            })
          }}
        </p>
      </template>
    </PageHeader>

    <!-- Profil -->
    <section class="flex flex-col gap-4 rounded-2xl bg-surface p-5 shadow-card">
      <h2 class="text-lg font-extrabold tracking-tight">
        {{ t("account.profile.section") }}
      </h2>
      <form
        class="flex flex-col gap-4"
        data-test="profile-form"
        @submit.prevent="saveProfile"
      >
        <div class="flex flex-wrap items-center gap-4">
          <UserAvatar :user="preview" :size="72" />
          <div class="flex flex-wrap gap-2">
            <label
              class="el-button el-button--default cursor-pointer"
              :for="`${id}-avatar`"
            >
              <i class="pi pi-image mr-2" aria-hidden="true"></i>
              {{ t("account.profile.avatarChoose") }}
            </label>
            <input
              :id="`${id}-avatar`"
              type="file"
              accept="image/*"
              class="sr-only"
              @change="uploadAvatar"
            />
            <ElButton v-if="avatar" text @click="avatar = ''">{{
              t("account.profile.avatarRemove")
            }}</ElButton>
          </div>
        </div>
        <div class="flex flex-col gap-1.5">
          <label :for="`${id}-name`" :class="labelClass">{{
            t("account.displayName")
          }}</label>
          <ElInput
            :id="`${id}-name`"
            v-model="displayName"
            maxlength="100"
            :placeholder="auth.user?.username"
          />
        </div>
        <div class="flex flex-col gap-1.5">
          <label :for="`${id}-bio`" :class="labelClass">{{
            t("account.bio")
          }}</label>
          <ElInput
            :id="`${id}-bio`"
            v-model="bio"
            type="textarea"
            :autosize="{ minRows: 3, maxRows: 10 }"
            maxlength="2000"
            show-word-limit
          />
        </div>
        <div class="flex flex-col gap-1.5" data-test="profile-visibility">
          <span :class="labelClass">{{ t("sharing.profile.title") }}</span>
          <p class="text-sm text-muted">{{ t("sharing.profile.help") }}</p>
          <VisibilitySelect
            v-model="profileVisibility"
            :label="t('sharing.profile.title')"
          />
          <RouterLink
            v-if="auth.user && auth.user.profile_visibility !== 'private'"
            :to="`/u/${auth.user.username}`"
            class="text-sm font-semibold text-link"
          >
            <i class="pi pi-external-link mr-1 text-xs" aria-hidden="true"></i>
            {{ t("sharing.profile.view") }}
          </RouterLink>
        </div>
        <ElButton
          type="primary"
          native-type="submit"
          class="self-start"
          :loading="savingProfile || uploading"
          >{{ t("common.save") }}</ElButton
        >
      </form>
    </section>

    <!-- Mot de passe -->
    <section class="flex flex-col gap-4 rounded-2xl bg-surface p-5 shadow-card">
      <h2 class="text-lg font-extrabold tracking-tight">
        {{ t("account.password.title") }}
      </h2>
      <form
        :key="passwordKey"
        class="flex max-w-sm flex-col gap-4"
        @submit.prevent="savePassword"
      >
        <div class="flex flex-col gap-1.5">
          <label :for="`${id}-current`" :class="labelClass">{{
            t("account.password.current")
          }}</label>
          <ElInput
            :id="`${id}-current`"
            v-model="current"
            size="large"
            type="password"
            show-password
            autocomplete="current-password"
          />
        </div>
        <PasswordFields v-model="next" />
        <ElButton
          type="primary"
          native-type="submit"
          class="self-start"
          :loading="savingPassword"
          :disabled="!current || !next"
          >{{ t("account.password.submit") }}</ElButton
        >
      </form>
    </section>

    <InvitationsCard v-if="auth.user?.can_invite" :admin="auth.isAdmin" />
  </div>
</template>

<script setup lang="ts">
// Mon profil (P4-0) : nom affiché, présentation, avatar ; mot de passe ; invitations si permises.
import { computed, ref, useId, watch } from "vue";
import { useI18n } from "vue-i18n";
import { meApi, uploadApi } from "@/api/client";
import { useAuthStore } from "@/stores/auth";
import { useNotify } from "@/composables/useNotify";
import PageHeader from "@/components/common/PageHeader.vue";
import InvitationsCard from "@/components/account/InvitationsCard.vue";
import PasswordFields from "@/components/account/PasswordFields.vue";
import UserAvatar from "@/components/account/UserAvatar.vue";
import VisibilitySelect from "@/components/sharing/VisibilitySelect.vue";
import type { Visibility } from "@/api/types";

const { t } = useI18n();
const auth = useAuthStore();
const { notifyError, notifySuccess } = useNotify();
const id = useId();
const labelClass = "font-mono text-[11px] uppercase tracking-wider text-muted";

const displayName = ref("");
const bio = ref("");
const avatar = ref("");
const profileVisibility = ref<Visibility>("private");
watch(
  () => auth.user,
  (u) => {
    displayName.value = u?.display_name ?? "";
    bio.value = u?.bio ?? "";
    avatar.value = u?.avatar ?? "";
    profileVisibility.value = u?.profile_visibility ?? "private";
  },
  { immediate: true },
);
/** Aperçu de l'avatar et du nom avant l'enregistrement. */
const preview = computed(() => ({
  username: auth.user?.username ?? "",
  display_name: displayName.value,
  avatar: avatar.value,
}));

const uploading = ref(false);
async function uploadAvatar(ev: Event) {
  const input = ev.target as HTMLInputElement;
  const file = input.files?.[0];
  input.value = "";
  if (!file) return;
  uploading.value = true;
  try {
    avatar.value = (await uploadApi.upload(file)).url;
  } catch (e) {
    notifyError(e);
  } finally {
    uploading.value = false;
  }
}

const savingProfile = ref(false);
async function saveProfile() {
  savingProfile.value = true;
  try {
    const canInvite = auth.user?.can_invite;
    auth.user = await meApi.updateProfile({
      display_name: displayName.value.trim(),
      bio: bio.value.trim(),
      avatar: avatar.value,
      profile_visibility: profileVisibility.value,
    });
    auth.user.can_invite = canInvite;
    notifySuccess(t("account.profile.saved"));
  } catch (e) {
    notifyError(e);
  } finally {
    savingProfile.value = false;
  }
}

const current = ref("");
const next = ref("");
/** Change après un enregistrement : le formulaire (et ses deux saisies) repart à vide. */
const passwordKey = ref(0);
const savingPassword = ref(false);
async function savePassword() {
  savingPassword.value = true;
  try {
    await auth.changePassword({
      current_password: current.value,
      new_password: next.value,
    });
    current.value = "";
    next.value = "";
    passwordKey.value++;
    notifySuccess(t("account.password.saved"));
  } catch (e) {
    notifyError(e);
  } finally {
    savingPassword.value = false;
  }
}
</script>
