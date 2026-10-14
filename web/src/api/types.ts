// Types matching the OpenAPI spec and Go server models

// ---------- Auth ----------
export interface LoginRequest {
  username: string;
  password: string;
}

export interface LoginResponse {
  token: string; // access token JWT (courte durée)
  refresh_token: string; // token opaque, rotation à chaque refresh
  expires_in: number; // durée de vie de `token` en secondes
  user: User;
}

export interface RefreshRequest {
  refresh_token: string;
}

// ---------- User ----------
export interface User {
  id: string;
  username: string;
  email: string;
  role: "admin" | "user";
  created_at: string;
  updated_at: string;
}

// ---------- ItemType ----------
export interface ItemType {
  id: string;
  name: string;
  description?: string;
  attributes_schema: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export interface ItemTypeCreate {
  name: string;
  description?: string;
  attributes_schema: Record<string, unknown>;
}

export interface ItemTypeUpdate {
  name?: string;
  description?: string;
  attributes_schema?: Record<string, unknown>;
}

// ---------- RelationType ----------
export interface RelationType {
  id: string;
  name: string;
  description?: string;
  source_item_type_ids: string[];
  target_item_type_ids: string[];
  source_item_types?: ItemType[];
  target_item_types?: ItemType[];
  attributes_schema: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export interface RelationTypeCreate {
  name: string;
  description?: string;
  source_item_type_ids: string[];
  target_item_type_ids: string[];
  attributes_schema?: Record<string, unknown>;
}

export interface RelationTypeUpdate {
  name?: string;
  description?: string;
  source_item_type_ids?: string[];
  target_item_type_ids?: string[];
  attributes_schema?: Record<string, unknown>;
}

// ---------- CollectionType ----------
export interface CollectionType {
  id: string;
  name: string;
  description?: string;
  attributes_schema: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export interface CollectionTypeCreate {
  name: string;
  description?: string;
  attributes_schema?: Record<string, unknown>;
}

export interface CollectionTypeUpdate {
  name?: string;
  description?: string;
  attributes_schema?: Record<string, unknown>;
}

// ---------- Tag ----------
export interface Tag {
  id: string;
  name: string;
}

// ---------- Item ----------
export interface Item {
  id: string;
  item_type_id: string;
  item_type?: ItemType;
  attributes: Record<string, unknown>;
  tags?: Tag[];
  created_at: string;
  updated_at: string;
}

export interface ItemCreate {
  item_type_id: string;
  attributes: Record<string, unknown>;
  tags?: string[];
}

export interface ItemUpdate {
  attributes?: Record<string, unknown>;
  tags?: string[];
}

// ---------- ItemRelation ----------
export interface ItemRelation {
  id: string;
  relation_type_id: string;
  relation_type?: RelationType;
  source_item_id: string;
  source_item?: Item;
  target_item_id: string;
  target_item?: Item;
  attributes: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export interface ItemRelationCreate {
  relation_type_id: string;
  source_item_id: string;
  target_item_id: string;
  attributes?: Record<string, unknown>;
}

export interface ItemRelationUpdate {
  attributes: Record<string, unknown>;
}

// ---------- Collection ----------
export interface Collection {
  id: string;
  user_id: string;
  collection_type_id: string;
  collection_type?: CollectionType;
  name: string;
  description?: string;
  item_count: number;
  created_at: string;
  updated_at: string;
}

export interface CollectionCreate {
  collection_type_id: string;
  name: string;
  description?: string;
}

export interface CollectionUpdate {
  name?: string;
  description?: string;
}

// ---------- CollectionItem ----------
export interface CollectionItem {
  id: string;
  collection_id: string;
  item_id: string;
  item?: Item;
  attributes: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export interface CollectionItemAdd {
  item_id: string;
  attributes?: Record<string, unknown>;
}

export interface CollectionItemUpdate {
  attributes: Record<string, unknown>;
}

// ---------- Item Collection Membership ----------
export interface ItemCollectionMembership {
  collection: Collection;
  attributes: Record<string, unknown>;
}

// ---------- Graph ----------
export interface GraphNode {
  id: string;
  type: string;
  label: string;
  attributes: Record<string, unknown>;
  depth: number;
}

export interface GraphEdge {
  source_id: string;
  target_id: string;
  relation_type: string;
  attributes: Record<string, unknown>;
}

export interface GraphResponse {
  root: GraphNode;
  nodes: GraphNode[];
  edges: GraphEdge[];
  truncated: boolean; // exploration arrêtée par la limite de nœuds du serveur
}

// ---------- Stats ----------
export interface ItemsByType {
  type_name: string;
  count: number;
}

export interface Stats {
  items_by_type: ItemsByType[];
  total_items: number;
  total_collections: number;
  total_relations: number;
}

// ---------- Pagination ----------
export interface PaginatedResponse<T> {
  data: T[];
  total: number;
  page: number;
  per_page: number;
  total_pages: number;
}

export interface PaginationParams {
  page?: number;
  per_page?: number;
}

// ---------- Item search (GET /items) ----------
export type ItemListParams = PaginationParams & {
  item_type_id?: string;
  search?: string;
  tag?: string;
  sort?: string;
  related_to?: string;
  relation_type_id?: string;
  /** Filtres d'attributs : `attr.<clé>`, `attr.<clé>.gte|lte|contains`. */
  [attr: `attr.${string}`]: string | string[] | undefined;
};

// ---------- API Error ----------
export interface APIError {
  error: string;
  message: string;
  details?: Record<string, unknown>;
}

// ---------- Upload ----------
export interface UploadResponse {
  url: string;
  filename: string;
  size: number;
}

// ---------- Import / export ----------
export type CsvDelimiter = "comma" | "semicolon" | "tab";

export interface ImportCounts {
  created: number;
  updated: number;
  unchanged: number;
  /** Import de collection : items retirés car absents du fichier. */
  removed?: number;
}

export type ImportEntity =
  | "file"
  | "item_types"
  | "collection_types"
  | "relation_types"
  | "items"
  | "relations"
  | "collection";

export interface ImportError {
  entity: ImportEntity;
  /** Position dans la liste (0-based), -1 pour une erreur de fichier. */
  index: number;
  id?: string;
  message: string;
}

/** Rapport de POST /import (200 si appliqué ou simulation sans erreur, 422 sinon). */
export interface ImportReport {
  dry_run: boolean;
  applied: boolean;
  /** Import de collection : collection créée ou mise à jour. */
  collection_id?: string;
  counts: Partial<Record<ImportEntity, ImportCounts>>;
  errors: ImportError[];
}

// ---------- Sources externes (connecteur BD / Manga) ----------
/** Œuvre trouvée sur Wikidata (GET /external/bd-manga/works). */
export interface ExternalWorkHit {
  wikidata_id: string;
  label: string;
  description?: string;
  item_type_id: string;
  start_year?: number;
  existing_item_id?: string;
  source_url: string;
}

export interface ExternalVolumeHit {
  bnf_ark: string;
  number?: number;
  title: string;
  isbn?: string;
}

/** Édition française de la BnF (GET /external/bd-manga/editions). */
export interface ExternalEditionHit {
  bnf_ark: string;
  title: string;
  publishers: string[];
  start_year?: number;
  existing_item_id?: string;
  source_url: string;
  /** Recherche par ISBN : le tome qui porte cet ISBN. */
  matched_volume?: ExternalVolumeHit;
}

export interface ExternalImportRequest {
  work?: string;
  editions?: string[];
  /** Import seulement : clés des nœuds à écrire. */
  selected?: string[];
  /** Import seulement : attributs corrigés dans l'aperçu (clé de nœud → attributs complets). */
  overrides?: Record<string, Record<string, unknown>>;
}

export type ExternalNodeRole =
  "work" | "universe" | "person" | "edition" | "publisher" | "volume";

export type ExternalNodeStatus = "new" | "update" | "unchanged";

export interface ExternalNode {
  key: string;
  role: ExternalNodeRole;
  item_type_id: string;
  /** Ce qui serait écrit : valeurs existantes complétées par la source. */
  attributes: Record<string, unknown>;
  existing_item_id?: string;
  status: ExternalNodeStatus;
  /** Sélection par défaut. */
  selected: boolean;
  parent?: string;
  source_url: string;
  note?: "reprint" | "no_number";
}

export interface ExternalRelation {
  relation_type_id: string;
  from: string;
  to: string;
  attributes: Record<string, unknown>;
  existing_id?: string;
}

/** Aperçu de POST /external/bd-manga/preview. */
export interface ExternalPlan {
  retrieved_at: string;
  nodes: ExternalNode[];
  relations: ExternalRelation[];
}

/** Rapport de POST /external/bd-manga/import : celui de /import + l'id de chaque item écrit. */
export interface ExternalImportReport extends ImportReport {
  item_ids: Record<string, string>;
}
