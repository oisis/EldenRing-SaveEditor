import { useMemo, useState } from "react";
import type { CatalogItemDatabaseRequest } from "../../../application/catalog/catalogPort";
import { useCatalogItemDatabase } from "../../../application/catalog/useCatalogItemDatabase";
import { useCatalogItemVariants } from "../../../application/catalog/useCatalogItemVariants";
import { useCatalogResource } from "../../../application/catalog/useCatalogResource";
import type { ResourceIdentity } from "../../../application/items/itemsPort";

/**
 * The identity of one catalog resource, carried verbatim. Nothing else of a row
 * is stored: the name, the family, the detail and the variants live in the
 * query cache and are never copied into a second store.
 */
export type ItemCatalogSelection = ResourceIdentity;

/**
 * The Item Database controller.
 *
 * It composes the authoritative list with the two detail queries and owns
 * nothing but presentational intent: which rows are ticked and which row is
 * open. Visibility, order and paging are the backend's answer under the active
 * Safety Profile and are never re-sorted, re-filtered or re-paged here, so the
 * screen works with no save session and no character.
 */
export function useItemCatalog(request: CatalogItemDatabaseRequest) {
  const resources = useCatalogItemDatabase(request);
  const [selected, setSelected] = useState<readonly ItemCatalogSelection[]>([]);
  const [opened, setOpened] = useState<ItemCatalogSelection | null>(null);

  // With nothing opened both hooks receive `undefined`, and their `skipToken`
  // guard keeps the port out of reach entirely.
  const detail = useCatalogResource(opened?.kind, opened?.key);
  const variants = useCatalogItemVariants(opened?.kind, opened?.key);

  const rows = resources.data?.resources ?? [];
  const tokens = useMemo(
    () => new Set(selected.map((entry) => `${entry.kind}/${entry.key}`)),
    [selected],
  );

  const isSelected = (identity: ItemCatalogSelection) =>
    tokens.has(`${identity.kind}/${identity.key}`);

  return {
    resources,
    rows,
    selected,
    isSelected,
    opened,
    detail,
    variants,
    /** The rows of the current page that are ticked, in the backend's order. */
    selectedRows: rows.filter((row) => isSelected(row)),
    toggleSelection: (identity: ItemCatalogSelection) =>
      setSelected((current) =>
        isSelected(identity)
          ? current.filter((entry) => entry.kind !== identity.kind || entry.key !== identity.key)
          : [...current, { kind: identity.kind, key: identity.key }],
      ),
    clearSelection: () => setSelected([]),
    openItem: (identity: ItemCatalogSelection) => setOpened(identity),
    // Presentational intent only: the global catalog cache stays untouched and
    // no bridge call is made.
    closeItem: () => setOpened(null),
  };
}
