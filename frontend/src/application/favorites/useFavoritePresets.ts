import { skipToken, useQuery } from "@tanstack/react-query";
import { queryKeys } from "../queryKeys";
import { useFavoritesPort } from "./favoritesClient";

export function useFavoritePresets(
  saveSessionID: string | undefined,
  saveRevision: string | undefined,
  favoriteSlotID?: number,
) {
  const port = useFavoritesPort();

  return useQuery({
    queryKey:
      saveSessionID !== undefined && saveRevision !== undefined
        ? queryKeys.favoritePresets(saveSessionID, saveRevision, favoriteSlotID)
        : ["save-session", "none", "favorite-presets"],
    queryFn:
      saveSessionID !== undefined && saveRevision !== undefined
        ? () => port.getFavoritePresets(saveSessionID, favoriteSlotID)
        : skipToken,
    retry: false,
  });
}
