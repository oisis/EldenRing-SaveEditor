import type {
  CatalogResourcePresentationIdentity,
  CatalogResourcesRequest,
} from "./catalog/catalogPort";

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
   * Presentation batches preserve order and duplicates, so every ordered pair
   * participates in the key. `null` is the unselected state; an empty array is
   * a real backend request and therefore a distinct key.
   */
  catalogResourcePresentationSummaries: (
    identities: readonly CatalogResourcePresentationIdentity[] | null,
  ) =>
    [
      "catalog",
      "resource-presentation-summaries",
      identities?.map(({ kind, key }) => [kind, key] as const) ?? null,
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
   * One validation report per slot and per revision, below the session scope,
   * so closing the session drops every report of every revision with it and two
   * slots never share an entry. The revision takes part in the key because a
   * report describes one exact save state: asking about another revision is a
   * different question, and the answer to the previous one may never stand in
   * for it. The revision is carried as the backend's own string and is never
   * parsed, ordered or compared numerically here.
   */
  saveValidationReport: (saveSessionID: string, characterID: CharacterKey, saveRevision: string) =>
    [
      "save-session",
      saveSessionID,
      "character",
      characterID,
      "validation-report",
      saveRevision,
    ] as const,
  /**
   * The per-character views are keyed by session and slot, so two characters of
   * one session and one character of two sessions stay separate entries.
   */
  characterProfile: (saveSessionID: string, characterID: CharacterKey) =>
    ["save-session", saveSessionID, "character", characterID, "profile"] as const,
  characterStats: (saveSessionID: string, characterID: CharacterKey) =>
    ["save-session", saveSessionID, "character", characterID, "stats"] as const,
  /**
   * The coherent loadout and five narrow equipped views are independent backend getters,
   * so each one keeps its own key below the same session and slot scope. They
   * are never merged into one key: a failure or a refetch of one must not
   * invalidate, replace or hide the other five.
   */
  equipment: (saveSessionID: string, characterID: CharacterKey) =>
    ["save-session", saveSessionID, "character", characterID, "equipment"] as const,
  characterLoadout: (saveSessionID: string, characterID: CharacterKey) =>
    ["save-session", saveSessionID, "character", characterID, "loadout"] as const,
  quickItems: (saveSessionID: string, characterID: CharacterKey) =>
    ["save-session", saveSessionID, "character", characterID, "quick-items"] as const,
  pouchItems: (saveSessionID: string, characterID: CharacterKey) =>
    ["save-session", saveSessionID, "character", characterID, "pouch-items"] as const,
  physickMixture: (saveSessionID: string, characterID: CharacterKey) =>
    ["save-session", saveSessionID, "character", characterID, "physick-mixture"] as const,
  equippedSpells: (saveSessionID: string, characterID: CharacterKey) =>
    ["save-session", saveSessionID, "character", characterID, "equipped-spells"] as const,
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
