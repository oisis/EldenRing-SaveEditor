import { useMemo, useState } from "react";
import type { ResourceIdentity } from "../../../application/items/itemsPort";
import { useOwnedItems } from "../../../application/items/useOwnedItems";

/** The two containers of the shared workspace. */
export type ItemContainerName = "inventory" | "storage";

/**
 * The filters the workspace applies. Every one of them is sent to the backend,
 * which applies it to the whole container: nothing here filters or sorts a
 * served page.
 */
export type ItemsWorkspaceFilters = {
  search: string;
  category: string;
  sort: string;
  favoritesOnly: boolean;
};

export type ItemsWorkspaceQuery = {
  saveSessionID: string | undefined;
  saveRevision: string | undefined;
  characterID: number | undefined;
  containerSection: string;
  filters: ItemsWorkspaceFilters;
  favorites: readonly ResourceIdentity[];
  inventoryPage: number;
  storagePage: number;
  pageSize: number;
};

/** One selected record, kept as an identity and the revision that minted it. */
type Selection = {
  container: ItemContainerName;
  ownedItemID: string;
};

type WorkspaceEntry = {
  saveSessionID: string | undefined;
  characterID: number | undefined;
  containerSection: string;
  saveRevision: string | undefined;
  selected: readonly Selection[];
  opened: Selection | null;
};

/**
 * The controller of the shared Inventory and Storage workspace.
 *
 * It owns two things and nothing else: the two authoritative container queries,
 * and the user's selection intent. Records, names, icons, limits, categories,
 * order and the allowed actions all come from the backend and are never copied
 * into local state, recomputed or filtered again here.
 *
 * Selection is scoped to one exact workspace identity — session, slot, section
 * and revision. A change in any of them empties the selection during render, so
 * an owned-item identity minted under one revision can never reach a mutation
 * that runs under another one.
 */
export function useItemsWorkspace(query: ItemsWorkspaceQuery) {
  const shared = {
    saveSessionID: query.saveSessionID,
    saveRevision: query.saveRevision,
    characterID: query.characterID,
    containerSection: query.containerSection,
    search: query.filters.search,
    category: query.filters.category,
    favoritesOnly: query.filters.favoritesOnly,
    favorites: query.favorites,
    sort: query.filters.sort,
    pageSize: query.pageSize,
  };
  const inventory = useOwnedItems({ ...shared, container: "inventory", page: query.inventoryPage });
  const storage = useOwnedItems({ ...shared, container: "storage", page: query.storagePage });

  const [entry, setEntry] = useState<WorkspaceEntry>({
    saveSessionID: query.saveSessionID,
    characterID: query.characterID,
    containerSection: query.containerSection,
    saveRevision: query.saveRevision,
    selected: [],
    opened: null,
  });

  // A session, slot, section or revision change starts a fresh workspace entry.
  // Resetting during render prevents one frame from exposing the previous
  // workspace's intent: an owned-item identity only ever means something inside
  // the exact revision that offered it.
  const isCurrentEntry =
    entry.saveSessionID === query.saveSessionID &&
    entry.characterID === query.characterID &&
    entry.containerSection === query.containerSection &&
    entry.saveRevision === query.saveRevision;
  if (!isCurrentEntry) {
    setEntry({
      saveSessionID: query.saveSessionID,
      characterID: query.characterID,
      containerSection: query.containerSection,
      saveRevision: query.saveRevision,
      selected: [],
      opened: null,
    });
  }
  const selected = isCurrentEntry ? entry.selected : [];
  const opened = isCurrentEntry ? entry.opened : null;

  const update = (change: Partial<Pick<WorkspaceEntry, "selected" | "opened">>) =>
    setEntry({
      saveSessionID: query.saveSessionID,
      characterID: query.characterID,
      containerSection: query.containerSection,
      saveRevision: query.saveRevision,
      selected,
      opened,
      ...change,
    });

  const rowsFor = (container: ItemContainerName) =>
    (container === "inventory" ? inventory.data?.records : storage.data?.records) ?? [];

  // The open record is derived from the current page rather than copied, so a
  // refetch that no longer contains it closes the dialog instead of keeping a
  // record the backend no longer reports.
  const openedRow = useMemo(() => {
    if (!opened) return null;
    const page = opened.container === "inventory" ? inventory.data : storage.data;
    return page?.records.find((record) => record.ownedItemID === opened.ownedItemID) ?? null;
  }, [inventory.data, opened, storage.data]);

  const isSelected = (container: ItemContainerName, ownedItemID: string) =>
    selected.some((entry) => entry.container === container && entry.ownedItemID === ownedItemID);

  const selectedIn = (container: ItemContainerName) =>
    selected.filter((entry) => entry.container === container).map((entry) => entry.ownedItemID);

  return {
    inventory,
    storage,
    /** The categories the two containers report, merged and counted once. */
    categories: useMemo(() => {
      const counts = new Map<string, number>();
      for (const page of [inventory.data, storage.data]) {
        for (const facet of page?.categories ?? []) {
          counts.set(facet.category, (counts.get(facet.category) ?? 0) + facet.count);
        }
      }
      return [...counts.entries()]
        .map(([category, count]) => ({ category, count }))
        .sort((left, right) => left.category.localeCompare(right.category));
    }, [inventory.data, storage.data]),
    selected,
    selectedIn,
    isSelected,
    openedRow,
    openedContainer: opened?.container ?? null,
    toggleSelection: (container: ItemContainerName, ownedItemID: string) => {
      // Only an identity the current page actually reports can be selected: a
      // stale one would reach a mutation as a record that no longer exists.
      if (!rowsFor(container).some((record) => record.ownedItemID === ownedItemID)) return;
      update({
        selected: isSelected(container, ownedItemID)
          ? selected.filter(
              (entry) => entry.container !== container || entry.ownedItemID !== ownedItemID,
            )
          : [...selected, { container, ownedItemID }],
      });
    },
    clearSelection: () => update({ selected: [] }),
    openItem: (container: ItemContainerName, ownedItemID: string) => {
      if (!rowsFor(container).some((record) => record.ownedItemID === ownedItemID)) return;
      update({ opened: { container, ownedItemID } });
    },
    closeItem: () => update({ opened: null }),
  };
}
