<template>
  <ElDialog
    :model-value="visible"
    :title="t('sharing.share.title', { name: collection.name })"
    width="36rem"
    @update:model-value="(v: boolean) => emit('update:visible', v)"
  >
    <div class="flex flex-col gap-6">
      <section class="flex flex-col gap-2">
        <h3 class="font-bold">{{ t("sharing.visibility.label") }}</h3>
        <VisibilitySelect v-model="visibility" />
        <p class="text-xs text-muted">
          {{ t("sharing.share.visibilityHelp") }}
        </p>
        <RouterLink
          v-if="collection.visibility !== 'private'"
          :to="`/c/${collection.id}`"
          target="_blank"
          class="text-sm font-semibold text-link"
          data-test="public-page"
        >
          <i class="pi pi-external-link mr-1 text-xs" aria-hidden="true"></i>
          {{ t("sharing.share.openPublic") }}
        </RouterLink>
      </section>

      <section class="flex flex-col gap-2" data-test="hidden-fields">
        <div class="flex flex-col gap-1">
          <h3 class="font-bold">{{ t("sharing.share.hidden.title") }}</h3>
          <p class="text-sm text-muted">{{ t("sharing.share.hidden.help") }}</p>
        </div>
        <p v-if="keys.length === 0" class="text-sm text-muted">
          {{ t("sharing.share.hidden.none") }}
        </p>
        <template v-else>
          <div class="flex flex-wrap gap-x-5 gap-y-1">
            <ElCheckbox
              v-for="k in keys"
              :key="k"
              :model-value="hidden.includes(k)"
              :data-test="`hide-${k}`"
              @update:model-value="toggle(k, Boolean($event))"
              >{{ fieldLabel(schema, k, lang, "pretty") }}</ElCheckbox
            >
          </div>
          <div class="flex flex-wrap items-center gap-3 text-xs text-muted">
            <span data-test="hidden-mode">{{
              custom
                ? t("sharing.share.hidden.custom")
                : t("sharing.share.hidden.byType")
            }}</span>
            <ElButton
              v-if="custom"
              text
              size="small"
              data-test="hidden-reset"
              @click="reset"
              >{{ t("sharing.share.hidden.reset") }}</ElButton
            >
          </div>
        </template>
      </section>

      <ShareLinksCard :collection-id="collection.id" />
    </div>
    <template #footer>
      <ElButton text @click="emit('update:visible', false)">{{
        t("common.cancel")
      }}</ElButton>
      <ElButton
        type="primary"
        :loading="saving"
        data-test="share-save"
        @click="save"
      >
        <i class="pi pi-check mr-2" aria-hidden="true"></i>
        {{ t("common.save") }}
      </ElButton>
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
// Partage d'une de mes collections (P4-2) : visibilité, champs cachés aux autres lecteurs
// (défaut du type `x-private`, ou choix propre à la collection), liens de partage.
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { collectionApi } from "@/api/client";
import { useLang } from "@/composables/useLang";
import { useNotify } from "@/composables/useNotify";
import {
  fieldLabel,
  isImageGallery,
  orderedKeys,
  privateKeys,
  type Schema,
} from "@/lib/schema";
import VisibilitySelect from "./VisibilitySelect.vue";
import ShareLinksCard from "./ShareLinksCard.vue";
import type { Collection, Visibility } from "@/api/types";

const props = defineProps<{ collection: Collection; visible: boolean }>();
const emit = defineEmits<{
  "update:visible": [value: boolean];
  saved: [collection: Collection];
}>();
const { t } = useI18n();
const { lang } = useLang();
const { notifyError, notifySuccess } = useNotify();

const schema = computed(
  () =>
    props.collection.collection_type?.attributes_schema as Schema | undefined,
);
const keys = computed(() =>
  orderedKeys(schema.value, {}).filter((k) => !isImageGallery(schema.value, k)),
);

const visibility = ref<Visibility>("private");
const hidden = ref<string[]>([]);
/** Choix propre à la collection (sinon le réglage du type, envoyé comme `null`). */
const custom = ref(false);

function reset() {
  custom.value = false;
  hidden.value = privateKeys(schema.value);
}

watch(
  () => [props.visible, props.collection] as const,
  ([open]) => {
    if (!open) return;
    visibility.value = props.collection.visibility ?? "private";
    if (props.collection.hidden_fields == null) reset();
    else {
      custom.value = true;
      hidden.value = [...props.collection.hidden_fields];
    }
  },
  { immediate: true },
);

function toggle(key: string, on: boolean) {
  custom.value = true;
  hidden.value = on
    ? [...hidden.value, key]
    : hidden.value.filter((k) => k !== key);
}

const saving = ref(false);
async function save() {
  saving.value = true;
  try {
    const updated = await collectionApi.update(props.collection.id, {
      visibility: visibility.value,
      hidden_fields: custom.value ? hidden.value : null,
    });
    notifySuccess(t("sharing.share.saved"));
    emit("saved", updated);
    emit("update:visible", false);
  } catch (e) {
    notifyError(e);
  } finally {
    saving.value = false;
  }
}
</script>
