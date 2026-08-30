import { useState } from "react";
import type { CatalogResourcesRequest } from "../../../application/catalog/catalogPort";
import { useCatalogItemVariants } from "../../../application/catalog/useCatalogItemVariants";
import { useCatalogResource } from "../../../application/catalog/useCatalogResource";
import { useCatalogResources } from "../../../application/catalog/useCatalogResources";

/**
 * The arguments the Item Database screen controls. They are the subset of the
 * backend page request the screen owns; `resourceType` and `endpointID` are not
 * part of it, because this screen always reads the item catalog and never
 * narrows it to one endpoint.
 *
 * Nothing here is trimmed, recased, defaulted or validated: an empty filter and
 * a zero page or page size are real backend arguments with a backend meaning.
 */
export type ItemDatabaseQuery = {
  family: string;
  capability: string;
  search: string;
  page: number;
  pageSize: number;
};

/**
 * The identity of the selected catalog resource, carried verbatim. Nothing else
 * of the selected row is stored: the name, the family, the detail and the
 * variants live in the query cache and are never copied into a second store.
 */
export type ItemDatabaseSelection = {
  kind: string;
  key: string;
};

export type ItemDatabase = {
  /** One page of the item catalog, in backend order, with its served paging. */
  resources: ReturnType<typeof useCatalogResources>;
  selected: ItemDatabaseSelection | null;
  /** The detail of the selection; idle while nothing is selected. */
  detail: ReturnType<typeof useCatalogResource>;
  /** The variants of the selection; idle while nothing is selected. */
  variants: ReturnType<typeof useCatalogItemVariants>;
  selectItem: (kind: string, key: string) => void;
  clearSelection: () => void;
};

/**
 * The read-only Item Database controller. It composes the three catalog hooks
 * and owns nothing but the user's presentational intent: which row is selected.
 *
 * The catalog is global and immutable under this contract, so the screen works
 * with no save session and no character. The page is the backend's answer and
 * is never sorted, filtered or rewritten here; the detail and the variants stay
 * two independent queries with independent statuses and independent errors,
 * because one of them failing is not the other one failing.
 */
export function useItemDatabase(query: ItemDatabaseQuery): ItemDatabase {
  const [selected, setSelected] = useState<ItemDatabaseSelection | null>(null);

  // The one place the page request is built. `resourceType` and `endpointID`
  // are this screen's intent, not a validation of the backend's contract, so
  // neither is exposed to the caller.
  const request: CatalogResourcesRequest = {
    resourceType: "item",
    endpointID: "",
    family: query.family,
    capability: query.capability,
    search: query.search,
    page: query.page,
    pageSize: query.pageSize,
  };

  const resources = useCatalogResources(request);

  // With nothing selected both hooks receive `undefined`, and their `skipToken`
  // guard keeps the port out of reach entirely.
  const detail = useCatalogResource(selected?.kind, selected?.key);
  const variants = useCatalogItemVariants(selected?.kind, selected?.key);

  return {
    resources,
    selected,
    detail,
    variants,
    selectItem: (kind, key) => setSelected({ kind, key }),
    // Presentational intent only: the global catalog cache stays untouched and
    // no bridge call is made.
    clearSelection: () => setSelected(null),
  };
}
