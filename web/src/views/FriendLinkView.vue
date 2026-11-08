<template>
  <div class="mx-auto flex w-full max-w-md flex-col gap-6">
    <PageHeader :title="t('friends.link.title')" />
    <AppSpinner v-if="loading" />
    <p
      v-else-if="!link"
      class="rounded-2xl bg-surface p-5 text-muted shadow-card"
      data-test="friend-link-invalid"
    >
      {{ t("friends.link.invalid") }}
    </p>
    <section
      v-else
      class="flex flex-col items-center gap-4 rounded-2xl bg-surface p-6 text-center shadow-card"
    >
      <UserAvatar :user="link.user ?? null" :size="64" />
      <p class="text-lg font-bold" data-test="friend-link-invites">
        {{ t("friends.link.invites", { name: ownerName }) }}
      </p>
      <p v-if="own" class="text-sm text-muted">{{ t("friends.link.own") }}</p>
      <template v-else>
        <p class="text-sm text-muted">{{ t("friends.link.help") }}</p>
        <div class="flex gap-2">
          <ElButton @click="router.push('/members')">{{
            t("friends.link.later")
          }}</ElButton>
          <ElButton
            type="primary"
            :loading="accepting"
            data-test="friend-link-accept"
            @click="accept"
            >{{ t("friends.link.accept") }}</ElButton
          >
        </div>
      </template>
    </section>
  </div>
</template>

<script setup lang="ts">
// Ouverture d'un lien d'ami (P7-2) : qui m'invite, puis accepter. Un lien inconnu, expiré, utilisé, ou d'un membre
// qui m'a bloqué répond 404 : même message pour tous.
import { computed, onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { friendLinkApi } from "@/api/client";
import { useNotify } from "@/composables/useNotify";
import { useAuthStore } from "@/stores/auth";
import AppSpinner from "@/components/common/AppSpinner.vue";
import PageHeader from "@/components/common/PageHeader.vue";
import UserAvatar from "@/components/account/UserAvatar.vue";
import type { FriendLink } from "@/api/types";

const props = defineProps<{ token: string }>();
const { t } = useI18n();
const router = useRouter();
const auth = useAuthStore();
const { notifyError, notifySuccess } = useNotify();

const link = ref<FriendLink | null>(null);
const loading = ref(true);
const accepting = ref(false);

const ownerName = computed(
  () => link.value?.user?.display_name || link.value?.user?.username || "",
);
const own = computed(
  () => !!link.value && link.value.user?.username === auth.user?.username,
);

onMounted(async () => {
  try {
    link.value = await friendLinkApi.open(props.token);
  } catch (e) {
    const status = (e as { response?: { status?: number } })?.response?.status;
    if (status !== 404) notifyError(e);
  } finally {
    loading.value = false;
  }
});

async function accept() {
  accepting.value = true;
  try {
    await friendLinkApi.accept(props.token);
    notifySuccess(t("friends.invite.accepted", { name: ownerName.value }));
    router.push("/members");
  } catch (e) {
    notifyError(e);
  } finally {
    accepting.value = false;
  }
}
</script>
