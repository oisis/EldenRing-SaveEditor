import type { ChangedScope } from "../changedScopes";

/**
 * The port the application layer needs in order to read the Inventory and the
 * Storage Box of one character slot. Infrastructure implements it; feature
 * modules depend on it through the hooks in this directory and never on the
 * transport that fulfils it.
 *
 * Both getters are read-only and share one record shape, because the backend
 * reports the same physical fields for both containers. Nothing is added here:
 * no name, no icon, no capacity, no capability, no favourite state and no
 * equipped state. A raw identifier stays a raw identifier until a backend
 * getter names it.
 */

/** One physical container row exactly as the backend reports it. */
export type ItemRecord = {
  ownedItemID: string;
  /** Backend resource kind, carried verbatim; the UI does not interpret it. */
  kind: string;
  /** Canonical catalog resource key; no display name is resolved for it here. */
  key: string;
  gameID: number;
  containerSection: string;
  physicalIndex: number;
  gaItemHandle: number;
  quantity: number;
  acquisitionIndex: number;
};

/**
 * One resolved page of container records. `saveRevision` is opaque: it is
 * carried so a later step can compare it, and it is neither generated,
 * interpreted nor compared here.
 */
export type ItemPage = {
  saveSessionID: string;
  saveRevision: string;
  characterID: number;
  active: boolean;
  records: readonly ItemRecord[];
  total: number;
  page: number;
  pageSize: number;
};

/**
 * The arguments of one container page. They are grouped so that five positional
 * values of three types cannot be transposed silently; the adapter still passes
 * them to the bridge in the order the backend contract defines.
 */
export type ItemPageRequest = {
  saveSessionID: string;
  characterID: number;
  containerSection: string;
  page: number;
  pageSize: number;
};

export type ItemsPort = ItemsMutationPort & {
  /**
   * Reads one page of the Inventory. Section names, the slot range and paging
   * are the backend's contract: the request is passed on exactly as received.
   */
  getInventory: (request: ItemPageRequest) => Promise<ItemPage>;
  /** Reads one page of the Storage Box under the same contract. */
  getStorage: (request: ItemPageRequest) => Promise<ItemPage>;
  /**
   * Reads one authoritative page of one container: the backend applies the
   * search, the category, the favourites and the sort order to the whole
   * container and returns the requested slice of the result.
   */
  getOwnedItems: (request: OwnedItemsRequest) => Promise<OwnedItemsPage>;
};

/**
 * One catalog identity, exactly as the backend names it. It is the only shape a
 * favourite ever takes above this port: a favourite is a presentational
 * preference identified by the canonical `(kind, key)` pair and never a copy of
 * a GameCatalog document or of save data.
 */
export type ResourceIdentity = {
  kind: string;
  key: string;
};

/**
 * The mutations the backend accepts for one owned record. The backend decides
 * every one of them; the interface renders only what is true here and derives
 * nothing from a family, a category or a section of its own.
 */
export type OwnedItemActions = {
  moveToStorage: boolean;
  moveToInventory: boolean;
  remove: boolean;
  setQuantity: boolean;
  reorder: boolean;
};

/**
 * One authoritative container row. Every field is the backend's own value:
 * names, icons, categories and the maximum are resolved by the backend under
 * the active Safety Profile, and nothing here is computed, defaulted or
 * clamped above the port.
 */
export type OwnedItemRow = {
  ownedItemID: string;
  kind: string;
  key: string;
  gameID: number;
  container: string;
  containerSection: string;
  physicalIndex: number;
  acquisitionIndex: number;
  /**
   * The zero-based rank inside the manual Inventory order, counted over the
   * whole container by the backend. Meaningless unless `orderPositionKnown`,
   * which is true exactly for the records a reorder accepts.
   */
  orderPosition: number;
  orderPositionKnown: boolean;
  quantity: number;
  /** The container limit under the active profile; meaningless unless known. */
  maxQuantity: number;
  maxQuantityKnown: boolean;
  family: string;
  category: string;
  subcategory: string;
  /** Empty when the catalog does not know the name; never a synthesised one. */
  name: string;
  /** Embedded catalog path only; turn it into a URL through catalogAssetURL. */
  iconPath: string;
  recordMode: string;
  banRisk: boolean;
  cutContent: boolean;
  dlc: boolean;
  preOrder: boolean;
  actions: OwnedItemActions;
};

