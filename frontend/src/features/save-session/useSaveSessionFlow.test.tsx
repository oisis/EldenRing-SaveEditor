import { act, renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";
import { queryKeys } from "../../application/queryKeys";
import type {
  RecentFile,
  SessionChangedEvent,
} from "../../application/save-session/saveSessionPort";
import {
  createTestQueryClient,
  makeCharacterPort,
  makeDiagnosticsPort,
  makeSaveSessionPort,
  makeSettingsPort,
  stubCleanValidationReport,
  stubSaveCharacters,
  stubSaveSession,
  stubHostSettings,
  TestProviders,
} from "../../test/renderWithProviders";
import { useSaveSessionFlow } from "./useSaveSessionFlow";

type Ports = Parameters<typeof TestProviders>[0];

function setup(overrides: Partial<Ports> = {}) {
  const queryClient = overrides.queryClient ?? createTestQueryClient();
  const wrapper = ({ children }: { children: ReactNode }) => (
    <TestProviders
      queryClient={queryClient}
      saveSessionPort={overrides.saveSessionPort ?? makeSaveSessionPort()}
      characterPort={overrides.characterPort ?? makeCharacterPort()}
      diagnosticsPort={overrides.diagnosticsPort ?? makeDiagnosticsPort()}
      settingsPort={overrides.settingsPort ?? makeSettingsPort()}
    >
      {children}
    </TestProviders>
  );
  return { queryClient, wrapper };
}

describe("useSaveSessionFlow", () => {
  it("reaches the clean state and never asks the backend to remember a selection", async () => {
    const loadSave = vi.fn(() => Promise.resolve(stubSaveSession));
    const { wrapper } = setup({ saveSessionPort: makeSaveSessionPort({ loadSave }) });

    const { result } = renderHook(() => useSaveSessionFlow(), { wrapper });
    expect(result.current.state).toBe("empty");

    act(() => result.current.openSave());

    await waitFor(() => expect(result.current.state).toBe("clean"));
    expect(result.current.session).toEqual(stubSaveSession);
    // The natively chosen file is a durable local one, and the host path is
    // forwarded without any normalisation of its own.
    expect(loadSave).toHaveBeenCalledExactlyOnceWith(stubSaveSession.sourcePath, "", "local");
  });

  it("moves the active session and revision-keyed queries to the state confirmed after an event", async () => {
    let listener: ((event: SessionChangedEvent | null) => void) | undefined;
    let currentRevision = "0";
    const refreshed = {
      ...stubSaveSession,
      saveRevision: "1",
      unsavedChanges: true,
      eventSequence: "1",
    };
    const getSaveCharacters = vi.fn((saveSessionID: string) =>
      Promise.resolve({ ...stubSaveCharacters, saveSessionID, saveRevision: currentRevision }),
    );
    const { wrapper, queryClient } = setup({
      saveSessionPort: makeSaveSessionPort({
        getLoadedSave: () => Promise.resolve(currentRevision === "0" ? stubSaveSession : refreshed),
        subscribeSessionChanged: (next) => {
          listener = next;
          return () => {};
        },
      }),
      characterPort: makeCharacterPort({ getSaveCharacters }),
      diagnosticsPort: makeDiagnosticsPort({
        getSaveValidationReport: ({ saveSessionID, characterID }) =>
          Promise.resolve({
            ...stubCleanValidationReport,
            saveSessionID,
            characterID,
            saveRevision: currentRevision,
          }),
      }),
    });

    const { result } = renderHook(() => useSaveSessionFlow(), { wrapper });
    act(() => result.current.openSave());
    await waitFor(() => expect(result.current.state).toBe("clean"));
    await waitFor(() => expect(listener).toBeDefined());

    currentRevision = "1";
    act(() => {
      listener?.({
        sequence: "1",
        operationID: "op-1",
        operationKind: "set_character_name",
        saveSessionID: stubSaveSession.saveSessionID,
        saveRevision: "1",
        changedScopes: ["character.list", "diagnostics.report", "save.session"],
      });
    });

    await waitFor(() => expect(result.current.session).toEqual(refreshed));
    await waitFor(() =>
      expect(queryClient.getQueryState(queryKeys.saveCharacters("session-1", "1"))).toBeDefined(),
    );
    expect(result.current.session?.unsavedChanges).toBe(true);
    expect(result.current.session?.eventSequence).toBe("1");
  });

  it("passes the dialog path to the backend without normalising it", async () => {
    const awkward = "  /Volumes/A B/Elden Ring/ER0000.SL2  ";
    const loadSave = vi.fn(() => Promise.resolve(stubSaveSession));
    const { wrapper } = setup({
      saveSessionPort: makeSaveSessionPort({
        selectSaveFile: () => Promise.resolve(awkward),
        loadSave,
      }),
    });

    const { result } = renderHook(() => useSaveSessionFlow(), { wrapper });
    act(() => result.current.openSave());

    await waitFor(() => expect(loadSave).toHaveBeenCalledTimes(1));
    expect(loadSave).toHaveBeenCalledWith(awkward, "", "local");
  });

  it("treats a cancelled dialog as an ordinary outcome and never calls LoadSave", async () => {
    const loadSave = vi.fn(() => Promise.resolve(stubSaveSession));
    const { wrapper } = setup({
      saveSessionPort: makeSaveSessionPort({
        selectSaveFile: () => Promise.resolve(""),
        loadSave,
      }),
    });

    const { result } = renderHook(() => useSaveSessionFlow(), { wrapper });
    act(() => result.current.openSave());

    await waitFor(() => expect(result.current.state).toBe("cancelled"));
    expect(loadSave).not.toHaveBeenCalled();
    expect(result.current.session).toBe(undefined);
  });

  it("reports a failed load as a failure and never infers why it failed", async () => {
    const { wrapper } = setup({
      saveSessionPort: makeSaveSessionPort({
        loadSave: () => Promise.reject(new Error("bridge_call_failed")),
      }),
    });

    const { result } = renderHook(() => useSaveSessionFlow(), { wrapper });
    act(() => result.current.openSave());

    // A failed call carries no reason: the adapter reduces every failure to one
    // opaque code, so the flow may not classify the container behind it. Only
    // the one established refusal may ever reach `blocked`.
    await waitFor(() => expect(result.current.state).toBe("failed"));
    expect(result.current.failure).toBe("load_failed");
    expect(result.current.blockedReason).toBe(undefined);
    expect(result.current.session).toBe(undefined);
  });

  it("blocks a save with no active slot and closes the session it had to create", async () => {
    const closeSave = vi.fn(() => Promise.resolve());
    const queryClient = createTestQueryClient();
    const { wrapper } = setup({
      queryClient,
      saveSessionPort: makeSaveSessionPort({ closeSave }),
      characterPort: makeCharacterPort({
        getSaveCharacters: () =>
          Promise.resolve({
            saveSessionID: stubSaveSession.saveSessionID,
            saveRevision: stubSaveSession.saveRevision,
            characters: [
              { characterID: 0, active: false, name: "", level: 0 },
              { characterID: 1, active: false, name: "", level: 0 },
            ],
            slots: [],
          }),
      }),
    });

    const { result } = renderHook(() => useSaveSessionFlow(), { wrapper });
    act(() => result.current.openSave());

    await waitFor(() => expect(result.current.state).toBe("blocked"));
    expect(result.current.blockedReason).toBe("no_active_character");
    // No editable session is left behind, in the flow or in the cache.
    expect(result.current.session).toBe(undefined);
    expect(closeSave).toHaveBeenCalledExactlyOnceWith(stubSaveSession.saveSessionID);
    await waitFor(() =>
      expect(
        queryClient.getQueryData(
          queryKeys.saveCharacters(stubSaveSession.saveSessionID, stubSaveSession.saveRevision),
        ),
      ).toBe(undefined),
    );
  });

  it("reports warnings from the backend counters alone, without judging the save", async () => {
    for (const [label, report] of [
      ["warningCount", { ...stubCleanValidationReport, warningCount: 1 }],
      ["errorCount", { ...stubCleanValidationReport, errorCount: 1 }],
      [
        "unchecked scope",
        {
          ...stubCleanValidationReport,
          coverage: [
            ...stubCleanValidationReport.coverage.slice(1),
            {
              scope: "spells",
              checked: false,
              reason: "the scope could not be decoded",
              recordsChecked: 0,
              unresolvedRecords: 0,
            },
          ],
        },
      ],
      [
        "unresolved record",
        {
          ...stubCleanValidationReport,
          coverage: [
            ...stubCleanValidationReport.coverage.slice(1),
            {
              scope: "inventory",
              checked: true,
              reason: "",
              recordsChecked: 5,
              unresolvedRecords: 2,
            },
          ],
        },
      ],
    ] as const) {
      const { wrapper } = setup({
        diagnosticsPort: makeDiagnosticsPort({
          getSaveValidationReport: () => Promise.resolve(report),
        }),
      });

      const { result } = renderHook(() => useSaveSessionFlow(), { wrapper });
      act(() => result.current.openSave());

      await waitFor(() => expect(result.current.state).toBe("warnings"), { timeout: 3000 });
      // The session stays editable: warnings are not a block.
      expect(result.current.session, label).toEqual(stubSaveSession);
    }
  });

  it("selects slot 0 when it is active and the first active slot otherwise", async () => {
    const { wrapper } = setup();
    const { result } = renderHook(() => useSaveSessionFlow(), { wrapper });
    act(() => result.current.openSave());

    await waitFor(() => expect(result.current.state).toBe("clean"));
    expect(result.current.selection.selectedCharacterID).toBe(0);

    const laterSlots = setup({
      characterPort: makeCharacterPort({
        getSaveCharacters: () =>
          Promise.resolve({
            saveSessionID: stubSaveSession.saveSessionID,
            saveRevision: stubSaveSession.saveRevision,
            characters: [
              { characterID: 0, active: false, name: "", level: 0 },
              { characterID: 3, active: true, name: "Second", level: 60 },
              { characterID: 5, active: true, name: "Third", level: 90 },
            ],
            slots: [],
          }),
      }),
    });
    const second = renderHook(() => useSaveSessionFlow(), { wrapper: laterSlots.wrapper });
    act(() => second.result.current.openSave());

    await waitFor(() => expect(second.result.current.state).toBe("clean"));
    // The backend's order decides, not the lowest identifier.
    expect(second.result.current.selection.selectedCharacterID).toBe(3);
  });

  it("drops the cache and the character selection when the session is closed", async () => {
    const queryClient = createTestQueryClient();
    const { wrapper } = setup({ queryClient });

    const { result } = renderHook(() => useSaveSessionFlow(), { wrapper });
    act(() => result.current.openSave());
    await waitFor(() => expect(result.current.state).toBe("clean"));
    expect(
      queryClient.getQueryData(
        queryKeys.saveCharacters(stubSaveSession.saveSessionID, stubSaveSession.saveRevision),
      ),
    ).toEqual(stubSaveCharacters);

    act(() => result.current.closeSave());

    await waitFor(() => expect(result.current.state).toBe("empty"));
    // Nothing keyed under the closed session survives. The idle placeholder
    // queries of "no session" are keyed under the empty identifier and are not
    // session data, so they are deliberately not counted here.
    await waitFor(() =>
      expect(
        queryClient
          .getQueryCache()
          .getAll()
          .filter((query) => query.queryKey[1] === stubSaveSession.saveSessionID),
      ).toEqual([]),
    );
    expect(result.current.selection.selectedCharacterID).toBe(undefined);
  });

  it("closes and forgets the replaced session when another save is opened", async () => {
    const queryClient = createTestQueryClient();
    const closeSave = vi.fn(() => Promise.resolve());
    const sessions = [
      stubSaveSession,
      { ...stubSaveSession, saveSessionID: "session-2", sourcePath: "/saves/Second.sl2" },
    ];
    let call = 0;
    const { wrapper } = setup({
      queryClient,
      saveSessionPort: makeSaveSessionPort({
        closeSave,
        loadSave: () => Promise.resolve(sessions[Math.min(call++, 1)]),
      }),
    });

    const { result } = renderHook(() => useSaveSessionFlow(), { wrapper });
    act(() => result.current.openSave());
    await waitFor(() => expect(result.current.session?.saveSessionID).toBe("session-1"));

    act(() => result.current.openSave());
    await waitFor(() => expect(result.current.session?.saveSessionID).toBe("session-2"));

    // The replaced session is closed in the backend, not merely dropped from
    // the cache: the application holds one session, so abandoning the old one
    // while it stays open would leak a session nothing can reach any more.
    await waitFor(() => expect(closeSave).toHaveBeenCalledExactlyOnceWith("session-1"));
    // Nothing of the replaced session survives into the new one.
    await waitFor(() =>
      expect(
        queryClient
          .getQueryCache()
          .getAll()
          .filter((query) => query.queryKey[1] === "session-1"),
      ).toEqual([]),
    );
  });

  it("keeps the open session when the replacement load is refused", async () => {
    const closeSave = vi.fn(() => Promise.resolve());
    let call = 0;
    const { wrapper } = setup({
      saveSessionPort: makeSaveSessionPort({
        closeSave,
        loadSave: () =>
          call++ === 0
            ? Promise.resolve(stubSaveSession)
            : Promise.reject(new Error("bridge_call_failed")),
      }),
    });

    const { result } = renderHook(() => useSaveSessionFlow(), { wrapper });
    act(() => result.current.openSave());
    await waitFor(() => expect(result.current.state).toBe("clean"));

    act(() => result.current.openSave());

    // The replacement never became a session, so destroying the working one
    // would lose an editable save for a failure that says nothing about it.
    await waitFor(() => expect(result.current.failure).toBe("load_failed"));
    expect(result.current.session).toEqual(stubSaveSession);
    expect(result.current.state).toBe("clean");
    expect(closeSave).not.toHaveBeenCalled();
  });

  it("keeps the open session when the file dialog itself fails", async () => {
    const closeSave = vi.fn(() => Promise.resolve());
    let call = 0;
    const { wrapper } = setup({
      saveSessionPort: makeSaveSessionPort({
        closeSave,
        selectSaveFile: () =>
          call++ === 0
            ? Promise.resolve(stubSaveSession.sourcePath)
            : Promise.reject(new Error("bridge_call_failed")),
      }),
    });

    const { result } = renderHook(() => useSaveSessionFlow(), { wrapper });
    act(() => result.current.openSave());
    await waitFor(() => expect(result.current.state).toBe("clean"));

    act(() => result.current.openSave());

    await waitFor(() => expect(result.current.failure).toBe("dialog_failed"));
    // The session stays editable and reachable: nothing was chosen, nothing was
    // loaded and nothing was closed.
    expect(result.current.session).toEqual(stubSaveSession);
    expect(result.current.state).toBe("clean");
    expect(closeSave).not.toHaveBeenCalled();
  });

  it("keeps the open session and closes the candidate that has no active slot", async () => {
    const closeSave = vi.fn(() => Promise.resolve());
    const candidate = { ...stubSaveSession, saveSessionID: "session-2" };
    let load = 0;
    const { wrapper } = setup({
      saveSessionPort: makeSaveSessionPort({
        closeSave,
        loadSave: () => Promise.resolve(load++ === 0 ? stubSaveSession : candidate),
      }),
      characterPort: makeCharacterPort({
        getSaveCharacters: (saveSessionID: string) =>
          Promise.resolve({
            saveSessionID,
            saveRevision:
              saveSessionID === candidate.saveSessionID
                ? candidate.saveRevision
                : stubSaveSession.saveRevision,
            characters:
              saveSessionID === candidate.saveSessionID
                ? [{ characterID: 0, active: false, name: "", level: 0 }]
                : stubSaveCharacters.characters,
            slots: [],
          }),
      }),
    });

    const { result } = renderHook(() => useSaveSessionFlow(), { wrapper });
    act(() => result.current.openSave());
    await waitFor(() => expect(result.current.state).toBe("clean"));

    act(() => result.current.openSave());

    await waitFor(() => expect(result.current.failure).toBe("no_active_character"));
    // Only the rejected candidate is closed. The previous session is untouched,
    // and it is not reported as blocked: it is a working session.
    await waitFor(() => expect(closeSave).toHaveBeenCalledExactlyOnceWith("session-2"));
    expect(result.current.session).toEqual(stubSaveSession);
    expect(result.current.blockedReason).toBe(undefined);
    expect(result.current.unclosedSessionID).toBe(undefined);
  });

  it("keeps the previous session identifier reachable when its close fails", async () => {
    const sessions = [stubSaveSession, { ...stubSaveSession, saveSessionID: "session-2" }];
    let load = 0;
    const { wrapper } = setup({
      saveSessionPort: makeSaveSessionPort({
        loadSave: () => Promise.resolve(sessions[Math.min(load++, 1)]),
        closeSave: () => Promise.reject(new Error("bridge_call_failed")),
      }),
    });

    const { result } = renderHook(() => useSaveSessionFlow(), { wrapper });
    act(() => result.current.openSave());
    await waitFor(() => expect(result.current.session?.saveSessionID).toBe("session-1"));

    act(() => result.current.openSave());
    await waitFor(() => expect(result.current.session?.saveSessionID).toBe("session-2"));

    // The swap is not reported as complete: the previous session is still open
    // in the backend and stays nameable, so the user can ask again.
    await waitFor(() => expect(result.current.unclosedSessionID).toBe("session-1"));
  });

  it("keeps the session open and retryable when CloseSave fails", async () => {
    const queryClient = createTestQueryClient();
    let closes = 0;
    const closeSave = vi.fn(() =>
      closes++ === 0 ? Promise.reject(new Error("bridge_call_failed")) : Promise.resolve(),
    );
    const { wrapper } = setup({ queryClient, saveSessionPort: makeSaveSessionPort({ closeSave }) });

    const { result } = renderHook(() => useSaveSessionFlow(), { wrapper });
    act(() => result.current.openSave());
    await waitFor(() => expect(result.current.state).toBe("clean"));

    act(() => result.current.closeSave());

    await waitFor(() => expect(result.current.unclosedSessionID).toBe("session-1"));
    // Nothing may claim the session was closed: it is still shown, and its
    // cached views are still there, because the backend never confirmed.
    expect(result.current.session).toEqual(stubSaveSession);
    expect(
      queryClient.getQueryData(
        queryKeys.saveCharacters(stubSaveSession.saveSessionID, stubSaveSession.saveRevision),
      ),
    ).toEqual(stubSaveCharacters);

    act(() => result.current.closeSave());

    await waitFor(() => expect(result.current.state).toBe("empty"));
    expect(result.current.session).toBe(undefined);
    expect(result.current.unclosedSessionID).toBe(undefined);
    expect(closeSave).toHaveBeenCalledTimes(2);
    await waitFor(() =>
      expect(
        queryClient.getQueryData(
          queryKeys.saveCharacters(stubSaveSession.saveSessionID, stubSaveSession.saveRevision),
        ),
      ).toBe(undefined),
    );
  });

  it("requires an explicit dirty-session decision before close or replace", async () => {
    const dirty = { ...stubSaveSession, unsavedChanges: true };
    const closeSave = vi.fn(() => Promise.resolve());
    const loadSave = vi.fn(() => Promise.resolve(dirty));
    const { wrapper } = setup({
      saveSessionPort: makeSaveSessionPort({ closeSave, loadSave }),
    });

    const { result } = renderHook(() => useSaveSessionFlow(), { wrapper });
    act(() => result.current.openSave());
    await waitFor(() => expect(result.current.state).toBe("clean"));
    expect(loadSave).toHaveBeenCalledTimes(1);

    act(() => result.current.closeSave());
    await waitFor(() => expect(result.current.pendingSessionAction).toEqual({ kind: "close" }));
    expect(closeSave).not.toHaveBeenCalled();

    act(() => result.current.cancelPendingAction());
    expect(result.current.pendingSessionAction).toBe(undefined);

    act(() => result.current.openSave());
    await waitFor(() =>
      expect(result.current.pendingSessionAction).toEqual({ kind: "open-dialog" }),
    );
    // Nothing is discarded and nothing is replaced: no second load was even
    // attempted, and the dirty session is exactly as it was.
    expect(loadSave).toHaveBeenCalledTimes(1);
    expect(closeSave).not.toHaveBeenCalled();
    expect(result.current.session).toEqual(dirty);
  });

  it("does not claim a clean save when a report could not be obtained", async () => {
    const { wrapper } = setup({
      diagnosticsPort: makeDiagnosticsPort({
        getSaveValidationReport: () => Promise.reject(new Error("bridge_call_failed")),
      }),
    });

    const { result } = renderHook(() => useSaveSessionFlow(), { wrapper });
    act(() => result.current.openSave());

    await waitFor(() => expect(result.current.state).toBe("failed"));
  });

  it("keeps the open session when a report of the candidate cannot be obtained", async () => {
    const closeSave = vi.fn(() => Promise.resolve());
    const candidate = { ...stubSaveSession, saveSessionID: "session-2" };
    let load = 0;
    const { wrapper } = setup({
      saveSessionPort: makeSaveSessionPort({
        closeSave,
        loadSave: () => Promise.resolve(load++ === 0 ? stubSaveSession : candidate),
      }),
      diagnosticsPort: makeDiagnosticsPort({
        getSaveValidationReport: ({ saveSessionID, characterID }) =>
          saveSessionID === candidate.saveSessionID
            ? Promise.reject(new Error("bridge_call_failed"))
            : Promise.resolve({ ...stubCleanValidationReport, saveSessionID, characterID }),
      }),
    });

    const { result } = renderHook(() => useSaveSessionFlow(), { wrapper });
    act(() => result.current.openSave());
    await waitFor(() => expect(result.current.state).toBe("clean"));

    act(() => result.current.openSave());

    // Validation is part of opening, not of showing what was opened: a
    // candidate nothing could check never becomes the session, and losing an
    // editable save over it would be the worst possible reading of a failure
    // that says nothing about either file.
    await waitFor(() => expect(result.current.failure).toBe("validation_failed"));
    await waitFor(() => expect(closeSave).toHaveBeenCalledExactlyOnceWith("session-2"));
    expect(result.current.session).toEqual(stubSaveSession);
    expect(result.current.state).toBe("clean");
    expect(result.current.blockedReason).toBe(undefined);
    expect(result.current.unclosedSessionID).toBe(undefined);
  });

  it("keeps the open session when the candidate is validated by a stale revision", async () => {
    const closeSave = vi.fn(() => Promise.resolve());
    const candidate = { ...stubSaveSession, saveSessionID: "session-2" };
    let load = 0;
    const { wrapper } = setup({
      saveSessionPort: makeSaveSessionPort({
        closeSave,
        loadSave: () => Promise.resolve(load++ === 0 ? stubSaveSession : candidate),
      }),
      diagnosticsPort: makeDiagnosticsPort({
        // A complete, perfectly clean report — about another revision of the
        // save. It answers a question nobody asked, so it is no verdict at all.
        getSaveValidationReport: ({ saveSessionID, characterID }) =>
          Promise.resolve({
            ...stubCleanValidationReport,
            saveSessionID,
            characterID,
            saveRevision: saveSessionID === candidate.saveSessionID ? "7" : "0",
          }),
      }),
    });

    const { result } = renderHook(() => useSaveSessionFlow(), { wrapper });
    act(() => result.current.openSave());
    await waitFor(() => expect(result.current.state).toBe("clean"));

    act(() => result.current.openSave());

    await waitFor(() => expect(result.current.failure).toBe("validation_failed"));
    await waitFor(() => expect(closeSave).toHaveBeenCalledExactlyOnceWith("session-2"));
    expect(result.current.session).toEqual(stubSaveSession);
    expect(result.current.state).toBe("clean");
  });

  it("leaves no editing session behind when the first candidate cannot be validated", async () => {
    const closeSave = vi.fn(() => Promise.resolve());
    const { wrapper } = setup({
      saveSessionPort: makeSaveSessionPort({ closeSave }),
      diagnosticsPort: makeDiagnosticsPort({
        getSaveValidationReport: () => Promise.reject(new Error("bridge_call_failed")),
      }),
    });

    const { result } = renderHook(() => useSaveSessionFlow(), { wrapper });
    act(() => result.current.openSave());

    // With no previous session to protect, an unvalidated candidate is still
    // not an editing session: it is closed and nothing is left open.
    await waitFor(() => expect(result.current.failure).toBe("validation_failed"));
    expect(result.current.session).toBe(undefined);
    expect(result.current.state).toBe("failed");
    await waitFor(() =>
      expect(closeSave).toHaveBeenCalledExactlyOnceWith(stubSaveSession.saveSessionID),
    );
  });

  it("closes the replaced session only after the candidate is fully validated", async () => {
    const closeSave = vi.fn(() => Promise.resolve());
    const candidate = { ...stubSaveSession, saveSessionID: "session-2" };
    let load = 0;
    let releaseReport = () => {};
    const validated = new Promise<void>((resolve) => {
      releaseReport = resolve;
    });
    const getSaveValidationReport = vi.fn(
      async ({ saveSessionID, characterID }: { saveSessionID: string; characterID: number }) => {
        if (saveSessionID === candidate.saveSessionID) {
          await validated;
        }
        // A warning is not a block: the candidate is accepted for it, and only
        // then is the session it replaces closed.
        return {
          ...stubCleanValidationReport,
          saveSessionID,
          characterID,
          warningCount: saveSessionID === candidate.saveSessionID ? 1 : 0,
        };
      },
    );
    const { wrapper } = setup({
      saveSessionPort: makeSaveSessionPort({
        closeSave,
        loadSave: () => Promise.resolve(load++ === 0 ? stubSaveSession : candidate),
      }),
      diagnosticsPort: makeDiagnosticsPort({ getSaveValidationReport }),
    });

    const { result } = renderHook(() => useSaveSessionFlow(), { wrapper });
    act(() => result.current.openSave());
    await waitFor(() => expect(result.current.state).toBe("clean"));

    act(() => result.current.openSave());
    await waitFor(() =>
      expect(
        getSaveValidationReport.mock.calls.some(
          ([request]) => request.saveSessionID === candidate.saveSessionID,
        ),
      ).toBe(true),
    );

    // The candidate is loaded and its report is still outstanding. Nothing may
    // have been swapped or closed yet: the previous session is still the one
    // the user is editing.
    expect(result.current.session).toEqual(stubSaveSession);
    expect(closeSave).not.toHaveBeenCalled();

    await act(async () => {
      releaseReport();
      await validated;
    });

    await waitFor(() => expect(result.current.session).toEqual(candidate));
    expect(result.current.state).toBe("warnings");
    await waitFor(() => expect(closeSave).toHaveBeenCalledExactlyOnceWith("session-1"));
  });

  it("refuses to open another save while a close is still unconfirmed", async () => {
    const loadSave = vi.fn(() => Promise.resolve(stubSaveSession));
    const selectSaveFile = vi.fn(() => Promise.resolve(stubSaveSession.sourcePath));
    const { wrapper } = setup({
      saveSessionPort: makeSaveSessionPort({
        loadSave,
        selectSaveFile,
        closeSave: () => Promise.reject(new Error("bridge_call_failed")),
      }),
    });

    const { result } = renderHook(() => useSaveSessionFlow(), { wrapper });
    act(() => result.current.openSave());
    await waitFor(() => expect(result.current.state).toBe("clean"));

    act(() => result.current.closeSave());
    await waitFor(() => expect(result.current.unclosedSessionID).toBe("session-1"));

    act(() => result.current.openSave());

    // Fail-closed: a session that is still open in the backend is unresolved
    // state, and starting a second one beside it could strand the first.
    await waitFor(() => expect(result.current.unclosedSessionID).toBe("session-1"));
    expect(selectSaveFile).toHaveBeenCalledTimes(1);
    expect(loadSave).toHaveBeenCalledTimes(1);
    expect(result.current.session).toEqual(stubSaveSession);
  });

  it("refuses to close the current session while another one is unconfirmed", async () => {
    const closeSave = vi.fn(() => Promise.reject(new Error("bridge_call_failed")));
    const candidate = { ...stubSaveSession, saveSessionID: "session-2" };
    let load = 0;
    const { wrapper } = setup({
      saveSessionPort: makeSaveSessionPort({
        closeSave,
        loadSave: () => Promise.resolve(load++ === 0 ? stubSaveSession : candidate),
      }),
    });

    const { result } = renderHook(() => useSaveSessionFlow(), { wrapper });
    act(() => result.current.openSave());
    await waitFor(() => expect(result.current.state).toBe("clean"));

    act(() => result.current.openSave());
    await waitFor(() => expect(result.current.session).toEqual(candidate));
    await waitFor(() => expect(result.current.unclosedSessionID).toBe("session-1"));

    act(() => result.current.closeSave());
    act(() => result.current.retryClose());
    await waitFor(() => expect(closeSave).toHaveBeenCalledTimes(2));

    // Only the unconfirmed session is ever asked about again. Closing the
    // current one would produce a second failure, and the identifier of the
    // first would have nothing left able to name it.
    expect(closeSave.mock.calls).toEqual([["session-1"], ["session-1"]]);
    expect(result.current.unclosedSessionID).toBe("session-1");
    expect(result.current.session).toEqual(candidate);
  });

  it("restores the ordinary operations after a confirmed retry", async () => {
    let closes = 0;
    const loadSave = vi.fn(() => Promise.resolve(stubSaveSession));
    const closeSave = vi.fn(() =>
      closes++ === 0 ? Promise.reject(new Error("bridge_call_failed")) : Promise.resolve(),
    );
    const { wrapper } = setup({ saveSessionPort: makeSaveSessionPort({ closeSave, loadSave }) });

    const { result } = renderHook(() => useSaveSessionFlow(), { wrapper });
    act(() => result.current.openSave());
    await waitFor(() => expect(result.current.state).toBe("clean"));

    act(() => result.current.closeSave());
    await waitFor(() => expect(result.current.unclosedSessionID).toBe("session-1"));

    act(() => result.current.retryClose());
    await waitFor(() => expect(result.current.unclosedSessionID).toBe(undefined));
    expect(result.current.session).toBe(undefined);

    act(() => result.current.openSave());

    // The block belonged to the unresolved cleanup, not to the session: once
    // the backend confirms, opening works again.
    await waitFor(() => expect(result.current.state).toBe("clean"));
    expect(loadSave).toHaveBeenCalledTimes(2);
  });

  it("asks the backend for every scope of every active slot", async () => {
    const getSaveValidationReport = vi.fn(({ characterID }: { characterID: number }) =>
      Promise.resolve({ ...stubCleanValidationReport, characterID }),
    );
    const { wrapper } = setup({
      diagnosticsPort: makeDiagnosticsPort({ getSaveValidationReport }),
      characterPort: makeCharacterPort({
        getSaveCharacters: () =>
          Promise.resolve({
            saveSessionID: stubSaveSession.saveSessionID,
            saveRevision: stubSaveSession.saveRevision,
            characters: [
              { characterID: 0, active: true, name: "First", level: 10 },
              { characterID: 2, active: true, name: "Second", level: 20 },
              { characterID: 3, active: false, name: "", level: 0 },
            ],
            slots: [],
          }),
      }),
    });

    const { result } = renderHook(() => useSaveSessionFlow(), { wrapper });
    act(() => result.current.openSave());

    await waitFor(() => expect(result.current.state).toBe("clean"));
    // Only the active slots, and always the backend's full scope: a narrowed
    // report could hide exactly the problem this gate exists to surface.
    expect(getSaveValidationReport.mock.calls.map(([request]) => request)).toEqual([
      { saveSessionID: "session-1", characterID: 0, scope: "" },
      { saveSessionID: "session-1", characterID: 2, scope: "" },
    ]);
  });

  it("validates the exact revision before Save and adopts the confirmed clean session", async () => {
    const dirty = { ...stubSaveSession, saveRevision: "1", unsavedChanges: true };
    const clean = { ...dirty, saveRevision: "2", unsavedChanges: false, eventSequence: "1" };
    let current = dirty;
    const validateReviewChanges = vi.fn(() =>
      Promise.resolve({
        saveSessionID: dirty.saveSessionID,
        saveRevision: "1",
        validationToken: "validation-1",
        valid: true,
        warningCount: 0,
        banRiskCount: 0,
        criticalCount: 0,
        stages: [{ stage: "validation", percent: 100 }],
        issues: [],
      }),
    );
    const save = vi.fn(() => {
      current = clean;
      return Promise.resolve({
        operationID: "operation-save",
        operationKind: "save",
        saveSessionID: dirty.saveSessionID,
        saveRevision: "2",
        changedScopes: ["save.session", "diagnostics.report"] as const,
        target: dirty.sourcePath,
        backupPath: `${dirty.sourcePath}.backup`,
        warnings: [],
        retentionNoticeRequired: false,
      });
    });
    let recentFiles: RecentFile[] = [];
    const recordRecentFile = vi.fn(() => {
      recentFiles = [
        {
          path: clean.sourcePath,
          platform: clean.platform,
          format: clean.format,
          lastOpenedAt: "2026-09-02T20:00:00Z",
        },
      ];
      return Promise.resolve(recentFiles);
    });
    const { wrapper } = setup({
      saveSessionPort: makeSaveSessionPort({
        loadSave: () => Promise.resolve(dirty),
        getLoadedSave: () => Promise.resolve(current),
        validateReviewChanges,
        save,
        recordRecentFile,
        getRecentFiles: () => Promise.resolve(recentFiles),
      }),
      characterPort: makeCharacterPort({
        getSaveCharacters: (saveSessionID) =>
          Promise.resolve({ ...stubSaveCharacters, saveSessionID, saveRevision: "1" }),
      }),
      diagnosticsPort: makeDiagnosticsPort({
        getSaveValidationReport: ({ saveSessionID, characterID }) =>
          Promise.resolve({
            ...stubCleanValidationReport,
            saveSessionID,
            saveRevision: "1",
            characterID,
          }),
      }),
    });
    const { result } = renderHook(() => useSaveSessionFlow(), { wrapper });
    act(() => result.current.openSave());
    await waitFor(() => expect(result.current.session).toEqual(dirty));

    act(() => result.current.openReview());
    await waitFor(() =>
      expect(result.current.reviewValidation?.validationToken).toBe("validation-1"),
    );
    act(() => result.current.saveReviewed(false, false, false));

    await waitFor(() => expect(result.current.session).toEqual(clean));
    expect(validateReviewChanges).toHaveBeenCalledWith(dirty.saveSessionID, "1");
    expect(save).toHaveBeenCalledWith(dirty.saveSessionID, "1", "validation-1", false, false);
    expect(recordRecentFile).toHaveBeenCalledWith(dirty.saveSessionID);
    expect(result.current.recentFiles[0]?.path).toBe(clean.sourcePath);
    expect(result.current.history?.operations).toEqual([]);
  });

  it("skips only the normal-risk Save review after validation when the host setting allows it", async () => {
    const dirty = { ...stubSaveSession, saveRevision: "1", unsavedChanges: true };
    const clean = { ...dirty, saveRevision: "2", unsavedChanges: false, eventSequence: "1" };
    let current = dirty;
    const validateReviewChanges = vi.fn(() =>
      Promise.resolve({
        saveSessionID: dirty.saveSessionID,
        saveRevision: dirty.saveRevision,
        validationToken: "validation-1",
        valid: true,
        warningCount: 0,
        banRiskCount: 0,
        criticalCount: 0,
        stages: [],
        issues: [],
      }),
    );
    const save = vi.fn(() => {
      current = clean;
      return Promise.resolve({
        operationID: "operation-save",
        operationKind: "save",
        saveSessionID: dirty.saveSessionID,
        saveRevision: clean.saveRevision,
        changedScopes: ["save.session", "diagnostics.report"] as const,
        target: dirty.sourcePath,
        warnings: [],
        retentionNoticeRequired: false,
      });
    });
    const { wrapper } = setup({
      settingsPort: makeSettingsPort({
        getHostSettings: () =>
          Promise.resolve({ ...stubHostSettings, skipReviewForNormalRisk: true }),
      }),
      saveSessionPort: makeSaveSessionPort({
        loadSave: () => Promise.resolve(dirty),
        getLoadedSave: () => Promise.resolve(current),
        validateReviewChanges,
        save,
      }),
      characterPort: makeCharacterPort({
        getSaveCharacters: (saveSessionID) =>
          Promise.resolve({ ...stubSaveCharacters, saveSessionID, saveRevision: "1" }),
      }),
      diagnosticsPort: makeDiagnosticsPort({
        getSaveValidationReport: ({ saveSessionID, characterID }) =>
          Promise.resolve({
            ...stubCleanValidationReport,
            saveSessionID,
            saveRevision: "1",
            characterID,
          }),
      }),
    });

    const { result } = renderHook(() => useSaveSessionFlow(), { wrapper });
    act(() => result.current.openSave());
    await waitFor(() => expect(result.current.session).toEqual(dirty));
    act(() => result.current.save());

    await waitFor(() => expect(save).toHaveBeenCalledTimes(1));
    expect(validateReviewChanges).toHaveBeenCalledWith(dirty.saveSessionID, dirty.saveRevision);
    expect(result.current.reviewOpen).toBe(false);
    await waitFor(() => expect(result.current.session).toEqual(clean));
  });

  it("releases a deployment staging file after attempting to open it", async () => {
    const stagedPath = "/private/tmp/saveforge-download-1/downloaded.sl2";
    const releaseDeploymentStaging = vi.fn(() => Promise.resolve());
    const recordRecentFile = vi.fn(() => Promise.resolve([]));
    const loadSave = vi.fn(() =>
      Promise.resolve({
        ...stubSaveSession,
        sourcePath: stagedPath,
        sourceKind: "temporary" as const,
      }),
    );
    const { wrapper } = setup({
      saveSessionPort: makeSaveSessionPort({
        loadSave,
        recordRecentFile,
        releaseDeploymentStaging,
      }),
    });
    const { result } = renderHook(() => useSaveSessionFlow(), { wrapper });

    act(() => result.current.openStagedFile(stagedPath));

    await waitFor(() => expect(releaseDeploymentStaging).toHaveBeenCalledWith(stagedPath));
    expect(loadSave).toHaveBeenCalledWith(stagedPath, "", "temporary");
    expect(recordRecentFile).not.toHaveBeenCalled();
  });

  it("discards through the backend before continuing a dirty close request", async () => {
    const dirty = { ...stubSaveSession, saveRevision: "1", unsavedChanges: true };
    const clean = { ...dirty, saveRevision: "2", unsavedChanges: false, eventSequence: "1" };
    let current = dirty;
    const closeSave = vi.fn(() => Promise.resolve());
    const discardChanges = vi.fn(() => {
      current = clean;
      return Promise.resolve({
        operationID: "operation-discard",
        operationKind: "discard_changes",
        saveSessionID: dirty.saveSessionID,
        saveRevision: "2",
        changedScopes: ["save.session", "diagnostics.report"] as const,
        discardedOperations: 1,
      });
    });
    const { wrapper } = setup({
      saveSessionPort: makeSaveSessionPort({
        loadSave: () => Promise.resolve(dirty),
        getLoadedSave: () => Promise.resolve(current),
        closeSave,
        discardChanges,
      }),
      characterPort: makeCharacterPort({
        getSaveCharacters: (saveSessionID) =>
          Promise.resolve({ ...stubSaveCharacters, saveSessionID, saveRevision: "1" }),
      }),
      diagnosticsPort: makeDiagnosticsPort({
        getSaveValidationReport: ({ saveSessionID, characterID }) =>
          Promise.resolve({
            ...stubCleanValidationReport,
            saveSessionID,
            saveRevision: "1",
            characterID,
          }),
      }),
    });
    const { result } = renderHook(() => useSaveSessionFlow(), { wrapper });
    act(() => result.current.openSave());
    await waitFor(() => expect(result.current.session).toEqual(dirty));
    act(() => result.current.closeSave());
    await waitFor(() => expect(result.current.pendingSessionAction).toEqual({ kind: "close" }));
    act(() => result.current.discardPendingChanges());

    await waitFor(() => expect(result.current.session).toBe(undefined));
    expect(discardChanges).toHaveBeenCalledWith(dirty.saveSessionID, "1");
    expect(closeSave).toHaveBeenCalledWith(dirty.saveSessionID);
  });
});
