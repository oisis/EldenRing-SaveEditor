import { QueryClient, type UseQueryResult } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";
import {
  makeCharacterPort,
  stubCharacterProfile,
  stubCharacterStats,
  stubSaveCharacters,
  TestProviders,
} from "../../test/renderWithProviders";
import { noCharacter, queryKeys } from "../queryKeys";
import type { CharacterPort } from "./characterPort";
import { useCharacterProfile } from "./useCharacterProfile";
import { useCharacterStats } from "./useCharacterStats";
import { useSaveCharacters } from "./useSaveCharacters";

/**
 * The hooks are exercised through an injected `CharacterPort` stub. The
 * generated bindings are never mocked here: that belongs to the adapter test.
 *
 * The client keeps the library defaults on purpose, so a hook that dropped its
 * own `retry: false` would be caught instead of being covered by a test-only
 * default.
 */
function setup(port: CharacterPort) {
  const queryClient = new QueryClient();
  const wrapper = ({ children }: { children: ReactNode }) => (
    <TestProviders queryClient={queryClient} characterPort={port}>
      {children}
    </TestProviders>
  );

  return { queryClient, wrapper };
}

describe("useSaveCharacters", () => {
  it("asks the backend for nothing without a session identifier", async () => {
    const getSaveCharacters = vi.fn(makeCharacterPort().getSaveCharacters);
    const { wrapper } = setup(makeCharacterPort({ getSaveCharacters }));

    const { result, rerender } = renderHook(
      ({ id }: { id?: string }) => useSaveCharacters(id, "0"),
      {
        wrapper,
        initialProps: {},
      },
    );

    expect(result.current.fetchStatus).toBe("idle");
    expect(getSaveCharacters).not.toHaveBeenCalled();

    // An empty identifier is just as absent as a missing one.
    rerender({ id: "" });
    expect(result.current.fetchStatus).toBe("idle");
    expect(getSaveCharacters).not.toHaveBeenCalled();

    rerender({ id: "session-1" });
    await waitFor(() => expect(result.current.data).toEqual(stubSaveCharacters));
    // The identifier reaches the port exactly as given.
    expect(getSaveCharacters).toHaveBeenCalledExactlyOnceWith("session-1");
  });

  it("reaches no port on a manual refetch without a session identifier", async () => {
    const getSaveCharacters = vi.fn(makeCharacterPort().getSaveCharacters);
    const { wrapper } = setup(makeCharacterPort({ getSaveCharacters }));

    const { result, rerender } = renderHook(
      ({ id }: { id?: string }) => useSaveCharacters(id, "0"),
      {
        wrapper,
        initialProps: {},
      },
    );

    expect(getSaveCharacters).not.toHaveBeenCalled();

    // A disabled query is not a guarded one: `refetch` runs the query function
    // regardless, so the missing identifier has to remove the function itself.
    await result.current.refetch();
    expect(getSaveCharacters).not.toHaveBeenCalled();

    rerender({ id: "" });
    await result.current.refetch();
    expect(getSaveCharacters).not.toHaveBeenCalled();
  });

  it("calls the port again on a manual refetch with a session identifier", async () => {
    const getSaveCharactersForSession = vi.fn((saveSessionID: string) =>
      Promise.resolve({ ...stubSaveCharacters, saveSessionID }),
    );
    const configured = setup(makeCharacterPort({ getSaveCharacters: getSaveCharactersForSession }));
    const { result } = renderHook(() => useSaveCharacters("  Session ID  ", "0"), {
      wrapper: configured.wrapper,
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    await result.current.refetch();

    expect(getSaveCharactersForSession).toHaveBeenCalledTimes(2);
    // Both calls carry the identifier unchanged.
    expect(getSaveCharactersForSession.mock.calls).toEqual([
      ["  Session ID  "],
      ["  Session ID  "],
    ]);
  });

  it("reports an inactive slot as an ordinary result", async () => {
    const { wrapper } = setup(makeCharacterPort());

    const { result } = renderHook(() => useSaveCharacters("session-1", "0"), { wrapper });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.isError).toBe(false);
    expect(result.current.data?.characters[1]).toEqual({
      characterID: 1,
      active: false,
      name: "",
      level: 0,
    });
  });

  it("reads the central query key and does not retry a failed call", async () => {
    const getSaveCharacters = vi.fn(() => Promise.reject(new Error("bridge_call_failed")));
    const { queryClient, wrapper } = setup(makeCharacterPort({ getSaveCharacters }));

    const { result } = renderHook(() => useSaveCharacters("session-1", "0"), { wrapper });

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error?.message).toBe("bridge_call_failed");
    expect(getSaveCharacters).toHaveBeenCalledTimes(1);
    expect(
      queryClient.getQueryCache().find({ queryKey: queryKeys.saveCharacters("session-1", "0") }),
    ).toBeDefined();
  });
});

