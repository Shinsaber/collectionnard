<template>
  <AppSpinner v-if="loading" />
  <div v-else class="pb-20 md:pb-0">
    <RouterLink
      :to="cfg.listPath"
      class="mb-4 inline-flex items-center gap-1.5 text-sm font-semibold text-link"
    >
      <i class="pi pi-arrow-left text-xs" aria-hidden="true"></i>
      {{ t(cfg.listTitleKey) }}
    </RouterLink>

    <PageHeader :eyebrow="eyebrow" :title="title">
      <template #actions>
        <span
          v-if="dirty"
          class="flex items-center gap-1.5 text-[13px] text-muted"
          role="status"
        >
          <span class="size-2 rounded-full bg-accent" aria-hidden="true"></span>
          {{ t("admin.shared.unsaved") }}
        </span>
        <ElButton class="max-md:hidden!" @click="router.push(cfg.listPath)">
          {{ t("common.cancel") }}
        </ElButton>
        <ElButton
          type="primary"
          class="max-md:hidden!"
          :loading="saving"
          :disabled="!!id && !dirty"
          @click="save"
        >
          {{ id ? t("common.save") : t("common.create") }}
        </ElButton>
      </template>
    </PageHeader>

    <SchemaBuilder
      ref="builderRef"
      v-model="form.attributes_schema"
      :features="{
        completeness: kind === 'relation',
        private: kind === 'collection',
        identifier: kind === 'item',
        criteria: kind === 'review',
      }"
    >
      <template #aside>
        <!-- Identité : identifiant technique, description, extrémités (relations) -->
        <div
          class="flex flex-col gap-4 rounded-2xl bg-surface p-4.5 shadow-card"
        >
          <h2 class="text-base font-extrabold">
            {{ t("admin.shared.identity") }}
          </h2>
          <div class="flex flex-col gap-1.5">
            <label :for="`${uid}-name`" class="text-[13px] font-bold">{{
              t("admin.shared.technicalName")
            }}</label>
            <ElInput
              :id="`${uid}-name`"
              v-model="form.name"
              class="font-mono"
              :placeholder="t(cfg.namePlaceholderKey)"
              :aria-describedby="`${uid}-name-help`"
            />
            <small :id="`${uid}-name-help`" class="text-xs text-muted">{{
              t("admin.shared.technicalNameHelp")
            }}</small>
          </div>
          <div class="flex flex-col gap-1.5">
            <label :for="`${uid}-desc`" class="text-[13px] font-bold">{{
              t("common.description")
            }}</label>
            <ElInput
              :id="`${uid}-desc`"
              v-model="form.description"
              type="textarea"
              :autosize="{ minRows: 2, maxRows: 6 }"
            />
          </div>
          <div v-if="kind === 'item'" class="flex flex-col gap-1.5">
            <label :for="`${uid}-order`" class="text-[13px] font-bold">{{
              t("admin.itemTypes.displayOrder")
            }}</label>
            <ElInputNumber
              :id="`${uid}-order`"
              v-model="form.display_order"
              :min="0"
              :step="10"
              controls-position="right"
              class="w-40!"
              :aria-describedby="`${uid}-order-help`"
            />
            <small :id="`${uid}-order-help`" class="text-xs text-muted">{{
              t("admin.itemTypes.displayOrderHelp")
            }}</small>
          </div>
          <div v-if="kind === 'item'" class="flex flex-col gap-1.5">
            <label :for="`${uid}-review`" class="text-[13px] font-bold">{{
              t("admin.itemTypes.reviewType")
            }}</label>
            <ElSelect
              :id="`${uid}-review`"
              v-model="form.review_type_id"
              clearable
              :value-on-clear="null"
              :placeholder="t('admin.itemTypes.reviewsClosed')"
              :aria-describedby="`${uid}-review-help`"
              data-test="review-type-select"
            >
              <ElOption
                v-for="rt in metaTypes.reviewTypes"
                :key="rt.id"
                :label="typeTitle(rt, lang)"
                :value="rt.id"
              />
            </ElSelect>
            <small :id="`${uid}-review-help`" class="text-xs text-muted">{{
              t("admin.itemTypes.reviewTypeHelp")
            }}</small>
          </div>
          <template v-if="kind === 'review'">
            <div class="flex flex-col gap-1.5">
              <label :for="`${uid}-context`" class="text-[13px] font-bold">{{
                t("admin.reviewTypes.contextRelations")
              }}</label>
              <ElSelect
                :id="`${uid}-context`"
                v-model="form.context_relation_type_ids"
                multiple
                collapse-tags
                collapse-tags-tooltip
                filterable
                :aria-describedby="`${uid}-context-help`"
                data-test="context-relations"
              >
                <ElOption
                  v-for="rt in metaTypes.relationTypes"
                  :key="rt.id"
                  :label="`${typeTitle(rt, lang)} (${rt.name})`"
                  :value="rt.id"
                />
              </ElSelect>
              <small :id="`${uid}-context-help`" class="text-xs text-muted">{{
                t("admin.reviewTypes.contextRelationsHelp")
              }}</small>
            </div>
            <div
              v-for="l in LANGS"
              v-show="form.context_relation_type_ids.length > 0"
              :key="l"
              class="flex flex-col gap-1.5"
            >
              <label
                :for="`${uid}-context-${l}`"
                class="text-[13px] font-bold"
                >{{
                  t("admin.reviewTypes.contextLabel", { lang: l.toUpperCase() })
                }}</label
              >
              <ElInput
                :id="`${uid}-context-${l}`"
                v-model="form.context_label[l]"
                maxlength="60"
                :placeholder="t('admin.reviewTypes.contextLabelPlaceholder')"
              />
            </div>
          </template>
          <div v-if="kind === 'collection'" class="flex flex-col gap-1.5">
            <label :for="`${uid}-allowed`" class="text-[13px] font-bold">{{
              t("admin.collectionTypes.allowedItemTypes")
            }}</label>
            <ElSelect
              :id="`${uid}-allowed`"
              v-model="form.item_type_ids"
              multiple
              collapse-tags
              collapse-tags-tooltip
              filterable
              :placeholder="
                t('admin.collectionTypes.allowedItemTypesPlaceholder')
              "
              :aria-describedby="`${uid}-allowed-help`"
            >
              <ElOption
                v-for="it in metaTypes.itemTypes"
                :key="it.id"
                :label="typeTitle(it, lang)"
                :value="it.id"
              />
            </ElSelect>
            <small :id="`${uid}-allowed-help`" class="text-xs text-muted">{{
              t("admin.collectionTypes.allowedItemTypesHelp")
            }}</small>
          </div>
          <template v-if="kind === 'relation'">
            <div
              v-for="side in sides"
              :key="side.field"
              class="flex flex-col gap-1.5"
            >
              <label
                :for="`${uid}-${side.field}`"
                class="text-[13px] font-bold"
                >{{ side.label }}</label
              >
              <ElSelect
                :id="`${uid}-${side.field}`"
                v-model="form[side.field]"
                multiple
                collapse-tags
                collapse-tags-tooltip
                filterable
                :placeholder="t('admin.relationTypes.choose')"
              >
                <ElOption
                  v-for="it in metaTypes.itemTypes"
                  :key="it.id"
                  :label="typeTitle(it, lang)"
                  :value="it.id"
                >
                  {{ typeTitle(it, lang) }}
                  <span
                    v-if="typeTitle(it, lang) !== it.name"
                    class="ml-1 font-mono text-xs text-muted"
                    >{{ it.name }}</span
                  >
                </ElOption>
              </ElSelect>
            </div>
          </template>
        </div>
      </template>
    </SchemaBuilder>

    <!-- Mobile : enregistrement en barre fixe au-dessus des onglets -->
    <div
      class="fixed inset-x-0 bottom-[calc(4.5rem+env(safe-area-inset-bottom))] z-10 flex gap-2.5 bg-surface px-4 py-3 shadow-bar-up md:hidden"
    >
      <ElButton class="flex-1" @click="router.push(cfg.listPath)">
        {{ t("common.cancel") }}
      </ElButton>
      <ElButton
        type="primary"
        class="flex-[1.6]"
        :loading="saving"
        :disabled="!!id && !dirty"
        @click="save"
      >
        {{ id ? t("common.save") : t("common.create") }}
      </ElButton>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, useId, watch } from "vue";
