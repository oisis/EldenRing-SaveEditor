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

export type CatalogPort = {
  /** Reads one page of the catalog under the backend's own filter contract. */
  getResources: (request: CatalogResourcesRequest) => Promise<CatalogResourcesPage>;
};
