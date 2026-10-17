import { computed } from "vue";
import { useI18n } from "vue-i18n";

/** Icônes de navigation (tracés au trait des maquettes, viewBox 24). */
export const NAV_ICONS = {
  home: "M3 11l9-7 9 7v9a1 1 0 0 1-1 1h-5v-6h-6v6H4a1 1 0 0 1-1-1z",
  catalog: "M4 3h6v18H4zM12 6h5v15h-5zM19 7l2 13",
  collections: "M6 3h12v18l-6-4-6 4z",
  more: "M4 12a1.5 1.5 0 1 0 3 0a1.5 1.5 0 1 0-3 0M10.5 12a1.5 1.5 0 1 0 3 0a1.5 1.5 0 1 0-3 0M17 12a1.5 1.5 0 1 0 3 0a1.5 1.5 0 1 0-3 0",
  itemTypes: "M12 2l9 5v10l-9 5-9-5V7zM12 12l9-5M12 12v10M12 12L3 7",
  relationTypes: "M4 12h16M14 6l6 6-6 6M10 6l-6 6 6 6",
  collectionTypes: "M3 12V4h8l10 10-8 8zM7.5 7.5h.01",
  stats: "M4 20V10M10 20V4M16 20v-7M22 20H2",
  offline:
    "M3 3l18 18M8.6 8.6A5 5 0 0 0 7 18h10.6M20.4 16.4A4 4 0 0 0 18 9a6 6 0 0 0-7.7-3.8",
  importExport: "M12 3v12M7 10l5 5 5-5M4 21h16",
  images: "M4 5h16v14H4zM4 16l5-5 4 4 3-3 4 4M15.5 9a1.5 1.5 0 1 0 0 .01",
  domains: "M12 3l9 5-9 5-9-5zM3 12.5l9 5 9-5M3 16.5l9 5 9-5",
  users:
    "M9 11a4 4 0 1 0 0-8a4 4 0 1 0 0 8M2 21v-1a6 6 0 0 1 6-6h2a6 6 0 0 1 6 6v1M16 3.5a4 4 0 0 1 0 7M22 21v-1a6 6 0 0 0-4-5.7",
  external:
    "M7 18a5 5 0 0 1-.6-9.96A6 6 0 0 1 18 9a4 4 0 0 1 0 9zM12 11v6M9 14l3 3 3-3",
  /** Graphe d'un item (hors navigation, même jeu de tracés). */
  graph:
    "M4 6a2 2 0 1 0 4 0a2 2 0 1 0-4 0M16 6a2 2 0 1 0 4 0a2 2 0 1 0-4 0M10 19a2 2 0 1 0 4 0a2 2 0 1 0-4 0M7 7.8l4 9.4M17 7.8l-4 9.4",
} as const;

export type NavIconName = keyof typeof NAV_ICONS;

export interface NavLink {
  to: string;
  label: string;
  icon: NavIconName;
}

/** Un lien est actif sur sa route et ses sous-routes (`/items` couvre `/items/:id`) ; `/` seulement sur elle-même. */
export function isNavActive(to: string, path: string): boolean {
  if (to === "/") return path === "/";
  return path === to || path.startsWith(`${to}/`);
}

/** Section « Plus » de la barre d'onglets mobile : statistiques, hors ligne et pages d'administration. */
export function isMoreActive(path: string): boolean {
  return (
    path.startsWith("/admin") ||
    isNavActive("/stats", path) ||
    isNavActive("/offline", path)
  );
}

export function useNavigation() {
  const { t } = useI18n();

  const mainLinks = computed<NavLink[]>(() => [
    { to: "/", label: t("layout.nav.dashboard"), icon: "home" },
    { to: "/items", label: t("layout.nav.catalog"), icon: "catalog" },
    {
      to: "/collections",
      label: t("layout.nav.collections"),
      icon: "collections",
    },
    { to: "/stats", label: t("layout.nav.stats"), icon: "stats" },
    { to: "/offline", label: t("layout.nav.offline"), icon: "offline" },
  ]);

  /** Liens de la feuille « Plus » (mobile), avant l'administration. */
  const sheetLinks = computed<NavLink[]>(() => [
    { to: "/stats", label: t("layout.nav.stats"), icon: "stats" },
    { to: "/offline", label: t("layout.nav.offline"), icon: "offline" },
  ]);

  /** Onglets mobiles : libellés courts (le 4ᵉ onglet, « Plus », ouvre une feuille). */
  const tabLinks = computed<NavLink[]>(() => [
    { to: "/", label: t("layout.nav.dashboard"), icon: "home" },
    { to: "/items", label: t("layout.nav.catalog"), icon: "catalog" },
    {
      to: "/collections",
      label: t("layout.nav.collectionsTab"),
      icon: "collections",
    },
  ]);

  const adminLinks = computed<NavLink[]>(() => [
    { to: "/admin/users", label: t("layout.nav.users"), icon: "users" },
    { to: "/admin/domains", label: t("layout.nav.domains"), icon: "domains" },
    {
      to: "/admin/item-types",
      label: t("layout.nav.itemTypes"),
      icon: "itemTypes",
    },
    {
      to: "/admin/relation-types",
      label: t("layout.nav.relationTypes"),
      icon: "relationTypes",
    },
    {
      to: "/admin/collection-types",
      label: t("layout.nav.collectionTypes"),
      icon: "collectionTypes",
    },
    {
      to: "/admin/import-export",
      label: t("layout.nav.importExport"),
      icon: "importExport",
    },
    {
      to: "/admin/external-import",
      label: t("layout.nav.externalImport"),
      icon: "external",
    },
    { to: "/admin/images", label: t("layout.nav.images"), icon: "images" },
  ]);

  return { mainLinks, tabLinks, sheetLinks, adminLinks };
}
