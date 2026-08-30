/**
 * The port the application layer needs in order to read the characters of a
 * save session. Infrastructure implements it; feature modules depend on it
 * through the hooks in this directory and never on the transport that fulfils
 * it.
 *
 * Every type here is a projection of the backend character contract as it
 * exists today, and it carries exactly the fields the backend reports. Nothing
 * is added: no revision, no starting class name, no gender label, no slot
 * status beyond `active`, and no derived value. A raw identifier stays a raw
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

export type SaveCharacters = {
  saveSessionID: string;
  characters: readonly CharacterSummary[];
};

export type CharacterProfile = {
  saveSessionID: string;
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
};
