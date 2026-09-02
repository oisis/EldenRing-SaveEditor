import { type ChangedScope, changedScopes } from "../../application/changedScopes";
import type { SessionChangedEvent } from "../../application/save-session/saveSessionPort";
import { isCanonicalDecimal } from "../../application/saveRevision";

/**
 * The backend's stable event name. It is the same constant the Go side emits
 * under, so neither side carries a literal of its own.
 */
export const sessionChangedEventName = "session.changed";

/** The closed scope vocabulary, as a set, so membership is a lookup. */
const knownScopes: ReadonlySet<string> = new Set<string>(changedScopes);

/**
 * Validates one payload the Wails event bus delivered.
 *
 * The validation is fail-closed for the same reason the error envelope's is: an
 * event is a trigger for invalidation, so a payload that is not fully understood
 * could refresh the wrong views or advance the sequence past an event nobody
 * read. Every member of the contract has to be present and of the exact shape
 * the backend documents:
 *
 *   - `sequence` and `saveRevision` are canonical decimal strings. A number, a
 *     signed value, a leading zero or a padded string is rejected rather than
 *     normalised, because the frontend compares these as strings;
 *   - `changedScopes` is a non-empty, ascending list of distinct values of the
 *     closed scope vocabulary. An unknown or non-canonical list means the
 *     backend and this build no longer agree on the contract, and acting on the
 *     understood fragment would silently skip a view that changed;
 *   - the operation identifiers and the session identifier are non-empty.
 *
 * An unusable payload yields `null`. The listener treats that as a gap and
 * resynchronises through GetLoadedSave, so the rejected event is never applied
 * and never silently dropped either.
 */
export function parseSessionChangedEvent(payload: unknown): SessionChangedEvent | null {
  if (typeof payload !== "object" || payload === null || Array.isArray(payload)) {
    return null;
  }
  const raw = payload as Record<string, unknown>;
  const operationID = nonEmptyString(raw.operationID);
  const operationKind = nonEmptyString(raw.operationKind);
  const saveSessionID = nonEmptyString(raw.saveSessionID);
  if (
    !isCanonicalDecimal(raw.sequence) ||
    !isCanonicalDecimal(raw.saveRevision) ||
    operationID === null ||
    operationKind === null ||
    saveSessionID === null
  ) {
    return null;
  }
  if (!Array.isArray(raw.changedScopes) || raw.changedScopes.length === 0) {
    return null;
  }
  const scopes: ChangedScope[] = [];
  for (const scope of raw.changedScopes) {
    if (
      typeof scope !== "string" ||
      !knownScopes.has(scope) ||
      scopes.includes(scope as ChangedScope) ||
      (scopes.length > 0 && scopes[scopes.length - 1] >= scope)
    ) {
      return null;
    }
    scopes.push(scope as ChangedScope);
  }
  return {
    sequence: raw.sequence,
    operationID,
    operationKind,
    saveSessionID,
    saveRevision: raw.saveRevision,
    changedScopes: scopes,
  };
}

function nonEmptyString(value: unknown): string | null {
  return typeof value === "string" && value !== "" ? value : null;
}
