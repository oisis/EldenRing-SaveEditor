/**
 * The one mapping from a backend invalidation scope onto the frontend query
 * keys it invalidates. It lives beside `queryKeys` on purpose: the keys and the
 * scopes that invalidate them are a single source of truth, and no component
 * builds either of them.
 *
 * A component never refreshes "the whole application" for an ordinary event.
 * The only place a session-wide refresh is allowed is a detected gap in the
 * event sequence, which is handled once in the synchronisation hook.
 */

/**
 * The closed, stable vocabulary of backend invalidation scopes. Every value the
 * backend can send appears here, including the scopes no frontend getter
 * consumes yet: an exhaustive map is what proves an unmapped scope is a
 * decision rather than an oversight.
 */
export const changedScopes = [
  "save.session",
  "character.list",
  "character.profile",
  "character.stats",
  "character.appearance",
  "inventory",
  "storage",
  "equipment.loadout",
  "world.flags",
  "network",
  "favorites",
  "diagnostics.report",
] as const;

export type ChangedScope = (typeof changedScopes)[number];

/**
 * The wildcard segment of a key pattern. It stands for exactly one segment of
 * any value and exists because a per-character key carries the slot index in
 * the middle: `["save-session", id, "character", 3, "profile", "7"]` cannot be
 * addressed by a plain prefix when every slot has to be invalidated.
 */
export const anySegment = Symbol("any query key segment");

/** One key pattern: literal segments, with `anySegment` matching any single one. */
export type QueryKeyPattern = readonly (string | number | typeof anySegment)[];

/**
 * True when `key` starts with `pattern`. Matching is by prefix, so the trailing
 * revision segment of a save-dependent key is covered without the pattern
 * naming it: every cached revision of the same view is invalidated together.
 */
export function matchesQueryKeyPattern(key: readonly unknown[], pattern: QueryKeyPattern): boolean {
  if (key.length < pattern.length) {
    return false;
  }
  return pattern.every((segment, index) => segment === anySegment || key[index] === segment);
}

/** Every per-character view of one session, whatever slot it belongs to. */
function characterView(saveSessionID: string, view: string): QueryKeyPattern {
  return ["save-session", saveSessionID, "character", anySegment, view];
}

/**
 * The exhaustive scope map. Each entry lists the key patterns one scope
 * invalidates for one session; an empty list is an explicit statement that no
 * frontend getter reads that scope yet, and it gains patterns in the same task
 * that adds such a getter.
 */
export const changedScopeQueryKeyPatterns: Record<
  ChangedScope,
  (saveSessionID: string) => readonly QueryKeyPattern[]
> = {
  // The session's own metadata: revision, unsaved changes and event sequence.
  "save.session": (id) => [["save-session", id, "loaded"]],
  "character.list": (id) => [["save-session", id, "characters"]],
  "character.profile": (id) => [characterView(id, "profile")],
  "character.stats": (id) => [characterView(id, "stats")],
  // No appearance getter is wired into the frontend yet.
  "character.appearance": () => [],
  // Both containers have two getters: the raw physical page and the
  // authoritative, backend-filtered workspace page. They answer different
  // questions about the same records, so a mutation of a container refreshes
  // both. The owned-items pattern carries the container as its next segment,
  // so `inventory` never invalidates the Storage workspace and the other way
  // round.
  inventory: (id) => [
    characterView(id, "inventory"),
    [...characterView(id, "owned-items"), "inventory"],
  ],
  storage: (id) => [characterView(id, "storage"), [...characterView(id, "owned-items"), "storage"]],
  // The coherent loadout and the five narrow equipped views are independent
  // getters of one scope, so all six are refreshed together.
  "equipment.loadout": (id) => [
    characterView(id, "loadout"),
    characterView(id, "equipment"),
    characterView(id, "quick-items"),
    characterView(id, "pouch-items"),
    characterView(id, "physick-mixture"),
    characterView(id, "equipped-spells"),
  ],
  // No World getter is wired into the frontend yet.
  "world.flags": () => [],
  // No Network getter is wired into the frontend yet.
  network: () => [],
  // No Favorites getter is wired into the frontend yet.
  favorites: () => [],
  "diagnostics.report": (id) => [characterView(id, "validation-report")],
};

/**
 * The key patterns a set of changed scopes invalidates, with duplicates
 * removed. The infrastructure boundary rejects an unknown scope before this
 * function is called. Keeping the parameter typed closes the second half of
 * that boundary: application code cannot silently drop a scope it does not
 * understand.
 */
export function queryKeyPatternsForScopes(
  saveSessionID: string,
  scopes: readonly ChangedScope[],
): readonly QueryKeyPattern[] {
  const patterns: QueryKeyPattern[] = [];
  const seen = new Set<string>();
  for (const scope of scopes) {
    const build = changedScopeQueryKeyPatterns[scope];
    for (const pattern of build(saveSessionID)) {
      const identity = pattern.map((segment) => String(segment)).join("\u0000");
      if (!seen.has(identity)) {
        seen.add(identity);
        patterns.push(pattern);
      }
    }
  }
  return patterns;
}
