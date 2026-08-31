import { useQueries } from "@tanstack/react-query";
import type { CharacterSummary } from "../character/characterPort";
import { queryKeys } from "../queryKeys";
import { useDiagnosticsPort } from "./diagnosticsClient";
import type { DiagnosticsPort, SaveValidationReport } from "./diagnosticsPort";

/**
 * The aggregate state of a session's validation. It is a flow state, not a
 * domain verdict:
 *
 *   - `clean`    — every active slot was reported complete and problem-free;
 *   - `warnings` — at least one report carries a warning, an error, an
 *                  unresolved record or a scope the backend could not check;
 *   - `failed`   — at least one report could not be obtained, or one that
 *                  arrived does not belong to the session, the revision and the
 *                  slot it was asked for.
 *
 * There is no `blocked` here: a save the backend refuses to load never reaches
 * this stage, and "no active slot" is decided before any report is requested.
 */
export type SaveValidationState = "pending" | "clean" | "warnings" | "failed";

/**
 * The backend's own counters, added up. Nothing here is recomputed from the
 * issue list and no severity is reclassified.
 */
export type SaveValidationTotals = {
  errorCount: number;
  warningCount: number;
  /** Backend counters too: scopes reported unchecked, and unresolved records. */
  uncheckedScopes: number;
  unresolvedRecords: number;
};

export type SaveValidationSummary = SaveValidationTotals & {
  state: SaveValidationState;
  /** The matching reports that arrived, in the backend's slot order. */
  reports: readonly SaveValidationReport[];
  /** The session and revision the reports had to belong to. */
  saveSessionID: string | undefined;
  saveRevision: string | undefined;
};

/**
 * The one description of a validation-report query: its key, its call, its
 * retry rule and how long the answer stays usable. The hook below and the
 * session flow that has to validate a candidate imperatively both build on it,
 * so one save state is never read through two differently configured queries.
 *
 * The empty scope is the backend's "every scope": a narrowed report could hide
 * exactly the problem this gate exists to surface.
 *
 * A report answers about one exact save state — one session, one revision, one
 * slot — and every part of that state is in the key, so the answer can never
 * become stale under it: another revision is a different key and therefore a
 * different question. `gcTime` is stated for the same reason. The flow fetches
 * a candidate's reports before the session exists, and the hook that displays
 * them mounts only afterwards; an entry evicted in between would send the
 * backend the identical question a second time. The entry still lives below the
 * session prefix, so a confirmed `CloseSave` removes it with every other view
 * of that session regardless of this window.
 */
export function saveValidationReportQuery(
  port: DiagnosticsPort,
  saveSessionID: string,
  saveRevision: string,
  characterID: number,
) {
  return {
    queryKey: queryKeys.saveValidationReport(saveSessionID, characterID, saveRevision),
    queryFn: () => port.getSaveValidationReport({ saveSessionID, characterID, scope: "" }),
    retry: false,
    staleTime: Number.POSITIVE_INFINITY,
    gcTime: 5 * 60 * 1000,
  };
}

/**
 * Whether one report is an answer about the state the caller is actually
 * showing. Every part of the identity is compared as the exact string or number
 * the backend reported: the revision is never parsed, ordered or compared
 * numerically, so "newer" and "older" are not decided here — only "the same" and
 * "not the same".
 *
 * A report that fails any part of this is not a weaker result. It describes a
 * different save state, so it can never be counted, summed or shown as this
 * session's verdict.
 */
export function reportAnswersFor(
  report: SaveValidationReport,
  saveSessionID: string,
  saveRevision: string,
  characterID: number,
): boolean {
  return (
    report.saveSessionID === saveSessionID &&
    report.saveRevision === saveRevision &&
    report.characterID === characterID &&
    report.active
  );
}

export type SaveValidationAggregate = SaveValidationTotals & {
  /** The reports that answer about the asked state, in the asked order. */
  matching: readonly SaveValidationReport[];
  /** Whether any report that arrived answers about another save state. */
  stale: boolean;
  /** The verdict of the matching reports alone. */
  verdict: "clean" | "warnings";
};

