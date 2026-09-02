import { describe, expect, it } from "vitest";
import { bridgeFailureCode } from "../../application/errors/appError";
import { bridgeErrorPrefix, parseBridgeError } from "./bridgeError";

function envelope(payload: unknown): Error {
  return new Error(bridgeErrorPrefix + JSON.stringify(payload));
}

const complete = {
  code: "revision_conflict",
  message: 'expectedRevision "1" does not match the current saveRevision "4"',
  params: { expectedRevision: "1", currentRevision: "4" },
  severity: "error",
  stage: "mutation",
  retryable: false,
  currentRevision: "4",
  diagnosticID: "diag-abc",
};

describe("bridge error envelope", () => {
  it("recovers the complete structured failure", () => {
    expect(parseBridgeError(envelope(complete))).toEqual({
      code: "revision_conflict",
      message: 'expectedRevision "1" does not match the current saveRevision "4"',
      params: { expectedRevision: "1", currentRevision: "4" },
      severity: "error",
      stage: "mutation",
      retryable: false,
      fieldErrors: [],
      currentRevision: "4",
      diagnosticID: "diag-abc",
    });
  });

  it("recovers field errors", () => {
    const parsed = parseBridgeError(
      envelope({
        code: "invalid_request",
        message: "saveSessionID is required",
        severity: "error",
        stage: "request",
        retryable: false,
        fieldErrors: [
          { field: "saveSessionID", code: "invalid_request", message: "saveSessionID is required" },
        ],
        diagnosticID: "diag-1",
      }),
    );

    expect(parsed.code).toBe("invalid_request");
    expect(parsed.fieldErrors).toEqual([
      { field: "saveSessionID", code: "invalid_request", message: "saveSessionID is required" },
    ]);
    // An absent optional member is normalised, never invented.
    expect(parsed.currentRevision).toBeNull();
  });

  it("reduces anything that is not a complete envelope to the safe fallback", () => {
    const rejected: [string, unknown][] = [
      ["not an error", "saveforge-error:{}"],
      ["unmarked message", new Error(JSON.stringify(complete))],
      ["marker only", new Error(bridgeErrorPrefix)],
      ["truncated json", new Error(`${bridgeErrorPrefix}{"code":"revision_confl`)],
      ["array payload", new Error(`${bridgeErrorPrefix}["revision_conflict"]`)],
      ["missing code", envelope({ ...complete, code: undefined })],
      ["missing message", envelope({ ...complete, message: undefined })],
      ["missing stage", envelope({ ...complete, stage: undefined })],
      ["wrong param type", envelope({ ...complete, params: { expectedRevision: 1 } })],
      ["wrong field error shape", envelope({ ...complete, fieldErrors: [{ field: "x" }] })],
      ["missing retryable", envelope({ ...complete, retryable: undefined })],
      ["wrong retryable type", envelope({ ...complete, retryable: "no" })],
      ["missing diagnostic ID", envelope({ ...complete, diagnosticID: undefined })],
      ["empty diagnostic ID", envelope({ ...complete, diagnosticID: "" })],
      ["wrong revision type", envelope({ ...complete, currentRevision: 4 })],
      ["undefined", undefined],
    ];

    for (const [name, reason] of rejected) {
      const parsed = parseBridgeError(reason);
      expect(parsed.code, name).toBe(bridgeFailureCode);
      expect(parsed.currentRevision, name).toBeNull();
    }
  });

  it("never carries a raw transport failure into the model", () => {
    const raw = new Error("goroutine 1 [running]: /Users/private/app.go:42");

    const parsed = parseBridgeError(raw);

    expect(parsed.code).toBe(bridgeFailureCode);
    expect(JSON.stringify(parsed)).not.toContain("/Users/private");
    expect(JSON.stringify(parsed)).not.toContain("goroutine");
  });
});
