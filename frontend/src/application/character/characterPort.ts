import type { MutationReceipt } from "../save-session/saveSessionPort";

/**
 * The port the application layer needs in order to read the characters of a
 * save session. Infrastructure implements it; feature modules depend on it
 * through the hooks in this directory and never on the transport that fulfils
 * it.
 *
 * Every type here is a projection of the backend character contract as it
 * exists today, and it carries exactly the fields the backend reports. Nothing
 * is added: no starting class name, no gender label, no slot state the backend
 * did not classify, and no derived value. A raw identifier stays a raw
 * identifier until a backend getter names it.
 */

/** One physical slot as reported by GetSaveCharacters, in slot order. */
export type CharacterSummary = {
  /** Backend slot index, carried verbatim; the UI does not interpret it. */
  characterID: number;
  active: boolean;
  name: string;
  level: number;
};

/**
 * The slot states of the slot-management projection. `unknown` is the fail-safe
 * state: the backend could not classify the slot, so it is neither empty nor a
 * target of any operation.
 */
export type CharacterSlotState = "active" | "residual" | "empty" | "unknown";

/**
 * What the backend allows for one slot. It is a presentation hint only: every
 * writer revalidates the slot and the expected revision, so a stale capability
 * cannot become an accepted mutation. `delete` covers deleting an active
 * character and clearing a residual slot alike, because one writer does both.
 */
export type CharacterSlotCapabilities = {
  activate: boolean;
  deactivate: boolean;
  cloneFrom: boolean;
  cloneInto: boolean;
  delete: boolean;
};

/**
 * One physical slot as the slot management sees it. `startingClassID` is
 * meaningful only while `startingClassKnown` is true; the backend reports it
 * for an active slot alone and never invents a default class for a slot whose
 * profile summary was cleared.
 */
export type CharacterSlot = {
  characterID: number;
  state: CharacterSlotState;
  startingClassID: number;
  startingClassKnown: boolean;
  capabilities: CharacterSlotCapabilities;
};

export type SaveCharacters = {
  saveSessionID: string;
  saveRevision: string;
  characters: readonly CharacterSummary[];
  slots: readonly CharacterSlot[];
};

export type CharacterProfile = {
  saveSessionID: string;
  saveRevision: string;
  characterID: number;
  active: boolean;
  name: string;
  level: number;
  /** Raw backend identifier; no name is resolved for it here. */
  startingClassID: number;
  /** Raw backend identifier; no label is resolved for it here. */
  gender: number;
  /** Raw seconds as stored in the save; formatting belongs to a later step. */
  secondsPlayed: number;
};

/**
 * The raw statistics of one slot. Current, maximum and base values are all
 * reported by the backend; none of them is computed or reconciled here.
 */
export type CharacterStats = {
  saveSessionID: string;
  saveRevision: string;
  characterID: number;
  active: boolean;

  vigor: number;
  mind: number;
  endurance: number;
  strength: number;
  dexterity: number;
  intelligence: number;
  faith: number;
  arcane: number;
  level: number;

  hp: number;
  maxHP: number;
  baseMaxHP: number;
  fp: number;
  maxFP: number;
  baseMaxFP: number;
  sp: number;
  maxSP: number;
  baseMaxSP: number;

  /** Held runes, the only editable value of this group. */
  runes: number;
  /** Lifetime runes (TotalGetSoul). The backend exposes no writer for it. */
  soulMemory: number;
};

export type CharacterAttributes = {
  vigor: number;
  mind: number;
  endurance: number;
  strength: number;
  dexterity: number;
  intelligence: number;
  faith: number;
  arcane: number;
};

export type SetCharacterNameInput = {
  saveSessionID: string;
  characterID: number;
  name: string;
  expectedRevision: string;
};

export type SetCharacterStatsInput = {
  saveSessionID: string;
  characterID: number;
  attributes: CharacterAttributes;
  levelPolicy: "recalculate";
  expectedRevision: string;
};

export type SetCharacterStartingClassInput = {
  saveSessionID: string;
  characterID: number;
  startingClassID: number;
  confirmReset: boolean;
  expectedRevision: string;
};

export type SetCharacterRunesInput = {
  saveSessionID: string;
  characterID: number;
  runes: number;
  expectedRevision: string;
};

export type SetCharacterGenderInput = {
  saveSessionID: string;
  characterID: number;
  gender: number;
  expectedRevision: string;
};

export type SetCharacterActiveInput = {
  saveSessionID: string;
  characterID: number;
  active: boolean;
  expectedRevision: string;
};

/**
 * The two success variants of SetCharacterActive. An idempotent request commits
 * nothing, so it carries no receipt at all: no history entry, no new revision
 * and nothing to apply.
 */
export type SetCharacterActiveResult =
  | { changed: true; receipt: MutationReceipt }
  | { changed: false };

export type CloneCharacterInput = {
  saveSessionID: string;
  sourceCharacterID: number;
  targetSlotID: number;
  expectedRevision: string;
};

export type DeleteCharacterInput = {
  saveSessionID: string;
  characterID: number;
  expectedRevision: string;
};

export type CharacterPort = {
  /**
   * Reads every slot of a session. Both this and the per-character getters pass
   * their arguments to the backend exactly as received: the backend owns
   * session resolution and the slot range.
   */
  getSaveCharacters: (saveSessionID: string) => Promise<SaveCharacters>;
  getCharacterProfile: (saveSessionID: string, characterID: number) => Promise<CharacterProfile>;
  getCharacterStats: (saveSessionID: string, characterID: number) => Promise<CharacterStats>;
  setCharacterName: (input: SetCharacterNameInput) => Promise<MutationReceipt>;
  setCharacterStats: (input: SetCharacterStatsInput) => Promise<MutationReceipt>;
  setCharacterStartingClass: (input: SetCharacterStartingClassInput) => Promise<MutationReceipt>;
  setCharacterGender: (input: SetCharacterGenderInput) => Promise<MutationReceipt>;
  setCharacterRunes: (input: SetCharacterRunesInput) => Promise<MutationReceipt>;
  setCharacterActive: (input: SetCharacterActiveInput) => Promise<SetCharacterActiveResult>;
  cloneCharacter: (input: CloneCharacterInput) => Promise<MutationReceipt>;
  /** Deletes an active character or clears a residual slot; one backend writer. */
  deleteCharacter: (input: DeleteCharacterInput) => Promise<MutationReceipt>;
};
