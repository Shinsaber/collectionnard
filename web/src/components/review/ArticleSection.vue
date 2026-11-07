<template>
  <div class="flex flex-col gap-4" data-test="article-section">
    <ElButton
      v-if="!editing"
      type="primary"
      class="self-start"
      data-test="article-new"
      @click="startNew"
    >
      <i class="pi pi-link mr-2" aria-hidden="true"></i>
      {{ t("articles.share") }}
    </ElButton>
    <form
      v-else
      class="flex flex-col gap-3 rounded-2xl bg-surface p-4 shadow-card"
      data-test="article-form"
      @submit.prevent="save"
    >
      <div class="flex flex-col gap-1.5">
        <label :for="`${uid}-title`" class="text-[13px] font-bold">{{
          t("articles.form.title")
        }}</label>
        <ElInput :id="`${uid}-title`" v-model="form.title" maxlength="200" />
      </div>
      <div class="flex flex-col gap-1.5">
        <label :for="`${uid}-url`" class="text-[13px] font-bold">{{
          t("articles.form.url")
        }}</label>
        <ElInput
          :id="`${uid}-url`"
          v-model="form.url"
          type="url"
          maxlength="2000"
          placeholder="https://…"
        />
      </div>
      <div class="flex flex-col gap-1.5">
        <label :for="`${uid}-message`" class="text-[13px] font-bold">{{
          t("articles.form.message")
        }}</label>
        <ElInput
          :id="`${uid}-message`"
          v-model="form.message"
          type="textarea"
          :autosize="{ minRows: 2, maxRows: 6 }"
          maxlength="1000"
          show-word-limit
        />
      </div>
      <div class="flex items-center gap-2">
        <label :for="`${uid}-lang`" class="text-[13px] font-bold">{{
          t("reviews.form.lang")
        }}</label>
        <ElSelect :id="`${uid}-lang`" v-model="form.lang" class="w-36!">
          <ElOption
            v-for="l in LANGS"
            :key="l"
            :value="l"
            :label="`${LANG_FLAGS[l]} ${LANG_NAMES[l]}`"
          />
        </ElSelect>
      </div>
      <p v-if="!auth.canModerate" class="text-xs text-muted">
        {{ t("articles.form.moderated") }}
      </p>
      <div class="flex justify-end gap-2">
        <ElButton @click="editing = false">{{ t("common.cancel") }}</ElButton>
        <ElButton
          type="primary"
          native-type="submit"
          :loading="saving"
          :disabled="!form.title.trim() || !form.url.trim()"
          data-test="article-save"
          >{{
            editingId ? t("common.save") : t("articles.form.send")
          }}</ElButton
        >
      </div>
    </form>

    <AppSpinner v-if="loading" />
    <template v-else>
      <p
        v-if="articles.length === 0"
        class="rounded-2xl bg-surface p-5 text-sm text-muted shadow-card"
      >
        {{ t("articles.none") }}
      </p>
      <ArticleCard
        v-for="a in articles"
        :key="a.id"
        :article="a"
        :can-write="auth.isAuthenticated"
        @edit="startEdit(a)"
        @remove="remove(a)"
      />
      <ElPagination
        class="justify-end"
        hide-on-single-page
        layout="prev, pager, next"
        :total="total"
        :page-size="PER_PAGE"
        :current-page="page"
        @current-change="(p: number) => ((page = p), load())"
      />
    </template>
  </div>
</template>

<script setup lang="ts">
// Onglet « Actualités » d'une fiche (I-3) : partager un article (titre, lien, un mot) ; celui d'un membre
// attend la validation d'un modérateur avant de paraître.
import { onMounted, reactive, ref, useId, watch } from "vue";
import { useI18n } from "vue-i18n";
import { articleApi } from "@/api/client";
import { useAuthStore } from "@/stores/auth";
import { useConfirmDialog } from "@/composables/useConfirmDialog";
import { useLang } from "@/composables/useLang";
import { useNotify } from "@/composables/useNotify";
import { LANGS, LANG_FLAGS, LANG_NAMES } from "@/i18n/langs";
import AppSpinner from "@/components/common/AppSpinner.vue";
import ArticleCard from "./ArticleCard.vue";
import type { Article, ArticleWrite } from "@/api/types";

const PER_PAGE = 20;

const props = defineProps<{ itemId: string }>();
const emit = defineEmits<{ count: [n: number] }>();

const { t } = useI18n();
const { lang } = useLang();
const auth = useAuthStore();
const { confirm } = useConfirmDialog();
const { notifyError, notifySuccess } = useNotify();
const uid = useId();

const articles = ref<Article[]>([]);
const total = ref(0);
const page = ref(1);
const loading = ref(true);
const editing = ref(false);
const editingId = ref<string | null>(null);
const saving = ref(false);
const form = reactive<ArticleWrite>({
  title: "",
  url: "",
  message: "",
  lang: lang.value,
});

async function load() {
  loading.value = true;
  try {
    const res = await articleApi.list(props.itemId, {
      page: page.value,
      per_page: PER_PAGE,
    });
    articles.value = res.data;
    total.value = res.total;
    emit("count", res.total);
  } catch (e) {
    notifyError(e);
  } finally {
    loading.value = false;
  }
}

function startNew() {
  Object.assign(form, { title: "", url: "", message: "", lang: lang.value });
  editingId.value = null;
  editing.value = true;
}

function startEdit(a: Article) {
  Object.assign(form, {
    title: a.title,
    url: a.url,
    message: a.message,
    lang: a.lang,
  });
  editingId.value = a.id;
  editing.value = true;
}

async function save() {
  saving.value = true;
  try {
    const a = editingId.value
      ? await articleApi.update(editingId.value, { ...form })
      : await articleApi.create(props.itemId, { ...form });
    notifySuccess(
      t(a.status === "published" ? "articles.published" : "articles.waiting"),
    );
    editing.value = false;
    page.value = 1;
    await load();
  } catch (e) {
    notifyError(e);
  } finally {
    saving.value = false;
  }
}

async function remove(a: Article) {
  const ok = await confirm({
    message: t("articles.confirmDelete"),
    confirmText: t("common.delete"),
    danger: true,
  });
  if (!ok) return;
  try {
    await articleApi.remove(a.id);
    await load();
  } catch (e) {
    notifyError(e);
  }
}

onMounted(load);
watch(
  () => props.itemId,
  () => ((page.value = 1), load()),
);
</script>
