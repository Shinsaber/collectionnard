<template>
  <form
    class="flex flex-col gap-4 rounded-2xl bg-surface p-4 shadow-card"
    data-test="review-form"
    @submit.prevent="save"
  >
    <ElRadioGroup
      v-if="keys.length > 0"
      v-model="mode"
      :aria-label="t('reviews.form.mode')"
      data-test="review-mode"
    >
      <ElRadioButton value="overall">{{
        t("reviews.form.overall")
      }}</ElRadioButton>
      <ElRadioButton value="criteria">{{
        t("reviews.form.byCriteria")
      }}</ElRadioButton>
    </ElRadioGroup>

    <div v-if="mode === 'overall'" class="flex flex-col gap-1.5">
      <span :id="`${uid}-rating`" class="text-[13px] font-bold">{{
        t("reviews.form.rating")
      }}</span>
      <div class="flex items-center gap-3">
        <ElSlider
          v-model="rating"
          :min="0"
          :max="10"
          :step="1"
          class="flex-1"
          :aria-labelledby="`${uid}-rating`"
        />
        <ReviewStars :rating="rating" />
      </div>
    </div>
    <div v-else class="flex flex-col gap-2" data-test="review-criteria-form">
      <DynamicForm v-model="criteria" :schema="schema" />
      <p class="flex items-center gap-2 text-[13px] text-muted">
        {{ t("reviews.form.average") }}
        <ReviewStars :rating="average" />
      </p>
    </div>

    <div v-if="contextItems.length > 0" class="flex flex-col gap-1.5">
      <label :for="`${uid}-context`" class="text-[13px] font-bold">{{
        contextLabelText
      }}</label>
      <ElSelect
        :id="`${uid}-context`"
        v-model="contextItemId"
        clearable
        filterable
        :value-on-clear="null"
        :placeholder="t('reviews.form.contextNone')"
        class="max-w-xl"
        :aria-describedby="`${uid}-context-help`"
        data-test="review-context"
      >
        <ElOption
          v-for="ci in contextItems"
          :key="ci.id"
          :value="ci.id"
          :label="itemLabel(ci, lang)"
        />
      </ElSelect>
      <small
        :id="`${uid}-context-help`"
        class="text-xs"
        :class="
          contextItemId && visibility === 'anonymous'
            ? 'font-semibold text-accent-ink'
            : 'text-muted'
        "
        data-test="review-context-help"
        >{{
          contextItemId && visibility === "anonymous"
            ? t("reviews.form.contextAnonymous")
            : t("reviews.form.contextHelp")
        }}</small
      >
    </div>

    <div class="flex flex-col gap-1.5">
      <label :for="`${uid}-body`" class="text-[13px] font-bold">{{
        t("reviews.form.body")
      }}</label>
      <ElInput
        :id="`${uid}-body`"
        v-model="body"
        type="textarea"
        :autosize="{ minRows: 3, maxRows: 14 }"
        :maxlength="MAX_BODY"
        show-word-limit
        data-test="review-body-input"
        :aria-describedby="`${uid}-body-help`"
      />
      <small :id="`${uid}-body-help`" class="text-xs text-muted">{{
        t("reviews.form.bodyHelp")
      }}</small>
    </div>

    <div class="flex flex-wrap items-center gap-x-6 gap-y-3">
      <ElCheckbox v-model="spoiler" data-test="review-spoiler-input">{{
        t("reviews.form.spoiler")
      }}</ElCheckbox>
      <div class="flex items-center gap-2">
        <label :for="`${uid}-lang`" class="text-[13px] font-bold">{{
          t("reviews.form.lang")
        }}</label>
        <ElSelect :id="`${uid}-lang`" v-model="reviewLang" class="w-36!">
          <ElOption
            v-for="l in LANGS"
            :key="l"
            :value="l"
            :label="`${LANG_FLAGS[l]} ${LANG_NAMES[l]}`"
          />
        </ElSelect>
      </div>
    </div>

    <fieldset class="flex flex-col gap-1.5" data-test="review-visibility">
      <legend class="mb-1 text-[13px] font-bold">
        {{ t("reviews.form.visibility") }}
      </legend>
      <ElRadioGroup v-model="visibility" class="flex-col! items-start! gap-2">
        <ElRadio
          v-for="v in REVIEW_VISIBILITIES"
          :key="v"
          :value="v"
          class="h-auto! items-start! whitespace-normal!"
        >
          <span class="font-semibold">{{ t(`reviews.visibility.${v}`) }}</span>
          <span class="block text-xs font-normal text-muted">{{
            t(`reviews.visibility.${v}Help`)
          }}</span>
        </ElRadio>
      </ElRadioGroup>
    </fieldset>

    <div class="flex flex-wrap justify-end gap-2">
      <ElButton v-if="existing" type="danger" plain @click="emit('remove')">
        {{ t("common.delete") }}
      </ElButton>
      <ElButton @click="emit('cancel')">{{ t("common.cancel") }}</ElButton>
      <ElButton
        type="primary"
        native-type="submit"
        :loading="saving"
        data-test="review-save"
      >
        {{ t("common.save") }}
      </ElButton>
    </div>
  </form>
