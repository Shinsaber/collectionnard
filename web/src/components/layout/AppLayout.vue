<template>
  <div class="min-h-screen flex flex-col bg-gray-50 dark:bg-gray-900">
    <header
      class="bg-white dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700 px-6 py-3 flex items-center justify-between shadow-sm"
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
            exact-active-class="!text-indigo-600 dark:!text-indigo-400 !bg-indigo-50 dark:!bg-gray-700"
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
                >Admin</span
              >
              <router-link
                v-for="link in adminLinks"
                :key="link.to"
                :to="link.to"
                class="px-2 py-1 rounded text-sm font-medium text-amber-700 dark:text-amber-300 hover:text-amber-900 dark:hover:text-amber-100 hover:bg-amber-100 dark:hover:bg-gray-600 transition-colors"
                exact-active-class="!text-amber-900 dark:!text-amber-100 !bg-amber-100 dark:!bg-gray-600"
              >
                <i :class="link.icon" class="mr-1"></i>
                {{ link.label }}
              </router-link>
            </div>
          </template>
        </nav>
      </div>

      <div class="flex items-center gap-4">
        <span class="text-sm text-gray-500 dark:text-gray-400">
          <i class="pi pi-user mr-1"></i>
          {{ auth.user?.username }}
          <Tag v-if="auth.isAdmin" value="admin" severity="warn" class="ml-1" />
        </span>
        <Button
          :icon="theme === 'dark' ? 'pi pi-sun' : 'pi pi-moon'"
          text
          severity="secondary"
          @click="toggleTheme"
          v-tooltip="theme === 'dark' ? 'Mode clair' : 'Mode sombre'"
        />
        <Button
          icon="pi pi-sign-out"
          text
          severity="secondary"
          @click="handleLogout"
        />
      </div>
    </header>

    <main class="flex-1 p-6 max-w-7xl mx-auto w-full">
      <router-view />
    </main>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from "vue-router";
import Button from "primevue/button";
import Tag from "primevue/tag";
import { useAuthStore } from "@/stores/auth";
import { useTheme } from "@/composables/useTheme";

const auth = useAuthStore();
const router = useRouter();
const { theme, toggle: toggleTheme } = useTheme();

const navLinks = [
  { to: "/", label: "Dashboard", icon: "pi pi-home" },
  { to: "/items", label: "Catalogue", icon: "pi pi-box" },
  { to: "/collections", label: "Collections", icon: "pi pi-bookmark" },
];

const adminLinks = [
  { to: "/admin/item-types", label: "Types", icon: "pi pi-cog" },
  { to: "/admin/relation-types", label: "Relations", icon: "pi pi-arrows-h" },
  { to: "/admin/collection-types", label: "Types coll.", icon: "pi pi-tags" },
];

function handleLogout() {
  auth.logout();
  router.push("/login");
}
</script>
