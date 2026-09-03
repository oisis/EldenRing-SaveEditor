import { useQueryClient } from "@tanstack/react-query";
import { useCallback, useEffect, useState } from "react";
import { matchesQueryKeyPattern, queryKeyPatternsForScopes } from "../../application/changedScopes";
import { useCharacterPort } from "../../application/character/characterClient";
import { saveCharactersQuery } from "../../application/character/useSaveCharacters";
import { useDiagnosticsPort } from "../../application/diagnostics/diagnosticsClient";
import {
  aggregateValidationReports,
  saveValidationReportQuery,
  useSaveValidationReports,
} from "../../application/diagnostics/useSaveValidationReports";
import { type AppError, toAppError } from "../../application/errors/appError";
import { queryKeys } from "../../application/queryKeys";
import { useSaveSessionPort } from "../../application/save-session/saveSessionClient";
import type {
  MutationReceipt,
  OperationHistory,
  RecentFile,
  RecoveryJournal,
  ReviewValidationResult,
  SaveLifecycleResult,
  SaveLifecycleSettings,
  SaveSession,
} from "../../application/save-session/saveSessionPort";
import { useCloseSave } from "../../application/save-session/useCloseSave";
import { useLoadSave } from "../../application/save-session/useLoadSave";
import {
  type SessionChangedSync,
  useSessionChangedSync,
} from "../../application/save-session/useSessionChangedSync";
import { type CharacterSelection, useCharacterSelection } from "../character/useCharacterSelection";

export type SaveSessionFlowState =
  | "empty"
  | "opening"
  | "cancelled"
  | "clean"
  | "warnings"
  | "blocked"
  | "failed";

export type SaveSessionBlockedReason = "no_active_character";
export type SaveSessionFailure =
  | "dialog_failed"
  | "load_failed"
  | "validation_failed"
  | "no_active_character";

export type PendingSessionAction =
  | { kind: "open-dialog" }
  | { kind: "open-recent"; path: string }
  | { kind: "restore-recovery"; journalID: string }
  | { kind: "close" }
  | { kind: "quit" };

export type SaveSessionFlow = {
  state: SaveSessionFlowState;
  blockedReason: SaveSessionBlockedReason | undefined;
  failure: SaveSessionFailure | undefined;
  appError: AppError | undefined;
  lifecycleError: AppError | undefined;
  lifecycleMessage: string | undefined;
  lastSaveResult: SaveLifecycleResult | undefined;
  unclosedSessionID: string | undefined;
  session: SaveSession | undefined;
  validation: ReturnType<typeof useSaveValidationReports>;
  selection: CharacterSelection;
  history: OperationHistory | undefined;
  reviewOpen: boolean;
  reviewValidation: ReviewValidationResult | undefined;
  pendingSessionAction: PendingSessionAction | undefined;
  recentFiles: readonly RecentFile[];
  recoveryJournals: readonly RecoveryJournal[];
  lifecycleSettings: SaveLifecycleSettings | undefined;
  /**
   * The single post-mutation step of the whole application: it notes the
   * mutation against the `session.changed` stream, maps the receipt's
   * `changedScopes` onto query keys, refreshes the session and re-reads the
   * operation history. Feature modules that commit a save mutation hand their
   * receipt to this function and invalidate nothing themselves.
   */
  applyMutationReceipt: (receipt: MutationReceipt) => Promise<SaveSession>;
  openSave: () => void;
  openRecent: (path: string) => void;
  closeSave: () => void;
  retryClose: () => void;
  openReview: () => void;
  closeReview: () => void;
  saveReviewed: (saveAs: boolean, confirmWarnings: boolean, confirmBanRisk: boolean) => void;
  undo: () => void;
  redo: () => void;
  revertOperation: (operationID: string) => void;
  discardPendingChanges: () => void;
  savePendingChanges: () => void;
  cancelPendingAction: () => void;
  removeRecent: (path: string) => void;
  clearRecent: () => void;
  restoreRecovery: (journalID: string) => void;
  discardRecovery: (journalID: string) => void;
  exportRecovery: (journalID: string) => void;
  setBackupRetention: (retention: number) => void;
  isBusy: boolean;
  sessionSync: SessionChangedSync;
};

