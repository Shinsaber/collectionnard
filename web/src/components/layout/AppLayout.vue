<template>
  <div class="min-h-screen flex flex-col bg-gray-50 dark:bg-gray-900">
    <header
      class="bg-white dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700 px-6 py-3 flex items-center justify-between shadow-xs"
    >
      <div class="flex items-center gap-4">
        <router-link
          to="/"
          class="text-xl font-bold text-indigo-600 dark:text-indigo-400"
        >
          📦 Collectionnard
        </router-link>
        <nav class="flex items-center gap-1 ml-6">
          <router-link
            v-for="link in navLinks"
            :key="link.to"
            :to="link.to"
            class="px-3 py-2 rounded-md text-sm font-medium text-gray-600 dark:text-gray-300 hover:text-indigo-600 dark:hover:text-indigo-400 hover:bg-indigo-50 dark:hover:bg-gray-700 transition-colors"
            exact-active-class="text-indigo-600! dark:text-indigo-400! bg-indigo-50! dark:bg-gray-700!"
          >
            <i :class="link.icon" class="mr-1"></i>
            {{ link.label }}
          </router-link>

          <!-- Admin section -->
          <template v-if="auth.isAdmin">
            <div class="w-px h-5 bg-gray-300 dark:bg-gray-600 mx-1"></div>
            <div
              class="flex items-center gap-1 px-2 py-1 rounded-md bg-amber-50 dark:bg-gray-700 border border-amber-200 dark:border-gray-600"
            >
              <span
                class="text-[10px] font-semibold text-amber-600 dark:text-amber-400 uppercase tracking-wide mr-1"
                >{{ t("layout.adminSection") }}</span
              >
              <router-link
                v-for="link in adminLinks"
                :key="link.to"
                :to="link.to"
                class="px-2 py-1 rounded-sm text-sm font-medium text-amber-700 dark:text-amber-300 hover:text-amber-900 dark:hover:text-amber-100 hover:bg-amber-100 dark:hover:bg-gray-600 transition-colors"
                exact-active-class="text-amber-900! dark:text-amber-100! bg-amber-100! dark:bg-gray-600!"
              >
                <i :class="link.icon" class="mr-1"></i>
                {{ link.label }}
              </router-link>
            </div>
          </template>
        </nav>
      </div>

      <div class="flex items-center gap-2">
        <span class="text-sm text-gray-500 dark:text-gray-400 mr-2">
          <i class="pi pi-user mr-1"></i>
          {{ auth.user?.username }}
          <ElTag v-if="auth.isAdmin" type="warning" size="small" class="ml-1">{{
            t("layout.adminBadge")
          }}</ElTag>
        </span>
        <!-- Sélecteur de langue : liste extensible issue du registre `langs` -->
        <ElDropdown trigger="click" @command="setLang">
          <ElButton
            text
            size="small"
            class="font-mono"
            :title="t('layout.language')"
            :aria-label="t('layout.language')"
            >{{ lang.toUpperCase() }}
            <i class="pi pi-angle-down ml-1" aria-hidden="true"></i
          ></ElButton>
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
        <ElButton
          text
          :title="themeLabel"
          :aria-label="themeLabel"
          @click="toggleTheme"
        >
          <i :class="theme === 'dark' ? 'pi pi-sun' : 'pi pi-moon'"></i>
        </ElButton>
        <ElButton
          text
          :title="t('layout.logout')"
          :aria-label="t('layout.logout')"
          @click="handleLogout"
        >
          <i class="pi pi-sign-out"></i>
        </ElButton>
      </div>
    </header>

    <main class="flex-1 p-6 max-w-7xl mx-auto w-full">
      <router-view />
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { useAuthStore } from "@/stores/auth";
import { useTheme } from "@/composables/useTheme";
import { useLang } from "@/composables/useLang";

const auth = useAuthStore();
const router = useRouter();
const { t } = useI18n();
const { theme, toggle: toggleTheme } = useTheme();
const { lang, setLang, langs, langNames } = useLang();

const themeLabel = computed(() =>
  theme.value === "dark" ? t("layout.lightMode") : t("layout.darkMode"),
);

const navLinks = computed(() => [
  { to: "/", label: t("layout.nav.dashboard"), icon: "pi pi-home" },
  { to: "/items", label: t("layout.nav.catalog"), icon: "pi pi-box" },
  {
    to: "/collections",
    label: t("layout.nav.collections"),
    icon: "pi pi-bookmark",
  },
]);

const adminLinks = computed(() => [
  {
    to: "/admin/item-types",
    label: t("layout.nav.itemTypes"),
    icon: "pi pi-cog",
  },
  {
    to: "/admin/relation-types",
    label: t("layout.nav.relationTypes"),
    icon: "pi pi-arrows-h",
  },
  {
    to: "/admin/collection-types",
    label: t("layout.nav.collectionTypes"),
    icon: "pi pi-tags",
  },
  {
    to: "/admin/import-export",
    label: t("layout.nav.importExport"),
    icon: "pi pi-database",
  },
]);

function handleLogout() {
  auth.logout();
  router.push("/login");
}
</script>
