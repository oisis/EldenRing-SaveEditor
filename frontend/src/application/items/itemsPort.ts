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

export type ItemsPort = {
  /**
   * Reads one page of the Inventory. Section names, the slot range and paging
   * are the backend's contract: the request is passed on exactly as received.
   */
  getInventory: (request: ItemPageRequest) => Promise<ItemPage>;
  /** Reads one page of the Storage Box under the same contract. */
  getStorage: (request: ItemPageRequest) => Promise<ItemPage>;
};
