import axios, { type InternalAxiosRequestConfig } from "axios";
import {
  clearTokens,
  getAccessToken,
  getRefreshToken,
  setTokens,
} from "./tokens";
import type {
  LoginRequest,
  LoginResponse,
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
  ItemRelation,
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
  Tag,
  PaginatedResponse,
  PaginationParams,
  UploadResponse,
} from "./types";

const BASE_URL = "/api/v1";

const http = axios.create({
  baseURL: BASE_URL,
  headers: { "Content-Type": "application/json" },
});

// Inject the access token
http.interceptors.request.use((config) => {
  const token = getAccessToken();
  if (token) {
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
    if (!newToken) {
      redirectToLogin();
      return Promise.reject(error);
    }
    config.headers.Authorization = `Bearer ${newToken}`;
    return http(config);
  },
);

// ── API Error Helper ──────────────────────────────────────────────────
export function extractApiError(e: unknown): {
  message: string;
  fieldErrors?: Record<string, string>;
} {
  if (axios.isAxiosError(e) && e.response?.data) {
    const data = e.response.data as Record<string, unknown>;
    const message = (data.message as string) ?? e.message;
    const details = data.details as
      | { fields?: Record<string, string> }
      | undefined;
    return {
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
  return { message: e instanceof Error ? e.message : "Erreur inconnue" };
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
export const itemApi = {
  list: (
    params?: PaginationParams & {
      item_type_id?: string;
      search?: string;
      tag?: string;
      sort?: string;
    },
  ) =>
    http.get<PaginatedResponse<Item>>("/items", { params }).then((r) => r.data),
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
      .get<
        PaginatedResponse<CollectionItem>
      >(`/collections/${collectionId}/items`, { params })
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
export const statsApi = {
  get: () => http.get<Stats>("/stats").then((r) => r.data),
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
};
