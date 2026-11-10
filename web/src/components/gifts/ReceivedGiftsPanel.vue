<template>
  <!-- Onglet « Reçus » (P7-4c) : les cadeaux remis à confirmer, toutes listes confondues, puis l'historique des cadeaux
       reçus. Un cadeau reçu ne se supprime pas. -->
  <div class="flex flex-col gap-4" data-test="received-gifts">
    <AppSpinner
      v-if="
        loading && toConfirm.length === 0 && received.data.value.length === 0
      "
    />
    <template v-else>
      <section v-if="toConfirm.length" class="flex flex-col gap-2">
        <h2 class="text-lg font-extrabold tracking-tight">
          {{ t("gifts.received.toConfirm") }}
        </h2>
        <ul
          class="flex flex-col gap-2 rounded-2xl bg-surface p-3 shadow-card"
          data-test="gifts-to-confirm"
        >
          <ReceivedGiftCard
            v-for="g in toConfirm"
            :key="g.id"
            :gift="g"
            @receive="openReceive"
            @decline="onDecline"
          />
        </ul>
      </section>
      <p
        v-if="received.data.value.length === 0"
        class="rounded-2xl bg-surface p-5 text-muted shadow-card"
        data-test="received-none"
      >
        {{ t("gifts.received.none") }}
      </p>
      <ul
        v-else
        class="flex flex-col gap-2 rounded-2xl bg-surface p-3 shadow-card"
        data-test="gifts-received"
      >
        <ReceivedGiftCard
          v-for="g in received.data.value"
          :key="g.id"
          :gift="g"
        />
      </ul>
      <ElPagination
        class="justify-end"
        hide-on-single-page
        layout="prev, pager, next"
        :total="received.total.value"
        :page-size="received.perPage.value"
        :current-page="received.page.value"
        @current-change="onPage"
      />
    </template>

    <ReceiveGiftDialog
      v-model:visible="receiveVisible"
      :collection-id="target?.collection_id ?? ''"
      :item-type-id="target?.item?.item_type_id"
      :gift="target"
      :label="target?.item ? itemLabel(target.item, lang) : ''"
      @done="reload"
    />
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { giftApi } from "@/api/client";
import { useGiftDecline } from "@/composables/useGiftDecline";
import { useLang } from "@/composables/useLang";
import { useNotify } from "@/composables/useNotify";
import { usePagination } from "@/composables/usePagination";
import { itemLabel } from "@/lib/schema";
import AppSpinner from "@/components/common/AppSpinner.vue";
import ReceiveGiftDialog from "./ReceiveGiftDialog.vue";
import ReceivedGiftCard from "./ReceivedGiftCard.vue";
import type { GiftReservation } from "@/api/types";

const { t } = useI18n();
const { lang } = useLang();
const { notifyError } = useNotify();
const { decline } = useGiftDecline();

/** Remises à confirmer : peu nombreuses, chargées d'un coup. */
const toConfirm = ref<GiftReservation[]>([]);
const received = usePagination((params: Record<string, unknown>) =>
  giftApi.received({ ...params, status: "received" }),
);
const loading = ref(false);

async function reload() {
  loading.value = true;
  try {
    const [pending] = await Promise.all([
      giftApi.received({ status: "given", per_page: 100 }),
      received.load(),
    ]);
    toConfirm.value = pending.data;
  } catch (e) {
    notifyError(e);
  } finally {
    loading.value = false;
  }
}

const receiveVisible = ref(false);
const target = ref<GiftReservation | null>(null);
function openReceive(g: GiftReservation) {
  target.value = g;
  receiveVisible.value = true;
}

async function onDecline(g: GiftReservation) {
  if (g.collection_id && (await decline(g.collection_id, g))) await reload();
}

function onPage(p: number) {
  received.goToPage(p);
  received.load().catch(notifyError);
}

onMounted(reload);
</script>
