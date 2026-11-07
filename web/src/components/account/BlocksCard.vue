<template>
  <section
    class="flex flex-col gap-4 rounded-2xl bg-surface p-5 shadow-card"
    data-test="blocks"
  >
    <h2 class="text-lg font-extrabold tracking-tight">
      {{ t("recommendations.blocks.title") }}
    </h2>
    <p class="text-sm text-muted">{{ t("recommendations.blocks.help") }}</p>
    <form class="flex max-w-sm gap-2" @submit.prevent="add">
      <label :for="`${id}-user`" class="sr-only">{{
        t("recommendations.blocks.addPlaceholder")
      }}</label>
      <ElInput
        :id="`${id}-user`"
        v-model="username"
        :placeholder="t('recommendations.blocks.addPlaceholder')"
      />
      <ElButton
        native-type="submit"
        :disabled="!username.trim()"
        :loading="adding"
        data-test="block-add"
        >{{ t("recommendations.blocks.add") }}</ElButton
      >
    </form>
    <p v-if="list.data.value.length === 0" class="text-sm text-muted">
      {{ t("recommendations.blocks.none") }}
    </p>
    <ul v-else class="flex flex-col gap-2">
      <li
        v-for="b in list.data.value"
        :key="b.id"
        class="flex items-center gap-3 rounded-xl bg-soft px-3 py-2"
        data-test="block"
      >
        <UserAvatar :user="b.user ?? null" :size="28" />
        <span class="flex min-w-0 flex-1 flex-col">
          <span class="truncate font-semibold">{{
            b.user
              ? b.user.display_name || b.user.username
              : t("recommendations.blocks.unknown")
          }}</span>
          <span class="font-mono text-[11px] text-muted">{{
            t("recommendations.blocks.since", {
              date: new Date(b.created_at).toLocaleDateString(lang),
            })
          }}</span>
        </span>
        <ElButton text size="small" @click="unblock(b)">{{
          t("recommendations.blocks.unblock")
        }}</ElButton>
      </li>
    </ul>
    <ElPagination
      class="justify-end"
      hide-on-single-page
      layout="prev, pager, next"
      :total="list.total.value"
      :page-size="list.perPage.value"
      :current-page="list.page.value"
      @current-change="onPage"
    />
  </section>
</template>

<script setup lang="ts">
// Membres bloqués (I-4) : bloquer par nom, débloquer. Un membre bloqué depuis un contenu qui ne le nommait pas
// reste « Un membre » : le serveur ne renvoie pas son nom.
import { onMounted, ref, useId } from "vue";
import { useI18n } from "vue-i18n";
import { blockApi } from "@/api/client";
import { useLang } from "@/composables/useLang";
import { useNotify } from "@/composables/useNotify";
import { usePagination } from "@/composables/usePagination";
import UserAvatar from "./UserAvatar.vue";
import type { Block } from "@/api/types";

const { t } = useI18n();
const { lang } = useLang();
const { notifyError, notifySuccess } = useNotify();
const id = useId();
const list = usePagination((params: Record<string, unknown>) =>
  blockApi.list(params),
);
const reload = () => list.load().catch(notifyError);

const username = ref("");
const adding = ref(false);
async function add() {
  adding.value = true;
  try {
    await blockApi.create({ username: username.value.trim() });
    username.value = "";
    notifySuccess(t("recommendations.block.done"));
    await reload();
  } catch (e) {
    notifyError(e);
  } finally {
    adding.value = false;
  }
}

async function unblock(b: Block) {
  try {
    await blockApi.remove(b.id);
    notifySuccess(t("recommendations.blocks.unblocked"));
    await reload();
  } catch (e) {
    notifyError(e);
  }
}

function onPage(p: number) {
  list.goToPage(p);
  reload();
}

onMounted(reload);
</script>