const staleValidationReport = Symbol("stale validation report");

/**
 * Owns the complete desktop session lifecycle. Save state, operation history,
 * Review Changes, recent files and recovery all stay behind the same backend
 * port; components only request actions and render authoritative results.
 */
export function useSaveSessionFlow(): SaveSessionFlow {
  const port = useSaveSessionPort();
  const characterPort = useCharacterPort();
  const diagnosticsPort = useDiagnosticsPort();
  const queryClient = useQueryClient();
  const load = useLoadSave();
  const close = useCloseSave();

  const [session, setSession] = useState<SaveSession>();
  const [cancelled, setCancelled] = useState(false);
  const [failure, setFailure] = useState<SaveSessionFailure>();
  const [appError, setAppError] = useState<AppError>();
  const [lifecycleError, setLifecycleError] = useState<AppError>();
  const [lifecycleMessage, setLifecycleMessage] = useState<string>();
  const [lastSaveResult, setLastSaveResult] = useState<SaveLifecycleResult>();
  const [unclosedSessionID, setUnclosedSessionID] = useState<string>();
  const [opening, setOpening] = useState(false);
  const [lifecycleBusy, setLifecycleBusy] = useState(false);
  const [history, setHistory] = useState<OperationHistory>();
  const [reviewOpen, setReviewOpen] = useState(false);
  const [reviewValidation, setReviewValidation] = useState<ReviewValidationResult>();
  const [pendingSessionAction, setPendingSessionAction] = useState<PendingSessionAction>();
  const [recentFiles, setRecentFiles] = useState<readonly RecentFile[]>([]);
  const [recoveryJournals, setRecoveryJournals] = useState<readonly RecoveryJournal[]>([]);
  const [lifecycleSettings, setLifecycleSettings] = useState<SaveLifecycleSettings>();

  const acceptSessionRefresh = useCallback((refreshed: SaveSession) => {
    setSession((current) =>
      current?.saveSessionID === refreshed.saveSessionID ? refreshed : current,
    );
  }, []);
  const sessionSync = useSessionChangedSync(session?.saveSessionID, acceptSessionRefresh);
  const selection = useCharacterSelection(session?.saveSessionID, session?.saveRevision);
  const validation = useSaveValidationReports(
    session?.saveSessionID,
    session?.saveRevision,
    selection.activeCharacters,
  );

  const loadSave = load.mutateAsync;
  const closeSave = close.mutateAsync;

  const refreshHostState = useCallback(async () => {
    const [recent, recovery, settings] = await Promise.allSettled([
      port.getRecentFiles(),
      port.getRecoveryJournals(),
      port.getSaveLifecycleSettings(),
    ]);
    if (recent.status === "fulfilled") setRecentFiles(recent.value);
    if (recovery.status === "fulfilled") setRecoveryJournals(recovery.value);
    if (settings.status === "fulfilled") setLifecycleSettings(settings.value);
    const failure = [recent, recovery, settings].find(
      (result): result is PromiseRejectedResult => result.status === "rejected",
    );
    if (failure !== undefined) setLifecycleError(toAppError(failure.reason));
  }, [port]);

  useEffect(() => {
    void refreshHostState().catch((reason) => setLifecycleError(toAppError(reason)));
  }, [refreshHostState]);

  const historySessionID = session?.saveSessionID;
  const historySaveRevision = session?.saveRevision;
  useEffect(() => {
    if (historySessionID === undefined || historySaveRevision === undefined) {
      setHistory(undefined);
      return;
    }
    let active = true;
    void port
      .getOperationHistory(historySessionID)
      .then((result) => {
        if (active && result.saveSessionID === historySessionID) setHistory(result);
      })
      .catch((reason) => {
        if (active) setLifecycleError(toAppError(reason));
      });
    return () => {
      active = false;
    };
  }, [historySaveRevision, historySessionID, port]);

  const retire = useCallback(
    async (saveSessionID: string): Promise<boolean> => {
      try {
        await closeSave(saveSessionID);
      } catch (reason) {
        setUnclosedSessionID((pending) => pending ?? saveSessionID);
        setAppError(toAppError(reason));
        return false;
      }
      setUnclosedSessionID((pending) => (pending === saveSessionID ? undefined : pending));
      setSession((open) => (open?.saveSessionID === saveSessionID ? undefined : open));
      return true;
    },
    [closeSave],
  );

  const validateCandidate = useCallback(
    async (candidate: SaveSession): Promise<void> => {
      const characters = await queryClient.fetchQuery(
        saveCharactersQuery(characterPort, candidate.saveSessionID, candidate.saveRevision),
      );
      const activeCharacters = characters.characters.filter((character) => character.active);
      if (activeCharacters.length === 0) throw new Error("no_active_character");
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
      if (aggregate.stale) throw staleValidationReport;
    },
    [characterPort, diagnosticsPort, queryClient],
  );

  const acceptCandidate = useCallback(
    async (candidate: SaveSession, replaced: string | undefined) => {
      try {
        await validateCandidate(candidate);
      } catch (reason) {
        await retire(candidate.saveSessionID);
        if (reason instanceof Error && reason.message === "no_active_character") {
          setFailure("no_active_character");
        } else {
          setFailure("validation_failed");
          setAppError(reason === staleValidationReport ? undefined : toAppError(reason));
        }
        return;
      }
      setSession(candidate);
      try {
        setRecentFiles(await port.recordRecentFile(candidate.saveSessionID));
      } catch (reason) {
        setLifecycleError(toAppError(reason));
      }
      if (replaced !== undefined && replaced !== candidate.saveSessionID) await retire(replaced);
    },
    [port, retire, validateCandidate],
  );

  const openPath = useCallback(
    async (source: string) => {
      const replaced = session?.saveSessionID;
      let candidate: SaveSession;
      try {
        candidate = await loadSave({ source, expectedPlatform: "", sourceKind: "local" });
      } catch (reason) {
        setFailure("load_failed");
        setAppError(toAppError(reason));
        return;
      }
      await acceptCandidate(candidate, replaced);
    },
    [acceptCandidate, loadSave, session?.saveSessionID],
  );

  const executeAction = useCallback(
    async (action: PendingSessionAction) => {
      setCancelled(false);
      setFailure(undefined);
      setAppError(undefined);
      setLifecycleError(undefined);
      setOpening(true);
      try {
        if (action.kind === "close") {
          if (session !== undefined) await retire(session.saveSessionID);
          return;
        }
        if (action.kind === "quit") {
          await port.quitApplication();
          return;
        }
        if (action.kind === "restore-recovery") {
          const replaced = session?.saveSessionID;
          let candidate: SaveSession;
          try {
            candidate = await port.restoreRecoveryJournal(action.journalID);
          } catch (reason) {
            setLifecycleError(toAppError(reason));
            return;
          }
          await acceptCandidate(candidate, replaced);
          await refreshHostState();
          return;
        }
        if (action.kind === "open-recent") {
          await openPath(action.path);
          return;
        }
        let source: string;
        try {
          source = await port.selectSaveFile();
        } catch (reason) {
          setFailure("dialog_failed");
          setAppError(toAppError(reason));
          return;
        }
        if (source === "") {
          setCancelled(true);
          return;
        }
        await openPath(source);
      } finally {
        setOpening(false);
      }
    },
    [acceptCandidate, openPath, port, refreshHostState, retire, session],
  );

  const cleanupPending = unclosedSessionID !== undefined;
  const isBusy = opening || close.isPending || lifecycleBusy;

  const requestAction = useCallback(
    (action: PendingSessionAction) => {
      if (isBusy) return;
      if (cleanupPending) {
        if (
          action.kind === "close" &&
          session !== undefined &&
          unclosedSessionID === session.saveSessionID
        ) {
          void executeAction(action);
        }
        return;
      }
      if (session?.unsavedChanges) {
        setPendingSessionAction(action);
        return;
      }
      void executeAction(action);
    },
    [cleanupPending, executeAction, isBusy, session, unclosedSessionID],
  );

  useEffect(
    () => port.subscribeApplicationCloseRequested(() => requestAction({ kind: "quit" })),
    [port, requestAction],
  );

  const refreshAfterMutation = useCallback(
    async (receipt: MutationReceipt): Promise<SaveSession> => {
      sessionSync.noteAppliedMutation(receipt);
      const patterns = queryKeyPatternsForScopes(receipt.saveSessionID, receipt.changedScopes);
      if (patterns.length > 0) {
        await queryClient.invalidateQueries({
          predicate: (query) =>
            patterns.some((pattern) => matchesQueryKeyPattern(query.queryKey, pattern)),
        });
      }
      const refreshed = await port.getLoadedSave(receipt.saveSessionID);
      queryClient.setQueryData(queryKeys.loadedSave(receipt.saveSessionID), refreshed);
      setSession(refreshed);
      setHistory(await port.getOperationHistory(receipt.saveSessionID));
      setReviewValidation(undefined);
      return refreshed;
    },
    [port, queryClient, sessionSync],
  );

  const runHistoryMutation = useCallback(
    async (mutation: (saveSessionID: string, revision: string) => Promise<MutationReceipt>) => {
      if (session === undefined || lifecycleBusy) return;
      setLifecycleBusy(true);
      setLifecycleError(undefined);
      setLifecycleMessage(undefined);
      setLastSaveResult(undefined);
      try {
        const refreshed = await refreshAfterMutation(
          await mutation(session.saveSessionID, session.saveRevision),
        );
        if (reviewOpen) {
          setReviewValidation(
            await port.validateReviewChanges(refreshed.saveSessionID, refreshed.saveRevision),
          );
        }
      } catch (reason) {
        setLifecycleError(toAppError(reason));
      } finally {
        setLifecycleBusy(false);
      }
    },
    [lifecycleBusy, port, refreshAfterMutation, reviewOpen, session],
  );

  const openReview = useCallback(() => {
    if (session === undefined || lifecycleBusy) return;
    setReviewOpen(true);
    setReviewValidation(undefined);
    setLifecycleError(undefined);
    setLifecycleBusy(true);
    void Promise.all([
      port.getOperationHistory(session.saveSessionID),
      port.validateReviewChanges(session.saveSessionID, session.saveRevision),
    ])
      .then(([nextHistory, result]) => {
        setHistory(nextHistory);
        setReviewValidation(result);
      })
      .catch((reason) => setLifecycleError(toAppError(reason)))
      .finally(() => setLifecycleBusy(false));
  }, [lifecycleBusy, port, session]);

  const saveReviewed = useCallback(
    (saveAs: boolean, confirmWarnings: boolean, confirmBanRisk: boolean) => {
      if (
        session === undefined ||
        lifecycleBusy ||
        reviewValidation?.valid !== true ||
        reviewValidation.validationToken === undefined ||
        reviewValidation.saveSessionID !== session.saveSessionID ||
        reviewValidation.saveRevision !== session.saveRevision
      )
        return;
      const validationToken = reviewValidation.validationToken;
      setLifecycleBusy(true);
      setLifecycleError(undefined);
      setLifecycleMessage(undefined);
      setLastSaveResult(undefined);
      void (async () => {
        try {
          let result: SaveLifecycleResult;
          if (saveAs) {
            const target = await port.selectSaveTarget(fileNameFromPath(session.sourcePath));
            if (target === "") return;
            result = await port.saveAs(
              session.saveSessionID,
              session.saveRevision,
              validationToken,
              confirmWarnings,
              confirmBanRisk,
              target,
            );
          } else {
            result = await port.save(
              session.saveSessionID,
              session.saveRevision,
              validationToken,
              confirmWarnings,
              confirmBanRisk,
            );
          }
          await refreshAfterMutation(result);
          try {
            setRecentFiles(await port.recordRecentFile(result.saveSessionID));
          } catch (reason) {
            setLifecycleError(toAppError(reason));
          }
          setReviewOpen(false);
          setLastSaveResult(result);
          setLifecycleMessage(result.warnings.length > 0 ? result.warnings.join(" ") : undefined);
          const nextAction = pendingSessionAction;
          setPendingSessionAction(undefined);
          await refreshHostState();
          if (nextAction !== undefined) await executeAction(nextAction);
        } catch (reason) {
          setLifecycleError(toAppError(reason));
        } finally {
          setLifecycleBusy(false);
        }
      })();
    },
    [
      executeAction,
      lifecycleBusy,
      pendingSessionAction,
      port,
      refreshAfterMutation,
      refreshHostState,
      reviewValidation,
      session,
    ],
  );

  const discardPendingChanges = useCallback(() => {
    if (session === undefined || pendingSessionAction === undefined || lifecycleBusy) return;
    const action = pendingSessionAction;
    setLifecycleBusy(true);
    setLifecycleError(undefined);
    void (async () => {
      try {
        const receipt = await port.discardChanges(session.saveSessionID, session.saveRevision);
        await refreshAfterMutation(receipt);
        setPendingSessionAction(undefined);
        await executeAction(action);
      } catch (reason) {
        setLifecycleError(toAppError(reason));
      } finally {
        setLifecycleBusy(false);
      }
    })();
  }, [executeAction, lifecycleBusy, pendingSessionAction, port, refreshAfterMutation, session]);

  const retryClose = useCallback(() => {
    if (unclosedSessionID === undefined || isBusy) return;
    setAppError(undefined);
    void retire(unclosedSessionID);
  }, [isBusy, retire, unclosedSessionID]);

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
    applyMutationReceipt: refreshAfterMutation,
    blockedReason,
    failure,
    appError,
    lifecycleError,
    lifecycleMessage,
    lastSaveResult,
    unclosedSessionID,
    session,
    validation,
    selection,
    history,
    reviewOpen,
    reviewValidation,
    pendingSessionAction,
    recentFiles,
    recoveryJournals,
    lifecycleSettings,
    openSave: () => requestAction({ kind: "open-dialog" }),
    openRecent: (path) => requestAction({ kind: "open-recent", path }),
    closeSave: () => requestAction({ kind: "close" }),
    retryClose,
    openReview,
    closeReview: () => setReviewOpen(false),
    saveReviewed,
    undo: () => void runHistoryMutation(port.undoLastOperation),
    redo: () => void runHistoryMutation(port.redoLastOperation),
    revertOperation: (operationID) =>
      void runHistoryMutation((id, revision) => port.revertOperation(id, operationID, revision)),
    discardPendingChanges,
    savePendingChanges: openReview,
    cancelPendingAction: () => setPendingSessionAction(undefined),
    removeRecent: (path) =>
      void port
        .removeRecentFile(path)
        .then(setRecentFiles)
        .catch((reason) => setLifecycleError(toAppError(reason))),
    clearRecent: () =>
      void port
        .clearRecentFiles()
        .then(() => setRecentFiles([]))
        .catch((reason) => setLifecycleError(toAppError(reason))),
    restoreRecovery: (journalID) => requestAction({ kind: "restore-recovery", journalID }),
    discardRecovery: (journalID) =>
      void port
        .discardRecoveryJournal(journalID)
        .then(refreshHostState)
        .catch((reason) => setLifecycleError(toAppError(reason))),
    exportRecovery: (journalID) => {
      void (async () => {
        try {
          const target = await port.selectSaveTarget(`${journalID}.saveforge-recovery.json`);
          if (target !== "") await port.exportRecoveryJournal(journalID, target);
        } catch (reason) {
          setLifecycleError(toAppError(reason));
        }
      })();
    },
    setBackupRetention: (retention) =>
      void port
        .setSaveLifecycleSettings(retention)
        .then(setLifecycleSettings)
        .catch((reason) => setLifecycleError(toAppError(reason))),
    isBusy,
    sessionSync,
  };
}

function fileNameFromPath(path: string): string {
  const parts = path.split(/[\\/]/);
  return parts.at(-1) || "ER0000.sl2";
}
