<template>
  <section class="flex flex-col gap-3" data-test="share-links">
    <div class="flex flex-col gap-1">
      <h3 class="font-bold">{{ t("sharing.share.links.title") }}</h3>
      <p class="text-sm text-muted">{{ t("sharing.share.links.help") }}</p>
    </div>
    <ul v-if="links.length > 0" class="flex flex-col gap-1.5">
      <li
        v-for="l in links"
        :key="l.id"
        class="flex items-center justify-between gap-3 rounded-xl bg-soft px-3 py-1.5 text-sm"
      >
        <span>{{
          t("sharing.share.links.created", { date: date(l.created_at) })
        }}</span>
        <ElButton
          text
          type="danger"
          :aria-label="
            t('sharing.share.links.revoke', { date: date(l.created_at) })
          "
          @click="revoke(l)"
        >
          <i class="pi pi-trash" aria-hidden="true"></i>
        </ElButton>
      </li>
    </ul>
    <p v-else class="text-sm text-muted">
      {{ t("sharing.share.links.none") }}
    </p>
    <ElButton
      class="self-start"
      :loading="creating"
      data-test="create-share-link"
      @click="create"
    >
      <i class="pi pi-link mr-2" aria-hidden="true"></i>
      {{ t("sharing.share.links.create") }}
    </ElButton>
    <OneTimeLinkDialog
      :url="link"
      :title="t('sharing.share.links.linkTitle')"
      @close="link = ''"
    />
  </section>
</template>

<script setup lang="ts">
// Liens de partage secrets d'une collection (P4-2) : le lien n'est montré qu'à sa création.
import { onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { shareLinkApi } from "@/api/client";
import { useLang } from "@/composables/useLang";
import { useNotify } from "@/composables/useNotify";
import { useConfirmDialog } from "@/composables/useConfirmDialog";
import OneTimeLinkDialog from "@/components/account/OneTimeLinkDialog.vue";
import type { ShareLink } from "@/api/types";

const props = defineProps<{ collectionId: string }>();
const { t } = useI18n();
const { lang } = useLang();
const { notifyError } = useNotify();
const { confirm } = useConfirmDialog();

const links = ref<ShareLink[]>([]);
const creating = ref(false);
const link = ref("");

function date(iso: string): string {
  return new Date(iso).toLocaleDateString(lang.value, { dateStyle: "medium" });
}

/** Lien construit ici : le serveur ne connaît que le hash du jeton. */
function shareURL(token: string): string {
  return `${window.location.origin}/share?token=${encodeURIComponent(token)}`;
}

async function load() {
  try {
    links.value = await shareLinkApi.list(props.collectionId);
  } catch (e) {
    notifyError(e);
  }
}

async function create() {
  creating.value = true;
  try {
    const res = await shareLinkApi.create(props.collectionId);
    link.value = shareURL(res.token);
    await load();
  } catch (e) {
    notifyError(e);
  } finally {
    creating.value = false;
  }
}

async function revoke(l: ShareLink) {
  const ok = await confirm({
    message: t("sharing.share.links.revokeConfirm"),
    danger: true,
  });
  if (!ok) return;
  try {
    await shareLinkApi.revoke(props.collectionId, l.id);
    await load();
  } catch (e) {
    notifyError(e);
  }
}

onMounted(load);
</script>
