import { useCallback, useMemo, useState } from "react";
import type {
  CatalogResourcePresentationIdentity,
  CatalogResourcePresentationSummary,
} from "../../../application/catalog/catalogPort";
import { useCatalogResourcePresentationSummaries } from "../../../application/catalog/useCatalogResourcePresentationSummaries";
import type { ItemPage, ItemRecord } from "../../../application/items/itemsPort";
import { useInventory, useStorage } from "../../../application/items/useItems";

export type ItemContainer = "inventory" | "storage";

export type PageWindow = {
  page: number;
  pageSize: number;
};

export type InventoryAndStorageQuery = {
  saveSessionID: string | undefined;
  characterID: number | undefined;
  containerSection: string;
  inventory: PageWindow;
  storage: PageWindow;
};

export type ContainerRevisionState = "unavailable" | "consistent" | "mismatch";

export type SelectedOwnedItem = {
  container: ItemContainer;
  saveRevision: string;
  record: ItemRecord;
};

type PresentableIdentity = Pick<ItemRecord, "kind" | "key">;

type Selection = {
  container: ItemContainer;
  ownedItemID: string;
  saveRevision: string;
};

type SessionEntry = {
  saveSessionID: string | undefined;
  characterID: number | undefined;
  selection: Selection | null;
};

/**
 * Coordinates the two read-only container pages of one character without
 * copying either page out of TanStack Query. Presentation owns the section and
 * page windows; the backend remains the source of records, totals and served
 * paging values. Once both container calls have settled, one lightweight
 * catalog batch resolves the distinct exact identities from both current pages.
 *
 * Selection stores only an opaque owned-item identity and the exact revision
 * that minted it. The selected record is derived from the current query page,
 * so changing session, slot, page contents or revision can never leave a stale
 * record available to a future action.
 */
export function useInventoryAndStorage(query: InventoryAndStorageQuery) {
  const inventory = useInventory({
    saveSessionID: query.saveSessionID,
    characterID: query.characterID,
    containerSection: query.containerSection,
    page: query.inventory.page,
    pageSize: query.inventory.pageSize,
  });
  const storage = useStorage({
    saveSessionID: query.saveSessionID,
    characterID: query.characterID,
    containerSection: query.containerSection,
    page: query.storage.page,
    pageSize: query.storage.pageSize,
  });
  const presentationIdentities = useMemo(
    () =>
      presentationIdentitiesForSettledPages(
        query.saveSessionID,
        query.characterID,
        inventory.isPending,
        inventory.data,
        storage.isPending,
        storage.data,
      ),
    [
      inventory.data,
      inventory.isPending,
      query.characterID,
      query.saveSessionID,
      storage.data,
      storage.isPending,
    ],
  );
  const presentations = useCatalogResourcePresentationSummaries(presentationIdentities);
  const presentationsByKind = useMemo(() => {
    const byKind = new Map<string, Map<string, CatalogResourcePresentationSummary>>();
    for (const presentation of presentations.data?.resources ?? []) {
      let byKey = byKind.get(presentation.kind);
      if (!byKey) {
        byKey = new Map();
        byKind.set(presentation.kind, byKey);
      }
      byKey.set(presentation.key, presentation);
    }
    return byKind;
  }, [presentations.data]);
  const presentationFor = useCallback(
    (identity: PresentableIdentity) =>
      presentationsByKind.get(identity.kind)?.get(identity.key) ?? null,
    [presentationsByKind],
  );
  const [entry, setEntry] = useState<SessionEntry>({
    saveSessionID: query.saveSessionID,
    characterID: query.characterID,
    selection: null,
  });

  // A session or character change creates a fresh workspace entry. Resetting
  // during render prevents one frame from exposing the previous slot's intent.
  if (entry.saveSessionID !== query.saveSessionID || entry.characterID !== query.characterID) {
    setEntry({
      saveSessionID: query.saveSessionID,
      characterID: query.characterID,
      selection: null,
    });
  }

  const isCurrentEntry =
    entry.saveSessionID === query.saveSessionID && entry.characterID === query.characterID;
  const selection = isCurrentEntry ? entry.selection : null;
  const revisionState = compareRevisions(inventory.data, storage.data);

  const selected = useMemo<SelectedOwnedItem | null>(() => {
    if (!selection) return null;
    const page = selection.container === "inventory" ? inventory.data : storage.data;
    if (!page || page.saveRevision !== selection.saveRevision) return null;
    const record = page.records.find(
      (candidate) => candidate.ownedItemID === selection.ownedItemID,
    );
    return record
      ? { container: selection.container, saveRevision: page.saveRevision, record }
      : null;
  }, [inventory.data, selection, storage.data]);

  const selectItem = (container: ItemContainer, ownedItemID: string) => {
    const page = container === "inventory" ? inventory.data : storage.data;
    const exists = page?.records.some((record) => record.ownedItemID === ownedItemID) ?? false;
    setEntry({
      saveSessionID: query.saveSessionID,
      characterID: query.characterID,
      selection:
        page && exists ? { container, ownedItemID, saveRevision: page.saveRevision } : null,
    });
  };

  return {
    inventory,
    storage,
    presentations,
    presentationFor,
    revisionState,
    selected,
    selectedPresentation: selected ? presentationFor(selected.record) : null,
    selectItem,
    clearSelection: () =>
      setEntry({
        saveSessionID: query.saveSessionID,
        characterID: query.characterID,
        selection: null,
      }),
  };
}

function presentationIdentitiesForSettledPages(
  saveSessionID: string | undefined,
  characterID: number | undefined,
  inventoryPending: boolean,
  inventory: ItemPage | undefined,
  storagePending: boolean,
  storage: ItemPage | undefined,
): readonly CatalogResourcePresentationIdentity[] | undefined {
  if (
    !saveSessionID ||
    characterID === undefined ||
    inventoryPending ||
    storagePending ||
    (!inventory && !storage)
  ) {
    return undefined;
  }

  const identities: CatalogResourcePresentationIdentity[] = [];
  const seenByKind = new Map<string, Set<string>>();
  for (const page of [inventory, storage]) {
    for (const record of page?.records ?? []) {
      let seenKeys = seenByKind.get(record.kind);
      if (!seenKeys) {
        seenKeys = new Set();
        seenByKind.set(record.kind, seenKeys);
      }
      if (seenKeys.has(record.key)) continue;
      seenKeys.add(record.key);
      identities.push({ kind: record.kind, key: record.key });
    }
  }
  return identities;
}

function compareRevisions(
  inventory: ItemPage | undefined,
  storage: ItemPage | undefined,
): ContainerRevisionState {
  if (!inventory || !storage) return "unavailable";
  return inventory.saveRevision === storage.saveRevision ? "consistent" : "mismatch";
}