/**
 * Folds one report per asked slot into the identity check, the backend's own
 * counters and the resulting verdict. It is the single place both the session
 * flow and the hook below decide what a set of reports means.
 *
 * The folding is arithmetic only. Every input is a number or a boolean the
 * backend already decided: this function opens no save, reads no record, judges
 * no item and knows no game rule. `undefined` stands for a report that has not
 * arrived; it is not stale, it is simply not there yet, and it contributes
 * nothing to the counters.
 */
export function aggregateValidationReports(
  reports: readonly (SaveValidationReport | undefined)[],
  saveSessionID: string,
  saveRevision: string,
  characterIDs: readonly number[],
): SaveValidationAggregate {
  const answers = reports.map(
    (report, index) =>
      report !== undefined &&
      reportAnswersFor(report, saveSessionID, saveRevision, characterIDs[index]),
  );
  const matching = reports.flatMap((report, index) =>
    report !== undefined && answers[index] ? [report] : [],
  );
  const stale = reports.some((report, index) => report !== undefined && !answers[index]);

  const totals = matching.reduce<SaveValidationTotals>(
    (sums, report) => ({
      errorCount: sums.errorCount + report.errorCount,
      warningCount: sums.warningCount + report.warningCount,
      uncheckedScopes:
        sums.uncheckedScopes + report.coverage.filter((scope) => !scope.checked).length,
      unresolvedRecords:
        sums.unresolvedRecords +
        report.coverage.reduce((count, scope) => count + scope.unresolvedRecords, 0),
    }),
    { errorCount: 0, warningCount: 0, uncheckedScopes: 0, unresolvedRecords: 0 },
  );

  const verdict =
    totals.errorCount > 0 ||
    totals.warningCount > 0 ||
    totals.uncheckedScopes > 0 ||
    totals.unresolvedRecords > 0
      ? "warnings"
      : "clean";

  return { matching, stale, verdict, ...totals };
}

/**
 * Reads one full-scope validation report per active slot and folds them into a
 * single flow state.
 *
 * The session and the revision the reports must belong to are supplied by the
 * caller rather than taken from the reports themselves. A report that names
 * another session, another revision, another slot or an inactive slot is stale
 * and makes the whole summary `failed`; it can never produce `clean` or
 * `warnings`. This is the staleness rule for this one flow: the general
 * revision-driven invalidation of every getter is a separate contract and is
 * deliberately not implemented here.
 *
 * The queries themselves, the identity rule and the aggregation are shared with
 * the session flow, which validates a candidate before it becomes the session;
 * neither side restates them.
 */
export function useSaveValidationReports(
  saveSessionID: string | undefined,
  saveRevision: string | undefined,
  activeCharacters: readonly CharacterSummary[],
): SaveValidationSummary {
  const port = useDiagnosticsPort();
  // Without a session or a revision there is nothing to validate against, and
  // with no active slot the flow has already been blocked before this point.
  const identifier = saveSessionID ?? "";
  const revision = saveRevision ?? "";
  const slots = identifier === "" || saveRevision === undefined ? [] : activeCharacters;

  const results = useQueries({
    queries: slots.map((character) =>
      saveValidationReportQuery(port, identifier, revision, character.characterID),
    ),
  });

  const aggregate = aggregateValidationReports(
    results.map((result) => result.data),
    identifier,
    revision,
    slots.map((character) => character.characterID),
  );

  // A report that never arrived, and one that answers about another save state,
  // are both reported as their own state so the interface can say so instead of
  // implying the save is fine.
  const state: SaveValidationState =
    slots.length === 0
      ? "pending"
      : results.some((result) => result.isError) || aggregate.stale
        ? "failed"
        : results.some((result) => result.isPending)
          ? "pending"
          : aggregate.verdict;

  return {
    state,
    reports: aggregate.matching,
    saveSessionID,
    saveRevision,
    errorCount: aggregate.errorCount,
    warningCount: aggregate.warningCount,
    uncheckedScopes: aggregate.uncheckedScopes,
    unresolvedRecords: aggregate.unresolvedRecords,
  };
}
