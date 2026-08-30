import { QueryClient } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";
import {
  makeSaveSessionPort,
  stubSaveSession,
  TestProviders,
} from "../../test/renderWithProviders";
import { queryKeys } from "../queryKeys";
import type { SaveSessionPort } from "./saveSessionPort";
import { useCloseSave } from "./useCloseSave";
import { useLoadedSave } from "./useLoadedSave";
import { useLoadSave } from "./useLoadSave";

/**
 * The hooks are exercised through an injected `SaveSessionPort` stub. The
 * generated bindings are never mocked here: that belongs to the adapter test.
 *
 * The client keeps the library defaults on purpose, so a hook that dropped its
 * own `retry: false` would be caught instead of being covered by a test-only
 * default.
 */
function setup(port: SaveSessionPort) {
  const queryClient = new QueryClient();
  const wrapper = ({ children }: { children: ReactNode }) => (
    <TestProviders queryClient={queryClient} saveSessionPort={port}>
      {children}
    </TestProviders>
  );

  return { queryClient, wrapper };
}

describe("useLoadedSave", () => {
  it("asks the backend for nothing without a session identifier", async () => {
    const getLoadedSave = vi.fn(makeSaveSessionPort().getLoadedSave);
    const { wrapper } = setup(makeSaveSessionPort({ getLoadedSave }));

    const { result, rerender } = renderHook(({ id }: { id?: string }) => useLoadedSave(id), {
      wrapper,
      initialProps: {},
    });

    expect(result.current.fetchStatus).toBe("idle");
    expect(getLoadedSave).not.toHaveBeenCalled();

    // An empty identifier is just as absent as a missing one.
    rerender({ id: "" });
    expect(result.current.fetchStatus).toBe("idle");
    expect(getLoadedSave).not.toHaveBeenCalled();

    rerender({ id: "session-1" });
    await waitFor(() => expect(result.current.data).toEqual(stubSaveSession));
    expect(getLoadedSave).toHaveBeenCalledExactlyOnceWith("session-1");
  });

  it("reaches no port on a manual refetch without a session identifier", async () => {
    const getLoadedSave = vi.fn(makeSaveSessionPort().getLoadedSave);
    const { wrapper } = setup(makeSaveSessionPort({ getLoadedSave }));

    const { result, rerender } = renderHook(({ id }: { id?: string }) => useLoadedSave(id), {
      wrapper,
      initialProps: {},
    });

    expect(getLoadedSave).not.toHaveBeenCalled();

    // A disabled query is not a guarded one: `refetch` runs the query function
    // regardless, so the missing identifier has to remove the function itself.
    await result.current.refetch();
    expect(getLoadedSave).not.toHaveBeenCalled();

    rerender({ id: "" });
    await result.current.refetch();
    expect(getLoadedSave).not.toHaveBeenCalled();
  });

  it("calls the port again on a manual refetch with a session identifier", async () => {
    const getLoadedSave = vi.fn(makeSaveSessionPort().getLoadedSave);
    const { wrapper } = setup(makeSaveSessionPort({ getLoadedSave }));

    const { result } = renderHook(() => useLoadedSave("  Session ID  "), { wrapper });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    await result.current.refetch();

    expect(getLoadedSave).toHaveBeenCalledTimes(2);
    // Both calls carry the identifier unchanged.
    expect(getLoadedSave.mock.calls).toEqual([["  Session ID  "], ["  Session ID  "]]);
  });

  it("reads the central query key and does not retry a failed call", async () => {
    const getLoadedSave = vi.fn(() => Promise.reject(new Error("bridge_call_failed")));
    const { queryClient, wrapper } = setup(makeSaveSessionPort({ getLoadedSave }));

    const { result } = renderHook(() => useLoadedSave("session-1"), { wrapper });

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(getLoadedSave).toHaveBeenCalledTimes(1);
    expect(
      queryClient.getQueryCache().find({ queryKey: queryKeys.loadedSave("session-1") }),
    ).toBeDefined();
  });

  it("keeps one source of truth for the session query keys", () => {
    expect(queryKeys.saveSession("session-1")).toEqual(["save-session", "session-1"]);
    expect(queryKeys.loadedSave("session-1")).toEqual(["save-session", "session-1", "loaded"]);
  });
});

