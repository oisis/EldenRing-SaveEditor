import { describe, expect, it } from "vitest";
import { parseSessionChangedEvent } from "./sessionChangedEvent";

const complete = {
  sequence: "4",
  operationID: "op-abc",
  operationKind: "set_character_name",
  saveSessionID: "session-1",
  saveRevision: "7",
  changedScopes: ["character.list", "diagnostics.report", "save.session"],
};

describe("session.changed payload validation", () => {
  it("accepts the complete event and carries every member verbatim", () => {
    expect(parseSessionChangedEvent(complete)).toEqual(complete);
  });

  it("drops a payload that does not carry the whole contract", () => {
    const rejected: Record<string, unknown> = {
      "not an object": "session.changed",
      null: null,
      array: [complete],
      "missing sequence": { ...complete, sequence: undefined },
      "empty sequence": { ...complete, sequence: "" },
      "numeric sequence": { ...complete, sequence: 4 },
      "leading-zero sequence": { ...complete, sequence: "04" },
      "signed sequence": { ...complete, sequence: "+4" },
      "missing operationID": { ...complete, operationID: undefined },
      "missing operationKind": { ...complete, operationKind: undefined },
      "missing session": { ...complete, saveSessionID: undefined },
      "missing revision": { ...complete, saveRevision: undefined },
      "leading-zero revision": { ...complete, saveRevision: "07" },
      "missing scopes": { ...complete, changedScopes: undefined },
      "empty scopes": { ...complete, changedScopes: [] },
      "non-string scope": { ...complete, changedScopes: ["save.session", 3] },
      "unknown scope": { ...complete, changedScopes: ["future.scope"] },
      "duplicate scope": { ...complete, changedScopes: ["save.session", "save.session"] },
      "non-canonical scope order": {
        ...complete,
        changedScopes: ["save.session", "character.list"],
      },
    };

    for (const [name, payload] of Object.entries(rejected)) {
      expect(parseSessionChangedEvent(payload), name).toBeNull();
    }
  });
});
