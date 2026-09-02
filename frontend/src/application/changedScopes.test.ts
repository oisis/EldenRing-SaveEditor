import { describe, expect, it } from "vitest";
import {
  type ChangedScope,
  changedScopeQueryKeyPatterns,
  changedScopes,
  matchesQueryKeyPattern,
  queryKeyPatternsForScopes,
} from "./changedScopes";
import { queryKeys } from "./queryKeys";

const session = "session-1";

/**
 * The expected mapping, written out by hand. Reading it back from the
 * implementation would prove only that the table was read; this is the reviewed
 * contract of every backend scope.
 *
 * Each entry lists the query keys the scope must invalidate and the keys it must
 * leave alone, so a pattern that is too wide fails just as loudly as one that is
 * too narrow.
 */
const expected: Record<ChangedScope, { invalidates: readonly (readonly unknown[])[] }> = {
  "save.session": { invalidates: [queryKeys.loadedSave(session)] },
  "character.list": { invalidates: [queryKeys.saveCharacters(session, "3")] },
  "character.profile": { invalidates: [queryKeys.characterProfile(session, 0, "3")] },
  "character.stats": { invalidates: [queryKeys.characterStats(session, 0, "3")] },
  "character.appearance": { invalidates: [] },
  inventory: { invalidates: [queryKeys.inventory(session, 0, "common", 1, 50, "3")] },
  storage: { invalidates: [queryKeys.storage(session, 0, "common", 1, 50, "3")] },
  "equipment.loadout": {
    invalidates: [
      queryKeys.characterLoadout(session, 0, "3"),
      queryKeys.equipment(session, 0, "3"),
      queryKeys.quickItems(session, 0, "3"),
      queryKeys.pouchItems(session, 0, "3"),
      queryKeys.physickMixture(session, 0, "3"),
      queryKeys.equippedSpells(session, 0, "3"),
    ],
  },
  "world.flags": { invalidates: [] },
  network: { invalidates: [] },
  favorites: { invalidates: [] },
  "diagnostics.report": {
    invalidates: [queryKeys.saveValidationReport(session, 0, "3")],
  },
};

/** Every save-dependent key of one session, used as the "must not match" set. */
const everyKey: readonly (readonly unknown[])[] = [
  queryKeys.loadedSave(session),
  queryKeys.saveCharacters(session, "3"),
  queryKeys.characterProfile(session, 0, "3"),
  queryKeys.characterStats(session, 0, "3"),
  queryKeys.inventory(session, 0, "common", 1, 50, "3"),
  queryKeys.storage(session, 0, "common", 1, 50, "3"),
  queryKeys.characterLoadout(session, 0, "3"),
  queryKeys.equipment(session, 0, "3"),
  queryKeys.quickItems(session, 0, "3"),
  queryKeys.pouchItems(session, 0, "3"),
  queryKeys.physickMixture(session, 0, "3"),
  queryKeys.equippedSpells(session, 0, "3"),
  queryKeys.saveValidationReport(session, 0, "3"),
];

/** Query keys are compared as their serialized form; they are fresh arrays. */
function identity(keys: readonly (readonly unknown[])[]): string[] {
  return keys.map((key) => JSON.stringify(key)).sort();
}

function matched(scopes: readonly ChangedScope[]): string[] {
  const patterns = queryKeyPatternsForScopes(session, scopes);
  return identity(
    everyKey.filter((key) => patterns.some((pattern) => matchesQueryKeyPattern(key, pattern))),
  );
}

describe("changed scope mapping", () => {
  it("maps every backend scope exactly, and nothing beyond it", () => {
    for (const scope of changedScopes) {
      expect(matched([scope]), scope).toEqual(identity(expected[scope].invalidates));
    }
  });

  it("covers the complete backend vocabulary", () => {
    expect(Object.keys(changedScopeQueryKeyPatterns).sort()).toEqual([...changedScopes].sort());
  });

  it("invalidates a per-character view for every slot, not only the selected one", () => {
    const patterns = queryKeyPatternsForScopes(session, ["character.profile"]);

    for (const slot of [0, 3, 9]) {
      const key = queryKeys.characterProfile(session, slot, "12");
      expect(
        patterns.some((pattern) => matchesQueryKeyPattern(key, pattern)),
        `slot ${slot}`,
      ).toBe(true);
    }
  });

  it("never reaches another session or the global catalog", () => {
    const patterns = queryKeyPatternsForScopes(session, [...changedScopes]);
    const foreign = [
      queryKeys.loadedSave("session-2"),
      queryKeys.characterProfile("session-2", 0, "3"),
      queryKeys.catalogResource("item", "key"),
      queryKeys.applicationInfo(),
    ];

    for (const key of foreign) {
      expect(
        patterns.some((pattern) => matchesQueryKeyPattern(key, pattern)),
        String(key),
      ).toBe(false);
    }
  });

  it("deduplicates patterns shared by several scopes", () => {
    const once = queryKeyPatternsForScopes(session, ["equipment.loadout"]);
    const twice = queryKeyPatternsForScopes(session, ["equipment.loadout", "equipment.loadout"]);

    expect(twice).toHaveLength(once.length);
  });
});
