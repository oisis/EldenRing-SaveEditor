/**
 * The port for the backend's non-mutating save diagnostics.
 *
 * Every type here mirrors the backend result exactly. The frontend judges no
 * save data: it may read the counts and the coverage the backend reports and
 * nothing else. No severity is reinterpreted, no issue is classified, no
 * coverage gap is explained away and no repair is ever proposed here.
 */

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

export type DiagnosticsPort = {
  getSaveValidationReport: (request: SaveValidationReportRequest) => Promise<SaveValidationReport>;
};
