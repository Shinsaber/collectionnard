<template>
  <div class="min-h-screen flex flex-col bg-gray-50">
    <header
      class="bg-white border-b border-gray-200 px-6 py-3 flex items-center justify-between shadow-sm"
    >
      <div class="flex items-center gap-4">
        <router-link to="/" class="text-xl font-bold text-indigo-600">
          📦 Collectionnard
        </router-link>
        <nav class="flex gap-1 ml-6">
          <router-link
            v-for="link in navLinks"
            :key="link.to"
            :to="link.to"
            class="px-3 py-2 rounded-md text-sm font-medium text-gray-600 hover:text-indigo-600 hover:bg-indigo-50 transition-colors"
            active-class="!text-indigo-600 !bg-indigo-50"
          >
            <i :class="link.icon" class="mr-1"></i>
            {{ link.label }}
          </router-link>
        </nav>
      </div>

      <div class="flex items-center gap-4">
        <span class="text-sm text-gray-500">
          <i class="pi pi-user mr-1"></i>
          {{ auth.user?.username }}
          <Tag v-if="auth.isAdmin" value="admin" severity="warn" class="ml-1" />
        </span>
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
import { computed } from "vue";
import { useRouter } from "vue-router";
import Button from "primevue/button";
import Tag from "primevue/tag";
import { useAuthStore } from "@/stores/auth";

const auth = useAuthStore();
const router = useRouter();

const navLinks = computed(() => {
  const links = [
    { to: "/", label: "Dashboard", icon: "pi pi-home" },
    { to: "/items", label: "Catalogue", icon: "pi pi-box" },
    { to: "/collections", label: "Collections", icon: "pi pi-bookmark" },
  ];
  if (auth.isAdmin) {
    links.push(
      { to: "/admin/item-types", label: "Types", icon: "pi pi-cog" },
      {
        to: "/admin/relation-types",
        label: "Relations",
        icon: "pi pi-arrows-h",
      },
      {
        to: "/admin/collection-types",
        label: "Types coll.",
        icon: "pi pi-tags",
      },
    );
  }
  return links;
});

function handleLogout() {
  auth.logout();
  router.push("/login");
}
</script>
