import axios, { type InternalAxiosRequestConfig } from "axios";
import {
  clearTokens,
  getAccessToken,
  getRefreshToken,
  setTokens,
} from "./tokens";
import type {
  LoginRequest,
  Invitation,
  InvitationCheck,
  InvitationCreate,
  InvitationCreated,
  LoginResponse,
  PasswordChange,
  ProfileUpdate,
  RegisterRequest,
  ResetLink,
  UserUpdate,
  RefreshRequest,
  User,
  ItemType,
  ItemTypeCreate,
  ItemTypeUpdate,
  RelationType,
  RelationTypeCreate,
  RelationTypeUpdate,
  CollectionType,
  CollectionTypeCreate,
  CollectionTypeUpdate,
  Item,
  ItemCreate,
  ItemUpdate,
  OfflineSnapshot,
  DomainInfo,
  DomainInstallReport,
  ItemRelation,
  RelatedItemCollections,
  ItemRelationCreate,
  ItemRelationUpdate,
  ItemCollectionMembership,
  Collection,
  CollectionCreate,
  CollectionUpdate,
  CollectionItem,
  CollectionItemAdd,
  CollectionItemUpdate,
  GraphResponse,
  Stats,
  MeStats,
  NotificationPage,
  Duplicate,
  Proposal,
  ProposalComment,
  ProposalConflict,
  ProposalInput,
  ProposalPayload,
  ItemHistory,
  CollectionStats,
  PublicCollection,
  PublicProfile,
  ShareLink,
  ShareLinkCreated,
  Tag,
  PaginatedResponse,
  PaginationParams,
  ItemListParams,
  CsvDelimiter,
  ConnectorInfo,
  ExternalEditionHit,
  ExternalImportReport,
  ExternalImportRequest,
  ExternalPlan,
  ExternalWorkHit,
  ImportReport,
  UploadPurgeReport,
  UploadResponse,
} from "./types";
import { i18n } from "@/i18n";

const BASE_URL = "/api/v1";

const http = axios.create({
  baseURL: BASE_URL,
  headers: { "Content-Type": "application/json" },
});

// Lectures publiques (P4-2) : servies aussi sans compte. Un jeton refusé et non
// renouvelable ne renvoie pas vers /login : la requête est rejouée en anonyme.
declare module "axios" {
  interface AxiosRequestConfig {
    /** Lecture publique : rejouée sans jeton si la session ne se renouvelle pas. */
    publicRead?: boolean;
    _anonymous?: boolean;
  }
}
const publicRead = { publicRead: true };

