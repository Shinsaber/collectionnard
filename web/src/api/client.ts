import axios from "axios";
import type {
  LoginRequest,
  LoginResponse,
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

const http = axios.create({
  baseURL: "/api/v1",
  headers: { "Content-Type": "application/json" },
});

// Inject JWT token from localStorage
http.interceptors.request.use((config) => {
  const token = localStorage.getItem("token");
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

// Redirect to login on 401
http.interceptors.response.use(
  (r) => r,
  (error) => {
    if (axios.isAxiosError(error) && error.response?.status === 401) {
      localStorage.removeItem("token");
      if (window.location.pathname !== "/login") {
        window.location.href = "/login";
      }
    }
    return Promise.reject(error);
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
