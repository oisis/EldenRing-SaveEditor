import { QueryClient } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";
import type { ItemPage, ItemsPort } from "../../../application/items/itemsPort";
import {
  makeItemsPort,
  stubInventoryPage,
  stubStoragePage,
  TestProviders,
} from "../../../test/renderWithProviders";
import { type InventoryAndStorageQuery, useInventoryAndStorage } from "./useInventoryAndStorage";

const query: InventoryAndStorageQuery = {
  saveSessionID: "session-1",
  characterID: 0,
  containerSection: "common",
  inventory: { page: 2, pageSize: 30 },
  storage: { page: 3, pageSize: 50 },
};

function setup(port: ItemsPort) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const wrapper = ({ children }: { children: ReactNode }) => (
    <TestProviders queryClient={queryClient} itemsPort={port}>
      {children}
    </TestProviders>
  );
  return { queryClient, wrapper };
}

function page(source: ItemPage, overrides: Partial<ItemPage> = {}): ItemPage {
  return { ...source, ...overrides };
}

describe("useInventoryAndStorage", () => {
  it("forwards independent page windows to both read-only getters", async () => {
    const getInventory = vi.fn(() => Promise.resolve(stubInventoryPage));
    const getStorage = vi.fn(() => Promise.resolve(stubStoragePage));
    const { wrapper } = setup(makeItemsPort({ getInventory, getStorage }));

    const { result } = renderHook(() => useInventoryAndStorage(query), { wrapper });

    await waitFor(() => expect(result.current.inventory.data).toBeDefined());
    await waitFor(() => expect(result.current.storage.data).toBeDefined());
    expect(getInventory).toHaveBeenCalledExactlyOnceWith({
      saveSessionID: "session-1",
      characterID: 0,
      containerSection: "common",
      page: 2,
      pageSize: 30,
    });
    expect(getStorage).toHaveBeenCalledExactlyOnceWith({
      saveSessionID: "session-1",
      characterID: 0,
      containerSection: "common",
      page: 3,
      pageSize: 50,
    });
  });

  it("does not reach either getter without a session or character", async () => {
    const getInventory = vi.fn(makeItemsPort().getInventory);
    const getStorage = vi.fn(makeItemsPort().getStorage);
    const { wrapper } = setup(makeItemsPort({ getInventory, getStorage }));

    const withoutSession = renderHook(
      () => useInventoryAndStorage({ ...query, saveSessionID: undefined }),
      { wrapper },
    );
    expect(withoutSession.result.current.inventory.fetchStatus).toBe("idle");
    expect(withoutSession.result.current.storage.fetchStatus).toBe("idle");
    await withoutSession.result.current.inventory.refetch();
    await withoutSession.result.current.storage.refetch();
    withoutSession.unmount();

    const withoutCharacter = renderHook(
      () => useInventoryAndStorage({ ...query, characterID: undefined }),
      { wrapper },
    );
    await withoutCharacter.result.current.inventory.refetch();
    await withoutCharacter.result.current.storage.refetch();

    expect(getInventory).not.toHaveBeenCalled();
    expect(getStorage).not.toHaveBeenCalled();
  });

  it("compares opaque revisions exactly without parsing or ordering them", async () => {
    const inventory = page(stubInventoryPage, { saveRevision: "  Revision 09  " });
    const sameStorage = page(stubStoragePage, { saveRevision: "  Revision 09  " });
    const { wrapper } = setup(
      makeItemsPort({
        getInventory: () => Promise.resolve(inventory),
        getStorage: () => Promise.resolve(sameStorage),
      }),
    );
    const { result } = renderHook(() => useInventoryAndStorage(query), { wrapper });

    expect(result.current.revisionState).toBe("unavailable");
    await waitFor(() => expect(result.current.revisionState).toBe("consistent"));
    expect(result.current.inventory.data?.saveRevision).toBe("  Revision 09  ");
  });

  it("reports mismatched revisions while preserving both independent results", async () => {
    const inventory = page(stubInventoryPage, { saveRevision: "9" });
    const storage = page(stubStoragePage, { saveRevision: "09" });
    const { wrapper } = setup(
      makeItemsPort({
        getInventory: () => Promise.resolve(inventory),
        getStorage: () => Promise.resolve(storage),
      }),
    );
    const { result } = renderHook(() => useInventoryAndStorage(query), { wrapper });

    await waitFor(() => expect(result.current.revisionState).toBe("mismatch"));
    expect(result.current.inventory.data).toBe(inventory);
    expect(result.current.storage.data).toBe(storage);
  });

  it("derives selection from the current cache page instead of copying a record", async () => {
    const inventory = page(stubInventoryPage, { saveRevision: "revision-7" });
    const { wrapper } = setup(
      makeItemsPort({
        getInventory: () => Promise.resolve(inventory),
        getStorage: () => Promise.resolve(page(stubStoragePage, { saveRevision: "revision-7" })),
      }),
    );
    const { result } = renderHook(() => useInventoryAndStorage(query), { wrapper });
    await waitFor(() => expect(result.current.inventory.data).toBe(inventory));

    result.current.selectItem("inventory", "owned-1");

    await waitFor(() => expect(result.current.selected).not.toBeNull());
    expect(result.current.selected).toEqual({
      container: "inventory",
      saveRevision: "revision-7",
      record: inventory.records[0],
    });
    expect(result.current.selected?.record).toBe(inventory.records[0]);
  });

  it("never creates a selection for an identity outside the current page", async () => {
    const { wrapper } = setup(makeItemsPort());
    const { result } = renderHook(() => useInventoryAndStorage(query), { wrapper });
    await waitFor(() => expect(result.current.inventory.data).toBeDefined());

    result.current.selectItem("inventory", "missing-owned-item");

    expect(result.current.selected).toBeNull();
  });

  it("invalidates selection when its backend revision changes", async () => {
    let inventory = page(stubInventoryPage, { saveRevision: "revision-1" });
    const getInventory = vi.fn(() => Promise.resolve(inventory));
    const { queryClient, wrapper } = setup(makeItemsPort({ getInventory }));
    const { result } = renderHook(() => useInventoryAndStorage(query), { wrapper });
    await waitFor(() => expect(result.current.inventory.data).toBeDefined());
    result.current.selectItem("inventory", "owned-1");
    await waitFor(() => expect(result.current.selected).not.toBeNull());

    inventory = page(stubInventoryPage, { saveRevision: "revision-2" });
    await queryClient.invalidateQueries({ queryKey: ["save-session", "session-1"] });

    await waitFor(() => expect(result.current.inventory.data?.saveRevision).toBe("revision-2"));
    expect(result.current.selected).toBeNull();
  });

  it("starts a fresh selection entry after changing session or character", async () => {
    const { wrapper } = setup(makeItemsPort());
    const { result, rerender } = renderHook(
      ({ current }: { current: InventoryAndStorageQuery }) => useInventoryAndStorage(current),
      { wrapper, initialProps: { current: query } },
    );
    await waitFor(() => expect(result.current.inventory.data).toBeDefined());
    result.current.selectItem("inventory", "owned-1");
    await waitFor(() => expect(result.current.selected).not.toBeNull());

    rerender({ current: { ...query, saveSessionID: "session-2", characterID: 1 } });
    expect(result.current.selected).toBeNull();
    rerender({ current: query });
    expect(result.current.selected).toBeNull();
  });

  it("keeps an Inventory error independent from a successful Storage page", async () => {
    const getInventory = vi.fn(() => Promise.reject(new Error("bridge_call_failed private")));
    const { wrapper } = setup(makeItemsPort({ getInventory }));
    const { result } = renderHook(() => useInventoryAndStorage(query), { wrapper });

    await waitFor(() => expect(result.current.inventory.isError).toBe(true));
    await waitFor(() => expect(result.current.storage.data).toBe(stubStoragePage));
    expect(result.current.revisionState).toBe("unavailable");
    expect(getInventory).toHaveBeenCalledTimes(1);
  });

  it("exposes only query state, revision coordination and selection intent", async () => {
    const { wrapper } = setup(makeItemsPort());
    const { result } = renderHook(() => useInventoryAndStorage(query), { wrapper });
    await waitFor(() => expect(result.current.inventory.data).toBeDefined());

    expect(Object.keys(result.current).sort()).toEqual([
      "clearSelection",
      "inventory",
      "revisionState",
      "selectItem",
      "selected",
      "storage",
    ]);
    expect(result.current).not.toHaveProperty("addItem");
    expect(result.current).not.toHaveProperty("moveItem");
    expect(result.current).not.toHaveProperty("removeItem");
    expect(result.current).not.toHaveProperty("sortItems");
  });
});
