import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";
import {
  makeCatalogPort,
  stubCatalogItemVariants,
  TestProviders,
} from "../../test/renderWithProviders";
import { queryKeys } from "../queryKeys";
import { useCloseSave } from "../save-session/useCloseSave";
import { CatalogPortProvider } from "./catalogClient";
import type { CatalogPort } from "./catalogPort";
import { useCatalogItemVariants } from "./useCatalogItemVariants";

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

describe("useCatalogItemVariants", () => {
  it("passes the kind and the key to the port exactly as given", async () => {
    const getItemVariants = vi.fn(makeCatalogPort().getItemVariants);
    const { wrapper } = setup(makeCatalogPort({ getItemVariants }));

    const { result } = renderHook(() => useCatalogItemVariants("  Item  ", " 000f4240 "), {
      wrapper,
    });

    await waitFor(() => expect(result.current.data).toEqual(stubCatalogItemVariants));
    // No trimming, no recasing and no alias: the backend owns the identity.
    expect(getItemVariants).toHaveBeenCalledExactlyOnceWith({
      kind: "  Item  ",
      key: " 000f4240 ",
    });
  });

  it("carries the five facts of every variant through with their provenance", async () => {
    const { wrapper } = setup(makeCatalogPort());

    const { result } = renderHook(() => useCatalogItemVariants("item", "000F4240"), { wrapper });

    await waitFor(() => expect(result.current.data).toBeDefined());
    const variants = result.current.data?.variants ?? [];
    for (const variant of variants) {
      expect(Object.keys(variant)).toEqual([
        "gameID",
        "kind",
        "affinity",
        "upgradeLevel",
        "sourceRowID",
      ]);
      expect(variant.gameID.provenance.source).toBe("legacy_db_data");
    }
    // An unresolved fact keeps its raw empty string and its raw zero.
    expect(variants[1]?.affinity).toEqual({
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
    expect(variants[1]?.sourceRowID.value).toBe(0);
    // A known fact whose value happens to be zero stays known.
    expect(variants[0]?.upgradeLevel.known).toBe(true);
    expect(variants[0]?.upgradeLevel.value).toBe(0);
    // The variant document data and the parameter records are not part of the
    // application contract of this step.
    expect(JSON.stringify(result.current.data)).not.toContain("sourceRecords");
    expect(JSON.stringify(result.current.data)).not.toContain('"data"');
  });

  it("keeps the catalog order the port reported", async () => {
    const { wrapper } = setup(makeCatalogPort());

    const { result } = renderHook(() => useCatalogItemVariants("item", "000F4240"), { wrapper });

    await waitFor(() => expect(result.current.data).toBeDefined());
    expect(result.current.data?.variants.map((variant) => variant.gameID.value)).toEqual([
      1000100, 1000001,
    ]);
  });

  it("reports an item without variants as an empty list, not as a failure", async () => {
    const getItemVariants = vi.fn(() => Promise.resolve({ variants: [] }));
    const { wrapper } = setup(makeCatalogPort({ getItemVariants }));

    const { result } = renderHook(() => useCatalogItemVariants("item", "10009C40"), { wrapper });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toEqual({ variants: [] });
    expect(result.current.isError).toBe(false);
  });

  it("treats an empty kind or key as a real request, not as a missing one", async () => {
    const getItemVariants = vi.fn(makeCatalogPort().getItemVariants);
    const { wrapper } = setup(makeCatalogPort({ getItemVariants }));

    const { result } = renderHook(() => useCatalogItemVariants("", ""), { wrapper });

    await waitFor(() => expect(result.current.data).toBeDefined());
    // The backend rejects the empty pair; the hook must not decide that itself.
    expect(getItemVariants).toHaveBeenCalledExactlyOnceWith({ kind: "", key: "" });
  });

  it("never reaches the port while nothing is selected", async () => {
    const getItemVariants = vi.fn(makeCatalogPort().getItemVariants);
    const { wrapper } = setup(makeCatalogPort({ getItemVariants }));

    const { result } = renderHook(() => useCatalogItemVariants(undefined, undefined), { wrapper });

    await waitFor(() => expect(result.current.fetchStatus).toBe("idle"));
    expect(getItemVariants).not.toHaveBeenCalled();

    // `enabled` would still run the query function here; `skipToken` cannot.
    await result.current.refetch();
    expect(getItemVariants).not.toHaveBeenCalled();
  });

  it("reads the variants with no save session and no character", async () => {
    const getItemVariants = vi.fn(makeCatalogPort().getItemVariants);
    // Deliberately no SaveSessionPortProvider, no CharacterPortProvider and no
    // session identifier anywhere: the catalog is global.
    const wrapper = ({ children }: { children: ReactNode }) => (
      <QueryClientProvider client={new QueryClient()}>
        <CatalogPortProvider port={makeCatalogPort({ getItemVariants })}>
          {children}
        </CatalogPortProvider>
      </QueryClientProvider>
    );

    const { result } = renderHook(() => useCatalogItemVariants("item", "000F4240"), { wrapper });

    await waitFor(() => expect(result.current.data).toEqual(stubCatalogItemVariants));
    expect(getItemVariants).toHaveBeenCalledExactlyOnceWith({ kind: "item", key: "000F4240" });
  });

  it("reports a rejected call without retrying it", async () => {
    const getItemVariants = vi.fn(() => Promise.reject(new Error("bridge_call_failed")));
    const { wrapper } = setup(makeCatalogPort({ getItemVariants }));

    const { result } = renderHook(() => useCatalogItemVariants("item", "000F4240"), { wrapper });

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(getItemVariants).toHaveBeenCalledTimes(1);
    // Only the stable code reaches the hook: no Go error text and no path.
    expect(result.current.error?.message).toBe("bridge_call_failed");
  });

  it("fails as a wiring error when no CatalogPortProvider is above the hook", () => {
    expect(() => renderHook(() => useCatalogItemVariants("item", "000F4240"))).toThrow(
      "CatalogPortProvider is missing above this component",
    );
  });
});

describe("catalog item variants query keys", () => {
  const base = queryKeys.catalogItemVariants("item", "000F4240");

  it("starts with the global catalog prefix, outside every save session", () => {
    expect(base.slice(0, 2)).toEqual(["catalog", "item-variants"]);
    expect(base).not.toContain("save-session");
  });

  it("tells the kind and the key apart, including the empty and unselected ones", () => {
    const distinct = [
      queryKeys.catalogItemVariants("class", "000F4240"),
      queryKeys.catalogItemVariants("item", "000F4241"),
      queryKeys.catalogItemVariants("item", ""),
      queryKeys.catalogItemVariants("", "000F4240"),
      // Nothing selected is its own entry and never collides with a real value.
      queryKeys.catalogItemVariants(null, null),
      queryKeys.catalogItemVariants("item", null),
    ];

    for (const key of distinct) {
      expect(key).not.toEqual(base);
    }
    expect(new Set(distinct.map((key) => JSON.stringify(key))).size).toBe(distinct.length);
    expect(queryKeys.catalogItemVariants("item", "000F4240")).toEqual(base);
    // The variants are their own branch of the catalog, never the detail entry.
    expect(base).not.toEqual(queryKeys.catalogResource("item", "000F4240"));
  });

  it("survives CloseSave, which only removes the closed session scope", async () => {
    const queryClient = new QueryClient();
    queryClient.setQueryData(base, stubCatalogItemVariants);
    queryClient.setQueryData(queryKeys.loadedSave("session-1"), { saveSessionID: "session-1" });
    const wrapper = ({ children }: { children: ReactNode }) => (
      <TestProviders queryClient={queryClient}>{children}</TestProviders>
    );

    const { result } = renderHook(() => useCloseSave(), { wrapper });
    result.current.mutate("session-1");

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    // The session scope is gone; the global catalog variants are untouched.
    expect(queryClient.getQueryData(queryKeys.loadedSave("session-1"))).toBeUndefined();
    expect(queryClient.getQueryData(base)).toEqual(stubCatalogItemVariants);
  });
});
