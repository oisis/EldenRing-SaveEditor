import { act, renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";
import type {
  CharacterPort,
  CharacterSummary,
  SaveCharacters,
} from "../../application/character/characterPort";
import {
  createTestQueryClient,
  makeCharacterPort,
  stubCharacterProfile,
  stubCharacterStats,
  stubSlotsFor,
  TestProviders,
} from "../../test/renderWithProviders";
import { useCharacterSelection } from "./useCharacterSelection";

/**
 * The controller is exercised through an injected `CharacterPort`. The
 * generated bindings are never mocked here: a selection rule must break these
 * tests when the application contract changes, not when the transport does.
 */
function summary(characterID: number, overrides: Partial<CharacterSummary> = {}): CharacterSummary {
  return { characterID, active: false, name: "", level: 0, ...overrides };
}

function saveCharacters(
  saveSessionID: string,
  characters: readonly CharacterSummary[],
): SaveCharacters {
  return { saveSessionID, saveRevision: "0", characters, slots: stubSlotsFor(characters) };
}

function setup(port: CharacterPort) {
  const queryClient = createTestQueryClient();
  const wrapper = ({ children }: { children: ReactNode }) => (
    <TestProviders queryClient={queryClient} characterPort={port}>
      {children}
    </TestProviders>
  );

  return { queryClient, wrapper };
}

function spyPort(overrides: Partial<CharacterPort>) {
  const getCharacterProfile = vi.fn((saveSessionID: string, characterID: number) =>
    Promise.resolve({ ...stubCharacterProfile, saveSessionID, characterID }),
  );
  const getCharacterStats = vi.fn((saveSessionID: string, characterID: number) =>
    Promise.resolve({ ...stubCharacterStats, saveSessionID, characterID }),
  );

  return {
    getCharacterProfile,
    getCharacterStats,
    port: makeCharacterPort({ getCharacterProfile, getCharacterStats, ...overrides }),
  };
}

describe("useCharacterSelection default selection", () => {
  it("selects slot 0 when it is active, wherever the backend lists it", async () => {
    // Slot 0 is deliberately the last element of the reported array.
    const characters = [
      summary(3, { active: true, name: "Third", level: 60 }),
      summary(1, { active: true, name: "First", level: 40 }),
      summary(0, { active: true, name: "Zero", level: 150 }),
    ];
    const { port, getCharacterProfile, getCharacterStats } = spyPort({
      getSaveCharacters: (id) => Promise.resolve(saveCharacters(id, characters)),
    });
    const { wrapper } = setup(port);

    const { result } = renderHook(() => useCharacterSelection("session-1", "0"), { wrapper });

    await waitFor(() => expect(result.current.selectedCharacterID).toBe(0));
    expect(getCharacterProfile).toHaveBeenCalledExactlyOnceWith("session-1", 0);
    expect(getCharacterStats).toHaveBeenCalledExactlyOnceWith("session-1", 0);
    // The reported order is kept inside the active group.
    expect(result.current.activeCharacters.map((c) => c.characterID)).toEqual([3, 1, 0]);
  });

  it("selects the first active character when slot 0 is inactive", async () => {
    const characters = [
      summary(0),
      summary(1),
      summary(4, { active: true, name: "Fourth", level: 80 }),
      summary(2, { active: true, name: "Second", level: 30 }),
    ];
    const { port, getCharacterProfile, getCharacterStats } = spyPort({
      getSaveCharacters: (id) => Promise.resolve(saveCharacters(id, characters)),
    });
    const { wrapper } = setup(port);

    const { result } = renderHook(() => useCharacterSelection("session-1", "0"), { wrapper });

    // "First active" is the first one the backend reports, not the lowest index.
    await waitFor(() => expect(result.current.selectedCharacterID).toBe(4));
    expect(getCharacterProfile).toHaveBeenCalledExactlyOnceWith("session-1", 4);
    expect(getCharacterStats).toHaveBeenCalledExactlyOnceWith("session-1", 4);
  });

  it("selects nothing and reaches no per-character port without an active character", async () => {
    const { port, getCharacterProfile, getCharacterStats } = spyPort({
      getSaveCharacters: (id) => Promise.resolve(saveCharacters(id, [summary(0), summary(1)])),
    });
    const { wrapper } = setup(port);

    const { result } = renderHook(() => useCharacterSelection("session-1", "0"), { wrapper });

    await waitFor(() => expect(result.current.characters.isSuccess).toBe(true));
    expect(result.current.selectedCharacterID).toBeUndefined();
    expect(result.current.profile.fetchStatus).toBe("idle");
    expect(result.current.stats.fetchStatus).toBe("idle");
    expect(getCharacterProfile).not.toHaveBeenCalled();
    expect(getCharacterStats).not.toHaveBeenCalled();

    // A manual refetch must not reach the port either.
    await act(async () => {
      await result.current.profile.refetch();
      await result.current.stats.refetch();
    });
    expect(getCharacterProfile).not.toHaveBeenCalled();
    expect(getCharacterStats).not.toHaveBeenCalled();
  });

  it("reaches no character port at all without a session", async () => {
    const getSaveCharacters = vi.fn(makeCharacterPort().getSaveCharacters);
    const { port, getCharacterProfile, getCharacterStats } = spyPort({ getSaveCharacters });
    const { wrapper } = setup(port);

    const { result, rerender } = renderHook(
      ({ id }: { id?: string }) => useCharacterSelection(id, "0"),
      {
        wrapper,
        initialProps: {},
      },
    );

    expect(result.current.hasSession).toBe(false);
    expect(result.current.selectedCharacterID).toBeUndefined();
    expect(result.current.characters.fetchStatus).toBe("idle");

    // An empty identifier is just as absent as a missing one.
    rerender({ id: "" });
    expect(result.current.hasSession).toBe(false);

    expect(getSaveCharacters).not.toHaveBeenCalled();
    expect(getCharacterProfile).not.toHaveBeenCalled();
    expect(getCharacterStats).not.toHaveBeenCalled();
  });
});

describe("useCharacterSelection manual selection", () => {
  const characters = [
    summary(0, { active: true, name: "Zero", level: 150 }),
    summary(1, { active: true, name: "One", level: 90 }),
    summary(2),
  ];

  it("drives the profile and the statistics of the chosen character", async () => {
    const { port, getCharacterProfile, getCharacterStats } = spyPort({
      getSaveCharacters: (id) => Promise.resolve(saveCharacters(id, characters)),
    });
    const { wrapper } = setup(port);

    const { result } = renderHook(() => useCharacterSelection("session-1", "0"), { wrapper });

    await waitFor(() => expect(result.current.selectedCharacterID).toBe(0));

    act(() => result.current.selectCharacter(1));

    await waitFor(() => expect(result.current.selectedCharacterID).toBe(1));
    expect(getCharacterProfile.mock.calls).toEqual([
      ["session-1", 0],
      ["session-1", 1],
    ]);
    expect(getCharacterStats.mock.calls).toEqual([
      ["session-1", 0],
      ["session-1", 1],
    ]);
    await waitFor(() => expect(result.current.profile.data?.characterID).toBe(1));
    expect(result.current.stats.data?.characterID).toBe(1);
  });

  it("keeps slot 0 a valid explicit selection", async () => {
    const { port, getCharacterProfile } = spyPort({
      getSaveCharacters: (id) => Promise.resolve(saveCharacters(id, characters)),
    });
    const { wrapper } = setup(port);

    const { result } = renderHook(() => useCharacterSelection("session-1", "0"), { wrapper });

    await waitFor(() => expect(result.current.selectedCharacterID).toBe(0));

    act(() => result.current.selectCharacter(1));
    await waitFor(() => expect(result.current.selectedCharacterID).toBe(1));

    act(() => result.current.selectCharacter(0));
    await waitFor(() => expect(result.current.selectedCharacterID).toBe(0));
    expect(getCharacterProfile.mock.calls.at(-1)).toEqual(["session-1", 0]);
  });

  it("ignores a slot the backend does not report as active", async () => {
    const { port, getCharacterProfile } = spyPort({
      getSaveCharacters: (id) => Promise.resolve(saveCharacters(id, characters)),
    });
    const { wrapper } = setup(port);

    const { result } = renderHook(() => useCharacterSelection("session-1", "0"), { wrapper });

    await waitFor(() => expect(result.current.selectedCharacterID).toBe(0));

    act(() => result.current.selectCharacter(2));

    // The default rule still holds and no query is made for the inactive slot.
    expect(result.current.selectedCharacterID).toBe(0);
    expect(getCharacterProfile).toHaveBeenCalledExactlyOnceWith("session-1", 0);
  });

  it("survives a refetch while the chosen slot stays active", async () => {
    let reported = characters;
    const { port, getCharacterProfile } = spyPort({
      getSaveCharacters: (id) => Promise.resolve(saveCharacters(id, reported)),
    });
    const { wrapper } = setup(port);

    const { result } = renderHook(() => useCharacterSelection("session-1", "0"), { wrapper });

    await waitFor(() => expect(result.current.selectedCharacterID).toBe(0));
    act(() => result.current.selectCharacter(1));
    await waitFor(() => expect(result.current.selectedCharacterID).toBe(1));

    // The same slot is still active after the refetch, with a changed level.
    reported = [
      summary(0, { active: true, name: "Zero", level: 151 }),
      summary(1, { active: true, name: "One", level: 91 }),
      summary(2),
    ];
    await act(async () => {
      await result.current.characters.refetch();
    });

    expect(result.current.selectedCharacterID).toBe(1);
    expect(getCharacterProfile.mock.calls.at(-1)).toEqual(["session-1", 1]);
  });

  it("returns to the default rule when the chosen slot stops being active", async () => {
    let reported = characters;
    const { port, getCharacterProfile } = spyPort({
      getSaveCharacters: (id) => Promise.resolve(saveCharacters(id, reported)),
    });
    const { wrapper } = setup(port);

    const { result } = renderHook(() => useCharacterSelection("session-1", "0"), { wrapper });

    await waitFor(() => expect(result.current.selectedCharacterID).toBe(0));
    act(() => result.current.selectCharacter(1));
    await waitFor(() => expect(result.current.selectedCharacterID).toBe(1));

    reported = [summary(0, { active: true, name: "Zero", level: 150 }), summary(1), summary(2)];
    await act(async () => {
      await result.current.characters.refetch();
    });

    // Slot 0 is active, so the first rule applies again.
    await waitFor(() => expect(result.current.selectedCharacterID).toBe(0));
    expect(getCharacterProfile.mock.calls.at(-1)).toEqual(["session-1", 0]);
  });

  it("does not resume a selection when an earlier session is entered again", async () => {
    const perSession: Record<string, readonly CharacterSummary[]> = {
      "session-1": characters,
      "session-2": [summary(0), summary(3, { active: true, name: "Other", level: 20 })],
    };
    const { port, getCharacterProfile, getCharacterStats } = spyPort({
      getSaveCharacters: (id) => Promise.resolve(saveCharacters(id, perSession[id] ?? [])),
    });
    const { wrapper } = setup(port);

    const { result, rerender } = renderHook(
      ({ id }: { id: string }) => useCharacterSelection(id, "0"),
      {
        wrapper,
        initialProps: { id: "session-1" },
      },
    );

    await waitFor(() => expect(result.current.selectedCharacterID).toBe(0));
    act(() => result.current.selectCharacter(1));
    await waitFor(() => expect(result.current.selectedCharacterID).toBe(1));

    rerender({ id: "session-2" });
    await waitFor(() => expect(result.current.selectedCharacterID).toBe(3));

    // Returning is a new entry into the session, not a resumed history: the
    // default rule applies again even though slot 1 is still active there.
    rerender({ id: "session-1" });
    await waitFor(() => expect(result.current.selectedCharacterID).toBe(0));
    expect(result.current.activeCharacters.map((c) => c.characterID)).toEqual([0, 1]);

    // The old slot is never asked for again after the return.
    expect(getCharacterProfile.mock.calls).toEqual([
      ["session-1", 0],
      ["session-1", 1],
      ["session-2", 3],
      ["session-1", 0],
    ]);
    expect(getCharacterStats.mock.calls.at(-1)).toEqual(["session-1", 0]);
  });

  it("does not resume a selection across a gap without a session", async () => {
    const { port, getCharacterProfile, getCharacterStats } = spyPort({
      getSaveCharacters: (id) => Promise.resolve(saveCharacters(id, characters)),
    });
    const { wrapper } = setup(port);

    const { result, rerender } = renderHook(
      ({ id }: { id?: string }) => useCharacterSelection(id, "0"),
      {
        wrapper,
        initialProps: { id: "session-1" } as { id?: string },
      },
    );

    await waitFor(() => expect(result.current.selectedCharacterID).toBe(0));
    act(() => result.current.selectCharacter(1));
    await waitFor(() => expect(result.current.selectedCharacterID).toBe(1));

    const callsBeforeTheGap = getCharacterProfile.mock.calls.length;
    rerender({});

    expect(result.current.hasSession).toBe(false);
    expect(result.current.selectedCharacterID).toBeUndefined();
    expect(getCharacterProfile).toHaveBeenCalledTimes(callsBeforeTheGap);
    expect(getCharacterStats).toHaveBeenCalledTimes(callsBeforeTheGap);

    rerender({ id: "session-1" });
    await waitFor(() => expect(result.current.selectedCharacterID).toBe(0));
    expect(getCharacterProfile.mock.calls.at(-1)).toEqual(["session-1", 0]);
    expect(getCharacterStats.mock.calls.at(-1)).toEqual(["session-1", 0]);
  });

  it("treats an empty identifier as a real gap between two entries", async () => {
    const { port, getCharacterProfile } = spyPort({
      getSaveCharacters: (id) => Promise.resolve(saveCharacters(id, characters)),
    });
    const { wrapper } = setup(port);

    const { result, rerender } = renderHook(
      ({ id }: { id?: string }) => useCharacterSelection(id, "0"),
      {
        wrapper,
        initialProps: { id: "session-1" } as { id?: string },
      },
    );

    await waitFor(() => expect(result.current.selectedCharacterID).toBe(0));
    act(() => result.current.selectCharacter(1));
    await waitFor(() => expect(result.current.selectedCharacterID).toBe(1));

    rerender({ id: "" });
    expect(result.current.hasSession).toBe(false);
    expect(result.current.selectedCharacterID).toBeUndefined();

    rerender({ id: "session-1" });
    await waitFor(() => expect(result.current.selectedCharacterID).toBe(0));
    expect(getCharacterProfile.mock.calls.at(-1)).toEqual(["session-1", 0]);
  });

  it("never carries a selection into another session", async () => {
    const perSession: Record<string, readonly CharacterSummary[]> = {
      "session-1": characters,
      "session-2": [summary(0), summary(3, { active: true, name: "Other", level: 20 })],
    };
    const { port, getCharacterProfile, getCharacterStats } = spyPort({
      getSaveCharacters: (id) => Promise.resolve(saveCharacters(id, perSession[id] ?? [])),
    });
    const { wrapper } = setup(port);

    const { result, rerender } = renderHook(
      ({ id }: { id: string }) => useCharacterSelection(id, "0"),
      {
        wrapper,
        initialProps: { id: "session-1" },
      },
    );

    await waitFor(() => expect(result.current.selectedCharacterID).toBe(0));
    act(() => result.current.selectCharacter(1));
    await waitFor(() => expect(result.current.selectedCharacterID).toBe(1));

    rerender({ id: "session-2" });

    // The previous slot is never effective in the new session, not even for one
    // render before the new list arrives.
    expect(result.current.selectedCharacterID).toBeUndefined();
    await waitFor(() => expect(result.current.selectedCharacterID).toBe(3));
    expect(getCharacterProfile.mock.calls).toEqual([
      ["session-1", 0],
      ["session-1", 1],
      ["session-2", 3],
    ]);
    expect(getCharacterStats.mock.calls.at(-1)).toEqual(["session-2", 3]);
  });
});
