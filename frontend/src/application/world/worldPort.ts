/**
 * The port the application layer needs in order to read the World state of one
 * character slot. Infrastructure implements it; feature modules depend on it
 * through the hooks in this directory and never on the transport that fulfils
 * it.
 *
 * Stage 9B adds the write side. The port carries the thirteen World getters,
 * the backend capability contract and the fifteen World mutations. The
 * capability contract is what makes a writer available: the risk level, the
 * risk reason and the bulk support of an operation are the backend's own
 * answers, and no risk, availability, bulk capability or "current step" is
 * invented above this port to fill a gap.
 *
 * Every field is the backend's own answer, carried as reported. Nothing above
 * this port resolves an event flag, derives a region from a name, normalises a
 * Spectral Steed conflict into one active attire or re-orders an entry list.
 */

import type { MutationReceipt, OperationRisk } from "../save-session/saveSessionPort";

/** The session and slot one World view is read for. */
export type WorldRequest = {
  saveSessionID: string;
  characterID: number;
};

/** The identity and slot state every World getter reports with its payload. */
export type WorldViewIdentity = {
  saveSessionID: string;
  saveRevision: string;
  characterID: number;
  active: boolean;
};

export type RegionEntry = {
  kind: string;
  key: string;
  name: string;
  area: string;
  unlocked: boolean;
};

export type MapRegionEntry = {
  kind: string;
  key: string;
  name: string;
  areaLabel: string;
  visible: boolean;
};

export type GraceEntry = {
  kind: string;
  key: string;
  name: string;
  regionLabel: string;
  bossArena: boolean;
  dungeonType: string;
  visited: boolean;
};

export type BossEntry = {
  kind: string;
  key: string;
  name: string;
  regionLabel: string;
  encounterType: string;
  remembrance: boolean;
  defeated: boolean;
};

/**
 * One declared step of a questline. `matched` is an independent per-step fact:
 * several steps of one questline can match at once, and no "current step" is
 * derived from them here or above this port.
 */
export type QuestStepEntry = {
  stepKind: string;
  stepKey: string;
  description: string;
  location: string;
  matched: boolean;
};

export type QuestEntry = {
  kind: string;
  key: string;
  name: string;
  steps: readonly QuestStepEntry[];
};

export type GestureEntry = {
  kind: string;
  key: string;
  slotID: number;
  name: string;
  category: string;
  unlocked: boolean;
};

export type CookbookEntry = {
  kind: string;
  key: string;
  name: string;
  category: string;
  unlocked: boolean;
};

export type BellBearingEntry = {
  kind: string;
  key: string;
  name: string;
  category: string;
  unlocked: boolean;
};

export type WhetbladeEntry = {
  kind: string;
  key: string;
  name: string;
  unlocked: boolean;
};

export type TutorialEntry = {
  kind: string;
  key: string;
  title: string;
  unlocked: boolean;
};

export type SummoningPoolEntry = {
  kind: string;
  key: string;
  name: string;
  regionLabel: string;
  activated: boolean;
};

export type ColosseumEntry = {
  kind: string;
  key: string;
  name: string;
  unlocked: boolean;
};

export type SpectralSteedAttireEntry = {
  attireKey: string;
  name: string;
  owned: boolean;
  requiredResourceKind: string;
  requiredResourceKey: string;
  iconPath: string;
};

export type WorldRegions = WorldViewIdentity & { regions: readonly RegionEntry[] };
export type WorldMapRegions = WorldViewIdentity & { mapRegions: readonly MapRegionEntry[] };
export type WorldGraces = WorldViewIdentity & { graces: readonly GraceEntry[] };
export type WorldBosses = WorldViewIdentity & { bosses: readonly BossEntry[] };
export type WorldQuests = WorldViewIdentity & { quests: readonly QuestEntry[] };
export type WorldGestures = WorldViewIdentity & { gestures: readonly GestureEntry[] };
export type WorldCookbooks = WorldViewIdentity & { cookbooks: readonly CookbookEntry[] };
export type WorldBellBearings = WorldViewIdentity & { bellBearings: readonly BellBearingEntry[] };
export type WorldWhetblades = WorldViewIdentity & { whetblades: readonly WhetbladeEntry[] };
export type WorldTutorials = WorldViewIdentity & { tutorials: readonly TutorialEntry[] };
export type WorldSummoningPools = WorldViewIdentity & {
  summoningPools: readonly SummoningPoolEntry[];
};
export type WorldColosseums = WorldViewIdentity & { colosseums: readonly ColosseumEntry[] };

/**
 * The backend's closed classification of the Spectral Steed attires. Any other
 * value is an unknown contract, not a fourth state, and is rejected at the
 * bridge boundary instead of being carried into the application.
 */
export type SpectralSteedAttireStatus = "resolved" | "legacy" | "conflict";

/**
 * The Spectral Steed view. `status` is the backend's own classification and it
 * is presented as reported: `legacy` and `conflict` are never collapsed into a
 * single active attire, and `activeAttireKey` is filled by the backend only
 * when the status is resolved.
 */
export type WorldSpectralSteedAttires = WorldViewIdentity & {
  status: SpectralSteedAttireStatus;
  activeAttireKey: string;
  attires: readonly SpectralSteedAttireEntry[];
};

/**
 * The closed set of World mutations this build can perform. It is the
 * vocabulary the backend publishes its capabilities under, and a value outside
 * it is an unknown contract that is rejected at the bridge boundary rather than
 * carried into the application.
 */
