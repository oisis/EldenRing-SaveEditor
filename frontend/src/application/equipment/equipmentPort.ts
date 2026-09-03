/**
 * The port the application layer needs in order to read the equipped state of
 * one character slot. Infrastructure implements it; feature modules depend on
 * it through the hooks in this directory and never on the transport that
 * fulfils it.
 *
 * The port carries both halves of the Equipment contract: seven read-only
 * getters and the seven mutations of `EquipmentMutationPort`. A getter never
 * writes and a mutation is reached only through the application layer's single
 * mutation runner, so no feature module calls a setter on this port directly.
 *
 * Four of them report raw save state. A raw identifier is carried exactly as
 * the backend reports it and nothing is derived from it here: no name, no icon,
 * no slot label, no item kind, no compatibility, no locked-slot rule and no
 * sentinel interpretation. `0`, `0xFFFFFFFF` and every unknown value stay
 * themselves.
 *
 * Among the five narrow getters, `getEquippedSpells` is the only result the
 * backend already resolved through GameCatalog. Its `resourceKey`, `name` and
 * `memorySlots` are the backend's answer, not a local lookup.
 *
 * The coherent `getCharacterLoadout` result is the frontend-facing projection.
 * It carries resolved identities and public slot
 * states. The five older getters stay available as narrow diagnostic reads;
 * consumers must not combine them into a second loadout model.
 */

import type { ChangedScope } from "../changedScopes";

/** The session and slot one equipped view is read for. */
export type EquipmentRequest = {
  saveSessionID: string;
  characterID: number;
};

export type LoadoutSlotState = "empty" | "occupied" | "locked";

export type LoadoutResource = {
  kind: string;
  key: string;
};

export type LoadoutSlot = {
  slotType: string;
  state: LoadoutSlotState;
  /**
   * The backend's own, revision-scoped identity of the Inventory record this
   * position references. It is present exactly for the occupied hand, armor and
   * talisman positions, which are the three groups whose setter addresses an
   * owned record; an empty, locked, ammunition or Physick position carries
   * none. It is never derived, minted or repaired above this port.
   */
  ownedItemID?: string;
  resource?: LoadoutResource;
  name?: string;
  iconPath?: string;
  rawValue: number;
};

export type LoadoutOwnedSlot = {
  slotType: "quick_item" | "pouch";
  state: LoadoutSlotState;
  ownedItemID?: string;
  resource?: LoadoutResource;
  name?: string;
  iconPath?: string;
  quantity?: number;
};

export type LoadoutSpellSlot = {
  state: LoadoutSlotState;
  resource?: LoadoutResource;
  name?: string;
  iconPath?: string;
  memorySlots?: number;
};

/**
 * One backend-owned, coherent loadout snapshot. Every group, identity,
 * presentation field, state and aggregate count is carried as reported; the
 * frontend neither resolves raw game IDs nor recomputes capacity or locking.
 */
export type CharacterLoadout = {
  saveSessionID: string;
  saveRevision: string;
  characterID: number;
  active: boolean;
  rightHand: readonly LoadoutSlot[];
  leftHand: readonly LoadoutSlot[];
  arrows: readonly LoadoutSlot[];
  bolts: readonly LoadoutSlot[];
  armor: readonly LoadoutSlot[];
  talismans: readonly LoadoutSlot[];
  quickItems: readonly LoadoutOwnedSlot[];
  pouch: readonly LoadoutOwnedSlot[];
  activeQuickItem: number;
  physick: readonly LoadoutSlot[];
  spells: readonly LoadoutSpellSlot[];
  activeSpellIndex: number;
  usedMemorySlots: number;
  availableMemorySlots: number;
  unlockedTalismanSlots: number;
};

/**
 * The 22 raw ChrAsmEquipment fields of one slot, in the backend's stored order.
 * The indexes are deliberately unnamed: which physical field each one is, and
 * which of them are still unknown, is not part of this stage. No entry is
 * dropped, reordered or replaced.
 */
export type CharacterEquipment = {
  saveSessionID: string;
  saveRevision: string;
  characterID: number;
  active: boolean;
  slots: readonly number[];
};

/** One raw EquipItemData record: both values are carried unresolved. */
export type EquipItemRecord = {
  itemID: number;
  equipIndex: number;
};

/**
 * The ten raw quick-item records and the raw active-slot value behind them.
 * `activeQuick` is signed in the backend contract, so a negative value is a
 * real reported state and is never clamped, zeroed or turned into an index.
 */
export type CharacterQuickItems = {
  saveSessionID: string;
  saveRevision: string;
  characterID: number;
  active: boolean;
  items: readonly EquipItemRecord[];
  activeQuick: number;
};

/** The six raw pouch records, in the backend's stored order. */
export type CharacterPouchItems = {
  saveSessionID: string;
  saveRevision: string;
  characterID: number;
  active: boolean;
  items: readonly EquipItemRecord[];
};

/** Both raw Crystal Tear identifiers of the current Physick mixture. */
export type CharacterPhysickMixture = {
  saveSessionID: string;
  saveRevision: string;
  characterID: number;
  active: boolean;
  tears: readonly number[];
};

/**
 * One memory slot. `rawMagicParamID` is the identifier exactly as the save
 * stores it, the empty sentinel included. The three resolved fields are what
 * the backend read from GameCatalog for an occupied record; an empty record
 * keeps the backend's own empty values and gets nothing invented for it.
 */
export type EquippedSpellSlot = {
  rawMagicParamID: number;
  resourceKey: string;
  name: string;
  memorySlots: number;
};

/**
 * The twelve public memory slots in the backend's stored order, with the two
 * capacity counts it reports. The counts are carried as they arrive: neither is
 * recomputed from the records, and they are not expected to agree.
 */
