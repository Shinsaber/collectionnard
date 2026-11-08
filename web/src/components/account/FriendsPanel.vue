<template>
  <div class="flex flex-col gap-6">
    <p class="text-sm text-muted">{{ t("friends.help") }}</p>

    <section class="flex flex-col gap-3 rounded-2xl bg-surface p-5 shadow-card">
      <label :for="`${uid}-user`" class="font-bold">{{
        t("friends.invite.label")
      }}</label>
      <form class="flex max-w-md gap-2" @submit.prevent="invite">
        <ElInput
          :id="`${uid}-user`"
          v-model="username"
          :placeholder="t('friends.invite.placeholder')"
          autocomplete="off"
          data-test="friend-username"
        />
        <ElButton
          type="primary"
          native-type="submit"
          :disabled="!username.trim()"
          :loading="inviting"
          data-test="friend-invite"
          >{{ t("friends.invite.submit") }}</ElButton
        >
      </form>
      <p class="text-sm text-muted">{{ t("friends.invite.help") }}</p>
    </section>

    <FriendLinksCard />

    <section
      v-for="group in requestGroups"
      :key="group.key"
      class="flex flex-col gap-3 rounded-2xl bg-surface p-5 shadow-card"
      :data-test="`friend-requests-${group.key}`"
    >
      <h2 class="text-lg font-extrabold tracking-tight">{{ group.title }}</h2>
      <ul class="flex flex-col gap-2">
        <li
          v-for="f in group.list"
          :key="f.id"
          class="flex items-center gap-3 rounded-xl bg-soft px-3 py-2"
          data-test="friend-request"
        >
          <UserAvatar :user="f.user" :size="32" />
          <span class="flex min-w-0 flex-1 flex-col">
            <span class="truncate font-semibold">{{ name(f) }}</span>
            <span class="font-mono text-[11px] text-muted">{{
              t("friends.requested", { date: day(f.created_at) })
            }}</span>
          </span>
          <template v-if="f.incoming">
            <ElButton
              type="primary"
              size="small"
              data-test="friend-accept"
              @click="accept(f)"
              >{{ t("friends.accept") }}</ElButton
            >
            <ElButton
              size="small"
              data-test="friend-refuse"
              @click="drop(f, 'friends.refused')"
              >{{ t("friends.refuse") }}</ElButton
            >
          </template>
          <ElButton
            v-else
            text
            size="small"
            data-test="friend-cancel"
            @click="drop(f, 'friends.cancelled')"
            >{{ t("friends.cancel") }}</ElButton
          >
        </li>
      </ul>
    </section>

    <section class="flex flex-col gap-3">
      <h2 class="text-lg font-extrabold tracking-tight">
        {{ t("friends.list") }}
      </h2>
      <AppSpinner
        v-if="friends.loading.value && friends.data.value.length === 0"
      />
      <p
        v-else-if="friends.data.value.length === 0"
        class="rounded-2xl bg-surface p-5 text-muted shadow-card"
        data-test="friends-none"
      >
        {{ t("friends.none") }}
      </p>
      <ul v-else class="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
        <li
          v-for="f in friends.data.value"
          :key="f.id"
          class="flex items-center gap-3 rounded-2xl bg-surface p-4 shadow-card"
          data-test="friend"
        >
          <UserAvatar :user="f.user" :size="48" />
          <span class="flex min-w-0 flex-1 flex-col">
            <RouterLink
              v-if="f.profile_visible"
              :to="`/u/${f.user.username}`"
              class="truncate font-bold text-ink hover:underline"
              :title="t('friends.profile')"
              >{{ name(f) }}</RouterLink
            >
            <span v-else class="truncate font-bold text-ink">{{
              name(f)
            }}</span>
            <span class="truncate font-mono text-xs text-muted"
              >@{{ f.user.username }}</span
            >
            <span v-if="f.accepted_at" class="text-xs text-muted">{{
              t("friends.since", { date: day(f.accepted_at) })
            }}</span>
          </span>
          <ElButton
            text
            size="small"
            data-test="friend-remove"
            @click="remove(f)"
            >{{ t("friends.remove") }}</ElButton
          >
        </li>
      </ul>
      <ElPagination
        class="justify-end"
        hide-on-single-page
        layout="prev, pager, next"
        :total="friends.total.value"
        :page-size="friends.perPage.value"
        :current-page="friends.page.value"
        @current-change="onPage"
      />
    </section>
  </div>
