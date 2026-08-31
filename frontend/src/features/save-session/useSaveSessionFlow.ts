import { useQueryClient } from "@tanstack/react-query";
import { useCallback, useState } from "react";
import { useCharacterPort } from "../../application/character/characterClient";
import { saveCharactersQuery } from "../../application/character/useSaveCharacters";
import { useDiagnosticsPort } from "../../application/diagnostics/diagnosticsClient";
import {
  aggregateValidationReports,
  saveValidationReportQuery,
  useSaveValidationReports,
} from "../../application/diagnostics/useSaveValidationReports";
import { useSaveSessionPort } from "../../application/save-session/saveSessionClient";
import type { SaveSession } from "../../application/save-session/saveSessionPort";
import { useCloseSave } from "../../application/save-session/useCloseSave";
import { useLoadSave } from "../../application/save-session/useLoadSave";
import { type CharacterSelection, useCharacterSelection } from "../character/useCharacterSelection";

/**
 * What the session flow currently shows.
 *
 *   - `empty`     — no session; the screen offers Open Save;
 *   - `opening`   — the dialog, the load or the validation is still running;
 *   - `cancelled` — the user closed the dialog; nothing was loaded;
 *   - `clean`     — an editable session whose active slots validated clean;
 *   - `warnings`  — an editable session with a persistent banner and a report;
 *   - `blocked`   — the one established refusal: a save with no active slot,
 *                   and no session left behind for it;
 *   - `failed`    — a call failed, so nothing can be claimed about the save.
 */
export type SaveSessionFlowState =
  | "empty"
  | "opening"
  | "cancelled"
  | "clean"
  | "warnings"
  | "blocked"
  | "failed";

/**
 * Why the flow refused to open an editing session.
 *
 * There is exactly one value, and it is the only refusal this build can
 * establish: the backend reported a character list with no active slot. Any
 * finer reason — a damaged container, an unrecognised format, an unsupported
 * version — would have to be inferred from a failed call, and a failed call
 * carries no reason at all: the desktop adapter reduces every transport and
 * backend failure to one opaque code, and the message text is never parsed.
 * Distinguishing those cases needs the structured backend error contract
 * (`tmp/sf-2.0/frontend-backend.md`, section 13), which does not exist yet.
 */
export type SaveSessionBlockedReason = "no_active_character";

/**
 * What went wrong in the last open or close attempt. It is an outcome of the
 * operation, not a claim about the save file:
 *
 *   - `dialog_failed`       — the host's file dialog itself failed;
 *   - `load_failed`         — LoadSave, or the character list of the candidate
 *                             it created, failed. Nothing is claimed about why;
 *   - `validation_failed`   — a validation report of the candidate could not be
 *                             obtained, or the one that arrived answered about
 *                             another save state. Nothing is claimed about the
 *                             contents of the save;
 *   - `no_active_character` — the backend reported no active slot;
 *   - `unsaved_changes`     — the session holds unsaved changes and Save and
 *                             Discard do not exist yet, so it may not be closed
 *                             or replaced.
 */
export type SaveSessionFailure =
  | "dialog_failed"
  | "load_failed"
  | "validation_failed"
  | "no_active_character"
  | "unsaved_changes";

export type SaveSessionFlow = {
  state: SaveSessionFlowState;
  blockedReason: SaveSessionBlockedReason | undefined;
  /** The outcome of the last open or close attempt, or undefined. */
  failure: SaveSessionFailure | undefined;
  /**
   * A backend session whose CloseSave failed. It stays reachable so the user can
   * retry: a session dropped from the interface while still open in the backend
   * would be unreachable for the rest of the run. While it is set, opening
   * another save is refused and so is closing any *other* session, so a second
   * failure can never take its place and strand it.
   */
  unclosedSessionID: string | undefined;
  /** The editable session, or undefined when none is open. */
  session: SaveSession | undefined;
  validation: ReturnType<typeof useSaveValidationReports>;
  selection: CharacterSelection;
  /** Opens the dialog and, unless it was cancelled, loads what it returned. */
  openSave: () => void;
  closeSave: () => void;
  /** Asks the backend again to close the session whose CloseSave failed. */
  retryClose: () => void;
  isBusy: boolean;
};

