import {
  createRouter,
  createWebHistory,
  type RouteLocationNormalized,
} from "vue-router";
import { useAuthStore } from "@/stores/auth";
import { useOfflineStore } from "@/stores/offline";

const PublicLayout = () => import("@/components/layout/PublicLayout.vue");

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
      path: "/register",
      name: "register",
      component: () => import("@/views/RegisterView.vue"),
      meta: { public: true },
    },
    {
      path: "/reset-password",
      name: "reset-password",
      component: () => import("@/views/ResetPasswordView.vue"),
      meta: { public: true },
    },
    // Pages publiques (P4-2) : sans compte, hors de la coquille de l'application.
    {
      path: "/u/:username",
      component: PublicLayout,
      meta: { public: true },
      children: [
        {
          path: "",
          name: "public-profile",
          component: () => import("@/views/public/PublicProfileView.vue"),
          props: true,
          meta: { public: true },
        },
      ],
    },
    {
      path: "/c/:id",
      component: PublicLayout,
      meta: { public: true },
      children: [
        {
          path: "",
          name: "public-collection",
          component: () => import("@/views/public/PublicCollectionView.vue"),
          props: true,
          meta: { public: true },
        },
      ],
    },
    {
      path: "/share",
      component: PublicLayout,
      meta: { public: true },
      children: [
        {
          path: "",
          name: "shared-collection",
          component: () => import("@/views/public/SharedCollectionView.vue"),
          meta: { public: true },
        },
      ],
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
        // ── Scanner (mobile.md, M3) ──
        {
          path: "scan",
          name: "scan",
          component: () => import("@/views/ScanView.vue"),
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
        // Édition d'un méta-type (page dédiée, commune aux trois sortes) : `new` ou un id.
        ...(["item", "relation", "collection"] as const).map((kind) => ({
          path: `admin/${kind}-types/:id`,
          name: `${kind}-type-editor`,
          component: () => import("@/views/admin/TypeEditorView.vue"),
          props: (route: RouteLocationNormalized) => ({
            kind,
            id: route.params.id === "new" ? undefined : route.params.id,
          }),
          meta: { admin: true },
        })),
        {
          path: "admin/import-export",
          name: "import-export",
          component: () => import("@/views/admin/ImportExportView.vue"),
          meta: { admin: true },
        },
        {
          path: "offline",
          name: "offline",
          component: () => import("@/views/OfflineView.vue"),
        },
        {
          path: "proposals",
          name: "proposals",
          component: () => import("@/views/ProposalsView.vue"),
        },
        {
          path: "proposals/:id",
          name: "proposal",
          component: () => import("@/views/ProposalDetailView.vue"),
          props: true,
        },
        {
          path: "members",
          name: "members",
          component: () => import("@/views/MembersView.vue"),
        },
        {
          path: "stats",
          name: "stats",
          component: () => import("@/views/StatsView.vue"),
        },
        {
          path: "profile",
          name: "profile",
          component: () => import("@/views/ProfileView.vue"),
        },
        {
          path: "admin/users",
          name: "admin-users",
          component: () => import("@/views/admin/UsersView.vue"),
          meta: { admin: true },
        },
        {
          path: "admin/domains",
          name: "domains",
          component: () => import("@/views/admin/DomainsView.vue"),
          meta: { admin: true },
        },
        {
          path: "admin/images",
          name: "images",
          component: () => import("@/views/admin/ImagesView.vue"),
          meta: { admin: true },
        },
        {
          path: "admin/external-import",
          name: "external-import",
          // Membres aussi (P4-3b) : ils proposent la sélection au lieu de l'importer.
          component: () => import("@/views/admin/ExternalImportView.vue"),
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

  // Serveur injoignable : seule la copie hors ligne des collections est consultable (lot E5c).
  if (!useOfflineStore().reachable && to.name !== "offline") {
    return { name: "offline" };
  }

  if (to.meta.catalog && !auth.canEditCatalog) {
    return { name: "dashboard" };
  }

  if (to.meta.admin && !auth.isAdmin) {
    return { name: "dashboard" };
  }

  return true;
});

export default router;
