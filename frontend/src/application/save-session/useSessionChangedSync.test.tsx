import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";
import { makeSaveSessionPort, stubSaveSession } from "../../test/renderWithProviders";
import { queryKeys } from "../queryKeys";
import { SaveSessionPortProvider } from "./saveSessionClient";
import type { SaveSession, SaveSessionPort, SessionChangedEvent } from "./saveSessionPort";
import { compareSequences, nextSequence, useSessionChangedSync } from "./useSessionChangedSync";

const session = stubSaveSession.saveSessionID;

function event(overrides: Partial<SessionChangedEvent> = {}): SessionChangedEvent {
  return {
    sequence: "1",
    operationID: "op-1",
    operationKind: "set_character_name",
    saveSessionID: session,
    saveRevision: "1",
    changedScopes: ["save.session", "character.list", "diagnostics.report"],
    ...overrides,
  };
}

/**
 * Renders the hook with a real query client and a port whose subscription this
 * test drives by hand. The cache is seeded so an invalidation is observable:
 * an invalidated entry reports it on its query state.
 */
function setup(port: Partial<SaveSessionPort> = {}, onSession = vi.fn()) {
  const listeners: ((published: SessionChangedEvent | null) => void)[] = [];
  const unsubscribe = vi.fn();
  // The caller may supply its own reader; the one the port actually uses is the
  // one this helper hands back, so a test always observes the calls it made.
  const getLoadedSave = port.getLoadedSave ?? vi.fn(() => Promise.resolve(stubSaveSession));
  const saveSessionPort = makeSaveSessionPort({
    ...port,
    getLoadedSave,
    subscribeSessionChanged: (listener) => {
      listeners.push(listener);
      return unsubscribe;
    },
  });

  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false, staleTime: Number.POSITIVE_INFINITY } },
  });
  const seed = () => {
    queryClient.setQueryData(queryKeys.saveCharacters(session, "0"), { characters: [] });
    queryClient.setQueryData(queryKeys.characterStats(session, 0, "0"), { level: 1 });
    queryClient.setQueryData(queryKeys.loadedSave(session), stubSaveSession);
  };
  seed();

  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>
      <SaveSessionPortProvider port={saveSessionPort}>{children}</SaveSessionPortProvider>
    </QueryClientProvider>
  );
  const rendered = renderHook(() => useSessionChangedSync(session, onSession), { wrapper });

  const publish = async (published: SessionChangedEvent | null) => {
    await act(async () => {
      for (const listener of listeners) {
        listener(published);
      }
    });
  };
  const invalidated = (key: readonly unknown[]) =>
    queryClient.getQueryState(key)?.isInvalidated === true;

  return {
    rendered,
    publish,
    invalidated,
    getLoadedSave,
    onSession,
    unsubscribe,
    queryClient,
    seed,
  };
}