/** One category the container holds, with the rows behind it. */
export type OwnedItemCategory = {
  category: string;
  count: number;
};

/** One resolved page of one container, filtered and sorted by the backend. */
export type OwnedItemsPage = {
  saveSessionID: string;
  saveRevision: string;
  characterID: number;
  active: boolean;
  safetyProfile: string;
  container: string;
  records: readonly OwnedItemRow[];
  categories: readonly OwnedItemCategory[];
  total: number;
  page: number;
  pageSize: number;
};

/**
 * The arguments of one authoritative container page. They are grouped so that
 * eleven positional values cannot be transposed silently; the adapter still
 * passes them to the bridge in the order the backend contract defines.
 *
 * Which container names exist, which sort orders are accepted, that an empty
 * filter never filters and which defaults zero paging resolves to are the
 * backend's contract. Nothing here is normalised, trimmed, recased or
 * defaulted, and no safety profile is sent: the backend reads its own.
 */
export type OwnedItemsRequest = {
  saveSessionID: string;
  characterID: number;
  container: string;
  containerSection: string;
  search: string;
  category: string;
  favoritesOnly: boolean;
  favorites: readonly ResourceIdentity[];
  sort: string;
  page: number;
  pageSize: number;
};

/** One requested resource of the shared Add dialog. */
export type AddItemsRequestEntry = {
  kind: string;
  key: string;
  variantID?: number;
  inventoryQuantity: number;
  storageQuantity: number;
};

/**
 * The committed receipt of one item mutation. It is the shared backend receipt
 * and is carried verbatim: the frontend never assembles a receipt, a revision
 * or a scope list of its own.
 */
export type ItemMutationReceipt = {
  operationID: string;
  operationKind: string;
  saveSessionID: string;
  saveRevision: string;
  changedScopes: readonly ChangedScope[];
};

export type ItemsMutationPort = {
  /** One atomic batch add into Inventory, Storage or both. */
  addItemsToContainers: (request: {
    saveSessionID: string;
    characterID: number;
    items: readonly AddItemsRequestEntry[];
    confirmBanRisk: boolean;
    expectedRevision: string;
  }) => Promise<ItemMutationReceipt>;
  /** One atomic batch move of Inventory common records into Storage. */
  moveOwnedItemsToStorage: (request: {
    saveSessionID: string;
    characterID: number;
    ownedItemIDs: readonly string[];
    expectedRevision: string;
  }) => Promise<ItemMutationReceipt>;
  /** One atomic batch move of Storage common records into Inventory. */
  moveOwnedItemsToInventory: (request: {
    saveSessionID: string;
    characterID: number;
    ownedItemIDs: readonly string[];
    expectedRevision: string;
  }) => Promise<ItemMutationReceipt>;
  /** One atomic batch removal. */
  removeOwnedItems: (request: {
    saveSessionID: string;
    characterID: number;
    ownedItemIDs: readonly string[];
    expectedRevision: string;
  }) => Promise<ItemMutationReceipt>;
  /** One atomic anchored group move inside the Inventory order. */
  reorderInventoryItems: (request: {
    saveSessionID: string;
    characterID: number;
    anchorOwnedItemID: string;
    groupOwnedItemIDs: readonly string[];
    targetPosition: number;
    expectedRevision: string;
  }) => Promise<ItemMutationReceipt>;
  /** Sets the stored quantity of one owned record. */
  setOwnedItemQuantity: (request: {
    saveSessionID: string;
    characterID: number;
    ownedItemID: string;
    quantity: number;
    expectedRevision: string;
  }) => Promise<ItemMutationReceipt>;
};
