import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";
import {
  makeCatalogPort,
  stubCatalogResourceDetail,
  TestProviders,
} from "../../test/renderWithProviders";
import { queryKeys } from "../queryKeys";
import { useCloseSave } from "../save-session/useCloseSave";
import { CatalogPortProvider } from "./catalogClient";
import type { CatalogPort } from "./catalogPort";
import { useCatalogResource } from "./useCatalogResource";

/**
 * The hook is exercised through an injected `CatalogPort` stub. The generated
 * bindings are never mocked here: that belongs to the adapter test.
 *
 * The client keeps the library defaults on purpose, so a hook that dropped its
 * own `retry: false` would be caught instead of being covered by a test-only
 * default.
 */
function setup(port: CatalogPort) {
  const queryClient = new QueryClient();
  const wrapper = ({ children }: { children: ReactNode }) => (
    <TestProviders queryClient={queryClient} catalogPort={port}>
      {children}
    </TestProviders>
  );

  return { queryClient, wrapper };
}

describe("useCatalogResource", () => {
  it("passes the kind and the key to the port exactly as given", async () => {
    const getResource = vi.fn(makeCatalogPort().getResource);
    const { wrapper } = setup(makeCatalogPort({ getResource }));

    const { result } = renderHook(() => useCatalogResource("  Item  ", " 000f4240 "), { wrapper });

    await waitFor(() => expect(result.current.data).toEqual(stubCatalogResourceDetail));
    // No trimming, no recasing and no alias: the backend owns the identity.
    expect(getResource).toHaveBeenCalledExactlyOnceWith({ kind: "  Item  ", key: " 000f4240 " });
  });

  it("carries the resolved identity and the common item detail through unchanged", async () => {
    const { wrapper } = setup(makeCatalogPort());

    const { result } = renderHook(() => useCatalogResource("item", "000F4240"), { wrapper });

    await waitFor(() => expect(result.current.data).toBeDefined());
    expect(result.current.data?.kind).toBe("item");
    expect(result.current.data?.key).toBe("000F4240");
    // Unknown facts keep their raw value and absent optional facts stay null.
    expect(result.current.data?.item?.subcategory).toEqual({
      known: false,
      value: "",
      provenance: {
        source: "legacy_db_data",
        method: "unresolved",
        table: "",
        row: "",
        field: "",
      },
    });
    expect(result.current.data?.item?.storage.maxInventorySFV).toBeNull();
    expect(result.current.data?.item?.capabilities.stack.rules).toBeNull();
  });

  it("treats an empty kind or key as a real request, not as a missing one", async () => {
    const getResource = vi.fn(makeCatalogPort().getResource);
    const { wrapper } = setup(makeCatalogPort({ getResource }));

    const { result } = renderHook(() => useCatalogResource("", ""), { wrapper });

    await waitFor(() => expect(result.current.data).toBeDefined());
    // The backend rejects the empty pair; the hook must not decide that itself.
    expect(getResource).toHaveBeenCalledExactlyOnceWith({ kind: "", key: "" });
  });

  it("never reaches the port while nothing is selected", async () => {
    const getResource = vi.fn(makeCatalogPort().getResource);
    const { wrapper } = setup(makeCatalogPort({ getResource }));

    const { result } = renderHook(() => useCatalogResource(undefined, undefined), { wrapper });

    await waitFor(() => expect(result.current.fetchStatus).toBe("idle"));
    expect(getResource).not.toHaveBeenCalled();

    // `enabled` would still run the query function here; `skipToken` cannot.
    await result.current.refetch();
    expect(getResource).not.toHaveBeenCalled();
  });

  it("reads a resource with no save session and no character", async () => {
    const getResource = vi.fn(makeCatalogPort().getResource);
    // Deliberately no SaveSessionPortProvider, no CharacterPortProvider and no
    // session identifier anywhere: the catalog is global.
    const wrapper = ({ children }: { children: ReactNode }) => (
      <QueryClientProvider client={new QueryClient()}>
        <CatalogPortProvider port={makeCatalogPort({ getResource })}>
          {children}
        </CatalogPortProvider>
      </QueryClientProvider>
    );

    const { result } = renderHook(() => useCatalogResource("item", "000F4240"), { wrapper });

    await waitFor(() => expect(result.current.data).toEqual(stubCatalogResourceDetail));
    expect(getResource).toHaveBeenCalledExactlyOnceWith({ kind: "item", key: "000F4240" });
  });

  it("reports a rejected call without retrying it", async () => {
    const getResource = vi.fn(() => Promise.reject(new Error("bridge_call_failed")));
    const { wrapper } = setup(makeCatalogPort({ getResource }));

    const { result } = renderHook(() => useCatalogResource("item", "000F4240"), { wrapper });

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(getResource).toHaveBeenCalledTimes(1);
  });

  it("fails as a wiring error when no CatalogPortProvider is above the hook", () => {
    // The port is read before any query is set up, so a tree without the
    // provider fails immediately instead of silently rendering nothing.
    expect(() => renderHook(() => useCatalogResource("item", "000F4240"))).toThrow(
      "CatalogPortProvider is missing above this component",
    );
  });
});

describe("catalog resource query keys", () => {
  const base = queryKeys.catalogResource("item", "000F4240");

  it("starts with the global catalog prefix, outside every save session", () => {
    expect(base.slice(0, 2)).toEqual(["catalog", "resource"]);
    expect(base).not.toContain("save-session");
  });

  it("tells the kind and the key apart, including the empty and unselected ones", () => {
    const distinct = [
      queryKeys.catalogResource("class", "000F4240"),
      queryKeys.catalogResource("item", "000F4241"),
      queryKeys.catalogResource("item", ""),
      queryKeys.catalogResource("", "000F4240"),
      // Nothing selected is its own entry and never collides with a real value.
      queryKeys.catalogResource(null, null),
      queryKeys.catalogResource("item", null),
    ];

    for (const key of distinct) {
      expect(key).not.toEqual(base);
    }
    expect(new Set(distinct.map((key) => JSON.stringify(key))).size).toBe(distinct.length);
    expect(queryKeys.catalogResource("item", "000F4240")).toEqual(base);
    // The list key and the detail key are separate scopes of the same prefix.
    expect(base).not.toEqual(
      queryKeys.catalogResources({
        resourceType: "item",
        family: "",
        capability: "",
        endpointID: "",
        search: "",
        page: 1,
        pageSize: 50,
      }),
    );
  });

  it("survives CloseSave, which only removes the closed session scope", async () => {
    const queryClient = new QueryClient();
    queryClient.setQueryData(base, stubCatalogResourceDetail);
    queryClient.setQueryData(queryKeys.loadedSave("session-1"), { saveSessionID: "session-1" });
    const wrapper = ({ children }: { children: ReactNode }) => (
      <TestProviders queryClient={queryClient}>{children}</TestProviders>
    );

    const { result } = renderHook(() => useCloseSave(), { wrapper });
    result.current.mutate("session-1");

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    // The session scope is gone; the global catalog detail is untouched.
    expect(queryClient.getQueryData(queryKeys.loadedSave("session-1"))).toBeUndefined();
    expect(queryClient.getQueryData(base)).toEqual(stubCatalogResourceDetail);
  });
});
