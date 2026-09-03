/**
 * The port the application layer needs in order to read GameCatalog resources.
 * Infrastructure implements it; feature modules depend on it through the hook
 * in this directory and never on the transport that fulfils it.
 *
 * The catalog is global: it belongs to no save session and is deliberately not
 * part of the Items module, because Item Database, Equipment, Templates and the
 * pickers all read the same list. Every type here is an exact projection of the
 * backend result. Nothing is added: no icon, no description, no capability, no
 * provenance, no limit, no favourite state and no safety level. `kind` and
 * `family` stay plain strings so a value the backend adds later cannot be
 * rejected by a frontend enum.
 */

/** One catalog row exactly as the backend reports it. */
export type CatalogResourceSummary = {
  /** Backend resource kind, carried verbatim; the UI does not interpret it. */
  kind: string;
  /** Canonical catalog resource key. */
  key: string;
  /** Item family, or the empty string for a resource that has none. */
  family: string;
  /**
   * The display name the catalog knows. An unknown name is the empty string and
   * stays empty: a name synthesised from the key would be indistinguishable
   * from a real one.
   */
  name: string;
};

/** One exact identity requested from the lightweight presentation batch. */
export type CatalogResourcePresentationIdentity = {
  kind: string;
  key: string;
};

/** Scalar-only presentation metadata returned for one exact identity. */
export type CatalogResourcePresentationSummary = CatalogResourcePresentationIdentity & {
  /** Empty when the catalog does not know the resource name. */
  name: string;
  /** Embedded catalog path only; turn it into a URL through catalogAssetURL. */
  iconPath: string;
};

export type CatalogResourcePresentationSummaries = {
  resources: readonly CatalogResourcePresentationSummary[];
};

/** One resolved page of catalog resources. */
export type CatalogResourcesPage = {
  resources: readonly CatalogResourceSummary[];
  /** Every resource that passed the filters, before paging. */
  total: number;
  /** The page the backend actually served, which may differ from the request. */
  page: number;
  /** The page size the backend actually applied, including its own default. */
  pageSize: number;
};

/**
 * The arguments of one catalog page. They are grouped so that seven positional
 * values of two types cannot be transposed silently; the adapter still passes
 * them to the bridge in the order the backend contract defines.
 *
 * Which filter values are accepted, which are rejected, that an empty filter
 * never filters and which defaults zero paging resolves to are the backend's
 * contract. Nothing here is normalised, trimmed, recased or defaulted.
 */
export type CatalogResourcesRequest = {
  resourceType: string;
  family: string;
  capability: string;
  endpointID: string;
  search: string;
  page: number;
  pageSize: number;
};

/**
 * Where one catalog value came from, carried verbatim. It is the backend's own
 * record and is never summarised, scored or turned into a trust level here: the
 * three optional parts are the empty string when the backend omits them, which
 * is exactly what its `omitempty` encoding of an empty string means.
 */
export type CatalogProvenance = {
  source: string;
  method: string;
  table: string;
  row: string;
  field: string;
};

/**
 * One catalog value together with what the backend knows about it. `known`
 * false keeps the raw value the backend sent — a zero, an empty string or a
 * false — because a substituted placeholder would be indistinguishable from a
 * real value.
 */
export type CatalogFact<T> = {
  known: boolean;
  value: T;
  provenance: CatalogProvenance;
};

/**
 * One capability of an item. A capability the backend reports without rules
 * keeps `rules` null; no rule set is invented for it, and `enabled` is never
 * derived from the presence of rules or the other way round.
 */
export type CatalogCapability<R> = {
  known: boolean;
  enabled: boolean;
  rules: R | null;
  provenance: CatalogProvenance;
};

export type CatalogUpgradeRules = {
  model: string;
  maxLevel: number;
  /** Null when the backend reports no SaveForge-verified maximum. */
  maxLevelSFV: CatalogFact<number> | null;
};

export type CatalogInfusionRules = {
  allowedAffinities: readonly string[];
};

export type CatalogAshOfWarMountRules = {
  mode: string;
  weaponType: string;
  compatibilityBit: number;
};

export type CatalogStackRules = {
  maxPerStack: number;
};

export type CatalogEquipmentRules = {
  allowedSlots: readonly string[];
};

/** The presentation texts of an item, each with its own provenance. */
export type CatalogItemPresentation = {
  name: CatalogFact<string>;
  caption: CatalogFact<string>;
  description: CatalogFact<string>;
  location: CatalogFact<string>;
  /**
   * The icon path as catalog metadata only. No icon is loaded, resolved or
   * turned into a URL anywhere above this port.
   */
  iconPath: CatalogFact<string>;
};

/**
 * The raw storage limits of an item. Nothing is combined here: no effective
 * limit is computed, and which of these the current mode applies is a decision
 * that belongs to a later step with a backend contract behind it.
 */
export type CatalogItemStorage = {
  recordMode: CatalogFact<string>;
  maxInventory: CatalogFact<number>;
  safeModeMaxInventory: CatalogFact<number> | null;
  maxInventorySFV: CatalogFact<number> | null;
  maxStorage: CatalogFact<number>;
  safeModeMaxStorage: CatalogFact<number> | null;
  maxStorageSFV: CatalogFact<number> | null;
};

/**
 * The safety facts of an item, exactly as the backend reports them. No risk
 * level, no severity and no ordering is derived from them here.
 */
export type CatalogItemSafety = {
  cutContent: CatalogFact<boolean>;
  banRisk: CatalogFact<boolean>;
  dlc: CatalogFact<boolean>;
  noDatabase: CatalogFact<boolean>;
  scalesWithNG: CatalogFact<boolean>;
  preOrder: CatalogFact<boolean>;
};

