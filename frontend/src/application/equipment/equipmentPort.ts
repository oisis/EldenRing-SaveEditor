/**
 * The port the application layer needs in order to read the equipped state of
 * one character slot. Infrastructure implements it; feature modules depend on
 * it through the hooks in this directory and never on the transport that
 * fulfils it.
 *
 * All five getters are read-only: the port declares no `set*` method, so no
 * layer above it can reach a mutation through this contract.
 *
 * Four of them report raw save state. A raw identifier is carried exactly as
 * the backend reports it and nothing is derived from it here: no name, no icon,
 * no slot label, no item kind, no compatibility, no locked-slot rule and no
 * sentinel interpretation. `0`, `0xFFFFFFFF` and every unknown value stay
 * themselves.
 *
 * `getEquippedSpells` is the only one whose result the backend already resolved
 * through GameCatalog, so `resourceKey`, `name` and `memorySlots` are the only
 * presentation this port carries. They are the backend's answer, not a local
 * lookup.
 *
 * None of the five returns a `saveRevision`: it is absent from the backend
 * contract of this stage and is therefore not invented here.
 */

/** The session and slot one equipped view is read for. */
export type EquipmentRequest = {
  saveSessionID: string;
  characterID: number;
};

/**
 * The 22 raw ChrAsmEquipment fields of one slot, in the backend's stored order.
 * The indexes are deliberately unnamed: which physical field each one is, and
 * which of them are still unknown, is not part of this stage. No entry is
 * dropped, reordered or replaced.
 */
export type CharacterEquipment = {
  saveSessionID: string;
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
  characterID: number;
  active: boolean;
  items: readonly EquipItemRecord[];
  activeQuick: number;
};

/** The six raw pouch records, in the backend's stored order. */
export type CharacterPouchItems = {
  saveSessionID: string;
  characterID: number;
  active: boolean;
  items: readonly EquipItemRecord[];
};

/** Both raw Crystal Tear identifiers of the current Physick mixture. */
export type CharacterPhysickMixture = {
  saveSessionID: string;
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
  characterID: number;
  active: boolean;
  spells: readonly EquippedSpellSlot[];
  usedMemorySlots: number;
  availableMemorySlots: number;
};

export type EquipmentPort = {
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