describe("useLoadSave", () => {
  it("caches a successful load under the session it created", async () => {
    const loaded = { ...stubSaveSession, saveSessionID: "session-42" };
    const loadSave = vi.fn(() => Promise.resolve(loaded));
    const { queryClient, wrapper } = setup(makeSaveSessionPort({ loadSave }));

    const { result } = renderHook(() => useLoadSave(), { wrapper });

    result.current.mutate({ source: "  /saves/ER0000.sl2  ", expectedPlatform: "  pc  " });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    // The arguments reach the port exactly as given.
    expect(loadSave).toHaveBeenCalledExactlyOnceWith("  /saves/ER0000.sl2  ", "  pc  ");
    expect(queryClient.getQueryData(queryKeys.loadedSave("session-42"))).toEqual(loaded);
    // Nothing is written under any other session.
    expect(queryClient.getQueryData(queryKeys.loadedSave(stubSaveSession.saveSessionID))).toBe(
      undefined,
    );
  });

  it("writes no partial state when the load fails", async () => {
    const loadSave = vi.fn(() => Promise.reject(new Error("bridge_call_failed")));
    const { queryClient, wrapper } = setup(makeSaveSessionPort({ loadSave }));

    const { result } = renderHook(() => useLoadSave(), { wrapper });

    result.current.mutate({ source: "/saves/ER0000.sl2", expectedPlatform: "pc" });

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(queryClient.getQueryCache().getAll()).toEqual([]);
  });
});

describe("useCloseSave", () => {
  it("drops every cached view of the session once the backend confirms the close", async () => {
    const { queryClient, wrapper } = setup(makeSaveSessionPort());
    queryClient.setQueryData(queryKeys.loadedSave("session-1"), stubSaveSession);
    queryClient.setQueryData(
      [...queryKeys.saveSession("session-1"), "characters"],
      ["a future per-session view"],
    );
    queryClient.setQueryData(queryKeys.loadedSave("session-2"), stubSaveSession);

    const { result } = renderHook(() => useCloseSave(), { wrapper });

    result.current.mutate("session-1");

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(queryClient.getQueryData(queryKeys.loadedSave("session-1"))).toBe(undefined);
    expect(queryClient.getQueryData([...queryKeys.saveSession("session-1"), "characters"])).toBe(
      undefined,
    );
    // Another session is untouched.
    expect(queryClient.getQueryData(queryKeys.loadedSave("session-2"))).toEqual(stubSaveSession);
  });

  it("drops the character views of the closed session and keeps another session's", async () => {
    const { queryClient, wrapper } = setup(makeSaveSessionPort());
    for (const saveSessionID of ["session-1", "session-2"]) {
      queryClient.setQueryData(queryKeys.saveCharacters(saveSessionID), { saveSessionID });
      queryClient.setQueryData(queryKeys.characterProfile(saveSessionID, 0), { saveSessionID });
      queryClient.setQueryData(queryKeys.characterStats(saveSessionID, 9), { saveSessionID });
    }

    const { result } = renderHook(() => useCloseSave(), { wrapper });

    result.current.mutate("session-1");

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    // The session prefix is the only cleanup rule the character views need.
    expect(queryClient.getQueryData(queryKeys.saveCharacters("session-1"))).toBe(undefined);
    expect(queryClient.getQueryData(queryKeys.characterProfile("session-1", 0))).toBe(undefined);
    expect(queryClient.getQueryData(queryKeys.characterStats("session-1", 9))).toBe(undefined);
    expect(queryClient.getQueryData(queryKeys.saveCharacters("session-2"))).toEqual({
      saveSessionID: "session-2",
    });
    expect(queryClient.getQueryData(queryKeys.characterProfile("session-2", 0))).toEqual({
      saveSessionID: "session-2",
    });
    expect(queryClient.getQueryData(queryKeys.characterStats("session-2", 9))).toEqual({
      saveSessionID: "session-2",
    });
  });

  it("keeps the cache and reports the failure when the close is rejected", async () => {
    const closeSave = vi.fn(() => Promise.reject(new Error("bridge_call_failed")));
    const { queryClient, wrapper } = setup(makeSaveSessionPort({ closeSave }));
    queryClient.setQueryData(queryKeys.loadedSave("session-1"), stubSaveSession);

    const { result } = renderHook(() => useCloseSave(), { wrapper });

    result.current.mutate("session-1");

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error?.message).toBe("bridge_call_failed");
    expect(queryClient.getQueryData(queryKeys.loadedSave("session-1"))).toEqual(stubSaveSession);
  });
});
