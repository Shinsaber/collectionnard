<template>
  <div class="flex max-w-5xl flex-col gap-6">
    <PageHeader :title="t('account.users.title')">
      <template #subtitle>
        <p class="text-[15px] text-muted">{{ t("account.users.help") }}</p>
      </template>
    </PageHeader>

    <section class="rounded-2xl bg-surface p-5 shadow-card">
      <ElTable v-loading="loading" :data="users" size="small" data-test="users">
        <ElTableColumn :label="t('account.users.account')" min-width="220">
          <template #default="{ row }">
            <div class="flex items-center gap-3">
              <UserAvatar :user="row as User" :size="32" />
              <div class="flex min-w-0 flex-col">
                <span class="truncate font-semibold">{{
                  (row as User).display_name || (row as User).username
                }}</span>
                <span class="truncate font-mono text-xs text-muted"
                  >{{ (row as User).username }} ·
                  {{ (row as User).email }}</span
                >
              </div>
            </div>
          </template>
        </ElTableColumn>
        <ElTableColumn :label="t('account.role')" width="150">
          <template #default="{ row }">
            <ElSelect
              :model-value="(row as User).role"
              size="small"
              :disabled="isSelf(row as User)"
              :aria-label="
                t('account.users.roleOf', { name: (row as User).username })
              "
              @update:model-value="
                (v: string) => update(row as User, { role: v as User['role'] })
              "
            >
              <ElOption :label="t('account.roles.user')" value="user" />
              <ElOption
                :label="t('account.roles.contributor')"
                value="contributor"
              />
              <ElOption :label="t('account.roles.admin')" value="admin" />
            </ElSelect>
          </template>
        </ElTableColumn>
        <ElTableColumn :label="t('account.users.active')" width="110">
          <template #default="{ row }">
            <ElSwitch
              :model-value="!(row as User).disabled_at"
              :disabled="isSelf(row as User)"
              :aria-label="
                t('account.users.activeOf', { name: (row as User).username })
              "
              @update:model-value="
                (v: string | number | boolean) =>
                  update(row as User, { disabled: !v })
              "
            />
          </template>
        </ElTableColumn>
        <ElTableColumn width="190" align="right">
          <template #default="{ row }">
            <ElButton size="small" @click="resetLink(row as User)">
              <i class="pi pi-key mr-2" aria-hidden="true"></i>
              {{ t("account.users.resetLink") }}
            </ElButton>
          </template>
        </ElTableColumn>
      </ElTable>
      <ElPagination
        v-if="total > perPage"
        class="mt-4"
        layout="prev, pager, next"
        :total="total"
        :page-size="perPage"
        :current-page="page"
        @current-change="(p: number) => load(p)"
      />
    </section>

    <InvitationsCard admin />

    <OneTimeLinkDialog
      :url="link"
      :title="t('account.users.resetLinkTitle', { name: linkFor })"
      :expires-at="linkExpires"
      @close="link = ''"
    />
  </div>
</template>

<script setup lang="ts">
// Comptes de l'instance (admin, P4-0) : rôle, activation, lien de réinitialisation ; invitations.
import { onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { userApi } from "@/api/client";
import { useAuthStore } from "@/stores/auth";
import { useNotify } from "@/composables/useNotify";
import { useConfirmDialog } from "@/composables/useConfirmDialog";
import PageHeader from "@/components/common/PageHeader.vue";
import InvitationsCard from "@/components/account/InvitationsCard.vue";
import OneTimeLinkDialog from "@/components/account/OneTimeLinkDialog.vue";
import UserAvatar from "@/components/account/UserAvatar.vue";
import type { User, UserUpdate } from "@/api/types";

const { t } = useI18n();
const auth = useAuthStore();
const { notifyError, notifySuccess } = useNotify();
const { confirm } = useConfirmDialog();

const users = ref<User[]>([]);
const total = ref(0);
const page = ref(1);
const perPage = 50;
const loading = ref(true);
const link = ref("");
const linkFor = ref("");
const linkExpires = ref<string | undefined>();

/** Un admin ne peut ni se désactiver ni se retirer le rôle admin (le serveur le refuse aussi). */
function isSelf(u: User): boolean {
  return u.id === auth.user?.id;
}

async function load(p = page.value) {
  loading.value = true;
  try {
    const res = await userApi.list({ page: p, per_page: perPage });
    users.value = res.data;
    total.value = res.total;
    page.value = p;
  } catch (e) {
    notifyError(e);
  } finally {
    loading.value = false;
  }
}

async function update(u: User, change: UserUpdate) {
  if (
    change.disabled &&
    !(await confirm({
      message: t("account.users.disableConfirm", { name: u.username }),
      danger: true,
    }))
  )
    return;
  try {
    const updated = await userApi.update(u.id, change);
    users.value = users.value.map((x) => (x.id === u.id ? updated : x));
    notifySuccess(t("account.users.saved", { name: u.username }));
  } catch (e) {
    notifyError(e);
  }
}

async function resetLink(u: User) {
  try {
    const res = await userApi.resetLink(u.id);
    linkFor.value = u.username;
    linkExpires.value = res.expires_at;
    link.value = `${window.location.origin}/reset-password?token=${encodeURIComponent(res.token)}`;
  } catch (e) {
    notifyError(e);
  }
}

onMounted(() => load(1));
</script>