</template>

<script setup lang="ts">
// Amis (P7-2), onglet de la page « Membres » : inviter un membre par son nom d'utilisateur exact, répondre aux demandes, retirer un ami.
// Une demande envoyée à un profil qui ne m'est pas visible ne montre que le nom saisi (le serveur l'élague).
import { computed, onMounted, ref, useId } from "vue";
import { useI18n } from "vue-i18n";
import { extractApiError, friendApi } from "@/api/client";
import { useConfirmDialog } from "@/composables/useConfirmDialog";
import { useLang } from "@/composables/useLang";
import { useNotify } from "@/composables/useNotify";
import { usePagination } from "@/composables/usePagination";
import AppSpinner from "@/components/common/AppSpinner.vue";
import FriendLinksCard from "./FriendLinksCard.vue";
import UserAvatar from "./UserAvatar.vue";
import type { Friendship } from "@/api/types";

const { t } = useI18n();
const { lang } = useLang();
const { notifyError, notifySuccess, notifyWarning } = useNotify();
const { confirm } = useConfirmDialog();
const uid = useId();

const friends = usePagination((params: Record<string, unknown>) =>
  friendApi.list(params),
);
/** Demandes en attente : peu nombreuses, chargées d'un coup. */
const requests = ref<Friendship[]>([]);

const requestGroups = computed(() =>
  [
    {
      key: "received",
      title: t("friends.received"),
      list: requests.value.filter((f) => f.incoming),
    },
    {
      key: "sent",
      title: t("friends.sent"),
      list: requests.value.filter((f) => !f.incoming),
    },
  ].filter((g) => g.list.length > 0),
);

const name = (f: Friendship) => f.user.display_name || f.user.username;
const day = (iso: string) => new Date(iso).toLocaleDateString(lang.value);

async function reload() {
  try {
    const [, pending] = await Promise.all([
      friends.load(),
      friendApi.requests({ per_page: 100 }),
    ]);
    requests.value = pending.data;
  } catch (e) {
    notifyError(e);
  }
}

const username = ref("");
const inviting = ref(false);
async function invite() {
  inviting.value = true;
  try {
    const f = await friendApi.request(username.value.trim());
    username.value = "";
    notifySuccess(
      t(
        f.status === "accepted"
          ? "friends.invite.accepted"
          : "friends.invite.sent",
        { name: name(f) },
      ),
    );
    await reload();
  } catch (e) {
    // Nom inconnu ou profil qui ne m'est pas visible : même réponse, on oriente vers le lien d'ami.
    if (extractApiError(e).code === "not_found")
      notifyWarning(t("friends.invite.notFound"));
    else notifyError(e);
  } finally {
    inviting.value = false;
  }
}

async function accept(f: Friendship) {
  try {
    await friendApi.accept(f.id);
    notifySuccess(t("friends.invite.accepted", { name: name(f) }));
    await reload();
  } catch (e) {
    notifyError(e);
  }
}

async function drop(f: Friendship, done: string) {
  try {
    await friendApi.remove(f.id);
    notifySuccess(t(done));
    await reload();
  } catch (e) {
    notifyError(e);
  }
}

async function remove(f: Friendship) {
  const ok = await confirm({
    message: t("friends.removeConfirm", { name: name(f) }),
    confirmText: t("friends.remove"),
    danger: true,
  });
  if (ok) await drop(f, "friends.removed");
}

function onPage(p: number) {
  friends.goToPage(p);
  friends.load().catch(notifyError);
}

onMounted(reload);
</script>
