<template>
  <!-- « Bien reçu » (P7-4b) : les exemplaires offerts quittent la liste, vers une autre de mes collections qui
       accepte l'item (attributs du nouveau lot), ou simplement retirés si je l'ai déjà ajouté à la main. -->
  <ElDialog
    v-model="visible"
    :title="t('gifts.owner.receiveTitle', { label })"
    width="36rem"
    @open="open"
  >
    <div v-if="gift" class="flex flex-col gap-4">
      <p class="text-sm text-muted">
        {{ t("gifts.owner.receiveHelp", { n: gift.quantity }, gift.quantity) }}
      </p>
      <AppSpinner v-if="loading" />
      <ElRadioGroup
        v-else
        v-model="target"
        class="flex! flex-col! items-start! gap-2"
        data-test="receive-target"
      >
        <ElRadio
          v-for="c in targets"
          :key="c.id"
          :value="c.id"
          :data-test="`receive-into-${c.id}`"
          >{{ t("gifts.owner.moveTo") }} :
          <strong>{{ c.name }}</strong>
          <span class="font-mono text-[11px] uppercase text-muted">
            · {{ typeTitle(c.collection_type, lang) }}</span
          ></ElRadio
        >
        <p v-if="targets.length === 0" class="text-xs text-muted">
          {{ t("gifts.owner.noTarget") }}
        </p>
        <ElRadio :value="REMOVE_ONLY" data-test="receive-remove-only">{{
          t("gifts.owner.removeOnly")
        }}</ElRadio>
      </ElRadioGroup>
      <div v-if="targetSchema">
        <h4 class="mb-2 text-sm font-medium">
          {{ t("gifts.owner.attributes") }}
        </h4>
        <DynamicForm
          :key="target"
          ref="formRef"
          v-model="attributes"
          :schema="targetSchema"
        />
      </div>
    </div>
    <template #footer>
      <ElButton text @click="visible = false">{{
        t("common.cancel")
      }}</ElButton>
      <ElButton
        type="primary"
        :loading="saving"
        :disabled="loading"
        data-test="receive-submit"
        @click="submit"
        >{{ t("gifts.owner.submit") }}</ElButton
      >
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { collectionApi, giftApi } from "@/api/client";
import { useLang } from "@/composables/useLang";
import { useNotify } from "@/composables/useNotify";
import { acceptedTypeIds } from "@/lib/collections";
import { isLendable, typeTitle, type Schema } from "@/lib/schema";
import AppSpinner from "@/components/common/AppSpinner.vue";
import DynamicForm from "@/components/common/DynamicForm.vue";
import type { Collection, CollectionItem, GiftReservation } from "@/api/types";

/** Choix « le retirer seulement » (aucune collection de destination). */
const REMOVE_ONLY = "";

const props = defineProps<{
  /** La liste de souhaits. */
  collectionId: string;
  lot: CollectionItem | null;
  gift: GiftReservation | null;
  label: string;
}>();
const emit = defineEmits<{ done: [] }>();
const visible = defineModel<boolean>("visible", { required: true });

const { t } = useI18n();
const { lang } = useLang();
const { notifyError, notifySuccess } = useNotify();

const collections = ref<Collection[]>([]);
const loading = ref(false);
const target = ref<string>(REMOVE_ONLY);
const attributes = ref<Record<string, unknown>>({});
const formRef = ref<InstanceType<typeof DynamicForm> | null>(null);
const saving = ref(false);

/** Mes autres collections qui acceptent l'item ; celles qui se prêtent (une possession) d'abord. */
const targets = computed(() => {
  const typeId = props.lot?.item?.item_type_id;
  return collections.value
    .filter((c) => c.id !== props.collectionId)
    .filter((c) => {
      const accepted = acceptedTypeIds(c);
      return !typeId || accepted.length === 0 || accepted.includes(typeId);
    })
    .sort(
      (a, b) =>
        Number(isLendable(b.collection_type?.attributes_schema)) -
        Number(isLendable(a.collection_type?.attributes_schema)),
    );
});
const targetSchema = computed(
  () =>
    collections.value.find((c) => c.id === target.value)?.collection_type
      ?.attributes_schema as Schema | undefined,
);

async function open() {
  attributes.value = {};
  loading.value = true;
  try {
    collections.value = (await collectionApi.list({ per_page: 100 })).data;
    target.value = targets.value[0]?.id ?? REMOVE_ONLY;
  } catch (e) {
    notifyError(e);
  } finally {
    loading.value = false;
  }
}

async function submit() {
  if (!props.gift) return;
  if (formRef.value && !formRef.value.validate()) return;
  saving.value = true;
  try {
    await giftApi.receive(
      props.collectionId,
      props.gift.id,
      target.value === REMOVE_ONLY
        ? {}
        : { collection_id: target.value, attributes: attributes.value },
    );
    notifySuccess(t("gifts.owner.received"));
    visible.value = false;
    emit("done");
  } catch (e) {
    notifyError(e);
  } finally {
    saving.value = false;
  }
}
</script>
