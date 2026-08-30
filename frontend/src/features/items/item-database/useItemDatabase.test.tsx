import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";
import { CatalogPortProvider } from "../../../application/catalog/catalogClient";
import type { CatalogPort, CatalogResourcesPage } from "../../../application/catalog/catalogPort";
import { queryKeys } from "../../../application/queryKeys";
import {
  makeCatalogPort,
  stubCatalogItemVariants,
  stubCatalogResourceDetail,
} from "../../../test/renderWithProviders";
import { type ItemDatabaseQuery, useItemDatabase } from "./useItemDatabase";

/**
 * The controller is exercised through an injected `CatalogPort` stub and the
 * global catalog provider only: the Item Database reads a catalog that belongs
 * to no save session, so no session and no character provider is mounted here.
 *
 * The client keeps the library defaults on purpose, so a controller that lost
 * the hooks' own `retry: false` or their cache lifetime would be caught instead
 * of being covered by a test-only default.
 */
function setup(port: CatalogPort) {
  const queryClient = new QueryClient();
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>
      <CatalogPortProvider port={port}>{children}</CatalogPortProvider>
    </QueryClientProvider>
  );

  return { queryClient, wrapper };
}

const query: ItemDatabaseQuery = {
  family: "weapon",
  capability: "upgrade",
  search: "  Uchi  ",
  page: 2,
  pageSize: 25,
};

function spyPort(overrides: Partial<CatalogPort> = {}) {
  const base = makeCatalogPort();
  const getResources = vi.fn(base.getResources);
  const getResource = vi.fn(base.getResource);
  const getItemVariants = vi.fn(base.getItemVariants);

  return {
    getResources,
    getResource,
    getItemVariants,
    port: makeCatalogPort({ getResources, getResource, getItemVariants, ...overrides }),
  };
}

describe("useItemDatabase list request", () => {
  it("reads the item catalog with no endpoint narrowing and forwards the screen arguments verbatim", async () => {
    const { port, getResources } = spyPort();
    const { wrapper } = setup(port);

    const { result } = renderHook(() => useItemDatabase(query), { wrapper });

    await waitFor(() => expect(result.current.resources.data).toBeDefined());
    // No trimming, no recasing, no default filter and no paging normalisation:
    // the backend owns all of them.
    expect(getResources).toHaveBeenCalledExactlyOnceWith({
      resourceType: "item",
      endpointID: "",
      family: "weapon",
      capability: "upgrade",
      search: "  Uchi  ",
      page: 2,
      pageSize: 25,
    });
  });

  it("treats empty filters and zero paging as a real request, not as a missing one", async () => {
    const { port, getResources } = spyPort();
    const { wrapper } = setup(port);

    const { result } = renderHook(
      () => useItemDatabase({ family: "", capability: "", search: "", page: 0, pageSize: 0 }),
      { wrapper },
    );

    await waitFor(() => expect(result.current.resources.data).toBeDefined());
    expect(getResources).toHaveBeenCalledExactlyOnceWith({
      resourceType: "item",
      endpointID: "",
      family: "",
      capability: "",
      search: "",
      page: 0,
      pageSize: 0,
    });
  });

  it("keeps the backend order and the served paging of the page", async () => {
    // Deliberately unsorted by name and by key, with a served window that
    // differs from the request.
    const page: CatalogResourcesPage = {
      resources: [
        { kind: "item", key: "weapon/uchigatana", family: "weapon", name: "Uchigatana" },
        { kind: "item", key: "goods/unnamed", family: "", name: "" },
        { kind: "item", key: "armor/blaidd", family: "armor", name: "Blaidd's Armor" },
      ],
      total: 7,
      page: 3,
      pageSize: 11,
    };
    const { wrapper } = setup(makeCatalogPort({ getResources: () => Promise.resolve(page) }));

    const { result, rerender } = renderHook((q: ItemDatabaseQuery) => useItemDatabase(q), {
      wrapper,
      initialProps: query,
    });

    await waitFor(() => expect(result.current.resources.data).toBeDefined());
    expect(result.current.resources.data).toEqual(page);

    // Changing the search, the filters or the page re-asks the backend; it
    // never re-sorts or re-filters the answer locally.
    rerender({ ...query, search: "blaidd", family: "", page: 9 });
    await waitFor(() => expect(result.current.resources.data).toBeDefined());
    expect(result.current.resources.data).toEqual(page);
  });
});

