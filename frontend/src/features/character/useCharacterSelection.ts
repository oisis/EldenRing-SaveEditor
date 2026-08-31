import { useMemo, useState } from "react";
import type { CharacterSummary } from "../../application/character/characterPort";
import { useCharacterProfile } from "../../application/character/useCharacterProfile";
import { useCharacterStats } from "../../application/character/useCharacterStats";
import { useSaveCharacters } from "../../application/character/useSaveCharacters";

/**
 * One entry into a session, holding the presentational intent made inside it.
 * Entering another session — including returning to an identifier used before —
 * replaces the entry, so a historical selection is never resumed. The entry is
 * read through the same session comparison it is written with, so the effective
 * selection stays derived and no render can show the previous session's slot.
 */
type SessionEntry = {
  saveSessionID: string | undefined;
  characterID: number | undefined;
};

export type CharacterSelection = {
  /** Whether a session identifier is present at all; no session means no query. */
  hasSession: boolean;
  characters: ReturnType<typeof useSaveCharacters>;
  /** Backend order, preserved; active slots only. */
  activeCharacters: readonly CharacterSummary[];
  /** Backend order, preserved; everything the backend did not report as active. */
  inactiveCharacters: readonly CharacterSummary[];
  selectedCharacterID: number | undefined;
  selectCharacter: (characterID: number) => void;
  profile: ReturnType<typeof useCharacterProfile>;
  stats: ReturnType<typeof useCharacterStats>;
};

/**
 * The character selection of one save session. It owns nothing but the user's
 * presentational intent for the current session entry: the slot list, the
 * profile and the statistics stay in the query cache and are never copied into
 * a second store.
 *
 * The effective selection is derived, not stored:
 *
 *   1. a manual selection wins while it belongs to the current session entry
 *      and the chosen slot is still reported as active;
 *   2. otherwise slot 0 is selected when it is active;
 *   3. otherwise the first active slot in backend order is selected;
 *   4. with no active slot at all nothing is selected.
 *
 * Rule 1 is also the only guard needed for "the user can only select an active
 * character": an identifier that is not in the active set is simply never
 * effective, whether it was inactive from the start or became inactive later.
 * The slot range itself stays the backend's contract and is not validated here.
 */
export function useCharacterSelection(
  saveSessionID: string | undefined,
  saveRevision: string | undefined,
): CharacterSelection {
  const characters = useSaveCharacters(saveSessionID, saveRevision);
  const [entry, setEntry] = useState<SessionEntry>({ saveSessionID, characterID: undefined });

  /**
   * Every real change of the identifier is a new entry into a session, so the
   * intent of the previous one is dropped. The state is adjusted during render
   * rather than in an effect: an effect would let one render pass through with
   * the previous entry still in place.
   */
  if (entry.saveSessionID !== saveSessionID) {
    setEntry({ saveSessionID, characterID: undefined });
  }

  const reported = characters.data?.characters;
  const activeCharacters = useMemo(
    () => (reported ?? []).filter((character) => character.active),
    [reported],
  );
  const inactiveCharacters = useMemo(
    () => (reported ?? []).filter((character) => !character.active),
    [reported],
  );

  // Read through the same comparison the reset is written with, so even the
  // render that triggers the reset already sees no intent for this session.
  const intent = entry.saveSessionID === saveSessionID ? entry.characterID : undefined;

  const intended =
    intent !== undefined && activeCharacters.some((character) => character.characterID === intent)
      ? intent
      : undefined;

  const isSlotZeroActive = activeCharacters.some((character) => character.characterID === 0);
  const fallback = isSlotZeroActive ? 0 : activeCharacters[0]?.characterID;

  // `??` and not `||`: slot 0 is an ordinary slot, never an absent selection.
  const selectedCharacterID = intended ?? fallback;

  // The two per-character queries follow the effective selection. With nothing
  // selected they receive `undefined` and their `skipToken` guard keeps the
  // port out of reach entirely.
  const profile = useCharacterProfile(saveSessionID, saveRevision, selectedCharacterID);
  const stats = useCharacterStats(saveSessionID, saveRevision, selectedCharacterID);

  return {
    hasSession: (saveSessionID ?? "") !== "",
    characters,
    activeCharacters,
    inactiveCharacters,
    selectedCharacterID,
    selectCharacter: (characterID) => setEntry({ saveSessionID, characterID }),
    profile,
    stats,
  };
}
