import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";
import { makeCatalogPort } from "../../test/renderWithProviders";
import { queryKeys } from "../queryKeys";
import { CatalogPortProvider } from "./catalogClient";
import type { CatalogPort, CatalogResourcePresentationIdentity } from "./catalogPort";
import { useCatalogResourcePresentationSummaries } from "./useCatalogResourcePresentationSummaries";

function setup(port: CatalogPort) {
  const queryClient = new QueryClient();
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>
      <CatalogPortProvider port={port}>{children}</CatalogPortProvider>
    </QueryClientProvider>
  );
  return { queryClient, wrapper };
}

const identities: readonly CatalogResourcePresentationIdentity[] = [
  { kind: " item ", key: " 000F4240 " },
  { kind: "class", key: "0" },
  { kind: " item ", key: " 000F4240 " },
];

describe("useCatalogResourcePresentationSummaries", () => {
  it("forwards order, duplicates and unnormalised identities exactly", async () => {
    const getResourcePresentationSummaries = vi.fn(
      makeCatalogPort().getResourcePresentationSummaries,
    );
    const { wrapper } = setup(makeCatalogPort({ getResourcePresentationSummaries }));

    const { result } = renderHook(() => useCatalogResourcePresentationSummaries(identities), {
      wrapper,
    });
    await waitFor(() => expect(result.current.data).toBeDefined());
    expect(getResourcePresentationSummaries).toHaveBeenCalledExactlyOnceWith(identities);
    expect(result.current.data?.resources.map(({ kind, key }) => ({ kind, key }))).toEqual(
      identities,
    );
  });

  it("treats an empty array as a real request and undefined as unselected", async () => {
    const getResourcePresentationSummaries = vi.fn(
      makeCatalogPort().getResourcePresentationSummaries,
    );
    const { wrapper } = setup(makeCatalogPort({ getResourcePresentationSummaries }));

    const empty = renderHook(() => useCatalogResourcePresentationSummaries([]), { wrapper });
    await waitFor(() => expect(empty.result.current.data).toEqual({ resources: [] }));
    expect(getResourcePresentationSummaries).toHaveBeenCalledExactlyOnceWith([]);

    const absent = renderHook(() => useCatalogResourcePresentationSummaries(undefined), {
      wrapper,
    });
    await waitFor(() => expect(absent.result.current.fetchStatus).toBe("idle"));
    await absent.result.current.refetch();
    expect(getResourcePresentationSummaries).toHaveBeenCalledTimes(1);
  });

  it("does not retry a rejected batch", async () => {
    const getResourcePresentationSummaries = vi.fn(() =>
      Promise.reject(new Error("bridge_call_failed")),
    );
    const { wrapper } = setup(makeCatalogPort({ getResourcePresentationSummaries }));
    const { result } = renderHook(() => useCatalogResourcePresentationSummaries(identities), {
      wrapper,
    });
    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(getResourcePresentationSummaries).toHaveBeenCalledTimes(1);
  });
});

describe("catalog presentation query keys", () => {
  it("distinguishes order, duplicates, values, empty and unselected batches", () => {
    const base = queryKeys.catalogResourcePresentationSummaries(identities);
    expect(base.slice(0, 2)).toEqual(["catalog", "resource-presentation-summaries"]);
    expect(base).not.toContain("save-session");
    expect(queryKeys.catalogResourcePresentationSummaries([...identities])).toEqual(base);
    expect(
      queryKeys.catalogResourcePresentationSummaries([identities[1], identities[0], identities[2]]),
    ).not.toEqual(base);
    expect(queryKeys.catalogResourcePresentationSummaries(identities.slice(0, 2))).not.toEqual(
      base,
    );
    expect(queryKeys.catalogResourcePresentationSummaries([])).not.toEqual(
      queryKeys.catalogResourcePresentationSummaries(null),
    );
  });
});