/**
 * The controller of the one save session the application holds.
 *
 * It is a flow controller and nothing more. It performs no validation of its
 * own: it asks the backend to load, asks the backend which slots are active,
 * asks the backend for one report per active slot, and turns those answers into
 * a screen state. It reads no save data, judges no record and repairs nothing.
 *
 * Opening is transactional, and so is replacing. A candidate is loaded, its
 * character list is read, and every active slot is validated in full *before*
 * it becomes the session and before the session it replaces is closed. A failed
 * dialog, a failed load, a save with no active slot, an unobtainable report and
 * a report about another save state therefore all leave the previous session
 * exactly as it was — and, with no previous session, leave no editing session
 * behind at all. The rejected candidate is closed either way. Every step that
 * depends on the previous one is awaited rather than chained through mutation
 * callbacks, because the order is the contract here.
 *
 * A session is only ever forgotten after the backend confirms CloseSave. The
 * cache is dropped by `useCloseSave` on success alone, and an unconfirmed close
 * keeps the identifier reachable instead of leaking a session nothing can name.
 *
 * The character selection stays presentational and stays where it already
 * lives: the slot-0-or-first-active rule belongs to `useCharacterSelection` and
 * is not restated here, and no selection is ever sent to the backend.
 */
export function useSaveSessionFlow(): SaveSessionFlow {
  const port = useSaveSessionPort();
  const characterPort = useCharacterPort();
  const diagnosticsPort = useDiagnosticsPort();
  const queryClient = useQueryClient();
  const load = useLoadSave();
  const close = useCloseSave();

  const [session, setSession] = useState<SaveSession | undefined>(undefined);
  const [cancelled, setCancelled] = useState(false);
  const [failure, setFailure] = useState<SaveSessionFailure | undefined>(undefined);
  const [unclosedSessionID, setUnclosedSessionID] = useState<string | undefined>(undefined);
  const [opening, setOpening] = useState(false);

  const selection = useCharacterSelection(session?.saveSessionID);
  const validation = useSaveValidationReports(
    session?.saveSessionID,
    session?.saveRevision,
    selection.activeCharacters,
  );

  // Stable across renders, so every callback below may depend on them honestly.
  const loadSave = load.mutateAsync;
  const closeSave = close.mutateAsync;

  /**
   * Asks the backend to close one session and reports whether it confirmed.
   *
   * Only a confirmed close retires anything: the cached views go with it
   * through `useCloseSave`, and the flow forgets the session only when the
   * closed one is the session it is showing. A refusal records the identifier
   * instead, so the user sees the failure and can ask again.
   */
  const retire = useCallback(
    async (saveSessionID: string): Promise<boolean> => {
      try {
        await closeSave(saveSessionID);
      } catch {
        // The first unconfirmed close keeps the slot. A later failure may not
        // take its place: overwriting it would leave the earlier session open
        // in the backend with nothing left able to name it.
        setUnclosedSessionID((pending) => pending ?? saveSessionID);
        return false;
      }
      setUnclosedSessionID((pending) => (pending === saveSessionID ? undefined : pending));
      setSession((open) => (open?.saveSessionID === saveSessionID ? undefined : open));
      return true;
    },
    [closeSave],
  );

  const isBusy = opening || close.isPending;
  // Fail-closed: an unconfirmed close is unresolved state about a session that
  // is still open in the backend, so no further session operation may run
  // beside it. Only the retry of that exact session stays available.
  const cleanupPending = unclosedSessionID !== undefined;

  const openSave = useCallback(() => {
    if (isBusy || cleanupPending) {
      return;
    }
    // Until Save and Discard exist there is no safe way to part with unsaved
    // work, so the flow refuses instead of choosing one for the user. It never
    // discards anything by itself.
    if (session?.unsavedChanges === true) {
      setFailure("unsaved_changes");
      return;
    }

    const replaced = session?.saveSessionID;
    setCancelled(false);
    setFailure(undefined);
    setOpening(true);

    void (async () => {
      try {
        let source: string;
        try {
          source = await port.selectSaveFile();
        } catch {
          // The dialog failed, so nothing was chosen and nothing was called.
          // The open session is untouched and stays fully usable.
          setFailure("dialog_failed");
          return;
        }

        // Cancelling is an ordinary outcome: nothing is loaded for it, so the
        // backend is never called and the open session stays exactly as it was.
        if (source === "") {
          setCancelled(true);
          return;
        }

        let candidate: SaveSession;
        try {
          candidate = await loadSave(
            // The path goes to the backend exactly as the host reported it, and
            // a natively chosen file is a durable local one.
            { source, expectedPlatform: "", sourceKind: "local" },
          );
        } catch {
          // The call failed and says nothing more than that. No session was
          // created, and the previous one is neither closed nor hidden.
          setFailure("load_failed");
          return;
        }

        // The character list decides which slots exist and which of them are
        // active. A save with no active slot is critical in this build and must
        // not leave an editing session open, and finding that out must not cost
        // the session the user already had.
        let activeCharacters: readonly { characterID: number }[] = [];
        try {
          const characters = await queryClient.fetchQuery(
            saveCharactersQuery(characterPort, candidate.saveSessionID),
          );
          activeCharacters = characters.characters.filter((character) => character.active);
        } catch {
          await retire(candidate.saveSessionID);
          setFailure("load_failed");
          return;
        }

        if (activeCharacters.length === 0) {
          await retire(candidate.saveSessionID);
          setFailure("no_active_character");
          return;
        }

        // The full validation is part of opening, not of showing what was
        // opened: the candidate only becomes the session once every active slot
        // has a report that belongs to this exact session, revision and slot,
        // and the previous session is still the working one throughout. The
        // same query, the same identity rule and the same aggregation the
        // display uses are applied here; the flow restates none of them.
        try {
          const reports = await Promise.all(
            activeCharacters.map((character) =>
              queryClient.fetchQuery(
                saveValidationReportQuery(
                  diagnosticsPort,
                  candidate.saveSessionID,
                  candidate.saveRevision,
                  character.characterID,
                ),
              ),
            ),
          );
          const aggregate = aggregateValidationReports(
            reports,
            candidate.saveSessionID,
            candidate.saveRevision,
            activeCharacters.map((character) => character.characterID),
          );
          // A report about another save state is no verdict about this one, so
          // the candidate is refused exactly as it is for a report that never
          // arrived. What is left is `clean` or `warnings`, and both open.
          if (aggregate.stale) {
            throw new Error("stale_validation_report");
          }
        } catch {
          await retire(candidate.saveSessionID);
          setFailure("validation_failed");
          return;
        }

        // Only now, with the candidate fully loaded and fully validated, does it
        // become the session and the replaced one get closed. A refused close is
        // recorded by `retire` and stays visible: the swap is not reported as
        // complete just because the new session opened.
        setSession(candidate);
        if (replaced !== undefined && replaced !== candidate.saveSessionID) {
          await retire(replaced);
        }
      } finally {
        setOpening(false);
      }
    })();
  }, [
    isBusy,
    cleanupPending,
    session,
    port,
    loadSave,
    queryClient,
    characterPort,
    diagnosticsPort,
    retire,
  ]);

  const closeSaveSession = useCallback(() => {
    if (session === undefined || isBusy) {
      return;
    }
    // An unconfirmed close of *another* session blocks this one: resolving that
    // cleanup comes first. Closing the session that is itself unconfirmed is the
    // retry, so it stays available.
    if (unclosedSessionID !== undefined && unclosedSessionID !== session.saveSessionID) {
      return;
    }
    if (session.unsavedChanges) {
      setFailure("unsaved_changes");
      return;
    }
    setFailure(undefined);
    void retire(session.saveSessionID);
  }, [session, isBusy, unclosedSessionID, retire]);

  const retryClose = useCallback(() => {
    if (unclosedSessionID === undefined || isBusy) {
      return;
    }
    void retire(unclosedSessionID);
  }, [unclosedSessionID, isBusy, retire]);

  const blockedReason: SaveSessionBlockedReason | undefined =
    session === undefined && failure === "no_active_character" ? "no_active_character" : undefined;

  const state: SaveSessionFlowState =
    session === undefined
      ? blockedReason !== undefined
        ? "blocked"
        : failure !== undefined
          ? "failed"
          : cancelled
            ? "cancelled"
            : isBusy
              ? "opening"
              : "empty"
      : selection.characters.isError
        ? "failed"
        : selection.characters.isPending || validation.state === "pending"
          ? "opening"
          : validation.state === "failed"
            ? "failed"
            : validation.state === "warnings"
              ? "warnings"
              : "clean";

  return {
    state,
    blockedReason,
    failure,
    unclosedSessionID,
    session,
    validation,
    selection,
    openSave,
    closeSave: closeSaveSession,
    retryClose,
    isBusy,
  };
}
