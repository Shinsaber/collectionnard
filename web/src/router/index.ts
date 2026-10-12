import { createRouter, createWebHistory } from "vue-router";
import { useAuthStore } from "@/stores/auth";

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: "/login",
      name: "login",
      component: () => import("@/views/LoginView.vue"),
      meta: { public: true },
    },
    {
      path: "/",
      component: () => import("@/components/layout/AppLayout.vue"),
      children: [
        {
          path: "",
          name: "dashboard",
          component: () => import("@/views/DashboardView.vue"),
        },
        // ── Catalogue ──
        {
          path: "items",
          name: "items",
          component: () => import("@/views/ItemsView.vue"),
        },
        {
          path: "items/:id",
          name: "item-detail",
          component: () => import("@/views/ItemDetailView.vue"),
          props: true,
        },
        {
          path: "items/:id/graph",
          name: "item-graph",
          component: () => import("@/views/GraphView.vue"),
          props: true,
        },
        // ── Collections ──
        {
          path: "collections",
          name: "collections",
          component: () => import("@/views/CollectionsView.vue"),
        },
        {
          path: "collections/:id",
          name: "collection-detail",
          component: () => import("@/views/CollectionDetailView.vue"),
          props: true,
        },
        // ── Administration ──
        {
          path: "admin/item-types",
          name: "item-types",
          component: () => import("@/views/admin/ItemTypesView.vue"),
          meta: { admin: true },
        },
        {
          path: "admin/relation-types",
          name: "relation-types",
          component: () => import("@/views/admin/RelationTypesView.vue"),
          meta: { admin: true },
        },
        {
          path: "admin/collection-types",
          name: "collection-types",
          component: () => import("@/views/admin/CollectionTypesView.vue"),
          meta: { admin: true },
        },
        {
          path: "admin/import-export",
          name: "import-export",
          component: () => import("@/views/admin/ImportExportView.vue"),
          meta: { admin: true },
        },
        {
          path: "admin/external-import",
          name: "external-import",
          component: () => import("@/views/admin/ExternalImportView.vue"),
          meta: { admin: true },
        },
      ],
    },
  ],
});

router.beforeEach(async (to) => {
  const auth = useAuthStore();

  if (to.meta.public) return true;

  if (!auth.isAuthenticated) {
    return { name: "login" };
  }

  // Load user if not yet fetched
  if (!auth.user) {
    await auth.fetchMe();
    if (!auth.isAuthenticated) return { name: "login" };
  }

  if (to.meta.admin && !auth.isAdmin) {
    return { name: "dashboard" };
  }

  return true;
});

export default router;
