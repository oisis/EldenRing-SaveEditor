import { QueryClient } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";
import {
  makeItemsPort,
  stubInventoryPage,
  stubStoragePage,
  TestProviders,
} from "../../test/renderWithProviders";
import { noCharacter, queryKeys } from "../queryKeys";
import type { ItemsPort } from "./itemsPort";
import { useInventory, useStorage } from "./useItems";

/**
 * The hooks are exercised through an injected `ItemsPort` stub. The generated
 * bindings are never mocked here: that belongs to the adapter test.
 *
 * The client keeps the library defaults on purpose, so a hook that dropped its
 * own `retry: false` would be caught instead of being covered by a test-only
 * default.
 */
function setup(port: ItemsPort) {
  const queryClient = new QueryClient();
  const wrapper = ({ children }: { children: ReactNode }) => (
    <TestProviders queryClient={queryClient} itemsPort={port}>
      {children}
    </TestProviders>
  );

  return { queryClient, wrapper };
}

const backendRequest = {
  saveSessionID: "session-1",
  characterID: 0,
  containerSection: "  Common  ",
  page: 2,
  pageSize: 30,
};
const request = { ...backendRequest, saveRevision: "0" };

describe("useInventory and useStorage", () => {
  it("passes every argument to its own port method exactly as given", async () => {
    const getInventory = vi.fn(makeItemsPort().getInventory);
    const getStorage = vi.fn(makeItemsPort().getStorage);
    const { wrapper } = setup(makeItemsPort({ getInventory, getStorage }));

    const inventory = renderHook(() => useInventory(request), { wrapper });
    const storage = renderHook(() => useStorage(request), { wrapper });

    await waitFor(() => expect(inventory.result.current.data).toEqual(stubInventoryPage));
    await waitFor(() => expect(storage.result.current.data).toEqual(stubStoragePage));

    // No trimming, no section default and no paging normalisation: the backend
    // owns all three.
    expect(getInventory).toHaveBeenCalledExactlyOnceWith(backendRequest);
    expect(getStorage).toHaveBeenCalledExactlyOnceWith(backendRequest);
  });

  it("carries the opaque save revision through untouched", async () => {
    const revised = { ...stubInventoryPage, saveRevision: "  Revision 7  " };
    const { wrapper } = setup(makeItemsPort({ getInventory: () => Promise.resolve(revised) }));

    const { result } = renderHook(
      () => useInventory({ ...request, saveRevision: "  Revision 7  " }),
      { wrapper },
    );

    await waitFor(() => expect(result.current.data).toBeDefined());
    // The revision is neither parsed, compared nor replaced here.
    expect(result.current.data?.saveRevision).toBe("  Revision 7  ");
  });

  it("asks the backend for nothing without a session identifier", async () => {
    const getInventory = vi.fn(makeItemsPort().getInventory);
    const getStorage = vi.fn(makeItemsPort().getStorage);
    const { wrapper } = setup(makeItemsPort({ getInventory, getStorage }));

    const inventory = renderHook(
      ({ id }: { id?: string }) => useInventory({ ...request, saveSessionID: id }),
      {
        wrapper,
        initialProps: {},
      },
    );
    const storage = renderHook(
      ({ id }: { id?: string }) => useStorage({ ...request, saveSessionID: id }),
      {
        wrapper,
        initialProps: {},
      },
    );

    expect(inventory.result.current.fetchStatus).toBe("idle");
    expect(storage.result.current.fetchStatus).toBe("idle");

    // An empty identifier is just as absent as a missing one.
    inventory.rerender({ id: "" });
    storage.rerender({ id: "" });
    expect(getInventory).not.toHaveBeenCalled();
    expect(getStorage).not.toHaveBeenCalled();

    inventory.rerender({ id: "session-1" });
    await waitFor(() => expect(inventory.result.current.data).toEqual(stubInventoryPage));
    expect(getInventory).toHaveBeenCalledExactlyOnceWith({
      ...backendRequest,
      saveSessionID: "session-1",
    });
  });

  it("asks the backend for nothing without a character slot", async () => {
    const getInventory = vi.fn(makeItemsPort().getInventory);
    const getStorage = vi.fn(makeItemsPort().getStorage);
    const { wrapper } = setup(makeItemsPort({ getInventory, getStorage }));

    const inventory = renderHook(
      ({ slot }: { slot?: number }) => useInventory({ ...request, characterID: slot }),
      { wrapper, initialProps: {} },
    );
    const storage = renderHook(
      ({ slot }: { slot?: number }) => useStorage({ ...request, characterID: slot }),
      { wrapper, initialProps: {} },
    );

    expect(inventory.result.current.fetchStatus).toBe("idle");
    expect(storage.result.current.fetchStatus).toBe("idle");
    expect(getInventory).not.toHaveBeenCalled();
    expect(getStorage).not.toHaveBeenCalled();

    // Slot 0 is an ordinary slot, not an absent one.
    inventory.rerender({ slot: 0 });
    await waitFor(() => expect(inventory.result.current.data).toEqual(stubInventoryPage));
    expect(getInventory).toHaveBeenCalledExactlyOnceWith(backendRequest);
  });

  it("cannot reach the port through a manual refetch while an identifier is missing", async () => {
    const getInventory = vi.fn(makeItemsPort().getInventory);
    const getStorage = vi.fn(makeItemsPort().getStorage);
    const { wrapper } = setup(makeItemsPort({ getInventory, getStorage }));

    const withoutSession = renderHook(
      () => useInventory({ ...request, saveSessionID: undefined }),
      { wrapper },
    );
    const withoutSlot = renderHook(() => useStorage({ ...request, characterID: undefined }), {
      wrapper,
    });

    // `enabled` would still run the query function here; `skipToken` cannot.
    await withoutSession.result.current.refetch();
    await withoutSlot.result.current.refetch();

    expect(getInventory).not.toHaveBeenCalled();
    expect(getStorage).not.toHaveBeenCalled();
  });

  it("reports a rejected call without retrying it", async () => {
    const getInventory = vi.fn(() => Promise.reject(new Error("bridge_call_failed")));
    const { wrapper } = setup(makeItemsPort({ getInventory }));

    const { result } = renderHook(() => useInventory(request), { wrapper });

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(getInventory).toHaveBeenCalledTimes(1);
  });

  it("fails as a wiring error when no ItemsPortProvider is above the hook", () => {
    // The port is read before any query is set up, so a tree without the
    // provider fails immediately instead of silently rendering an empty view.
    expect(() => renderHook(() => useInventory(request))).toThrow(
      "ItemsPortProvider is missing above this component",
    );
    expect(() => renderHook(() => useStorage(request))).toThrow(
      "ItemsPortProvider is missing above this component",
    );
  });
});

