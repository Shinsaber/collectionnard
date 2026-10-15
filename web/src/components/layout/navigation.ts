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
  importExport: "M12 3v12M7 10l5 5 5-5M4 21h16",
  external:
    "M7 18a5 5 0 0 1-.6-9.96A6 6 0 0 1 18 9a4 4 0 0 1 0 9zM12 11v6M9 14l3 3 3-3",
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

/** Section « Plus » de la barre d'onglets mobile : réglages et pages d'administration. */
export function isMoreActive(path: string): boolean {
  return path.startsWith("/admin");
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
  ]);

  return { mainLinks, tabLinks, adminLinks };
}
