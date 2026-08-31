import { describe, expect, it } from "vitest";
import { requireCurrentSaveResponse } from "./saveRevision";

const response = {
  saveSessionID: "session-1",
  saveRevision: "  Revision 09  ",
  value: "payload",
};

describe("requireCurrentSaveResponse", () => {
  it("returns the original response for the exact opaque session identity", () => {
    expect(requireCurrentSaveResponse(response, "session-1", "  Revision 09  ")).toBe(response);
  });

  it("rejects a response from another session", () => {
    expect(() => requireCurrentSaveResponse(response, "session-2", "  Revision 09  ")).toThrowError(
      "stale_save_response",
    );
  });

  it("compares revisions exactly without parsing or normalising them", () => {
    expect(() => requireCurrentSaveResponse(response, "session-1", "Revision 09")).toThrowError(
      "stale_save_response",
    );
    expect(() =>
      requireCurrentSaveResponse({ ...response, saveRevision: "9" }, "session-1", "09"),
    ).toThrowError("stale_save_response");
  });
});
