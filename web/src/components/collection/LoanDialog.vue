<template>
  <!-- Prêter des exemplaires d'un lot (P7-3) : à un ami (notifié) ou à un simple nom. -->
  <ElDialog
    v-model="visible"
    :title="t('loans.lendTitle', { label })"
    width="32rem"
    @open="reset"
  >
    <form
      v-if="lot"
      class="flex flex-col gap-4"
      data-test="loan-form"
      @submit.prevent="submit"
    >
      <div class="flex flex-col gap-1.5">
        <span class="text-sm font-medium">{{ t("loans.borrower") }}</span>
        <ElRadioGroup v-model="mode" data-test="loan-mode">
          <ElRadio value="friend">{{ t("loans.friend") }}</ElRadio>
          <ElRadio value="name">{{ t("loans.name") }}</ElRadio>
        </ElRadioGroup>
        <template v-if="mode === 'friend'">
          <ElSelect
            v-model="friend"
            filterable
            :placeholder="t('loans.friendPlaceholder')"
            :aria-label="t('loans.friend')"
            :loading="loadingFriends"
            data-test="loan-friend"
          >
            <ElOption
              v-for="f in friends"
              :key="f.user.username"
              :label="f.user.display_name || f.user.username"
              :value="f.user.username"
            />
          </ElSelect>
          <p class="text-xs text-muted">
            {{
              !loadingFriends && friends.length === 0
                ? t("loans.noFriends")
                : t("loans.friendHelp")
            }}
          </p>
        </template>
        <template v-else>
          <ElInput
            v-model="name"
            maxlength="100"
            :placeholder="t('loans.namePlaceholder')"
            :aria-label="t('loans.name')"
            data-test="loan-name"
          />
          <p class="text-xs text-muted">{{ t("loans.nameHelp") }}</p>
        </template>
      </div>

      <div class="flex flex-wrap items-center gap-3">
        <label :for="`${uid}-qty`" class="text-sm font-medium">{{
          t("loans.quantity")
        }}</label>
        <ElInputNumber
          :id="`${uid}-qty`"
          v-model="quantity"
          :min="1"
          :max="Math.max(1, available)"
          size="small"
          data-test="loan-quantity"
        />
        <span class="text-xs text-muted">{{
          t("loans.available", available)
        }}</span>
      </div>

      <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <div class="flex flex-col gap-1">
          <label :for="`${uid}-lent`" class="text-sm font-medium">{{
            t("loans.lentAt")
          }}</label>
          <ElInput
            :id="`${uid}-lent`"
            v-model="lentAt"
            type="date"
            :max="today()"
          />
        </div>
        <div class="flex flex-col gap-1">
          <label :for="`${uid}-due`" class="text-sm font-medium">{{
            t("loans.dueAt")
          }}</label>
          <ElInput
            :id="`${uid}-due`"
            v-model="dueAt"
            type="date"
            :min="lentAt"
            data-test="loan-due"
          />
        </div>
      </div>

      <div class="flex flex-col gap-1">
        <label :for="`${uid}-note`" class="text-sm font-medium">{{
          t("loans.note")
        }}</label>
        <ElInput
          :id="`${uid}-note`"
          v-model="note"
          type="textarea"
          :rows="2"
          maxlength="500"
        />
      </div>
    </form>
    <template #footer>
      <ElButton text @click="visible = false">{{
        t("common.cancel")
      }}</ElButton>
      <ElButton
        type="primary"
        :loading="saving"
        :disabled="!ready"
        data-test="loan-submit"
        @click="submit"
        >{{ t("loans.submit") }}</ElButton
      >
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
import { computed, ref, useId } from "vue";
import { useI18n } from "vue-i18n";
import { friendApi, loanApi } from "@/api/client";
import { useNotify } from "@/composables/useNotify";
import { availableCopies, today } from "@/lib/loans";
import type { CollectionItem, Friendship } from "@/api/types";

const props = defineProps<{
  collectionId: string;
  lot: CollectionItem | null;
  /** Libellé de l'item, pour le titre. */
  label: string;
}>();
const emit = defineEmits<{ done: [] }>();
const visible = defineModel<boolean>("visible", { required: true });

const { t } = useI18n();
const { notifyError, notifySuccess } = useNotify();
const uid = useId();

const mode = ref<"friend" | "name">("friend");
const friends = ref<Friendship[]>([]);
const loadingFriends = ref(false);
const friend = ref("");
const name = ref("");
const quantity = ref(1);
const lentAt = ref(today());
const dueAt = ref("");
const note = ref("");
const saving = ref(false);

const available = computed(() => (props.lot ? availableCopies(props.lot) : 0));
const ready = computed(
  () =>
    available.value > 0 &&
    !!lentAt.value &&
    (mode.value === "friend" ? !!friend.value : !!name.value.trim()),
);

async function reset() {
  friend.value = name.value = note.value = "";
  quantity.value = 1;
  lentAt.value = today();
  dueAt.value = "";
  loadingFriends.value = true;
  try {
    friends.value = (await friendApi.list({ per_page: 100 })).data;
    mode.value = friends.value.length ? "friend" : "name";
  } catch (e) {
    notifyError(e);
  } finally {
    loadingFriends.value = false;
  }
}

async function submit() {
  if (!props.lot || !ready.value) return;
  saving.value = true;
  try {
    await loanApi.create(props.collectionId, props.lot.id, {
      quantity: quantity.value,
      ...(mode.value === "friend"
        ? { borrower_username: friend.value }
        : { borrower_name: name.value.trim() }),
      lent_at: lentAt.value,
      ...(dueAt.value ? { due_at: dueAt.value } : {}),
      note: note.value.trim(),
    });
    notifySuccess(t("loans.created"));
    visible.value = false;
    emit("done");
  } catch (e) {
    notifyError(e);
  } finally {
    saving.value = false;
  }
}
</script>
