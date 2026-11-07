<template>
  <ElDropdown trigger="click" @command="onCommand">
    <button
      type="button"
      class="flex h-11 items-center gap-2 rounded-[10px] bg-soft px-4 text-[15px] font-semibold text-ink"
      :aria-label="t('items.detail.moreActions')"
      data-test="item-more"
    >
      <span class="max-sm:hidden">{{ t("items.detail.more") }}</span>
      <!-- Les classes responsives sur un <span> : la feuille de primeicons force le display des <i>. -->
      <span class="max-sm:hidden" aria-hidden="true"
        ><i class="pi pi-chevron-down text-xs"></i
      ></span>
      <span class="sm:hidden" aria-hidden="true"
        ><i class="pi pi-ellipsis-h"></i
      ></span>
    </button>
    <template #dropdown>
      <ElDropdownMenu>
        <ElDropdownItem command="edit" data-test="more-edit">
          <i class="pi pi-pencil mr-2" aria-hidden="true"></i>
          {{
            auth.canEditCatalog
              ? t("items.detail.editItem")
              : t("proposals.suggest")
          }}
        </ElDropdownItem>
        <ElDropdownItem command="related">
          <i class="pi pi-list mr-2" aria-hidden="true"></i>
          {{ t("items.search.relatedItems") }}
        </ElDropdownItem>
        <ElDropdownItem command="graph">
          <NavIcon name="graph" :size="16" class="mr-2" />
          {{ t("items.detail.graph") }}
        </ElDropdownItem>
        <ElDropdownItem command="recommend" data-test="more-recommend">
          <NavIcon name="recommendations" :size="16" class="mr-2" />
          {{ t("recommendations.send.action") }}
        </ElDropdownItem>
        <ElDropdownItem
          v-if="!auth.isAdmin"
          command="report"
          divided
          data-test="more-report"
        >
          <i class="pi pi-flag mr-2" aria-hidden="true"></i>
          {{ t("proposals.report.title") }}
        </ElDropdownItem>
      </ElDropdownMenu>
    </template>
  </ElDropdown>
  <TextProposalDialog
    v-model:visible="reportOpen"
    kind="report"
    :item-id="itemId"
    :item-label="itemLabel"
  />
  <RecommendDialog v-model:visible="recommendOpen" :item-id="itemId" />
</template>

<script setup lang="ts">
// Menu « Plus » d'une fiche : modifier (ou suggérer une correction pour un membre, P4-3b),
// items liés, graphe, recommander à un membre (I-4), signaler (P4-3c ; l'admin corrige ou supprime lui-même).
import { ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { useAuthStore } from "@/stores/auth";
import NavIcon from "@/components/layout/NavIcon.vue";
import TextProposalDialog from "@/components/proposals/TextProposalDialog.vue";
import RecommendDialog from "./RecommendDialog.vue";

const props = defineProps<{ itemId: string; itemLabel: string }>();
const emit = defineEmits<{ edit: [] }>();
const { t } = useI18n();
const auth = useAuthStore();
const router = useRouter();
const reportOpen = ref(false);
const recommendOpen = ref(false);

function onCommand(cmd: "edit" | "related" | "graph" | "recommend" | "report") {
  if (cmd === "edit") emit("edit");
  else if (cmd === "recommend") recommendOpen.value = true;
  else if (cmd === "related")
    router.push({ path: "/items", query: { related_to: props.itemId } });
  else if (cmd === "graph") router.push(`/items/${props.itemId}/graph`);
  else reportOpen.value = true;
}
</script>
