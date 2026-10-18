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
  /** Profil (P4-0) : nom affiché (repli sur username), présentation, avatar (URL d'un envoi). */
  display_name?: string;
  bio?: string;
  avatar?: string;
  /** Compte désactivé depuis cette date. */
  disabled_at?: string;
  /** GET /auth/me seulement : peut créer des invitations. */
  profile_visibility?: Visibility;
  can_invite?: boolean;
  created_at: string;
  updated_at: string;
}

// ---------- Comptes, invitations, profils (P4-0, social.md) ----------
export interface UserUpdate {
  role?: "admin" | "user";
  disabled?: boolean;
}

export interface ProfileUpdate {
  display_name: string;
  bio: string;
  avatar: string;
  profile_visibility: Visibility;
}

export interface PasswordChange {
  current_password: string;
  new_password: string;
}

export type InvitationStatus = "pending" | "used" | "expired";

export interface Invitation {
  id: string;
  role: "admin" | "user";
  note?: string;
  created_by?: string;
  created_by_username?: string;
  expires_at: string;
  status: InvitationStatus;
  used_at?: string;
  used_by_username?: string;
  created_at: string;
}

export interface InvitationCreate {
  role?: "admin" | "user";
  note?: string;
  expires_in_days?: number;
}

/** Le jeton n'est montré qu'une fois : le lien `/register?token=…` se construit côté client. */
export interface InvitationCreated {
  invitation: Invitation;
  token: string;
}

export interface InvitationCheck {
  valid: boolean;
  expires_at?: string;
}

export interface RegisterRequest {
  token: string;
  username: string;
  email: string;
  password: string;
}

export interface ResetLink {
  token: string;
  expires_at: string;
}

// ---------- ItemType ----------
export interface ItemType {
  id: string;
  name: string;
  description?: string;
  attributes_schema: Record<string, unknown>;
  /** Ordre d'affichage (petit = en premier, puis par nom) ; défaut 100. */
  display_order: number;
  created_at: string;
  updated_at: string;
}

export interface ItemTypeCreate {
  name: string;
  description?: string;
  attributes_schema: Record<string, unknown>;
  display_order?: number;
}

export interface ItemTypeUpdate {
  name?: string;
  description?: string;
  attributes_schema?: Record<string, unknown>;
  display_order?: number;
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
  /** Types d'item permis dans ses collections ; vide = tous. */
  item_type_ids: string[];
  created_at: string;
  updated_at: string;
}

export interface CollectionTypeCreate {
  name: string;
  description?: string;
  attributes_schema?: Record<string, unknown>;
  item_type_ids?: string[];
}

export interface CollectionTypeUpdate {
  name?: string;
  description?: string;
  attributes_schema?: Record<string, unknown>;
  /** Remplace la liste ; `[]` = tous les types. */
  item_type_ids?: string[];
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
  visibility: Visibility;
  /** Champs cachés aux autres lecteurs ; `null` = les propriétés `x-private` du type. */
  hidden_fields: string[] | null;
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
  visibility?: Visibility;
  /** Absent : inchangé ; `null` : défaut du type (`x-private`). */
  hidden_fields?: string[] | null;
}

// ---------- Visibilité et partage (P4-2) : miroir de server/internal/model/sharing.go ----------

export type Visibility = "private" | "members" | "public";

export interface Owner {
  username: string;
  display_name?: string;
  avatar?: string;
}

/** Collection lue par un autre que son propriétaire : schéma du type sans les champs cachés. */
export interface PublicCollection {
  id: string;
  name: string;
  description?: string;
  collection_type: CollectionType;
  item_count: number;
  owner: Owner;
  visibility: Visibility;
  created_at: string;
  updated_at: string;
}

export interface PublicProfile {
  username: string;
  display_name?: string;
  bio?: string;
  avatar?: string;
  profile_visibility: Visibility;
  created_at: string;
}

export interface ShareLink {
  id: string;
  collection_id: string;
  created_at: string;
}

export interface ShareLinkCreated {
  share_link: ShareLink;
  token: string;
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
/** Collection qui contient un item lié, avec les attributs contextuels de l'item. */
export interface RelatedCollection {
  id: string;
  name: string;
  collection_type_id: string;
  attributes: Record<string, unknown>;
}

/** Item lié présent dans des collections de l'utilisateur (GET /items/{id}/relations/collections). */
export interface RelatedItemCollections {
  item_id: string;
  collections: RelatedCollection[];
}

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
  type_id: string;
  type_name: string;
  count: number;
}

// --- Statistiques personnelles (P4-1) : miroir de server/internal/model/stats.go ---

export interface NumberStat {
  field: string;
  count: number;
  sum: number;
  avg: number;
  min: number;
  max: number;
}

export interface ValueCount {
  value: string;
  count: number;
}

export interface Distribution {
  field: string;
  values: ValueCount[];
  missing: number;
}

export interface MonthCount {
  /** « 2026-10 » */
  month: string;
  count: number;
}

export interface Timeline {
  field: string;
  months: MonthCount[];
}

export interface StatsItem {
  id: string;
  item_type_id: string;
  attributes: Record<string, unknown>;
}

export interface Completeness {
  item: StatsItem;
  owned: number;
  total: number;
}

