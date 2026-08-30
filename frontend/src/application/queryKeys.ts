/**
 * The placeholder a per-character query is keyed under while no character is
 * selected. Such a query never runs, so it must not share a key with any real
 * slot index; a string can never collide with the numeric identifiers the
 * backend uses, including negative ones.
 */
export const noCharacter = "none";

/** A slot index as reported by the backend, or the unselected placeholder. */
type CharacterKey = number | typeof noCharacter;

/**
 * The single source of truth for TanStack Query keys. Components and feature
 * modules never build a key inline: invalidation scopes are mapped onto these
 * keys in one place.
 */
export const queryKeys = {
  applicationInfo: () => ["application", "info"] as const,
  /**
   * The prefix covering every cached view of one save session. Closing a
   * session removes this whole scope, so a later per-session query only has to
   * be keyed below it to be cleaned up with the session.
   */
  saveSession: (saveSessionID: string) => ["save-session", saveSessionID] as const,
  loadedSave: (saveSessionID: string) => ["save-session", saveSessionID, "loaded"] as const,
  saveCharacters: (saveSessionID: string) => ["save-session", saveSessionID, "characters"] as const,
  /**
   * The per-character views are keyed by session and slot, so two characters of
   * one session and one character of two sessions stay separate entries.
   */
  characterProfile: (saveSessionID: string, characterID: CharacterKey) =>
    ["save-session", saveSessionID, "character", characterID, "profile"] as const,
  characterStats: (saveSessionID: string, characterID: CharacterKey) =>
    ["save-session", saveSessionID, "character", characterID, "stats"] as const,
} as const;