describe("container query keys", () => {
  const key = (session: string, slot: number | typeof noCharacter) => ({
    inventory: queryKeys.inventory(session, slot, "common", 1, 30, "0"),
    storage: queryKeys.storage(session, slot, "common", 1, 30, "0"),
  });

  it("keeps Inventory and Storage apart", () => {
    const keys = key("session-1", 0);
    expect(keys.inventory).not.toEqual(keys.storage);
  });

  it("keeps sessions, slots, sections and page windows apart", () => {
    const base = queryKeys.inventory("session-1", 0, "common", 1, 30, "0");

    expect(queryKeys.inventory("session-2", 0, "common", 1, 30, "0")).not.toEqual(base);
    expect(queryKeys.inventory("session-1", 1, "common", 1, 30, "0")).not.toEqual(base);
    expect(queryKeys.inventory("session-1", noCharacter, "common", 1, 30, "0")).not.toEqual(base);
    expect(queryKeys.inventory("session-1", 0, "key", 1, 30, "0")).not.toEqual(base);
    expect(queryKeys.inventory("session-1", 0, "common", 2, 30, "0")).not.toEqual(base);
    expect(queryKeys.inventory("session-1", 0, "common", 1, 60, "0")).not.toEqual(base);
    expect(queryKeys.inventory("session-1", 0, "common", 1, 30, "1")).not.toEqual(base);
    expect(queryKeys.inventory("session-1", 0, "common", 1, 30, "0")).toEqual(base);
  });

  it("nests both containers under the session prefix that CloseSave removes", () => {
    const prefix = queryKeys.saveSession("session-1");

    for (const built of Object.values(key("session-1", 0))) {
      expect(built.slice(0, prefix.length)).toEqual([...prefix]);
    }
  });
});
