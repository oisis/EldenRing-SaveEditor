/**
 * The single source of truth for TanStack Query keys. Components and feature
 * modules never build a key inline: invalidation scopes are mapped onto these
 * keys in one place.
 */
export const queryKeys = {
  applicationInfo: () => ["application", "info"] as const,
} as const;
