<template>
  <section class="flex flex-col gap-4 rounded-2xl bg-surface p-5 shadow-card">
    <div class="flex flex-col gap-1">
      <h2 class="text-lg font-extrabold tracking-tight">
        {{ t("account.invitations.title") }}
      </h2>
      <p class="text-sm text-muted">{{ t("account.invitations.help") }}</p>
    </div>

    <form
      class="flex flex-wrap items-end gap-3"
      data-test="invitation-form"
      @submit.prevent="create"
    >
      <div v-if="admin" class="flex flex-col gap-1.5">
        <label :for="`${id}-role`" :class="labelClass">{{
          t("account.role")
        }}</label>
        <ElSelect :id="`${id}-role`" v-model="role" class="w-36!">
          <ElOption :label="t('account.roles.user')" value="user" />
          <ElOption :label="t('account.roles.admin')" value="admin" />
        </ElSelect>
      </div>
      <div class="flex flex-col gap-1.5">
        <label :for="`${id}-days`" :class="labelClass">{{
          t("account.invitations.days")
        }}</label>
        <ElInputNumber
          :id="`${id}-days`"
          v-model="days"
          :min="1"
          :max="90"
          controls-position="right"
          class="w-28!"
        />
      </div>
      <div class="flex min-w-48 flex-1 flex-col gap-1.5">
        <label :for="`${id}-note`" :class="labelClass">{{
          t("account.invitations.note")
        }}</label>
        <ElInput
          :id="`${id}-note`"
          v-model="note"
          maxlength="255"
          :placeholder="t('account.invitations.notePlaceholder')"
        />
      </div>
      <ElButton type="primary" native-type="submit" :loading="creating">
        <i class="pi pi-send mr-2" aria-hidden="true"></i>
        {{ t("account.invitations.create") }}
      </ElButton>
    </form>

    <ElTable
      :data="list"
      size="small"
      :empty-text="t('account.invitations.none')"
      data-test="invitations"
    >
      <ElTableColumn :label="t('account.invitations.note')" min-width="160">
        <template #default="{ row }">
          {{ (row as Invitation).note || "—" }}
          <span v-if="admin" class="block text-xs text-muted">{{
            t("account.invitations.by", {
              name: (row as Invitation).created_by_username || "?",
            })
          }}</span>
        </template>
      </ElTableColumn>
      <ElTableColumn :label="t('account.role')" width="110">
        <template #default="{ row }">
          {{ t(`account.roles.${(row as Invitation).role}`) }}
        </template>
      </ElTableColumn>
      <ElTableColumn :label="t('account.invitations.status')" min-width="170">
        <template #default="{ row }">
          <span
            class="rounded-full px-2 py-0.5 text-xs"
            :class="badge(row as Invitation)"
            >{{ statusText(row as Invitation) }}</span
          >
        </template>
      </ElTableColumn>
      <ElTableColumn width="60" align="center">
        <template #default="{ row }">
          <ElButton
            v-if="(row as Invitation).status !== 'used'"
            text
            :aria-label="t('account.invitations.withdraw')"
            @click="withdraw(row as Invitation)"
          >
            <i class="pi pi-trash" aria-hidden="true"></i>
          </ElButton>
        </template>
      </ElTableColumn>
    </ElTable>

    <OneTimeLinkDialog
      :url="link"
      :title="t('account.invitations.linkTitle')"
      :expires-at="linkExpires"
      @close="link = ''"
    />
  </section>
</template>

<script setup lang="ts">
// Invitations à créer un compte (P4-0) : création (lien montré une fois), liste, retrait.
import { onMounted, ref, useId } from "vue";
import { useI18n } from "vue-i18n";
import { invitationApi } from "@/api/client";
import { useLang } from "@/composables/useLang";
import { useNotify } from "@/composables/useNotify";
import { useConfirmDialog } from "@/composables/useConfirmDialog";
import OneTimeLinkDialog from "./OneTimeLinkDialog.vue";
import type { Invitation } from "@/api/types";

/** admin : choix du rôle, toutes les invitations (avec leur auteur). */
defineProps<{ admin?: boolean }>();
const { t } = useI18n();
const { lang } = useLang();
const { notifyError } = useNotify();
const { confirm } = useConfirmDialog();
const id = useId();
const labelClass = "font-mono text-[11px] uppercase tracking-wider text-muted";

const list = ref<Invitation[]>([]);
const role = ref<"user" | "admin">("user");
const days = ref(7);
const note = ref("");
const creating = ref(false);
const link = ref("");
const linkExpires = ref<string | undefined>();

/** Lien d'inscription : construit ici, le serveur ne connaît que le hash du jeton. */
function registerURL(token: string): string {
  return `${window.location.origin}/register?token=${encodeURIComponent(token)}`;
}

function date(iso: string): string {
  return new Date(iso).toLocaleDateString(lang.value, { dateStyle: "medium" });
}

function statusText(inv: Invitation): string {
  if (inv.status === "used")
    return t("account.invitations.used", { name: inv.used_by_username || "?" });
  if (inv.status === "expired") return t("account.invitations.expired");
  return t("account.invitations.pending", { date: date(inv.expires_at) });
}

function badge(inv: Invitation): string {
  if (inv.status === "used") return "bg-tint text-tint-ink";
  if (inv.status === "expired") return "bg-soft text-muted";
  return "bg-accent text-accent-ink";
}

async function load() {
  try {
    list.value = await invitationApi.list();
  } catch (e) {
    notifyError(e);
  }
}

async function create() {
  creating.value = true;
  try {
    const res = await invitationApi.create({
      role: role.value,
      expires_in_days: days.value,
      note: note.value.trim() || undefined,
    });
    link.value = registerURL(res.token);
    linkExpires.value = res.invitation.expires_at;
    note.value = "";
    await load();
  } catch (e) {
    notifyError(e);
  } finally {
    creating.value = false;
  }
}

async function withdraw(inv: Invitation) {
  if (
    !(await confirm({
      message: t("account.invitations.withdrawConfirm"),
      danger: true,
    }))
  )
    return;
  try {
    await invitationApi.delete(inv.id);
    await load();
  } catch (e) {
    notifyError(e);
  }
}

onMounted(load);
</script>
