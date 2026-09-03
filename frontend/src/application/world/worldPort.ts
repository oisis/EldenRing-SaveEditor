/**
 * The port the application layer needs in order to read the World state of one
 * character slot. Infrastructure implements it; feature modules depend on it
 * through the hooks in this directory and never on the transport that fulfils
 * it.
 *
 * Stage 9A is read-only. The port carries the thirteen World getters and no
 * mutation at all: writing a World flag needs the operation risk level, the
 * risk reason and the per-action capabilities the backend does not publish yet,
 * and those, together with the safe bulk operations, belong to stage 9B. No
 * risk, capability or "current step" is invented here to fill the gap.
 *
 * Every field is the backend's own answer, carried as reported. Nothing above
 * this port resolves an event flag, derives a region from a name, normalises a
 * Spectral Steed conflict into one active attire or re-orders an entry list.
 */

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
};
