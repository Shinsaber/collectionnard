<template>
  <AppSpinner v-if="loading" />
  <p
    v-else-if="!profile"
    class="rounded-2xl bg-surface p-6 text-muted shadow-card"
    data-test="public-not-found"
  >
    {{ t("sharing.public.notFound") }}
  </p>
  <div v-else class="flex flex-col gap-6">
    <section
      class="flex flex-col gap-4 rounded-2xl bg-surface p-5 shadow-card sm:flex-row sm:items-center"
    >
      <UserAvatar :user="profile" :size="88" />
      <div class="flex min-w-0 flex-col gap-1">
        <h1 class="text-2xl font-extrabold tracking-tight">
          {{ profile.display_name || profile.username }}
        </h1>
        <span class="font-mono text-xs text-muted"
          >@{{ profile.username }} ·
          {{
            t("sharing.public.memberSince", {
              date: new Date(profile.created_at).toLocaleDateString(lang, {
                month: "long",
                year: "numeric",
              }),
            })
          }}</span
        >
        <p v-if="profile.bio" class="mt-1 whitespace-pre-line text-[15px]">
          {{ profile.bio }}
        </p>
      </div>
      <div
        v-if="auth.isAuthenticated && auth.user?.username !== profile.username"
        class="flex gap-2 sm:ml-auto sm:self-start"
      >
        <ElButton
          :loading="inviting"
          data-test="profile-invite-friend"
          @click="inviteFriend"
          >{{ t("friends.invite.button") }}</ElButton
        >
        <BlockButton :username="profile.username" @blocked="blocks++" />
      </div>
    </section>

    <section class="flex flex-col gap-3">
      <h2 class="text-lg font-extrabold tracking-tight">
        {{ t("sharing.public.collections") }}
      </h2>
      <p
        v-if="collections.length === 0"
        class="rounded-2xl bg-surface p-5 text-muted shadow-card"
      >
        {{ t("sharing.public.noCollections") }}
      </p>
      <ul v-else class="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
        <li v-for="c in collections" :key="c.id">
          <RouterLink
            :to="`/c/${c.id}`"
            class="flex h-full flex-col gap-1.5 rounded-2xl bg-surface p-4 shadow-card hover:shadow-raised"
            data-test="public-collection"
          >
            <span class="font-bold text-ink">{{ c.name }}</span>
            <span class="font-mono text-[11px] uppercase text-muted"
              >{{ typeTitle(c.collection_type, lang) }} ·
              {{ t("collections.itemCount", c.item_count) }}</span
            >
            <span
              v-if="c.description"
              class="line-clamp-2 text-sm text-muted"
              >{{ c.description }}</span
            >
          </RouterLink>
        </li>
      </ul>
    </section>

    <section class="flex flex-col gap-3" data-test="public-reviews">
      <h2 class="text-lg font-extrabold tracking-tight">
        {{ t("reviews.publicReviews") }}
      </h2>
      <ReviewUserList
        :key="`${profile.username}-${blocks}`"
        :fetch="fetchReviews"
        :empty-text="t('reviews.noPublicReviews')"
      />
    </section>
  </div>
</template>

<script setup lang="ts">
// Profil public d'un membre (/u/:username) et ses collections visibles (P4-2).
import { onMounted, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { friendApi, publicApi } from "@/api/client";
import { useLang } from "@/composables/useLang";
import { useNotify } from "@/composables/useNotify";
import { typeTitle } from "@/lib/schema";
import AppSpinner from "@/components/common/AppSpinner.vue";
import UserAvatar from "@/components/account/UserAvatar.vue";
import ReviewUserList from "@/components/review/ReviewUserList.vue";
import BlockButton from "@/components/review/BlockButton.vue";
import { useAuthStore } from "@/stores/auth";
import type {
  PaginationParams,
  PublicCollection,
  PublicProfile,
} from "@/api/types";

const props = defineProps<{ username: string }>();
const fetchReviews = (params: PaginationParams) =>
  publicApi.reviews(props.username, params);
const { t } = useI18n();
const { lang } = useLang();
const { notifyError, notifySuccess } = useNotify();

const profile = ref<PublicProfile | null>(null);
const collections = ref<PublicCollection[]>([]);
const loading = ref(true);
/** Change après un blocage : la liste des avis se recharge (ceux du membre bloqué me sont masqués). */
const blocks = ref(0);
const auth = useAuthStore();
const inviting = ref(false);

/** Inviter en ami (P7-2) : le profil m'est visible, l'invitation par nom est permise. */
async function inviteFriend() {
  if (!profile.value) return;
  inviting.value = true;
  try {
    const f = await friendApi.request(profile.value.username);
    const name = f.user.display_name || f.user.username;
    notifySuccess(
      t(
        f.status === "accepted"
          ? "friends.invite.accepted"
          : "friends.invite.sent",
        { name },
      ),
    );
  } catch (e) {
    notifyError(e);
  } finally {
    inviting.value = false;
  }
}

async function load() {
  loading.value = true;
  try {
    [profile.value, collections.value] = await Promise.all([
      publicApi.profile(props.username),
      publicApi.collections(props.username),
    ]);
  } catch (e) {
    profile.value = null;
    const status = (e as { response?: { status?: number } })?.response?.status;
    if (status !== 404) notifyError(e);
  } finally {
    loading.value = false;
  }
}

onMounted(load);
watch(() => props.username, load);
</script>
