<template>
  <div class="flex flex-col gap-6">
    <PageHeader :title="t('sharing.members.title')">
      <template #subtitle>
        <p class="text-[15px] text-muted">{{ t("sharing.members.help") }}</p>
      </template>
    </PageHeader>
    <AppSpinner
      v-if="members.loading.value && members.data.value.length === 0"
    />
    <p
      v-else-if="members.data.value.length === 0"
      class="rounded-2xl bg-surface p-5 text-muted shadow-card"
    >
      {{ t("sharing.members.none") }}
    </p>
    <ul v-else class="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
      <li v-for="m in members.data.value" :key="m.username">
        <RouterLink
          :to="`/u/${m.username}`"
          class="flex h-full items-center gap-3 rounded-2xl bg-surface p-4 shadow-card hover:shadow-raised"
          data-test="member"
        >
          <UserAvatar :user="m" :size="48" />
          <span class="flex min-w-0 flex-col">
            <span class="truncate font-bold text-ink">{{
              m.display_name || m.username
            }}</span>
            <span class="truncate font-mono text-xs text-muted"
              >@{{ m.username }}</span
            >
            <span v-if="m.bio" class="line-clamp-1 text-sm text-muted">{{
              m.bio
            }}</span>
          </span>
        </RouterLink>
      </li>
    </ul>
    <ElPagination
      class="justify-end"
      hide-on-single-page
      layout="prev, pager, next"
      :total="members.total.value"
      :page-size="members.perPage.value"
      :current-page="members.page.value"
      @current-change="onPage"
    />
  </div>
</template>

<script setup lang="ts">
// Annuaire des membres (P4-2) : profils ouverts aux membres ou publics.
import { onMounted } from "vue";
import { useI18n } from "vue-i18n";
import { memberApi } from "@/api/client";
import { useNotify } from "@/composables/useNotify";
import { usePagination } from "@/composables/usePagination";
import AppSpinner from "@/components/common/AppSpinner.vue";
import PageHeader from "@/components/common/PageHeader.vue";
import UserAvatar from "@/components/account/UserAvatar.vue";

const { t } = useI18n();
const { notifyError } = useNotify();
const members = usePagination((params: Record<string, unknown>) =>
  memberApi.list(params),
);

function onPage(p: number) {
  members.goToPage(p);
  members.load().catch((e) => notifyError(e));
}

onMounted(() => members.load().catch((e) => notifyError(e)));
</script>
