<template>
  <ElTag closable @close="emit('close')">
    {{
      relationTypeName
        ? t("items.search.relatedVia", {
            label: relatedLabel,
            relation: relationTypeName,
          })
        : t("items.search.relatedTo", { label: relatedLabel })
    }}
  </ElTag>
</template>

<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { itemApi } from "@/api/client";
import type { Item } from "@/api/types";
import { useLang } from "@/composables/useLang";
import { useNotify } from "@/composables/useNotify";
import { useMetaTypes } from "@/stores/metaTypes";
import { itemLabel, typeTitle } from "@/lib/schema";

// Pastille du filtre `related_to` du catalogue : « Liés à <item> (via <relation>) ».
const props = defineProps<{ itemId: string; relationTypeId?: string }>();
const emit = defineEmits<{ close: [] }>();

const { t } = useI18n();
const { lang } = useLang();
const { notifyError } = useNotify();
const metaTypes = useMetaTypes();

const item = ref<Item | null>(null);
watch(
  () => props.itemId,
  async (id) => {
    item.value = null;
    try {
      item.value = await itemApi.get(id);
    } catch (e) {
      notifyError(e, t("items.search.relatedUnavailable"));
    }
  },
  { immediate: true },
);
watch(
  () => props.relationTypeId,
  (id) => {
    if (id) metaTypes.loadRelationTypes().catch((e) => notifyError(e));
  },
  { immediate: true },
);

const relatedLabel = computed(() =>
  item.value ? itemLabel(item.value, lang.value) : props.itemId.slice(0, 8),
);
const relationTypeName = computed(() => {
  const id = props.relationTypeId;
  if (!id) return "";
  const rt = metaTypes.relationTypes.find((r) => r.id === id);
  return rt ? typeTitle(rt, lang.value) : id.slice(0, 8);
});
</script>