export type CatalogItemCapabilities = {
  upgrade: CatalogCapability<CatalogUpgradeRules>;
  infusion: CatalogCapability<CatalogInfusionRules>;
  ashOfWarMount: CatalogCapability<CatalogAshOfWarMountRules>;
  stack: CatalogCapability<CatalogStackRules>;
  equipment: CatalogCapability<CatalogEquipmentRules>;
};

/**
 * The common part of one item document: the data every item family carries.
 * Acquisition, modifiers, links, variants, aliases, unlocks, related technical
 * records, source records, the family-specific statistics and the capability
 * rules evidence are deliberately absent; they belong to later steps with their
 * own contracts, and adding them here early would mean guessing which of them a
 * screen needs.
 *
 * `family`, `category`, `subcategory` and `recordMode` stay plain strings so a
 * value the backend adds later cannot be rejected by a frontend enum.
 */
export type CatalogItemDetail = {
  gameID: CatalogFact<number>;
  family: CatalogFact<string>;
  category: CatalogFact<string>;
  subcategory: CatalogFact<string>;
  presentation: CatalogItemPresentation;
  storage: CatalogItemStorage;
  safety: CatalogItemSafety;
  capabilities: CatalogItemCapabilities;
};

/**
 * One resolved catalog resource. `kind` and `key` are the identity the backend
 * resolved, carried verbatim. `item` is null for every resource of another
 * kind: no other document is mapped onto the item shape.
 */
export type CatalogResourceDetail = {
  kind: string;
  key: string;
  item: CatalogItemDetail | null;
};

/**
 * The identity of one catalog resource. Neither value is trimmed, recased,
 * parsed or matched through an alias: the exact pair is the backend's contract,
 * and an unknown or malformed one is its rejection to make.
 */
export type CatalogResourceRequest = {
  kind: string;
  key: string;
};

/**
 * One item variant exactly as the backend reports it. Only the five identity
 * and provenance facts of the variant are carried: `data` and `sourceRecords`
 * stay in the transport result, and no variant is materialised into an item
 * document here. `kind` and `affinity` stay plain strings so a value the
 * backend adds later cannot be rejected by a frontend enum.
 */
export type CatalogItemVariantSummary = {
  gameID: CatalogFact<number>;
  kind: CatalogFact<string>;
  affinity: CatalogFact<string>;
  upgradeLevel: CatalogFact<number>;
  sourceRowID: CatalogFact<number>;
};

/**
 * Every variant of one item, in catalog order. An item that carries none is a
 * valid answer and arrives as an empty list, never as a rejection and never as
 * a synthesised base variant.
 */
export type CatalogItemVariantsResult = {
  variants: readonly CatalogItemVariantSummary[];
};

/**
 * The identity whose variants are read. Neither value is trimmed, recased,
 * parsed or matched through an alias, and only the exact item kind carries
 * variants: that is the backend's contract, and every rejection of it is its
 * own.
 */
export type CatalogItemVariantsRequest = {
  kind: string;
  key: string;
};

/**
 * One Item Database row exactly as the backend reports it. Visibility, order
 * and paging are the backend's decisions under the active Safety Profile; the
 * three safety flags are carried so a row can be badged without a second call
 * and are never turned into a risk level or an ordering here.
 */
export type CatalogItemDatabaseEntry = {
  kind: string;
  key: string;
  /** Meaningless unless `gameIDKnown`; the raw value is carried regardless. */
  gameID: number;
  gameIDKnown: boolean;
  family: string;
  category: string;
  subcategory: string;
  /** Empty when the catalog does not know the name; never a synthesised one. */
  name: string;
  /** Embedded catalog path only; turn it into a URL through catalogAssetURL. */
  iconPath: string;
  banRisk: boolean;
  cutContent: boolean;
  dlc: boolean;
  preOrder: boolean;
};

/** One category the profile can reach, with the rows behind it. */
export type CatalogItemDatabaseCategory = {
  category: string;
  count: number;
};

/** One resolved page of the Item Database. */
export type CatalogItemDatabasePage = {
  /** The profile the backend resolved this page under, carried verbatim. */
  safetyProfile: string;
  resources: readonly CatalogItemDatabaseEntry[];
  categories: readonly CatalogItemDatabaseCategory[];
  total: number;
  page: number;
  pageSize: number;
};

/**
 * The arguments of one Item Database page. Which filters are accepted, which
 * sort orders exist, that an empty filter never filters and which defaults zero
 * paging resolves to are the backend's contract; no safety profile is sent,
 * because the backend reads its own host setting.
 */
export type CatalogItemDatabaseRequest = {
  family: string;
  category: string;
  search: string;
  favoritesOnly: boolean;
  favorites: readonly CatalogResourcePresentationIdentity[];
  sort: string;
  page: number;
  pageSize: number;
};

export type CatalogPort = {
  /** Reads one authoritative page of the Item Database under the active profile. */
  getItemDatabase: (request: CatalogItemDatabaseRequest) => Promise<CatalogItemDatabasePage>;
  /** Reads one page of the catalog under the backend's own filter contract. */
  getResources: (request: CatalogResourcesRequest) => Promise<CatalogResourcesPage>;
  /** Reads scalar presentation metadata for an ordered batch of exact identities. */
  getResourcePresentationSummaries: (
    identities: readonly CatalogResourcePresentationIdentity[],
  ) => Promise<CatalogResourcePresentationSummaries>;
  /** Reads the common detail of one resource under the backend's own identity contract. */
  getResource: (request: CatalogResourceRequest) => Promise<CatalogResourceDetail>;
  /** Reads the variants of one item under the backend's own identity contract. */
  getItemVariants: (request: CatalogItemVariantsRequest) => Promise<CatalogItemVariantsResult>;
};
