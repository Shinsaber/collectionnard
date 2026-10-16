<template>
  <div class="flex flex-col gap-3 rounded-xl bg-surface p-3 shadow-card">
    <div class="flex items-center gap-3">
      <span
        class="flex size-9 shrink-0 items-center justify-center rounded-full bg-ink text-sm font-bold text-surface"
        aria-hidden="true"
        >{{ initial }}</span
      >
      <div class="flex min-w-0 flex-col">
        <span class="truncate text-sm font-bold">{{
          auth.user?.username
        }}</span>
        <span
          class="font-mono text-[11px] uppercase tracking-wider text-muted"
          >{{
            auth.isAdmin ? t("layout.adminBadge") : t("layout.userBadge")
          }}</span
        >
      </div>
    </div>
    <div class="grid grid-cols-3 gap-1.5">
      <!-- Sélecteur de langue : liste extensible issue du registre `langs` -->
      <ElDropdown trigger="click" class="block!" @command="setLang">
        <button
          type="button"
          class="flex h-11 w-full items-center justify-center gap-1 rounded-lg bg-soft font-mono text-xs font-medium text-ink"
          :title="t('layout.language')"
          :aria-label="`${t('layout.language')} : ${langNames[lang]}`"
        >
          {{ lang.toUpperCase() }}
          <i class="pi pi-angle-down text-[10px]" aria-hidden="true"></i>
        </button>
        <template #dropdown>
          <ElDropdownMenu>
            <ElDropdownItem
              v-for="l in langs"
              :key="l"
              :command="l"
              :lang="l"
              :class="{ 'font-semibold': l === lang }"
              :aria-current="l === lang ? 'true' : undefined"
            >
              <i
                class="pi pi-check mr-2"
                :class="{ invisible: l !== lang }"
                aria-hidden="true"
              ></i>
              {{ langNames[l] }}
            </ElDropdownItem>
          </ElDropdownMenu>
        </template>
      </ElDropdown>
      <button
        type="button"
        class="flex h-11 items-center justify-center rounded-lg bg-soft text-ink"
        :title="themeLabel"
        :aria-label="themeLabel"
        @click="toggleTheme"
      >
        <i
          :class="theme === 'dark' ? 'pi pi-sun' : 'pi pi-moon'"
          aria-hidden="true"
        ></i>
      </button>
      <button
        type="button"
        class="flex h-11 items-center justify-center rounded-lg bg-soft text-ink"
        :title="t('layout.logout')"
        :aria-label="t('layout.logout')"
        @click="handleLogout"
      >
        <i class="pi pi-sign-out" aria-hidden="true"></i>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { useAuthStore } from "@/stores/auth";
import { useOfflineStore } from "@/stores/offline";
import { useTheme } from "@/composables/useTheme";
import { useLang } from "@/composables/useLang";

const auth = useAuthStore();
const router = useRouter();
const { t } = useI18n();
const { theme, toggle: toggleTheme } = useTheme();
const { lang, setLang, langs, langNames } = useLang();

const initial = computed(
  () => auth.user?.username?.charAt(0).toUpperCase() ?? "?",
);
const themeLabel = computed(() =>
  theme.value === "dark" ? t("layout.lightMode") : t("layout.darkMode"),
);

async function handleLogout() {
  await auth.logout();
  useOfflineStore().clear();
  router.push("/login");
}
</script>
