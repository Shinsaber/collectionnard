<template>
  <!-- Réserver un cadeau sur la liste d'un ami (P7-4), ou modifier / annuler ma réservation. -->
  <ElDialog
    v-model="visible"
    :title="t('gifts.reserveTitle', { label })"
    width="30rem"
    @open="reset"
  >
    <div v-if="lot" class="flex flex-col gap-4">
      <p class="text-sm text-muted">{{ t("gifts.help") }}</p>
      <div class="flex flex-wrap items-center gap-3">
        <label :for="`${uid}-qty`" class="text-sm font-medium">{{
          t("gifts.quantity")
        }}</label>
        <ElInputNumber
          :id="`${uid}-qty`"
          v-model="quantity"
          :min="1"
          :max="Math.max(1, left)"
          size="small"
          data-test="reserve-quantity"
        />
        <span class="text-xs text-muted">{{ t("gifts.left", left) }}</span>
      </div>
      <div class="flex flex-col gap-1">
        <label :for="`${uid}-until`" class="text-sm font-medium">{{
          t("gifts.until")
        }}</label>
        <ElInput
          :id="`${uid}-until`"
          v-model="expiresAt"
          type="date"
          :min="today()"
          :max="maxExpiry()"
          data-test="reserve-until"
        />
        <p
          v-if="mine?.expired && expiresAt === mine.expires_at"
          class="text-xs text-danger"
        >
          {{
            t("gifts.expiredHelp", { date: formatDay(mine.expires_at, lang) })
          }}
        </p>
        <p v-else class="text-xs text-muted">{{ t("gifts.untilHelp") }}</p>
      </div>
      <div class="flex flex-col gap-1">
        <label :for="`${uid}-note`" class="text-sm font-medium">{{
          t("gifts.note")
        }}</label>
        <ElInput
          :id="`${uid}-note`"
          v-model="note"
          type="textarea"
          :rows="2"
          maxlength="500"
        />
      </div>
    </div>
    <template #footer>
      <ElButton
        v-if="mine"
        text
        type="danger"
        class="mr-auto"
        data-test="reserve-cancel"
        @click="cancel"
        >{{ t("gifts.cancel") }}</ElButton
      >
      <ElButton text @click="visible = false">{{ t("common.close") }}</ElButton>
      <ElButton
        type="primary"
        :loading="saving"
        :disabled="left < 1 || !expiresAt"
        data-test="reserve-submit"
        @click="submit"
        >{{ mine ? t("gifts.save") : t("gifts.submit") }}</ElButton
      >
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
import { computed, ref, useId } from "vue";
import { useI18n } from "vue-i18n";
import { giftApi } from "@/api/client";
import { useConfirmDialog } from "@/composables/useConfirmDialog";
import { useNotify } from "@/composables/useNotify";
import {
  defaultExpiry,
  maxExpiry,
  myReservation,
  reservableCopies,
} from "@/lib/gifts";
import { formatDay, today } from "@/lib/loans";
import { useLang } from "@/composables/useLang";
import type { CollectionItem } from "@/api/types";

const props = defineProps<{
  collectionId: string;
  lot: CollectionItem | null;
  label: string;
}>();
const emit = defineEmits<{ done: [] }>();
const visible = defineModel<boolean>("visible", { required: true });

const { t } = useI18n();
const { lang } = useLang();
const { notifyError, notifySuccess } = useNotify();
const { confirm } = useConfirmDialog();
const uid = useId();
const quantity = ref(1);
const note = ref("");
const expiresAt = ref(defaultExpiry());
const saving = ref(false);

const mine = computed(() => (props.lot ? myReservation(props.lot) : undefined));
/** Exemplaires que ma réservation peut porter. */
const left = computed(() =>
  props.lot ? reservableCopies(props.lot, mine.value) : 0,
);

function reset() {
  quantity.value = mine.value?.quantity ?? 1;
  note.value = mine.value?.note ?? "";
  expiresAt.value = mine.value?.expires_at ?? defaultExpiry();
}

async function submit() {
  if (!props.lot) return;
  saving.value = true;
  try {
    if (mine.value) {
      await giftApi.update(mine.value.id, {
        quantity: quantity.value,
        status: mine.value.status,
        expires_at: expiresAt.value,
        note: note.value.trim(),
      });
      notifySuccess(t("gifts.saved"));
    } else {
      await giftApi.reserve(props.collectionId, props.lot.id, {
        quantity: quantity.value,
        expires_at: expiresAt.value,
        note: note.value.trim(),
      });
      notifySuccess(t("gifts.reserved"));
    }
    visible.value = false;
    emit("done");
  } catch (e) {
    notifyError(e);
  } finally {
    saving.value = false;
  }
}

async function cancel() {
  if (!mine.value) return;
  const ok = await confirm({
    message: t("gifts.cancelConfirm"),
    confirmText: t("gifts.cancel"),
    danger: true,
  });
  if (!ok) return;
  try {
    await giftApi.cancel(mine.value.id);
    notifySuccess(t("gifts.cancelled"));
    visible.value = false;
    emit("done");
  } catch (e) {
    notifyError(e);
  }
}
</script>
