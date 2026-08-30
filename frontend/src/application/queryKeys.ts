import type { CatalogResourcesRequest } from "./catalog/catalogPort";

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
   * The catalog is global: it belongs to no save session, so its keys live
   * outside the `save-session` prefix and closing a save leaves them cached.
   * All seven backend arguments take part in the key, because every one of them
   * selects a different page of the catalog.
   */
  catalogResources: ({
    resourceType,
    family,
    capability,
    endpointID,
    search,
    page,
    pageSize,
  }: CatalogResourcesRequest) =>
    [
      "catalog",
      "resources",
      resourceType,
      family,
      capability,
      endpointID,
      search,
      page,
      pageSize,
    ] as const,
  /**
   * One resource detail is keyed by its exact identity, so two kinds and two
   * keys never share an entry. `null` stands for "nothing selected" and cannot
   * collide with any backend value, because every kind and key is a string —
   * the empty string included, which is a real request the backend rejects.
   */
  catalogResource: (kind: string | null, key: string | null) =>
    ["catalog", "resource", kind, key] as const,
  /**
   * The variants of one item are a separate branch of the same global catalog
   * prefix, so they never share an entry with the resource detail of the same
   * identity. `null` stands for "nothing selected" and cannot collide with any
   * backend value: the empty string is a real request the backend rejects.
   */
  catalogItemVariants: (kind: string | null, key: string | null) =>
    ["catalog", "item-variants", kind, key] as const,
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
  /**
   * A container page is identified by its session, slot, container, section and
   * page window: two sections, two pages or two page sizes are different views
   * of the same slot and must not share a cache entry. Inventory and Storage
   * are separate containers of the same slot, so the container name is part of
   * the key rather than an argument of one shared key.
   */
  inventory: (
    saveSessionID: string,
    characterID: CharacterKey,
    containerSection: string,
    page: number,
    pageSize: number,
  ) =>
    [
      "save-session",
      saveSessionID,
      "character",
      characterID,
      "inventory",
      containerSection,
      page,
      pageSize,
    ] as const,
  storage: (
    saveSessionID: string,
    characterID: CharacterKey,
    containerSection: string,
    page: number,
    pageSize: number,
  ) =>
    [
      "save-session",
      saveSessionID,
      "character",
      characterID,
      "storage",
      containerSection,
      page,
      pageSize,
    ] as const,
} as const;