export type CharacterEquippedSpells = {
  saveSessionID: string;
  saveRevision: string;
  characterID: number;
  active: boolean;
  spells: readonly EquippedSpellSlot[];
  usedMemorySlots: number;
  availableMemorySlots: number;
};

/**
 * One resource the backend accepts for the requested slot type. Compatibility,
 * visibility and ordering are the backend's answer: nothing here is filtered,
 * re-sorted or re-checked against a local rule.
 *
 * `ownedItemID` is present exactly for the slot types whose setter addresses an
 * owned record. `memorySlots` is the confirmed capacity cost of a spell and is
 * absent for every other slot type.
 */
export type EquipmentCandidate = {
  resource: LoadoutResource;
  ownedItemID?: string;
  name: string;
  iconPath: string;
  quantity?: number;
  memorySlots?: number;
  banRisk: boolean;
  cutContent: boolean;
};

/** One page of the candidates of one slot type, exactly as the backend served it. */
export type EquipmentCandidatesPage = {
  saveSessionID: string;
  saveRevision: string;
  characterID: number;
  active: boolean;
  safetyProfile: string;
  slotType: string;
  candidates: readonly EquipmentCandidate[];
  total: number;
  page: number;
  pageSize: number;
};

/**
 * The arguments of one candidate page. The slot type is one value of the closed
 * backend dictionary and is passed on exactly as received; no safety profile is
 * sent, because the backend reads the host setting itself.
 */
export type EquipmentCandidatesRequest = {
  saveSessionID: string;
  characterID: number;
  slotType: string;
  search: string;
  page: number;
  pageSize: number;
};

/**
 * The committed receipt of one Equipment mutation. It is the shared backend
 * receipt carried verbatim: no revision, scope list or operation identity is
 * ever assembled above this port.
 */
export type EquipmentMutationReceipt = {
  operationID: string;
  operationKind: string;
  saveSessionID: string;
  saveRevision: string;
  changedScopes: readonly ChangedScope[];
};

/**
 * The seven Equipment mutations. Every one of them replaces a complete group in
 * the backend's own order, so a caller always sends the whole group and never
 * only the position it touched. `null` is the backend's own empty position and
 * is never replaced by a placeholder here.
 */
export type EquipmentMutationPort = {
  /** Six hand positions in left 1, right 1, left 2, right 2, left 3, right 3 order. */
  setEquippedArmaments: (request: {
    saveSessionID: string;
    characterID: number;
    slotAssignments: readonly (string | null)[];
    expectedRevision: string;
  }) => Promise<EquipmentMutationReceipt>;
  /** Four armor positions in head, chest, arms, legs order. */
  setEquippedArmor: (request: {
    saveSessionID: string;
    characterID: number;
    slotAssignments: readonly (string | null)[];
    expectedRevision: string;
  }) => Promise<EquipmentMutationReceipt>;
  /** The compact talisman list; the unlocked-slot limit is the backend's. */
  setEquippedTalismans: (request: {
    saveSessionID: string;
    characterID: number;
    orderedOwnedItemIDs: readonly string[];
    expectedRevision: string;
  }) => Promise<EquipmentMutationReceipt>;
  /** The compact spell list; the Memory Slots capacity rule is the backend's. */
  setEquippedSpells: (request: {
    saveSessionID: string;
    characterID: number;
    orderedResources: readonly LoadoutResource[];
    expectedRevision: string;
  }) => Promise<EquipmentMutationReceipt>;
  /** Both Physick positions; clearing one never left-packs the other. */
  setPhysickMixture: (request: {
    saveSessionID: string;
    characterID: number;
    crystalTearResources: readonly (LoadoutResource | null)[];
    expectedRevision: string;
  }) => Promise<EquipmentMutationReceipt>;
  /** All six Pouch positions, empty ones included. */
  setPouchItems: (request: {
    saveSessionID: string;
    characterID: number;
    slotAssignments: readonly (string | null)[];
    expectedRevision: string;
  }) => Promise<EquipmentMutationReceipt>;
  /** All ten Quick Items positions, empty ones included. */
  setQuickItems: (request: {
    saveSessionID: string;
    characterID: number;
    slotAssignments: readonly (string | null)[];
    expectedRevision: string;
  }) => Promise<EquipmentMutationReceipt>;
};

export type EquipmentPort = EquipmentMutationPort & {
  /**
   * Reads one page of the resources the requested slot type accepts. Which slot
   * types exist, which resources each one accepts and how paging resolves are
   * the backend's contract.
   */
  getEquipmentCandidates: (request: EquipmentCandidatesRequest) => Promise<EquipmentCandidatesPage>;
  /** Reads the coherent, catalog-resolved loadout used by frontend screens. */
  getCharacterLoadout: (request: EquipmentRequest) => Promise<CharacterLoadout>;
  /** Reads the 22 raw equipment fields of one slot. */
  getEquipment: (request: EquipmentRequest) => Promise<CharacterEquipment>;
  /** Reads the ten raw quick-item records and the raw active-slot value. */
  getQuickItems: (request: EquipmentRequest) => Promise<CharacterQuickItems>;
  /** Reads the six raw pouch records. */
  getPouchItems: (request: EquipmentRequest) => Promise<CharacterPouchItems>;
  /** Reads both raw Crystal Tear identifiers of the Physick mixture. */
  getPhysickMixture: (request: EquipmentRequest) => Promise<CharacterPhysickMixture>;
  /** Reads the twelve memory slots the backend already resolved. */
  getEquippedSpells: (request: EquipmentRequest) => Promise<CharacterEquippedSpells>;
};
