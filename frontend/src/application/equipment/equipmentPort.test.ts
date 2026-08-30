import { describe, expect, it } from "vitest";
import { makeEquipmentPort } from "../../test/renderWithProviders";

/**
 * The port surface itself is protected here. Stage 8A exposes read-only
 * getters, so a mutation reaching the application layer through this contract
 * has to fail as a test, not as a review comment.
 */
describe("EquipmentPort surface", () => {
  const methods = Object.keys(makeEquipmentPort()).sort();

  it("declares exactly the six backend getters", () => {
    expect(methods).toEqual([
      "getCharacterLoadout",
      "getEquipment",
      "getEquippedSpells",
      "getPhysickMixture",
      "getPouchItems",
      "getQuickItems",
    ]);
  });

  it("declares no mutating method", () => {
    expect(methods.filter((method) => !method.startsWith("get"))).toEqual([]);
    expect(methods.filter((method) => /^(set|apply|save|delete|clear|equip)/.test(method))).toEqual(
      [],
    );
  });
});