import { useI18n } from "vue-i18n";
import { onBeforeRouteLeave, useRouter } from "vue-router";
import { useNotify } from "@/composables/useNotify";
import { useConfirmDialog } from "@/composables/useConfirmDialog";
import { useLang } from "@/composables/useLang";
import { useMetaTypes } from "@/stores/metaTypes";
import { typeTitle } from "@/lib/schema";
import { LANGS } from "@/i18n/langs";
import AppSpinner from "@/components/common/AppSpinner.vue";
import PageHeader from "@/components/common/PageHeader.vue";
import SchemaBuilder from "@/components/common/SchemaBuilder.vue";
import { TYPE_KINDS, formOf, type TypeForm, type TypeKind } from "./typeKinds";

// Page d'édition d'un méta-type (item, relation, collection) : identité et schéma des
// attributs. Remplace les anciens dialogues ; une sortie avec des changements non
// enregistrés demande confirmation.
const props = defineProps<{ kind: TypeKind; id?: string }>();

const { t } = useI18n();
const { lang } = useLang();
const router = useRouter();
const { notifyError, notifySuccess } = useNotify();
const { confirm } = useConfirmDialog();
const metaTypes = useMetaTypes();
const uid = useId();

const cfg = computed(() => TYPE_KINDS[props.kind]);
const loading = ref(true);
const saving = ref(false);
const form = ref<TypeForm>(formOf());
const builderRef = ref<InstanceType<typeof SchemaBuilder> | null>(null);
/** Instantané de la dernière version enregistrée (détection des changements). */
const saved = ref("");
const dirty = computed(() => JSON.stringify(form.value) !== saved.value);

