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