</template>

<script setup lang="ts">
import { computed, ref, useId } from "vue";
import { useI18n } from "vue-i18n";
import { reviewApi } from "@/api/client";
import { useAuthStore } from "@/stores/auth";
import { useLang } from "@/composables/useLang";
import { useNotify } from "@/composables/useNotify";
import { LANGS, LANG_FLAGS, LANG_NAMES, isLang } from "@/i18n/langs";
import {
  REVIEW_VISIBILITIES,
  criteriaAverage,
  criteriaKeys,
  defaultReviewVisibility,
} from "@/lib/reviews";
import DynamicForm from "@/components/common/DynamicForm.vue";
import ReviewStars from "./ReviewStars.vue";
import { itemLabel } from "@/lib/schema";
import type { Item, Review, ReviewPut, ReviewVisibility } from "@/api/types";

/** Limite du serveur (`internal/review.MaxBody`). */
const MAX_BODY = 10000;
/** Point de départ d'un curseur encore jamais touché. */
const START = 5;

const props = defineProps<{
  itemId: string;
  /** Schéma du type d'avis : ses propriétés sont les critères. */
  schema: Record<string, unknown>;
  existing: Review | null;
  /** Éditions qu'il peut préciser, et le libellé de ce choix. */
  contextItems: Item[];
  contextLabelText: string;
}>();
const emit = defineEmits<{
  saved: [review: Review];
  cancel: [];
  remove: [];
}>();

const { t } = useI18n();
const { lang } = useLang();
const auth = useAuthStore();
const { notifyError, notifySuccess } = useNotify();
const uid = useId();

const keys = computed(() => criteriaKeys(props.schema));
// Un avis détaillé dont un critère a été retiré du type repart de la note globale.
const existingCriteria = props.existing?.criteria;
const mode = ref<"overall" | "criteria">(
  existingCriteria ? "criteria" : "overall",
);
/** Note saisie, préremplie avec l'ancienne moyenne en repassant en note globale. */
const rating = ref(Math.round(props.existing?.rating ?? START));
const criteria = ref<Record<string, unknown>>(
  Object.fromEntries(
    keys.value.map((k) => [k, existingCriteria?.[k] ?? START]),
  ),
);
const average = computed(() => criteriaAverage(criteria.value, keys.value));
const body = ref(props.existing?.body ?? "");
const spoiler = ref(props.existing?.spoiler ?? false);
const reviewLang = ref<string>(
  props.existing && isLang(props.existing.lang)
    ? props.existing.lang
    : lang.value,
);
const visibility = ref<ReviewVisibility>(
  props.existing?.visibility ??
    defaultReviewVisibility(auth.user?.profile_visibility),
);
const contextItemId = ref<string | null>(
  props.existing?.context_item_id ?? null,
);
const saving = ref(false);

async function save() {
  const payload: ReviewPut = {
    body: body.value,
    spoiler: spoiler.value,
    lang: reviewLang.value,
    visibility: visibility.value,
  };
  if (contextItemId.value) payload.context_item_id = contextItemId.value;
  if (mode.value === "criteria") {
    payload.criteria = Object.fromEntries(
      keys.value.map((k) => [k, Number(criteria.value[k] ?? START)]),
    );
  } else {
    payload.rating = rating.value;
  }
  saving.value = true;
  try {
    const review = await reviewApi.put(props.itemId, payload);
    notifySuccess(t("reviews.saved"));
    emit("saved", review);
  } catch (e) {
    notifyError(e);
  } finally {
    saving.value = false;
  }
}
</script>
