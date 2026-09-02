import {
  type AppError,
  type AppErrorFieldError,
  bridgeCallFailed,
} from "../../application/errors/appError";

/**
 * The marker the backend bridge puts in front of the JSON error model. Wails 2
 * carries only `error.Error()` back from a bound method, so the structured
 * contract travels inside that one string; this prefix is the same constant the
 * Go side writes and nothing here guesses at an undocumented serialization.
 */
export const bridgeErrorPrefix = "saveforge-error:";

/**
 * Recovers the backend's structured failure from a caught rejection.
 *
 * The validation is strict on purpose: an envelope that is absent, truncated,
 * not valid JSON, not an object, or missing a member of the contract is not
 * "partly usable". It is reduced to `bridge_call_failed`, because the
 * alternative is showing the user a failure the frontend invented from a
 * fragment.
 */
export function parseBridgeError(reason: unknown): AppError {
  const message = reason instanceof Error ? reason.message : null;
  if (message === null || !message.startsWith(bridgeErrorPrefix)) {
    return bridgeCallFailed();
  }
  const payload = message.slice(bridgeErrorPrefix.length);
  if (payload === "") {
    return bridgeCallFailed();
  }

  let decoded: unknown;
  try {
    decoded = JSON.parse(payload);
  } catch {
    return bridgeCallFailed();
  }
  if (typeof decoded !== "object" || decoded === null || Array.isArray(decoded)) {
    return bridgeCallFailed();
  }

  const raw = decoded as Record<string, unknown>;
  const code = stringMember(raw.code);
  const errorMessage = stringMember(raw.message);
  const severity = stringMember(raw.severity);
  const stage = stringMember(raw.stage);
  const diagnosticID = stringMember(raw.diagnosticID);
  if (
    code === null ||
    errorMessage === null ||
    severity === null ||
    stage === null ||
    diagnosticID === null ||
    typeof raw.retryable !== "boolean"
  ) {
    return bridgeCallFailed();
  }
  const params = stringMap(raw.params);
  const fieldErrors = fieldErrorList(raw.fieldErrors);
  if (params === null || fieldErrors === null) {
    return bridgeCallFailed();
  }
  const currentRevision = optionalStringMember(raw.currentRevision);
  if (currentRevision === undefined) {
    return bridgeCallFailed();
  }

  return {
    code,
    message: errorMessage,
    params,
    severity,
    stage,
    retryable: raw.retryable,
    fieldErrors,
    currentRevision,
    diagnosticID,
  };
}

function stringMember(value: unknown): string | null {
  return typeof value === "string" && value !== "" ? value : null;
}

/** Returns the string, `null` for an absent member, `undefined` for a wrong type. */
function optionalStringMember(value: unknown): string | null | undefined {
  if (value === undefined || value === null) {
    return null;
  }
  return typeof value === "string" ? value : undefined;
}

function stringMap(value: unknown): Record<string, string> | null {
  if (value === undefined || value === null) {
    return {};
  }
  if (typeof value !== "object" || Array.isArray(value)) {
    return null;
  }
  const result: Record<string, string> = {};
  for (const [key, member] of Object.entries(value as Record<string, unknown>)) {
    if (typeof member !== "string") {
      return null;
    }
    result[key] = member;
  }
  return result;
}

function fieldErrorList(value: unknown): AppErrorFieldError[] | null {
  if (value === undefined || value === null) {
    return [];
  }
  if (!Array.isArray(value)) {
    return null;
  }
  const result: AppErrorFieldError[] = [];
  for (const entry of value) {
    if (typeof entry !== "object" || entry === null || Array.isArray(entry)) {
      return null;
    }
    const raw = entry as Record<string, unknown>;
    const field = stringMember(raw.field);
    const code = stringMember(raw.code);
    const message = stringMember(raw.message);
    if (field === null || code === null || message === null) {
      return null;
    }
    result.push({ field, code, message });
  }
  return result;
}
