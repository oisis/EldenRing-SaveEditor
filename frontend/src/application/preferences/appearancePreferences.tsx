import {
  createContext,
  type ReactNode,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";

/**
 * Host preferences for appearance presets: which presets the user marked as local
 * favorites.
 *
 * This is pure client/host interface state, persisted in localStorage.
 * It is completely distinct from Mirror Favorites stored in the save file.
 * Presets are identified strictly by their stable string id.
 */
export type AppearancePreferences = {
  favorites: readonly string[];
  isFavorite: (id: string) => boolean;
  toggleFavorite: (id: string) => void;
};

const appearanceFavoritesStorageKey = "saveforge.appearance.favorites";

const AppearancePreferencesContext = createContext<AppearancePreferences | null>(null);

function readStoredFavorites(): string[] {
  try {
    const raw = window.localStorage.getItem(appearanceFavoritesStorageKey);
    if (raw === null) return [];
    const parsed: unknown = JSON.parse(raw);
    if (!Array.isArray(parsed)) return [];
    return parsed.filter((id): id is string => typeof id === "string" && id.length > 0);
  } catch {
    return [];
  }
}

function persist(storageKey: string, value: string) {
  try {
    window.localStorage.setItem(storageKey, value);
  } catch {
    // Storage unavailable or disabled
  }
}

export function AppearancePreferencesProvider({
  children,
  initialFavorites,
}: {
  children: ReactNode;
  initialFavorites?: readonly string[];
}) {
  const [favorites, setFavorites] = useState<readonly string[]>(
    () => initialFavorites ?? readStoredFavorites(),
  );
  const isInitialMount = useRef(true);

  useEffect(() => {
    if (isInitialMount.current) {
      isInitialMount.current = false;
      return;
    }
    persist(appearanceFavoritesStorageKey, JSON.stringify(favorites));
  }, [favorites]);

  const favoriteSet = useMemo(() => new Set(favorites), [favorites]);

  const toggleFavorite = useCallback((id: string) => {
    setFavorites((current) =>
      current.includes(id) ? current.filter((entry) => entry !== id) : [...current, id],
    );
  }, []);

  const value = useMemo<AppearancePreferences>(
    () => ({
      favorites,
      isFavorite: (id) => favoriteSet.has(id),
      toggleFavorite,
    }),
    [favorites, favoriteSet, toggleFavorite],
  );

  return (
    <AppearancePreferencesContext value={value}>
      {children}
    </AppearancePreferencesContext>
  );
}

/**
 * Hook to access host appearance preferences.
 * Throws a configuration error when used outside of an AppearancePreferencesProvider.
 */
export function useAppearancePreferences(): AppearancePreferences {
  const context = useContext(AppearancePreferencesContext);
  if (context === null) {
    throw new Error("AppearancePreferencesProvider is missing above this component");
  }
  return context;
}
