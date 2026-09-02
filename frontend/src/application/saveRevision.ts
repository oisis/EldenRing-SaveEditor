/** The identity every save-dependent getter must return with its payload. */
export type SaveSnapshotIdentity = {
  saveSessionID: string;
  saveRevision: string;
};

/**
 * Rejects a response that was produced for another session or revision before
 * TanStack Query can cache it. Revisions are opaque backend strings: equality
 * is exact and no parsing, trimming or ordering is applied.
 */
export function requireCurrentSaveResponse<T extends SaveSnapshotIdentity>(
  response: T,
  saveSessionID: string,
  saveRevision: string,
): T {
  if (response.saveSessionID !== saveSessionID || response.saveRevision !== saveRevision) {
    throw new Error("stale_save_response");
  }
  return response;
}

/**
 * True for a canonical decimal string: no sign, no leading zero, no separator
 * and no exponent. Both `saveRevision` and the `session.changed` sequence are
 * rendered this way by the backend, so both are validated here rather than by
 * two regular expressions that could drift apart.
 *
 * The value stays a string throughout: parsing one into a JavaScript number
 * could not represent every value the backend can produce.
 */
export function isCanonicalDecimal(value: unknown): value is string {
  return typeof value === "string" && /^(0|[1-9][0-9]*)$/.test(value);
}
