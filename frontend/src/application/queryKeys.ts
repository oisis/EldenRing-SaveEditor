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
} as const;