/**
 * The two per-character getters share one enabling rule, one key shape and one
 * failure contract, so they share one suite instead of two copies of it. The
 * port override is built by the caller, which keeps each case bound to the real
 * method of the port rather than to a computed name.
 */
function characterGetterSuite<T extends object>(config: {
  name: string;
  hook: (
    saveSessionID?: string,
    saveRevision?: string,
    characterID?: number,
  ) => UseQueryResult<T, Error>;
  key: (
    saveSessionID: string,
    characterID: number | typeof noCharacter,
    saveRevision: string,
  ) => readonly unknown[];
  stub: T;
  portWith: (call: (saveSessionID: string, characterID: number) => Promise<T>) => CharacterPort;
}) {
  const { name, hook, key, stub, portWith } = config;

  describe(name, () => {
    const resolving = () => vi.fn((_id: string, _characterID: number) => Promise.resolve(stub));

    it("asks the backend for nothing without a session or without a character", () => {
      const call = resolving();
      const { wrapper } = setup(portWith(call));

      const { result, rerender } = renderHook(
        ({ id, characterID }: { id?: string; characterID?: number }) => hook(id, "0", characterID),
        { wrapper, initialProps: { characterID: 0 } as { id?: string; characterID?: number } },
      );

      // No session: nothing to read.
      expect(result.current.fetchStatus).toBe("idle");
      rerender({ id: "", characterID: 0 });
      expect(result.current.fetchStatus).toBe("idle");
      // A session but no character: still nothing to read.
      rerender({ id: "session-1" });
      expect(result.current.fetchStatus).toBe("idle");
      expect(call).not.toHaveBeenCalled();
    });

    it("reaches no port on a manual refetch without a session or without a character", async () => {
      const call = resolving();
      const { wrapper } = setup(portWith(call));

      const { result, rerender } = renderHook(
        ({ id, characterID }: { id?: string; characterID?: number }) => hook(id, "0", characterID),
        { wrapper, initialProps: { characterID: 0 } as { id?: string; characterID?: number } },
      );

      // A disabled query is not a guarded one: `refetch` runs the query function
      // regardless, so a missing parameter has to remove the function itself.
      await result.current.refetch();
      expect(call).not.toHaveBeenCalled();

      rerender({ id: "", characterID: 0 });
      await result.current.refetch();
      expect(call).not.toHaveBeenCalled();

      // A known session with no character selected must not fall back to a slot.
      rerender({ id: "session-1" });
      await result.current.refetch();
      expect(call).not.toHaveBeenCalled();
    });

    it("calls the port again on a manual refetch with both parameters", async () => {
      const call = resolving();
      const { wrapper } = setup(portWith(call));

      const { result } = renderHook(() => hook("session-1", "0", 0), { wrapper });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      await result.current.refetch();

      expect(call).toHaveBeenCalledTimes(2);
      // Both calls carry the arguments unchanged, slot 0 included.
      expect(call.mock.calls).toEqual([
        ["session-1", 0],
        ["session-1", 0],
      ]);
    });

    it("treats character 0 as a character and runs the query", async () => {
      const call = resolving();
      const { wrapper } = setup(portWith(call));

      const { result } = renderHook(() => hook("session-1", "0", 0), { wrapper });

      await waitFor(() => expect(result.current.data).toEqual(stub));
      expect(call).toHaveBeenCalledExactlyOnceWith("session-1", 0);
    });

    it("passes a slot index outside the backend range on instead of rejecting it", async () => {
      const call = resolving();
      const { wrapper } = setup(portWith(call));

      const { result } = renderHook(() => hook("session-1", "0", -1), { wrapper });

      // The backend owns the slot range; the frontend does not pre-empt it.
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(call).toHaveBeenCalledExactlyOnceWith("session-1", -1);

      // A negative slot is a real value, so a manual refetch reaches the port
      // with it unchanged instead of being skipped.
      await result.current.refetch();
      expect(call.mock.calls).toEqual([
        ["session-1", -1],
        ["session-1", -1],
      ]);
    });

    it("reads the central query key and does not retry a failed call", async () => {
      const call = vi.fn((_id: string, _characterID: number) =>
        Promise.reject(new Error("bridge_call_failed")),
      );
      const { queryClient, wrapper } = setup(portWith(call));

      const { result } = renderHook(() => hook("session-1", "0", 0), { wrapper });

      await waitFor(() => expect(result.current.isError).toBe(true));
      expect(result.current.error?.message).toBe("bridge_call_failed");
      expect(call).toHaveBeenCalledTimes(1);
      expect(
        queryClient.getQueryCache().find({ queryKey: key("session-1", 0, "0") }),
      ).toBeDefined();
    });

    it("keeps two characters and two sessions apart in the cache", async () => {
      const call = vi.fn((saveSessionID: string, characterID: number) =>
        Promise.resolve({ ...stub, saveSessionID, characterID }),
      );
      const { queryClient, wrapper } = setup(portWith(call));

      const first = renderHook(() => hook("session-1", "0", 0), { wrapper });
      const second = renderHook(() => hook("session-1", "0", 1), { wrapper });
      const other = renderHook(() => hook("session-2", "0", 0), { wrapper });

      await waitFor(() => expect(first.result.current.isSuccess).toBe(true));
      await waitFor(() => expect(second.result.current.isSuccess).toBe(true));
      await waitFor(() => expect(other.result.current.isSuccess).toBe(true));

      expect(queryClient.getQueryData(key("session-1", 0, "0"))).toMatchObject({
        saveSessionID: "session-1",
        characterID: 0,
      });
      expect(queryClient.getQueryData(key("session-1", 1, "0"))).toMatchObject({
        saveSessionID: "session-1",
        characterID: 1,
      });
      expect(queryClient.getQueryData(key("session-2", 0, "0"))).toMatchObject({
        saveSessionID: "session-2",
        characterID: 0,
      });
      expect(call).toHaveBeenCalledTimes(3);
    });

    it("keys a disabled query away from every real slot index", () => {
      // A slot index is a number, so the unselected placeholder can never
      // collide with one, negative indices included.
      expect(key("session-1", noCharacter, "0")).not.toEqual(key("session-1", -1, "0"));
    });
  });
}