export interface CollectionStats {
  collection_id: string;
  total_items: number;
  items_by_type: ItemsByType[];
  numbers: NumberStat[];
  distributions: Distribution[];
  timelines: Timeline[];
  added: MonthCount[];
  completeness: Completeness[];
  completeness_truncated: boolean;
  /** Types d'item cités (libellés, y compris pour un lecteur anonyme). */
  item_types: ItemType[];
}

export interface CollectionSummary {
  id: string;
  name: string;
  collection_type_id: string;
  item_count: number;
}

export interface MeStats {
  total_collections: number;
  total_entries: number;
  distinct_items: number;
  items_by_type: ItemsByType[];
  collections: CollectionSummary[];
  added: MonthCount[];
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
  /** Fichier identique déjà présent : même URL, rien de réécrit. */
  reused: boolean;
}

// ---------- Purge des images (POST /uploads/purge) ----------
export interface UploadFileInfo {
  filename: string;
  size: number;
  modified_at: string;
}

export interface UploadPurgeReport {
  dry_run: boolean;
  applied: boolean;
  /** Images inutilisées plus anciennes que le délai de grâce (supprimées si `applied`). */
  files: UploadFileInfo[];
  total_size: number;
  referenced: number;
  recent: number;
  grace_hours: number;
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
/** Œuvre trouvée par un connecteur : œuvre Wikidata (livres), groupe de parutions MusicBrainz (album). */
export interface ExternalWorkHit {
  /** Identifiant à la source : le `work` d'une sélection. */
  id: string;
  wikidata_id?: string;
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

/** Édition trouvée par un connecteur : notice BnF (livres), parution MusicBrainz (musique). */
export interface ExternalEditionHit {
  /** Identifiant à la source : l'un des `editions` d'une sélection. */
  id: string;
  bnf_ark?: string;
  title: string;
  publishers: string[];
  start_year?: number;
  /** Langue du texte (ISO 639-1, sinon code BnF à trois lettres). */
  language?: string;
  /** Collection éditoriale et numéro : « Folio 2 ». */
  collection?: string;
  /** Roman en un volume : l'ISBN du livre. */
  isbn?: string;
  /** Musique : artiste crédité, supports (« 2×12" Vinyl »), pays, code-barres, référence catalogue. */
  artist?: string;
  format?: string;
  country?: string;
  barcode?: string;
  catalog_number?: string;
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
  | "work"
  | "universe"
  | "person"
  | "edition"
  | "season"
  | "publisher"
  | "volume";

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

/** Aperçu de POST /external/{connector}/preview. */
export interface ExternalPlan {
  retrieved_at: string;
  nodes: ExternalNode[];
  relations: ExternalRelation[];
}

/** Rapport de POST /external/{connector}/import : celui de /import + l'id de chaque item écrit. */
export interface ExternalImportReport extends ImportReport {
  item_ids: Record<string, string>;
}

// ---------- Copie hors ligne (GET /me/offline, lot E5) ----------
export interface OfflineEntry {
  collection_id: string;
  item_id: string;
  attributes: Record<string, unknown>;
  added_at: string;
}

export interface OfflineSnapshot {
  generated_at: string;
  user: User;
  item_types: ItemType[];
  collection_types: CollectionType[];
  /** Sans `collection_type` joint (voir `collection_types`). */
  collections: Collection[];
  /** Une fois chacun, sans `item_type` joint (voir `item_types`). */
  items: Item[];
  entries: OfflineEntry[];
}

// ---------- Domaines installables (lot P3-0) ----------
export interface DomainInfo {
  id: string;
  title: Record<string, string>;
  description: Record<string, string>;
  package_version: number;
  installed_version?: number;
  /** Paquets requis, installés avant celui-ci. */
  requires: string[];
  status: "not_installed" | "up_to_date" | "update_available";
  /** Au moins un type installé diffère de la version du paquet. */
  modified_locally: boolean;
  item_types: number;
  relation_types: number;
  collection_types: number;
}

export type DomainOutcome =
  "created" | "updated" | "merged" | "unchanged" | "skipped";

export interface DomainInstallEntry {
  entity: "item_types" | "collection_types" | "relation_types";
  id: string;
  name: string;
  outcome: DomainOutcome;
  /** Chemins modifiés des deux côtés : valeur locale gardée. */
  conflicts?: string[];
  missing_types?: string[];
}

export interface DomainInstallReport {
  dry_run: boolean;
  applied: boolean;
  domain: string;
  from_version?: number;
  to_version: number;
  entries: DomainInstallEntry[];
  errors: ImportReport["errors"];
  /** Paquets requis non installés : installés d'abord, dans la même transaction. */
  required?: DomainInstallReport[];
}

/** Connecteur de sources externes (GET /external/connectors). */
export interface ConnectorInfo {
  id: string;
  /** Paquet de domaine qui porte les types écrits par le connecteur. */
  domain: string;
  /** Les éditions se listent depuis une œuvre (`editions?work=`). */
  editions_by_work: boolean;
  /** Code de recherche d'une édition : `editions?isbn=` ou `editions?barcode=`. */
  edition_code: "isbn" | "barcode";
}
