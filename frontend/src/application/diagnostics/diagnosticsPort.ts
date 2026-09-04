/**
 * The port for the backend's non-mutating save diagnostics.
 *
 * Every type here mirrors the backend result exactly. The frontend judges no
 * save data: it may read the counts and the coverage the backend reports and
 * nothing else. No severity is reinterpreted, no issue is classified, no
 * coverage gap is explained away and no repair is ever proposed here.
 */

import type { CharacterAttributes } from "../character/characterPort";
import type { MutationReceipt } from "../save-session/saveSessionPort";

/** One finding, carried verbatim. The frontend never rewrites or ranks it. */
export type SaveValidationIssue = {
  id: string;
  /** Stable backend code. The UI may branch on it; it never renders it raw. */
  code: string;
  /** `error` or `warning`, exactly as the backend classified it. */
  severity: string;
  scope: string;
  /** The backend's safe fallback message. */
  message: string;
  /** Empty for an issue that names no container record. */
  ownedItemID: string;
};

/**
 * What one scope actually checked. `checked: false` means the scope could not
 * be judged at all, so an empty issue list does not make the save clean;
 * `unresolvedRecords` counts what stayed unjudged inside a checked scope.
 */
export type SaveValidationScopeCoverage = {
  scope: string;
  checked: boolean;
  reason: string;
  recordsChecked: number;
  unresolvedRecords: number;
};

export type SaveValidationReport = {
  saveSessionID: string;
  /** The revision the report was built from, carried as the string it is. */
  saveRevision: string;
  characterID: number;
  active: boolean;
  coverage: readonly SaveValidationScopeCoverage[];
  issues: readonly SaveValidationIssue[];
  errorCount: number;
  warningCount: number;
};

export type SaveValidationReportRequest = {
  saveSessionID: string;
  characterID: number;
  /** One backend scope, or the empty string for every scope. Sent verbatim. */
  scope: string;
};

/**
 * One planned change of a repair plan, carried verbatim.
 *
 * `issueIDs` lists every finding the one action resolves, in report order: the
 * backend may answer several findings with a single atomic write, and the
 * frontend neither splits nor merges that grouping.
 */
export type RepairAction = {
  issueIDs: readonly string[];
  scope: string;
  operation: string;
  /** Empty for an action that names no container record. */
  ownedItemID: string;
  /** The backend's own target value; absent when the operation carries none. */
  targetValue: number | undefined;
  /**
   * The statistics block a `set_character_stats` action will write, carried
   * exactly as the backend derived it and absent for every other operation. It
   * is never recomputed here: ApplyRepairs addresses a plan by its findings and
   * its token, so this is a value to read, never one this frontend sends back.
   */
  attributes?: CharacterAttributes;
  /** The backend's safe description of the change. */
  description: string;
};

/**
 * One requested finding the backend refuses to plan for, with its reason. A
 * rejection is a result, not an omission: it exists so an absent action is
 * never read as "already fine" or "handled silently".
 */
export type RepairRejection = {
  issueID: string;
  code: string;
  scope: string;
  reason: string;
};

/**
 * A non-mutating plan bound to one exact save state.
 *
 * `planToken` seals the plan. It is opaque here: the frontend never derives,
 * verifies or rebuilds it, it only returns it unchanged to ApplyRepairs so the
 * backend can prove it is executing this plan of this save version.
 */
export type RepairPlan = {
  saveSessionID: string;
  saveRevision: string;
  characterID: number;
  planToken: string;
  actions: readonly RepairAction[];
  rejected: readonly RepairRejection[];
};

export type RepairPlanRequest = {
  saveSessionID: string;
  characterID: number;
  /** The revision the selected findings were reported for. Sent verbatim. */
  saveRevision: string;
  /** The identifiers of the selected findings, in the order they were shown. */
  issueIDs: readonly string[];
};

export type ApplyRepairsRequest = {
  saveSessionID: string;
  characterID: number;
  issueIDs: readonly string[];
  planToken: string;
  expectedRevision: string;
};

/**
 * What every ApplyRepairs answer carries, whichever variant it is: the session
 * it addressed, the revision it left behind and the freshly derived plan the
 * backend accounted for.
 */
type ApplyRepairsOutcome = {
  saveSessionID: string;
  saveRevision: string;
  characterID: number;
  actions: readonly RepairAction[];
  rejected: readonly RepairRejection[];
};

/** A committed transaction: the whole mutation receipt of one execution. */
export type ApplyRepairsCommitted = MutationReceipt & ApplyRepairsOutcome & { applied: true };

/**
 * A verified selection with no executable action. The backend describes no
 * execution here, so this variant carries no operation identifier, no operation
 * kind and no changed scopes — not empty ones.
 */
export type ApplyRepairsNotApplied = ApplyRepairsOutcome & { applied: false };

/**
 * The result of applying a plan. `applied` is the backend's own discriminator
 * and the only thing that tells the two variants apart.
 */
export type ApplyRepairsResult = ApplyRepairsCommitted | ApplyRepairsNotApplied;

export type DiagnosticsPort = {
  getSaveValidationReport: (request: SaveValidationReportRequest) => Promise<SaveValidationReport>;
  /**
   * Builds the plan for the selected findings. It is non-mutating: nothing is
   * written, reserved or repaired by asking for it, and the frontend never
   * proposes a repair the plan does not contain.
   */
  getRepairPlan: (request: RepairPlanRequest) => Promise<RepairPlan>;
  /** Executes exactly the plan sealed by `planToken`, or fails without change. */
  applyRepairs: (request: ApplyRepairsRequest) => Promise<ApplyRepairsResult>;
};