const sides = computed(() => [
  {
    field: "source_item_type_ids" as const,
    label: t("admin.relationTypes.sourceTypes"),
  },
  {
    field: "target_item_type_ids" as const,
    label: t("admin.relationTypes.targetTypes"),
  },
]);

const eyebrow = computed(
  () =>
    `${t(`admin.shared.kinds.${props.kind}`)}${form.value.name ? ` · ${form.value.name}` : ""}`,
);
/** Nom affiché (x-title du schéma en cours d'édition), sinon identifiant, sinon « Nouveau type ». */
const title = computed(
  () =>
    typeTitle(
      {
        name: form.value.name,
        attributes_schema: form.value.attributes_schema,
      },
      lang.value,
    ) || t("admin.shared.newType"),
);

async function load() {
  loading.value = true;
  try {
    const type = props.id ? await cfg.value.get(props.id) : null;
    form.value = formOf(type);
    builderRef.value?.resetFromSchema(form.value.attributes_schema);
  } catch (e) {
    notifyError(e, t("admin.shared.notFound"));
  } finally {
    loading.value = false;
  }
  // Le constructeur normalise le schéma à l'ouverture : l'instantané suit ce premier rendu.
  setTimeout(() => {
    saved.value = JSON.stringify(form.value);
  });
}

async function save() {
  saving.value = true;
  try {
    await cfg.value.save(form.value, props.id);
    notifySuccess(t(props.id ? cfg.value.updatedKey : cfg.value.createdKey));
    metaTypes.invalidate();
    saved.value = JSON.stringify(form.value);
    router.push(cfg.value.listPath);
  } catch (e) {
    notifyError(e);
  } finally {
    saving.value = false;
  }
}

onBeforeRouteLeave(async () => {
  if (!dirty.value || saving.value) return true;
  return confirm({
    message: t("admin.shared.leaveConfirm"),
    confirmText: t("admin.shared.leave"),
    danger: true,
  });
});

onMounted(() => {
  if (props.kind === "item")
    metaTypes.loadReviewTypes().catch((e) => notifyError(e));
  if (props.kind === "review")
    metaTypes.loadRelationTypes().catch((e) => notifyError(e));
  if (props.kind !== "item")
    metaTypes
      .loadItemTypes()
      .catch((e) => notifyError(e, t("admin.relationTypes.itemTypesFailed")));
  load();
});
watch(
  () => [props.kind, props.id],
  () => load(),
);
</script>
