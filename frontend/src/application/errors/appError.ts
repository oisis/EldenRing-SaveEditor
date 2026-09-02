/**
 * The single frontend model of a backend failure.
 *
 * It mirrors the backend's public error model one to one and adds nothing: the
 * frontend never invents a code, never derives one from the wording of a
 * message and never classifies a failure by matching text. `code` is the only
 * thing a caller branches on.
 *
 * `message` is a fallback and not the text the user sees. The interface owns the
 * final localized wording and resolves it from `code` and `params`; the backend
 * message is what it falls back to for a code it does not know, next to the
 * `diagnosticID` that correlates the failure with the backend log.
 */
export type AppErrorFieldError = {
  /** The rejected request field, exactly as the backend named it. */
  field: string;
  /** Stable code of the field failure. */
  code: string;
  /** Safe English fallback for the field failure. */
  message: string;
};

export type AppError = {
  code: string;
  message: string;
  params: Readonly<Record<string, string>>;
  severity: string;
  stage: string;
  retryable: boolean;
  fieldErrors: readonly AppErrorFieldError[];
  /**
   * The session's current revision, present only on a revision conflict. It is
   * the backend's exact string and is never parsed or compared numerically.
   */
  currentRevision: string | null;
  diagnosticID: string;
};

/**
 * The stable codes the frontend reacts to. Every other code is valid and is
 * rendered through the unknown-code fallback; this list is what the application
 * has behaviour for, not the closed backend vocabulary.
 */
export const appErrorCodes = {
  invalidRequest: "invalid_request",
  invalidRevision: "invalid_revision",
  revisionConflict: "revision_conflict",
  unknownSaveSession: "unknown_save_session",
  operationFailed: "operation_failed",
  internalError: "internal_error",
  /**
   * The one code the frontend itself produces. It means the call did not
   * complete and the backend said nothing the frontend could validate: a
   * transport failure, a host rejection, or an error envelope that was absent,
   * truncated or malformed. It is never used to describe a domain outcome.
   */
  bridgeCallFailed: "bridge_call_failed",
} as const;

/** Kept as its own export because it is the code the bridge falls back to. */
export const bridgeFailureCode = appErrorCodes.bridgeCallFailed;

/**
 * The failure reported when nothing trustworthy came back. It carries no
 * backend message, because there is none to trust.
 */
export function bridgeCallFailed(): AppError {
  return {
    code: bridgeFailureCode,
    message: "The call to the application backend failed.",
    params: {},
    severity: "error",
    stage: "transport",
    retryable: false,
    fieldErrors: [],
    currentRevision: null,
    diagnosticID: "",
  };
}

/**
 * The error thrown by the application ports. It stays an ordinary `Error` so
 * every existing consumer and every test helper keeps working, and its message
 * is the stable code rather than a sentence, so nothing is ever tempted to
 * parse it.
 */
export class AppErrorException extends Error {
  readonly appError: AppError;

  constructor(appError: AppError) {
    super(appError.code);
    this.name = "AppErrorException";
    this.appError = appError;
  }
}

/**
 * Recovers the structured failure from a caught value. Anything that is not an
 * `AppErrorException` is reduced to `bridge_call_failed`, so a caller always
 * has a code and never has to inspect a raw rejection.
 */
export function toAppError(reason: unknown): AppError {
  return reason instanceof AppErrorException ? reason.appError : bridgeCallFailed();
}

/** True when the failure is a revision conflict carrying the current revision. */
export function isRevisionConflict(error: AppError): boolean {
  return error.code === appErrorCodes.revisionConflict && error.currentRevision !== null;
}
