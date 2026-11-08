<template>
  <section
    class="flex flex-col gap-3 rounded-2xl bg-surface p-5 shadow-card"
    data-test="friend-links"
  >
    <div class="flex flex-wrap items-center gap-2">
      <h2 class="text-lg font-extrabold tracking-tight">
        {{ t("friends.links.title") }}
      </h2>
      <ElButton
        class="ml-auto"
        :loading="creating"
        data-test="friend-link-create"
        @click="create"
        >{{ t("friends.links.create") }}</ElButton
      >
    </div>
    <p class="text-sm text-muted">{{ t("friends.links.help") }}</p>
    <p v-if="links.length === 0" class="text-sm text-muted">
      {{ t("friends.links.none") }}
    </p>
    <ul v-else class="flex flex-col gap-2">
      <li
        v-for="l in links"
        :key="l.id"
        class="flex items-center gap-3 rounded-xl bg-soft px-3 py-2"
        data-test="friend-link"
      >
        <span class="flex-1 font-mono text-[11px] text-muted">{{
          t("friends.links.expires", { date: day(l.expires_at) })
        }}</span>
        <ElButton
          text
          size="small"
          data-test="friend-link-revoke"
          @click="revoke(l)"
          >{{ t("friends.links.revoke") }}</ElButton
        >
      </li>
    </ul>

    <OneTimeLinkDialog
      :url="url"
      :title="t('friends.links.linkTitle')"
      :expires-at="expires"
      @close="url = ''"
    >
      <template v-if="canShare" #actions>
        <ElButton data-test="friend-link-share" @click="share">{{
          t("friends.links.share")
        }}</ElButton>
      </template>
    </OneTimeLinkDialog>
  </section>
</template>

<script setup lang="ts">
// Liens d'ami (P7-2) : un lien secret, à usage unique, pour devenir ami d'un membre sans deviner son nom. Le jeton
// n'est montré qu'à la création ; la liste ne garde que l'échéance de chaque lien valide, pour le révoquer.
import { onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { friendLinkApi } from "@/api/client";
import { useLang } from "@/composables/useLang";
import { useNotify } from "@/composables/useNotify";
import OneTimeLinkDialog from "./OneTimeLinkDialog.vue";
import type { FriendLink } from "@/api/types";

const { t } = useI18n();
const { lang } = useLang();
const { notifyError, notifySuccess } = useNotify();

const links = ref<FriendLink[]>([]);
const creating = ref(false);
const url = ref("");
const expires = ref<string | undefined>();
const canShare = typeof navigator !== "undefined" && !!navigator.share;

const day = (iso: string) => new Date(iso).toLocaleDateString(lang.value);

async function reload() {
  try {
    links.value = await friendLinkApi.list();
  } catch (e) {
    notifyError(e);
  }
}

async function create() {
  creating.value = true;
  try {
    const l = await friendLinkApi.create();
    url.value = `${window.location.origin}/friend/${encodeURIComponent(l.token ?? "")}`;
    expires.value = l.expires_at;
    await reload();
  } catch (e) {
    notifyError(e);
  } finally {
    creating.value = false;
  }
}

/** Feuille de partage du téléphone (messagerie…) ; une annulation n'est pas une erreur. */
async function share() {
  try {
    await navigator.share({
      text: t("friends.links.shareText"),
      url: url.value,
    });
  } catch {
    // partage annulé
  }
}

async function revoke(l: FriendLink) {
  try {
    await friendLinkApi.remove(l.id);
    notifySuccess(t("friends.links.revoked"));
    await reload();
  } catch (e) {
    notifyError(e);
  }
}

onMounted(reload);
</script>