describe("useItemDatabase selection", () => {
  it("asks for no detail and no variants while nothing is selected", async () => {
    const { port, getResource, getItemVariants } = spyPort();
    const { wrapper } = setup(port);

    const { result } = renderHook(() => useItemDatabase(query), { wrapper });

    await waitFor(() => expect(result.current.resources.data).toBeDefined());
    expect(result.current.selected).toBeNull();
    expect(getResource).not.toHaveBeenCalled();
    expect(getItemVariants).not.toHaveBeenCalled();
    expect(result.current.detail.fetchStatus).toBe("idle");
    expect(result.current.variants.fetchStatus).toBe("idle");
  });

  it("passes the exact kind and key of the selection to both getters", async () => {
    const { port, getResource, getItemVariants } = spyPort();
    const { wrapper } = setup(port);

    const { result } = renderHook(() => useItemDatabase(query), { wrapper });

    act(() => result.current.selectItem("  Item  ", " 000f4240 "));

    await waitFor(() => expect(result.current.detail.data).toEqual(stubCatalogResourceDetail));
    await waitFor(() => expect(result.current.variants.data).toEqual(stubCatalogItemVariants));
    // No trimming, no recasing and no alias: the backend owns the identity.
    const identity = { kind: "  Item  ", key: " 000f4240 " };
    expect(result.current.selected).toEqual(identity);
    expect(getResource).toHaveBeenCalledExactlyOnceWith(identity);
    expect(getItemVariants).toHaveBeenCalledExactlyOnceWith(identity);
  });

  it("keeps the detail and the variants independent in status and in failure", async () => {
    const { wrapper } = setup(
      makeCatalogPort({
        getResource: () => Promise.reject(new Error("resource_not_found")),
      }),
    );

    const { result } = renderHook(() => useItemDatabase(query), { wrapper });

    act(() => result.current.selectItem("item", "000F4240"));

    await waitFor(() => expect(result.current.detail.isError).toBe(true));
    await waitFor(() => expect(result.current.variants.isSuccess).toBe(true));
    // No synthesised joint error and no joint status: one query failing is not
    // the other one failing.
    expect(result.current.detail.error).toEqual(new Error("resource_not_found"));
    expect(result.current.variants.error).toBeNull();
    expect(result.current.variants.data).toEqual(stubCatalogItemVariants);
  });

  it("switches both queries to the exact identity of a second selection", async () => {
    const { port, getResource, getItemVariants } = spyPort();
    const { queryClient, wrapper } = setup(port);

    const { result } = renderHook(() => useItemDatabase(query), { wrapper });

    act(() => result.current.selectItem("item", "000F4240"));
    await waitFor(() => expect(result.current.detail.data).toBeDefined());

    act(() => result.current.selectItem("item", "000F4241"));
    await waitFor(() => expect(result.current.selected?.key).toBe("000F4241"));
    await waitFor(() => expect(result.current.detail.data).toBeDefined());
    await waitFor(() => expect(result.current.variants.data).toBeDefined());

    expect(getResource.mock.calls).toEqual([
      [{ kind: "item", key: "000F4240" }],
      [{ kind: "item", key: "000F4241" }],
    ]);
    expect(getItemVariants.mock.calls).toEqual([
      [{ kind: "item", key: "000F4240" }],
      [{ kind: "item", key: "000F4241" }],
    ]);
    expect(queryClient.getQueryData(queryKeys.catalogResource("item", "000F4241"))).toEqual(
      stubCatalogResourceDetail,
    );
    expect(queryClient.getQueryData(queryKeys.catalogItemVariants("item", "000F4241"))).toEqual(
      stubCatalogItemVariants,
    );
  });

  it("drops only the presentational selection on clearSelection", async () => {
    const { port, getResources, getResource, getItemVariants } = spyPort();
    const { queryClient, wrapper } = setup(port);

    const { result } = renderHook(() => useItemDatabase(query), { wrapper });

    act(() => result.current.selectItem("item", "000F4240"));
    await waitFor(() => expect(result.current.detail.data).toBeDefined());
    await waitFor(() => expect(result.current.variants.data).toBeDefined());

    act(() => result.current.clearSelection());
    await waitFor(() => expect(result.current.selected).toBeNull());

    // No new bridge call, and the global catalog cache survives untouched.
    expect(getResources).toHaveBeenCalledOnce();
    expect(getResource).toHaveBeenCalledOnce();
    expect(getItemVariants).toHaveBeenCalledOnce();
    expect(queryClient.getQueryData(queryKeys.catalogResource("item", "000F4240"))).toEqual(
      stubCatalogResourceDetail,
    );
    expect(queryClient.getQueryData(queryKeys.catalogItemVariants("item", "000F4240"))).toEqual(
      stubCatalogItemVariants,
    );
    expect(
      queryClient.getQueryData(
        queryKeys.catalogResources({ ...query, resourceType: "item", endpointID: "" }),
      ),
    ).toBeDefined();
  });
});

describe("useItemDatabase model", () => {
  it("exposes the query results themselves and copies no catalog data into local state", async () => {
    const page: CatalogResourcesPage = {
      resources: [{ kind: "item", key: "goods/unnamed", family: "", name: "" }],
      total: 1,
      page: 1,
      pageSize: 50,
    };
    const { queryClient, wrapper } = setup(
      makeCatalogPort({ getResources: () => Promise.resolve(page) }),
    );

    const { result } = renderHook(() => useItemDatabase(query), { wrapper });

    await waitFor(() => expect(result.current.resources.data).toBeDefined());
    act(() => result.current.selectItem("item", "000F4240"));
    await waitFor(() => expect(result.current.detail.data).toBeDefined());
    await waitFor(() => expect(result.current.variants.data).toBeDefined());

    expect(Object.keys(result.current).sort()).toEqual([
      "clearSelection",
      "detail",
      "resources",
      "selectItem",
      "selected",
      "variants",
    ]);
    // The selection carries the identity and nothing else: no name, no family,
    // no detail and no variants are copied beside it.
    expect(Object.keys(result.current.selected ?? {}).sort()).toEqual(["key", "kind"]);
    // The three results are the cache's own values, not a second copy of them.
    expect(result.current.resources.data).toBe(
      queryClient.getQueryData(
        queryKeys.catalogResources({ ...query, resourceType: "item", endpointID: "" }),
      ),
    );
    expect(result.current.detail.data).toBe(
      queryClient.getQueryData(queryKeys.catalogResource("item", "000F4240")),
    );
    expect(result.current.variants.data).toBe(
      queryClient.getQueryData(queryKeys.catalogItemVariants("item", "000F4240")),
    );
  });

  it("stays a wiring error without the catalog provider", () => {
    const queryClient = new QueryClient();
    const wrapper = ({ children }: { children: ReactNode }) => (
      <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    );
    // React logs the thrown render error; the assertion is on the throw itself.
    const consoleError = vi.spyOn(console, "error").mockImplementation(() => undefined);

    expect(() => renderHook(() => useItemDatabase(query), { wrapper })).toThrow(
      "CatalogPortProvider is missing above this component",
    );

    consoleError.mockRestore();
  });
});