describe("session.changed synchronisation", () => {
  it("compares canonical decimal sequences without turning them into numbers", () => {
    expect(compareSequences("2", "10")).toBe(-1);
    expect(compareSequences("10", "9")).toBe(1);
    expect(compareSequences("7", "7")).toBe(0);
    // Beyond the exact integer range of a JavaScript number.
    expect(compareSequences("9007199254740993", "9007199254740992")).toBe(1);
    expect(nextSequence("9")).toBe("10");
    expect(nextSequence("9007199254740992")).toBe("9007199254740993");
    expect(nextSequence("not a sequence")).toBeNull();
  });

  it("reads the current session once on start to establish its baseline", async () => {
    const { getLoadedSave } = setup();

    await waitFor(() => expect(getLoadedSave).toHaveBeenCalledWith(session));
  });

  it("refreshes only the getters the changed scopes name", async () => {
    const { publish, invalidated, getLoadedSave, seed } = setup();
    await waitFor(() => expect(getLoadedSave).toHaveBeenCalled());
    seed();

    await publish(event({ sequence: "1", changedScopes: ["character.list"] }));

    expect(invalidated(queryKeys.saveCharacters(session, "0"))).toBe(true);
    // A scope that was not reported must keep its cached view.
    expect(invalidated(queryKeys.characterStats(session, 0, "0"))).toBe(false);
  });

  it("publishes the authoritative session after an ordinary event", async () => {
    const sessions: SaveSession[] = [
      stubSaveSession,
      { ...stubSaveSession, saveRevision: "1", unsavedChanges: true, eventSequence: "1" },
    ];
    const onSession = vi.fn();
    const { publish, getLoadedSave } = setup(
      {
        getLoadedSave: vi.fn(() => Promise.resolve(sessions.shift() ?? stubSaveSession)),
      },
      onSession,
    );
    await waitFor(() => expect(getLoadedSave).toHaveBeenCalledTimes(1));

    await publish(event({ sequence: "1", saveRevision: "1" }));

    await waitFor(() =>
      expect(onSession).toHaveBeenCalledExactlyOnceWith({
        ...stubSaveSession,
        saveRevision: "1",
        unsavedChanges: true,
        eventSequence: "1",
      }),
    );
  });

  it("never lets an older overlapping resynchronisation replace newer state", async () => {
    let resolveStale: (value: SaveSession) => void = () => {};
    const stale = new Promise<SaveSession>((resolve) => {
      resolveStale = resolve;
    });
    let call = 0;
    const getLoadedSave = vi.fn(() => {
      call += 1;
      if (call === 1) {
        return Promise.resolve(stubSaveSession);
      }
      if (call === 2) {
        return stale;
      }
      return Promise.resolve({
        ...stubSaveSession,
        saveRevision: "1",
        unsavedChanges: true,
        eventSequence: "1",
      });
    });
    const onSession = vi.fn();
    const { publish } = setup({ getLoadedSave }, onSession);
    await waitFor(() => expect(getLoadedSave).toHaveBeenCalledTimes(1));

    await publish(null);
    await waitFor(() => expect(getLoadedSave).toHaveBeenCalledTimes(2));
    await publish(event({ sequence: "1", saveRevision: "1" }));
    await waitFor(() => expect(onSession).toHaveBeenCalledTimes(1));

    await act(async () => {
      resolveStale(stubSaveSession);
      await stale;
    });

    expect(onSession).toHaveBeenCalledTimes(1);
    expect(onSession.mock.calls[0][0].eventSequence).toBe("1");
  });

  it("ignores a duplicate and an out-of-order event", async () => {
    const { publish, invalidated, getLoadedSave, seed } = setup();
    await waitFor(() => expect(getLoadedSave).toHaveBeenCalled());
    seed();

    await publish(event({ sequence: "1", changedScopes: ["character.list"] }));

    // The same sequence again, and an older one, must change nothing: neither
    // may reach a scope that was not invalidated before.
    await publish(event({ sequence: "1", changedScopes: ["character.stats"] }));
    await publish(event({ sequence: "0", changedScopes: ["character.stats"] }));

    expect(invalidated(queryKeys.characterStats(session, 0, "0"))).toBe(false);
  });

  it("ignores an event addressed to another session", async () => {
    const { publish, invalidated, getLoadedSave, seed } = setup();
    await waitFor(() => expect(getLoadedSave).toHaveBeenCalled());
    seed();

    await publish(event({ saveSessionID: "other-session", changedScopes: ["character.list"] }));

    expect(invalidated(queryKeys.saveCharacters(session, "0"))).toBe(false);
  });

  it("resynchronises the whole session when the sequence has a gap", async () => {
    // The listener starts on a session at sequence 0 and only afterwards falls
    // behind: the second read is where the backend confirms it moved to 3.
    const sequences = ["0", "3"];
    const { publish, invalidated, getLoadedSave, seed } = setup({
      getLoadedSave: vi.fn(() =>
        Promise.resolve({ ...stubSaveSession, eventSequence: sequences.shift() ?? "3" }),
      ),
    });
    await waitFor(() => expect(getLoadedSave).toHaveBeenCalledTimes(1));
    seed();

    // Sequence 3 arrives while the baseline is still 0: events were lost, so the
    // scopes of this one event cannot describe what changed.
    await publish(event({ sequence: "3", changedScopes: ["character.list"] }));

    await waitFor(() => expect(getLoadedSave).toHaveBeenCalledTimes(2));
    await waitFor(() => expect(invalidated(queryKeys.characterStats(session, 0, "0"))).toBe(true));
    expect(invalidated(queryKeys.saveCharacters(session, "0"))).toBe(true);
  });

  it("does not invalidate twice for the mutation its own caller already refreshed", async () => {
    const { rendered, publish, invalidated, getLoadedSave, seed } = setup();
    await waitFor(() => expect(getLoadedSave).toHaveBeenCalled());
    seed();

    const receipt = {
      operationID: "op-1",
      saveRevision: "1",
      changedScopes: ["character.list"] as const,
    };
    act(() => {
      rendered.result.current.noteAppliedMutation(receipt);
    });

    await publish(event({ sequence: "1", changedScopes: receipt.changedScopes }));

    // The initiator already refreshed from its receipt, so the event only moves
    // the sequence forward.
    expect(invalidated(queryKeys.saveCharacters(session, "0"))).toBe(false);

    // The next event is an ordinary one again and does invalidate.
    await publish(event({ sequence: "2", operationID: "op-2", changedScopes: ["character.list"] }));
    expect(invalidated(queryKeys.saveCharacters(session, "0"))).toBe(true);
  });

  it("resynchronises when the window becomes visible again", async () => {
    const { getLoadedSave } = setup();
    await waitFor(() => expect(getLoadedSave).toHaveBeenCalledTimes(1));

    await act(async () => {
      document.dispatchEvent(new Event("visibilitychange"));
    });

    await waitFor(() => expect(getLoadedSave).toHaveBeenCalledTimes(2));
  });

  it("unsubscribes when it stops listening", async () => {
    const { rendered, unsubscribe, getLoadedSave } = setup();
    await waitFor(() => expect(getLoadedSave).toHaveBeenCalled());

    rendered.unmount();

    expect(unsubscribe).toHaveBeenCalledTimes(1);
  });
});