characterGetterSuite({
  name: "useCharacterProfile",
  hook: useCharacterProfile,
  key: queryKeys.characterProfile,
  stub: stubCharacterProfile,
  portWith: (getCharacterProfile) => makeCharacterPort({ getCharacterProfile }),
});

characterGetterSuite({
  name: "useCharacterStats",
  hook: useCharacterStats,
  key: queryKeys.characterStats,
  stub: stubCharacterStats,
  portWith: (getCharacterStats) => makeCharacterPort({ getCharacterStats }),
});

describe("character query keys", () => {
  it("keeps one source of truth below the session prefix", () => {
    expect(queryKeys.saveCharacters("session-1", "0")).toEqual([
      "save-session",
      "session-1",
      "characters",
      "0",
    ]);
    expect(queryKeys.characterProfile("session-1", 0, "0")).toEqual([
      "save-session",
      "session-1",
      "character",
      0,
      "profile",
      "0",
    ]);
    expect(queryKeys.characterStats("session-1", 0, "0")).toEqual([
      "save-session",
      "session-1",
      "character",
      0,
      "stats",
      "0",
    ]);

    // Every character key sits under the session prefix, so closing the session
    // removes them without a second cleanup rule.
    const prefix = queryKeys.saveSession("session-1");
    for (const characterKey of [
      queryKeys.saveCharacters("session-1", "0"),
      queryKeys.characterProfile("session-1", 0, "0"),
      queryKeys.characterStats("session-1", 0, "0"),
    ]) {
      expect(characterKey.slice(0, prefix.length)).toEqual([...prefix]);
    }
  });
});