export const worldOperationKinds = [
  "lock_all_spectral_steed_attires",
  "set_bell_bearing_unlocked",
  "set_boss_defeated",
  "set_colosseum_unlocked",
  "set_cookbook_unlocked",
  "set_fog_of_war_removed",
  "set_gesture_unlocked",
  "set_grace_visited",
  "set_map_region_revealed",
  "set_quest_step",
  "set_region_unlocked",
  "set_spectral_steed_attire",
  "set_summoning_pool_activated",
  "set_tutorial_unlocked",
  "set_whetblade_unlocked",
] as const;

export type WorldOperationKind = (typeof worldOperationKinds)[number];

/**
 * One supported World mutation as the backend describes it. Its presence in the
 * answer is what declares the operation available; `risk` and `riskReason` are
 * the backend's own values, shown before the operation runs and never derived,
 * defaulted or upgraded locally.
 */
export type WorldMutationCapability = {
  operationKind: WorldOperationKind;
  risk: OperationRisk;
  riskReason: string;
  supportsBulk: boolean;
};

/**
 * The receipt of a committed World mutation is the shared save-session receipt,
 * carried verbatim. World does not own a second receipt model, so the scopes it
 * reports go through the one invalidation path every other workspace uses.
 */
export type WorldMutationReceipt = MutationReceipt;

/** The session, slot and revision every World mutation is committed under. */
export type WorldMutationScope = {
  saveSessionID: string;
  characterID: number;
  expectedRevision: string;
};

/**
 * The eleven World mutations that assign one boolean to one catalog resource.
 * `resourceKind` and `resourceKey` are the endpoint's own pair and `value` is
 * the state being written, forwarded unchanged; the adapter only places them in
 * the positions the endpoint declares.
 */
export type WorldResourceToggleRequest = WorldMutationScope & {
  resourceKind: string;
  resourceKey: string;
  value: boolean;
};

export type WorldPort = {
  getRegions: (request: WorldRequest) => Promise<WorldRegions>;
  getMapRegions: (request: WorldRequest) => Promise<WorldMapRegions>;
  getGraces: (request: WorldRequest) => Promise<WorldGraces>;
  getBosses: (request: WorldRequest) => Promise<WorldBosses>;
  getQuests: (request: WorldRequest) => Promise<WorldQuests>;
  getGestures: (request: WorldRequest) => Promise<WorldGestures>;
  getCookbooks: (request: WorldRequest) => Promise<WorldCookbooks>;
  getBellBearings: (request: WorldRequest) => Promise<WorldBellBearings>;
  getWhetblades: (request: WorldRequest) => Promise<WorldWhetblades>;
  getTutorials: (request: WorldRequest) => Promise<WorldTutorials>;
  getSummoningPools: (request: WorldRequest) => Promise<WorldSummoningPools>;
  getColosseums: (request: WorldRequest) => Promise<WorldColosseums>;
  getSpectralSteedAttires: (request: WorldRequest) => Promise<WorldSpectralSteedAttires>;

  /**
   * The World mutation contract of this build. It takes no session and no slot,
   * so it is read once and is not tied to a save revision.
   */
  getWorldMutationCapabilities: () => Promise<readonly WorldMutationCapability[]>;

  setRegionUnlocked: (request: WorldResourceToggleRequest) => Promise<WorldMutationReceipt>;
  setMapRegionRevealed: (request: WorldResourceToggleRequest) => Promise<WorldMutationReceipt>;
  setGraceVisited: (request: WorldResourceToggleRequest) => Promise<WorldMutationReceipt>;
  setBossDefeated: (request: WorldResourceToggleRequest) => Promise<WorldMutationReceipt>;
  setGestureUnlocked: (request: WorldResourceToggleRequest) => Promise<WorldMutationReceipt>;
  setCookbookUnlocked: (request: WorldResourceToggleRequest) => Promise<WorldMutationReceipt>;
  setBellBearingUnlocked: (request: WorldResourceToggleRequest) => Promise<WorldMutationReceipt>;
  setWhetbladeUnlocked: (request: WorldResourceToggleRequest) => Promise<WorldMutationReceipt>;
  setTutorialUnlocked: (request: WorldResourceToggleRequest) => Promise<WorldMutationReceipt>;
  setSummoningPoolActivated: (request: WorldResourceToggleRequest) => Promise<WorldMutationReceipt>;
  setColosseumUnlocked: (request: WorldResourceToggleRequest) => Promise<WorldMutationReceipt>;

  /**
   * The one-way Fog of War removal. `removed` is typed as the literal `true`
   * because the backend accepts no other value: the inverse has no confirmed
   * contract, so it cannot be expressed here at all.
   */
  setFogOfWarRemoved: (
    request: WorldMutationScope & { removed: true },
  ) => Promise<WorldMutationReceipt>;

  /**
   * One curated quest step, addressed by its own kind and key pair. A step is
   * chosen explicitly: `matched` is an independent per-step fact and is never
   * treated as a current step that could be toggled.
   */
  setQuestStep: (
    request: WorldMutationScope & {
      questKind: string;
      questKey: string;
      stepKind: string;
      stepKey: string;
    },
  ) => Promise<WorldMutationReceipt>;

  /** Activates one appearance; the ownership rule is the backend's. */
  setSpectralSteedAttire: (
    request: WorldMutationScope & { attireKey: string },
  ) => Promise<WorldMutationReceipt>;

  /**
   * The one bulk World mutation: a single atomic call that removes the three
   * attire items and restores the default appearance together. It is never
   * emulated by a sequence of single mutations.
   */
  lockAllSpectralSteedAttires: (request: WorldMutationScope) => Promise<WorldMutationReceipt>;
};
