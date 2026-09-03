import { createContext, type ReactNode, useCallback, useContext, useMemo, useState } from "react";
import type { ResourceIdentity } from "../items/itemsPort";

/**
 * The presentational item preferences of this host: which catalog resources the
 * user marked as favourites, and whether item identifiers are shown.
 *
 * Both are interface state and nothing else. A favourite is identified by the
 * canonical `(kind, key)` pair only — no name, no icon, no limit, no safety
 * flag and no owned record is stored with it — so the preference can never
 * become a stale copy of a GameCatalog document or of save data. `Show Item ID`
 * changes what is rendered and never what is sent to a save mutation.
 *
 * These favourites are unrelated to the `favorites` endpoints, which own the
 * Mirror Favorites of character appearance.
 *
 * The values are persisted per host in `localStorage`. That is deliberate and
 * allowed: it stores the user's own preference, not save data and not a catalog
 * response.
 */
export type ItemPreferences = {
  /** The favourites, in insertion order, as the backend filter expects them. */
  favorites: readonly ResourceIdentity[];
  isFavorite: (identity: ResourceIdentity) => boolean;
  toggleFavorite: (identity: ResourceIdentity) => void;
  showItemID: boolean;
  setShowItemID: (value: boolean) => void;
};

const favoritesStorageKey = "saveforge.items.favorites";
const showItemIDStorageKey = "saveforge.items.showItemID";

const ItemPreferencesContext = createContext<ItemPreferences | null>(null);

/**
 * The comparison token of one favourite. The separator is a NUL character,
 * which cannot occur in a backend kind or key, so two different pairs can never
 * collapse into one entry.
 */
function identityToken({ kind, key }: ResourceIdentity): string {
  return `${kind}\u0000${key}`;
}

function readStoredFavorites(): ResourceIdentity[] {
  try {
    const raw = window.localStorage.getItem(favoritesStorageKey);
    if (raw === null) return [];
    const parsed: unknown = JSON.parse(raw);
    if (!Array.isArray(parsed)) return [];
    // Anything that is not an exact pair of strings is dropped rather than
    // repaired: a preference document is not a contract the interface can trust.
    return parsed.flatMap((entry) =>
      typeof entry === "object" &&
      entry !== null &&
      typeof (entry as ResourceIdentity).kind === "string" &&
      typeof (entry as ResourceIdentity).key === "string"
        ? [{ kind: (entry as ResourceIdentity).kind, key: (entry as ResourceIdentity).key }]
        : [],
    );
  } catch {
    // Unreadable or disabled storage is not an error the user has to act on;
    // the session simply starts with no favourites.
    return [];
  }
}

function readStoredShowItemID(): boolean {
  try {
    return window.localStorage.getItem(showItemIDStorageKey) === "true";
  } catch {
    return false;
  }
}

function persist(storageKey: string, value: string) {
  try {
    window.localStorage.setItem(storageKey, value);
  } catch {
    // A host that refuses storage still gets working preferences for this
    // session; nothing above depends on the write having happened.
  }
}

/**
 * `initialFavorites` and `initialShowItemID` let the composition root state the
 * starting values explicitly instead of reading the host store. Nothing else
 * overrides the stored preferences, and both stay ordinary React state
 * afterwards.
 */
export function ItemPreferencesProvider({
  children,
  initialFavorites,
  initialShowItemID,
}: {
  children: ReactNode;
  initialFavorites?: readonly ResourceIdentity[];
  initialShowItemID?: boolean;
}) {
  const [favorites, setFavorites] = useState<readonly ResourceIdentity[]>(
    () => initialFavorites ?? readStoredFavorites(),
  );
  const [showItemID, setShowItemIDState] = useState<boolean>(
    () => initialShowItemID ?? readStoredShowItemID(),
  );

  const tokens = useMemo(() => new Set(favorites.map(identityToken)), [favorites]);

  const toggleFavorite = useCallback((identity: ResourceIdentity) => {
    setFavorites((current) => {
      const token = identityToken(identity);
      const next = current.some((entry) => identityToken(entry) === token)
        ? current.filter((entry) => identityToken(entry) !== token)
        : [...current, { kind: identity.kind, key: identity.key }];
      persist(favoritesStorageKey, JSON.stringify(next));
      return next;
    });
  }, []);

  const setShowItemID = useCallback((value: boolean) => {
    setShowItemIDState(value);
    persist(showItemIDStorageKey, String(value));
  }, []);

  const value = useMemo<ItemPreferences>(
    () => ({
      favorites,
      isFavorite: (identity) => tokens.has(identityToken(identity)),
      toggleFavorite,
      showItemID,
      setShowItemID,
    }),
    [favorites, tokens, toggleFavorite, showItemID, setShowItemID],
  );

  return <ItemPreferencesContext value={value}>{children}</ItemPreferencesContext>;
}

export function useItemPreferences(): ItemPreferences {
  const preferences = useContext(ItemPreferencesContext);
  if (preferences === null) {
    // A wiring mistake, never a user-facing condition.
    throw new Error("ItemPreferencesProvider is missing above this component");
  }
  return preferences;
}