// Inject the access token
http.interceptors.request.use((config) => {
  const token = getAccessToken();
  if (token && !config._anonymous) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

// ── Refresh automatique ──────────────────────────────────────────────
// Sur un 401, on tente une fois de renouveler l'access token avec le refresh
// token, puis on rejoue la requête. Un seul refresh est en vol à la fois
// (les requêtes concurrentes attendent la même promesse). En cas d'échec,
// la session est purgée et l'utilisateur renvoyé sur /login.

type RetriableConfig = InternalAxiosRequestConfig & { _retried?: boolean };

// Endpoints dont un 401 ne doit pas déclencher de refresh.
const NO_REFRESH_PATHS = ["/auth/login", "/auth/refresh", "/auth/logout"];

let refreshing: Promise<string | null> | null = null;

async function refreshAccessToken(): Promise<string | null> {
  const refreshToken = getRefreshToken();
  if (!refreshToken) return null;
  try {
    // axios « nu » : on ne veut pas repasser par les intercepteurs de `http`.
    const { data } = await axios.post<LoginResponse>(
      `${BASE_URL}/auth/refresh`,
      { refresh_token: refreshToken } satisfies RefreshRequest,
    );
    setTokens(data.token, data.refresh_token);
    return data.token;
  } catch {
    return null;
  }
}

function redirectToLogin(): void {
  clearTokens();
  if (window.location.pathname !== "/login") {
    window.location.href = "/login";
  }
}

http.interceptors.response.use(
  (r) => r,
  async (error) => {
    if (!axios.isAxiosError(error) || error.response?.status !== 401) {
      return Promise.reject(error);
    }
    const config = error.config as RetriableConfig | undefined;
    const url = config?.url ?? "";
    if (NO_REFRESH_PATHS.some((p) => url.startsWith(p))) {
      // Identifiants invalides ou refresh token mort : pas de retry.
      if (!url.startsWith("/auth/login")) redirectToLogin();
      return Promise.reject(error);
    }
    if (!config || config._retried) {
      redirectToLogin();
      return Promise.reject(error);
    }
    config._retried = true;

    refreshing ??= refreshAccessToken().finally(() => {
      refreshing = null;
    });
    const newToken = await refreshing;
    if (!newToken && config.publicRead) {
      clearTokens();
      config._anonymous = true;
      delete config.headers.Authorization;
      return http(config);
    }
    if (!newToken) {
      redirectToLogin();
      return Promise.reject(error);
    }
    config.headers.Authorization = `Bearer ${newToken}`;
    return http(config);
  },
);

/** Instance HTTP, exposée aux seuls tests (adaptateur simulé) : les vues passent par les `*Api`. */
export const httpForTests = http;

// ── API Error Helper ──────────────────────────────────────────────────
/**
 * Erreur d'appel API prête à afficher : le code `error` de la réponse est traduit
 * (`errors.api.<code>`) ; à défaut, le `message` du serveur (en anglais), puis celui d'axios.
 * `code` vaut `network` quand le serveur n'a pas répondu.
 */
export function extractApiError(e: unknown): {
  code?: string;
  message: string;
  fieldErrors?: Record<string, string>;
} {
  const { t, te } = i18n.global;
  if (axios.isAxiosError(e) && e.response?.data) {
    const data = e.response.data as Record<string, unknown>;
    const code = typeof data.error === "string" ? data.error : undefined;
    const key = `errors.api.${code}`;
    const details = data.details as
      { fields?: Record<string, string>; missing?: string[] } | undefined;
    // `missing` (connector_unavailable) : noms introuvables, insérés dans le message traduit.
    const params = { missing: details?.missing?.join(", ") ?? "?" };
    const message =
      code && te(key)
        ? t(key, params)
        : ((data.message as string) ?? e.message);
    return {
      code,
      message,
      fieldErrors: details?.fields
        ? Object.fromEntries(
            Object.entries(details.fields).map(([k, v]) => [
              k.replace(/^\//, ""),
              v,
            ]),
          )
        : undefined,
    };
  }
  if (axios.isAxiosError(e) && !e.response && e.request) {
    return { code: "network", message: t("errors.network") };
  }
  return { message: e instanceof Error ? e.message : t("errors.unknown") };
}

// ── Auth ──────────────────────────────────────────────────────────────
export const authApi = {
  login: (data: LoginRequest) =>
    http.post<LoginResponse>("/auth/login", data).then((r) => r.data),
  refresh: (refreshToken: string) =>
    http
      .post<LoginResponse>("/auth/refresh", {
        refresh_token: refreshToken,
      } satisfies RefreshRequest)
      .then((r) => r.data),
  logout: (refreshToken: string) =>
    http.post<void>("/auth/logout", {
      refresh_token: refreshToken,
    } satisfies RefreshRequest),
  me: () => http.get<User>("/auth/me").then((r) => r.data),
  /** Invitation encore utilisable ? (public) */
  checkInvitation: (token: string) =>
    http
      .get<InvitationCheck>("/auth/invitation", { params: { token } })
      .then((r) => r.data),
  /** Crée le compte d'une invitation et connecte (public). */
  register: (data: RegisterRequest) =>
    http.post<LoginResponse>("/auth/register", data).then((r) => r.data),
  /** Nouveau mot de passe depuis un lien de réinitialisation (public). */
  resetPassword: (token: string, password: string) =>
    http.post<void>("/auth/reset-password", { token, password }),
};

// ── Mon compte, invitations, comptes (P4-0) ──────────────────────────
export const meApi = {
  updateProfile: (data: ProfileUpdate) =>
    http.put<User>("/me/profile", data).then((r) => r.data),
  /** Révoque toutes les sessions ; renvoie les nouveaux jetons de la session courante. */
  changePassword: (data: PasswordChange) =>
    http.put<LoginResponse>("/me/password", data).then((r) => r.data),
};

export const invitationApi = {
  list: () => http.get<Invitation[]>("/invitations").then((r) => r.data),
  create: (data: InvitationCreate) =>
    http.post<InvitationCreated>("/invitations", data).then((r) => r.data),
  delete: (id: string) => http.delete(`/invitations/${id}`),
};

export const userApi = {
  list: (params?: PaginationParams) =>
    http.get<PaginatedResponse<User>>("/users", { params }).then((r) => r.data),
  update: (id: string, data: UserUpdate) =>
    http.put<User>(`/users/${id}`, data).then((r) => r.data),
  resetLink: (id: string) =>
    http.post<ResetLink>(`/users/${id}/reset-link`).then((r) => r.data),
};

// ── Item Types ────────────────────────────────────────────────────────
export const itemTypeApi = {
  list: (params?: PaginationParams) =>
    http
      .get<PaginatedResponse<ItemType>>("/item-types", { params })
      .then((r) => r.data),
  get: (id: string) =>
    http.get<ItemType>(`/item-types/${id}`).then((r) => r.data),
  create: (data: ItemTypeCreate) =>
    http.post<ItemType>("/item-types", data).then((r) => r.data),
  update: (id: string, data: ItemTypeUpdate) =>
    http.put<ItemType>(`/item-types/${id}`, data).then((r) => r.data),
  delete: (id: string) => http.delete(`/item-types/${id}`),
};

// ── Relation Types ────────────────────────────────────────────────────
export const relationTypeApi = {
  list: (
    params?: PaginationParams & {
      source_item_type_id?: string;
      target_item_type_id?: string;
    },
  ) =>
    http
      .get<PaginatedResponse<RelationType>>("/relation-types", { params })
      .then((r) => r.data),
  get: (id: string) =>
    http.get<RelationType>(`/relation-types/${id}`).then((r) => r.data),
  create: (data: RelationTypeCreate) =>
    http.post<RelationType>("/relation-types", data).then((r) => r.data),
  update: (id: string, data: RelationTypeUpdate) =>
    http.put<RelationType>(`/relation-types/${id}`, data).then((r) => r.data),
  delete: (id: string) => http.delete(`/relation-types/${id}`),
};

// ── Collection Types ──────────────────────────────────────────────────
export const collectionTypeApi = {
  list: (params?: PaginationParams) =>
    http
      .get<PaginatedResponse<CollectionType>>("/collection-types", { params })
      .then((r) => r.data),
  get: (id: string) =>
    http.get<CollectionType>(`/collection-types/${id}`).then((r) => r.data),
  create: (data: CollectionTypeCreate) =>
    http.post<CollectionType>("/collection-types", data).then((r) => r.data),
  update: (id: string, data: CollectionTypeUpdate) =>
    http
      .put<CollectionType>(`/collection-types/${id}`, data)
      .then((r) => r.data),
  delete: (id: string) => http.delete(`/collection-types/${id}`),
};

// ── Items ─────────────────────────────────────────────────────────────
/** Notifications dans l'application (P4-3c). */
export const notificationApi = {
  list: (params?: PaginationParams) =>
    http
      .get<NotificationPage>("/me/notifications", { params })
      .then((r) => r.data),
  /** `ids` absent : toutes. */
  read: (ids?: string[]) =>
    http.post("/me/notifications/read", ids ? { ids } : {}),
};

/** Propositions et modération (P4-3b). */
export const proposalApi = {
  list: (
    params: PaginationParams & { scope?: "mine" | "review"; status?: string },
  ) =>
    http
      .get<PaginatedResponse<Proposal>>("/proposals", { params })
      .then((r) => r.data),
  create: (data: ProposalInput) =>
    http.post<Proposal>("/proposals", data).then((r) => r.data),
  get: (id: string) =>
    http.get<Proposal>(`/proposals/${id}`).then((r) => r.data),
  update: (id: string, data: ProposalInput) =>
    http.put<Proposal>(`/proposals/${id}`, data).then((r) => r.data),
  submit: (id: string) =>
    http.post<Proposal>(`/proposals/${id}/submit`).then((r) => r.data),
  withdraw: (id: string) =>
    http.post<Proposal>(`/proposals/${id}/withdraw`).then((r) => r.data),
  conflicts: (id: string) =>
    http
      .get<ProposalConflict[]>(`/proposals/${id}/conflicts`)
      .then((r) => r.data),
  accept: (
    id: string,
    body: {
      payload?: ProposalPayload;
      conflicts_resolved?: boolean;
      comment?: string;
    } = {},
  ) => http.post<Proposal>(`/proposals/${id}/accept`, body).then((r) => r.data),
  reject: (id: string, comment: string) =>
    http
      .post<Proposal>(`/proposals/${id}/reject`, { comment })
      .then((r) => r.data),
  requestChanges: (id: string, comment: string) =>
    http
      .post<Proposal>(`/proposals/${id}/request-changes`, { comment })
      .then((r) => r.data),
  comments: (id: string) =>
    http
      .get<ProposalComment[]>(`/proposals/${id}/comments`)
      .then((r) => r.data),
  comment: (id: string, body: string) =>
    http
      .post<ProposalComment>(`/proposals/${id}/comments`, { body })
      .then((r) => r.data),
  reviewCount: () =>
    http
      .get<{ count: number }>("/proposals/review-count")
      .then((r) => r.data.count),
};

export const itemApi = {
  /** Items du même type portant déjà un identifiant (`x-identifier`) des attributs (P4-3b). */
  duplicates: (
    itemTypeId: string,
    attributes: Record<string, unknown>,
    excludeId?: string,
  ) =>
    http
      .post<Duplicate[]>("/items/duplicates", {
        item_type_id: itemTypeId,
        attributes,
        exclude_id: excludeId,
      })
      .then((r) => r.data),
  /** Historique de l'item et de ses relations (P4-3a). */
  history: (id: string) =>
    http.get<ItemHistory>(`/items/${id}/history`).then((r) => r.data),
  /** Retour à une version (admins, contributeurs). */
  revert: (id: string, version: number) =>
    http.post<Item>(`/items/${id}/revert`, { version }).then((r) => r.data),
  /** Recherche avancée : `attr.<clé>[.<op>]`, `related_to`, `sort=attr.<clé>` (voir lib/itemSearch). */
  list: (params?: ItemListParams) =>
    http
      .get<PaginatedResponse<Item>>("/items", {
        params,
        // Filtres répétés en `attr.k=a&attr.k=b` (et non `attr.k[]=a`), comme l'attend l'API.
        paramsSerializer: { indexes: null },
      })
      .then((r) => r.data),
  get: (id: string) => http.get<Item>(`/items/${id}`).then((r) => r.data),
  create: (data: ItemCreate) =>
    http.post<Item>("/items", data).then((r) => r.data),
  update: (id: string, data: ItemUpdate) =>
    http.put<Item>(`/items/${id}`, data).then((r) => r.data),
  delete: (id: string) => http.delete(`/items/${id}`),
  relations: (id: string, params?: { relation_type_id?: string }) =>
    http
      .get<ItemRelation[]>(`/items/${id}/relations`, { params })
      .then((r) => r.data),
  /** Items liés présents dans mes collections (indicateur de la liste des relations). */
  relatedCollections: (id: string) =>
    http
      .get<RelatedItemCollections[]>(`/items/${id}/relations/collections`)
      .then((r) => r.data),
  graph: (
    id: string,
    params?: { max_depth?: number; relation_type_ids?: string },
  ) =>
    http
      .get<GraphResponse>(`/items/${id}/graph`, { params })
      .then((r) => r.data),
  collections: (id: string) =>
    http
      .get<ItemCollectionMembership[]>(`/items/${id}/collections`)
      .then((r) => r.data),
};

// ── Relations ─────────────────────────────────────────────────────────
export const relationApi = {
  get: (id: string) =>
    http.get<ItemRelation>(`/relations/${id}`).then((r) => r.data),
  create: (data: ItemRelationCreate) =>
    http.post<ItemRelation>("/relations", data).then((r) => r.data),
  update: (id: string, data: ItemRelationUpdate) =>
    http.put<ItemRelation>(`/relations/${id}`, data).then((r) => r.data),
  delete: (id: string) => http.delete(`/relations/${id}`),
};

// ── Collections ───────────────────────────────────────────────────────
export const collectionApi = {
  list: (params?: PaginationParams & { collection_type_id?: string }) =>
    http
      .get<PaginatedResponse<Collection>>("/collections", { params })
      .then((r) => r.data),
  get: (id: string) =>
    http.get<Collection>(`/collections/${id}`).then((r) => r.data),
  create: (data: CollectionCreate) =>
    http.post<Collection>("/collections", data).then((r) => r.data),
  update: (id: string, data: CollectionUpdate) =>
    http.put<Collection>(`/collections/${id}`, data).then((r) => r.data),
  delete: (id: string) => http.delete(`/collections/${id}`),
  listItems: (
    collectionId: string,
    params?: PaginationParams & { item_type_id?: string; sort?: string },
  ) =>
    http
      .get<PaginatedResponse<CollectionItem>>(
        `/collections/${collectionId}/items`,
        { params },
      )
      .then((r) => r.data),
  addItem: (collectionId: string, data: CollectionItemAdd) =>
    http
      .post<CollectionItem>(`/collections/${collectionId}/items`, data)
      .then((r) => r.data),
  updateItem: (
    collectionId: string,
    itemId: string,
    data: CollectionItemUpdate,
  ) =>
    http
      .put<CollectionItem>(`/collections/${collectionId}/items/${itemId}`, data)
      .then((r) => r.data),
  removeItem: (collectionId: string, itemId: string) =>
    http.delete(`/collections/${collectionId}/items/${itemId}`),
};

// ── Tags ──────────────────────────────────────────────────────────────
export const tagApi = {
  list: (params?: { search?: string }) =>
    http.get<Tag[]>("/tags", { params }).then((r) => r.data),
};

// ── Stats ─────────────────────────────────────────────────────────────
/** Liens de partage d'une de mes collections (P4-2). */
export const shareLinkApi = {
  list: (collectionId: string) =>
    http
      .get<ShareLink[]>(`/collections/${collectionId}/share-links`)
      .then((r) => r.data),
  create: (collectionId: string) =>
    http
      .post<ShareLinkCreated>(`/collections/${collectionId}/share-links`)
      .then((r) => r.data),
  revoke: (collectionId: string, linkId: string) =>
    http.delete(`/collections/${collectionId}/share-links/${linkId}`),
};

/** Lectures publiques (P4-2) : champs cachés déjà retirés par le serveur. */
export const publicApi = {
  profile: (username: string) =>
    http
      .get<PublicProfile>(
        `/public/users/${encodeURIComponent(username)}`,
        publicRead,
      )
      .then((r) => r.data),
  collections: (username: string) =>
    http
      .get<PublicCollection[]>(
        `/public/users/${encodeURIComponent(username)}/collections`,
        publicRead,
      )
      .then((r) => r.data),
  collection: (id: string) =>
    http
      .get<PublicCollection>(`/public/collections/${id}`, publicRead)
      .then((r) => r.data),
  items: (id: string, params?: PaginationParams) =>
    http
      .get<PaginatedResponse<CollectionItem>>(
        `/public/collections/${id}/items`,
        {
          ...publicRead,
          params,
        },
      )
      .then((r) => r.data),
  stats: (id: string) =>
    http
      .get<CollectionStats>(`/public/collections/${id}/stats`, publicRead)
      .then((r) => r.data),
};

/** Collection ouverte par un lien de partage (jeton en paramètre de requête). */
export const sharedApi = {
  collection: (token: string) =>
    http
      .get<PublicCollection>("/share", { ...publicRead, params: { token } })
      .then((r) => r.data),
  items: (token: string, params?: PaginationParams) =>
    http
      .get<PaginatedResponse<CollectionItem>>("/share/items", {
        ...publicRead,
        params: { ...params, token },
      })
      .then((r) => r.data),
  stats: (token: string) =>
    http
      .get<CollectionStats>("/share/stats", {
        ...publicRead,
        params: { token },
      })
      .then((r) => r.data),
};

/** Annuaire des membres (profils visibles des membres ou publics). */
export const memberApi = {
  list: (params?: PaginationParams) =>
    http
      .get<PaginatedResponse<PublicProfile>>("/members", { params })
      .then((r) => r.data),
};

export const statsApi = {
  get: () => http.get<Stats>("/stats").then((r) => r.data),
  /** Synthèse de mes collections (P4-1). */
  mine: () => http.get<MeStats>("/me/stats").then((r) => r.data),
  /** Statistiques d'une de mes collections (P4-1). */
  collection: (id: string) =>
    http.get<CollectionStats>(`/collections/${id}/stats`).then((r) => r.data),
};

// ── Domaines installables ───────────────────────────────────────────────
export const domainApi = {
  list: () => http.get<DomainInfo[]>("/domains").then((r) => r.data),
  /** Installe ou met à jour par fusion ; `dryRun` simule. Le rapport revient aussi en 422. */
  install: (id: string, dryRun: boolean) =>
    http
      .post<DomainInstallReport>(`/domains/${id}/install`, null, {
        params: { dry_run: dryRun },
        validateStatus: (s) => s === 200 || s === 422,
      })
      .then((r) => r.data),
};

// ── Copie hors ligne ───────────────────────────────────────────────────
export const offlineApi = {
  /** Mes collections, leurs items et les méta-types, pour la consultation sans réseau. */
  snapshot: () => http.get<OfflineSnapshot>("/me/offline").then((r) => r.data),
};

// ── Uploads ───────────────────────────────────────────────────────────
export const uploadApi = {
  upload: (file: File) => {
    const form = new FormData();
    form.append("file", file);
    return http
      .post<UploadResponse>("/uploads", form, {
        headers: { "Content-Type": "multipart/form-data" },
      })
      .then((r) => r.data);
  },
  /** Images qu'aucun attribut ne référence (admin) ; `dryRun` liste sans supprimer. */
  purge: (dryRun: boolean) =>
    http
      .post<UploadPurgeReport>("/uploads/purge", null, {
        params: { dry_run: dryRun },
      })
      .then((r) => r.data),
};

// ── Import / export ───────────────────────────────────────────────────
/** Nom de fichier de l'en-tête Content-Disposition (`attachment; filename="x.csv"`). */
export function filenameFromDisposition(
  header: string | undefined,
  fallback: string,
): string {
  const m = /filename\*?=(?:UTF-8'')?"?([^";]+)"?/i.exec(header ?? "");
  return m ? decodeURIComponent(m[1]!) : fallback;
}

/** Télécharge une réponse en pièce jointe (le JWT impose de passer par axios, pas par un lien). */
async function download(
  url: string,
  params: object,
  fallbackName: string,
): Promise<void> {
  const res = await http
    .get<Blob>(url, {
      params,
      responseType: "blob",
      paramsSerializer: { indexes: null },
    })
    .catch(async (e: unknown) => {
      // En `blob`, le corps d'une erreur (APIError JSON) arrive aussi en Blob : on le relit
      // pour que extractApiError / notifyError affichent le code traduit.
      if (axios.isAxiosError(e) && e.response?.data instanceof Blob) {
        try {
          e.response.data = JSON.parse(await e.response.data.text());
        } catch {
          /* corps non JSON : erreur laissée telle quelle */
        }
      }
      throw e;
    });
  const name = filenameFromDisposition(
    res.headers["content-disposition"] as string | undefined,
    fallbackName,
  );
  const href = URL.createObjectURL(res.data);
  const a = document.createElement("a");
  a.href = href;
  a.download = name;
  document.body.appendChild(a);
  a.click();
  a.remove();
  URL.revokeObjectURL(href);
}

export const importExportApi = {
  /** Catalogue complet en JSON (admin). */
  exportCatalog: () => download("/export", {}, "collectionnard-catalog.json"),
  /**
   * Importe un fichier d'export (admin). Renvoie le rapport aussi pour un 422 (erreurs de
   * validation, rien d'appliqué) ; les autres erreurs sont levées.
   */
  importCatalog: (file: Blob, dryRun: boolean) =>
    http
      .post<ImportReport>("/import", file, {
        params: { dry_run: dryRun },
        headers: { "Content-Type": "application/json" },
        validateStatus: (s) => s === 200 || s === 422,
      })
      .then((r) => r.data),
  /** Importe un export de collection dans le compte courant (même contrat que importCatalog). */
  importCollection: (file: Blob, dryRun: boolean) =>
    http
      .post<ImportReport>("/collections/import", file, {
        params: { dry_run: dryRun },
        headers: { "Content-Type": "application/json" },
        validateStatus: (s) => s === 200 || s === 422,
      })
      .then((r) => r.data),
  /** Items d'un type en CSV, avec les filtres de la recherche (item_type_id requis). */
  exportItemsCsv: (
    params: ItemListParams & { delimiter?: CsvDelimiter; lang?: string },
  ) => download("/items/export", params, "items.csv"),
  /** Une collection de l'utilisateur, en JSON ou CSV. */
  exportCollection: (
    id: string,
    format: "json" | "csv",
    opts: { delimiter?: CsvDelimiter; lang?: string } = {},
  ) =>
    download(
      `/collections/${id}/export`,
      { format, ...opts },
      `collection.${format}`,
    ),
};

// ── Sources externes (un connecteur par sorte de collection, admin) ───
const external = (connector: string) => `/external/${connector}`;

export const externalApi = {
  /** Connecteurs disponibles et domaine (paquet de types) de chacun. */
  connectors: () =>
    http.get<ConnectorInfo[]>("/external/connectors").then((r) => r.data),
  /** Œuvres (Wikidata, albums MusicBrainz…), par titre. */
  searchWorks: (connector: string, q: string) =>
    http
      .get<ExternalWorkHit[]>(`${external(connector)}/works`, {
        params: { q },
      })
      .then((r) => r.data),
  /** Éditions, par titre (`q`), ISBN, code-barres ou œuvre. */
  searchEditions: (
    connector: string,
    params:
      { q: string } | { isbn: string } | { barcode: string } | { work: string },
  ) =>
    http
      .get<ExternalEditionHit[]>(`${external(connector)}/editions`, { params })
      .then((r) => r.data),
  /** Items et relations proposés pour une sélection, rapprochés du catalogue. */
  preview: (connector: string, req: ExternalImportRequest) =>
    http
      .post<ExternalPlan>(`${external(connector)}/preview`, req)
      .then((r) => r.data),
  /** Simule ou importe les nœuds de `req.selected` ; renvoie aussi le rapport d'un 422. */
  importSelection: (
    connector: string,
    req: ExternalImportRequest,
    dryRun: boolean,
  ) =>
    http
      .post<ExternalImportReport>(`${external(connector)}/import`, req, {
        params: { dry_run: dryRun },
        validateStatus: (s) => s === 200 || s === 422,
      })
      .then((r) => r.data),
  /** Membre (P4-3b) : la sélection devient une proposition soumise. */
  propose: (connector: string, req: ExternalImportRequest) =>
    http
      .post<Proposal>(`${external(connector)}/propose`, req)
      .then((r) => r.data),
};
