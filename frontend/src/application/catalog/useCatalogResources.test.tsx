import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";
import { makeCatalogPort, stubCatalogPage, TestProviders } from "../../test/renderWithProviders";
import { queryKeys } from "../queryKeys";
import { useCloseSave } from "../save-session/useCloseSave";
import { CatalogPortProvider } from "./catalogClient";
import type { CatalogPort, CatalogResourcesRequest } from "./catalogPort";
import { useCatalogResources } from "./useCatalogResources";

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

const request: CatalogResourcesRequest = {
  resourceType: "item",
  family: "weapon",
  capability: "upgrade",
  endpointID: "",
  search: "  Uchi  ",
  page: 2,
  pageSize: 25,
};

describe("useCatalogResources", () => {
  it("passes the whole request to the port exactly as given", async () => {
    const getResources = vi.fn(makeCatalogPort().getResources);
    const { wrapper } = setup(makeCatalogPort({ getResources }));

    const { result } = renderHook(() => useCatalogResources(request), { wrapper });

    await waitFor(() => expect(result.current.data).toEqual(stubCatalogPage));
    // No trimming, no recasing, no filter default and no paging normalisation:
    // the backend owns all of them.
    expect(getResources).toHaveBeenCalledExactlyOnceWith(request);
  });

  it("treats empty filters and zero paging as a real request, not as a missing one", async () => {
    const getResources = vi.fn(makeCatalogPort().getResources);
    const { wrapper } = setup(makeCatalogPort({ getResources }));
    const empty: CatalogResourcesRequest = {
      resourceType: "",
      family: "",
      capability: "",
      endpointID: "",
      search: "",
      page: 0,
      pageSize: 0,
    };

    const { result } = renderHook(() => useCatalogResources(empty), { wrapper });

    await waitFor(() => expect(result.current.data).toEqual(stubCatalogPage));
    expect(getResources).toHaveBeenCalledExactlyOnceWith(empty);
  });

  it("keeps the page and page size the backend served, not the ones requested", async () => {
    const served = { ...stubCatalogPage, page: 1, pageSize: 50, total: 412 };
    const { wrapper } = setup(makeCatalogPort({ getResources: () => Promise.resolve(served) }));

    const { result } = renderHook(() => useCatalogResources({ ...request, page: 0, pageSize: 0 }), {
      wrapper,
    });

    await waitFor(() => expect(result.current.data).toBeDefined());
    expect(result.current.data?.page).toBe(1);
    expect(result.current.data?.pageSize).toBe(50);
    expect(result.current.data?.total).toBe(412);
  });

  it("carries an empty name and an empty family through without a fallback", async () => {
    const { wrapper } = setup(makeCatalogPort());

    const { result } = renderHook(() => useCatalogResources(request), { wrapper });

    await waitFor(() => expect(result.current.data).toBeDefined());
    const nameless = result.current.data?.resources[1];
    // The key is never promoted to a name and the family is never guessed.
    expect(nameless?.name).toBe("");
    expect(nameless?.family).toBe("");
    expect(nameless?.key).toBe("goods/unnamed");
  });

  it("reads the catalog with no save session and no character", async () => {
    const getResources = vi.fn(makeCatalogPort().getResources);
    // Deliberately no SaveSessionPortProvider, no CharacterPortProvider and no
    // session identifier anywhere: the catalog is global.
    const wrapper = ({ children }: { children: ReactNode }) => (
      <QueryClientProvider client={new QueryClient()}>
        <CatalogPortProvider port={makeCatalogPort({ getResources })}>
          {children}
        </CatalogPortProvider>
      </QueryClientProvider>
    );

    const { result } = renderHook(() => useCatalogResources(request), { wrapper });

    await waitFor(() => expect(result.current.data).toEqual(stubCatalogPage));
    expect(getResources).toHaveBeenCalledExactlyOnceWith(request);
  });

  it("reports a rejected call without retrying it", async () => {
    const getResources = vi.fn(() => Promise.reject(new Error("bridge_call_failed")));
    const { wrapper } = setup(makeCatalogPort({ getResources }));

    const { result } = renderHook(() => useCatalogResources(request), { wrapper });

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(getResources).toHaveBeenCalledTimes(1);
  });

  it("fails as a wiring error when no CatalogPortProvider is above the hook", () => {
    // The port is read before any query is set up, so a tree without the
    // provider fails immediately instead of silently rendering an empty list.
    expect(() => renderHook(() => useCatalogResources(request))).toThrow(
      "CatalogPortProvider is missing above this component",
    );
  });
});

describe("catalog query keys", () => {
  const base = queryKeys.catalogResources(request);

  it("starts with the global catalog prefix, outside every save session", () => {
    expect(base.slice(0, 2)).toEqual(["catalog", "resources"]);
    expect(base).not.toContain("save-session");
  });

  it("tells every one of the seven backend arguments apart", () => {
    const changes: CatalogResourcesRequest[] = [
      { ...request, resourceType: "class" },
      { ...request, family: "armor" },
      { ...request, capability: "equipment" },
      { ...request, endpointID: "get_resources" },
      { ...request, search: "Uchi" },
      { ...request, page: 3 },
      { ...request, pageSize: 50 },
    ];

    for (const changed of changes) {
      expect(queryKeys.catalogResources(changed)).not.toEqual(base);
    }
    expect(queryKeys.catalogResources({ ...request })).toEqual(base);
  });

  it("survives CloseSave, which only removes the closed session scope", async () => {
    const queryClient = new QueryClient();
    queryClient.setQueryData(base, stubCatalogPage);
    queryClient.setQueryData(queryKeys.loadedSave("session-1"), { saveSessionID: "session-1" });
    const wrapper = ({ children }: { children: ReactNode }) => (
      <TestProviders queryClient={queryClient}>{children}</TestProviders>
    );

    const { result } = renderHook(() => useCloseSave(), { wrapper });
    result.current.mutate("session-1");

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    // The session scope is gone; the global catalog page is untouched.
    expect(queryClient.getQueryData(queryKeys.loadedSave("session-1"))).toBeUndefined();
    expect(queryClient.getQueryData(base)).toEqual(stubCatalogPage);
  });
});
