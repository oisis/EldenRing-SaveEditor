import { skipToken, useQuery } from "@tanstack/react-query";
import { noCharacter, queryKeys, type WorldView } from "../queryKeys";
import { requireCurrentSaveResponse, type SaveSnapshotIdentity } from "../saveRevision";
import { useWorldPort } from "./worldClient";
import type {
  WorldBellBearings,
  WorldBosses,
  WorldColosseums,
  WorldCookbooks,
  WorldGestures,
  WorldGraces,
  WorldMapRegions,
  WorldQuests,
  WorldRegions,
  WorldRequest,
  WorldSpectralSteedAttires,
  WorldSummoningPools,
  WorldTutorials,
  WorldWhetblades,
} from "./worldPort";

/**
 * A World view as a screen asks for it: the session, the revision and the slot
 * may still be unknown while nothing has been selected.
 */
export type WorldQuery = {
  saveSessionID: string | undefined;
  saveRevision: string | undefined;
  characterID: number | undefined;
};

/**
 * The thirteen getters share one lifecycle and one argument pair, so they share
 * one query builder. Only the view segment of the key and the port call differ,
 * and each getter keeps its own cache entry below the shared `world` branch.
 *
 * Without a session, a revision or a slot there is nothing to ask for, so no
 * query function is built at all. `skipToken` rather than `enabled` is what
 * guards the port: a manual `refetch()` runs the query function even while the
 * query is disabled, so a missing identifier has to make the call impossible,
 * not merely inactive. No `placeholderData` and no previous data is configured:
 * a view of another session, character or revision must never stand in for the
 * one being loaded, so switching any of the three shows a pending state instead
 * of a stale answer. The query is not retried: a failing desktop bridge call
 * does not become healthy by repeating it.
 */
function useWorldQuery<T extends SaveSnapshotIdentity>(
  query: WorldQuery,
  view: WorldView,
  call: (request: WorldRequest) => Promise<T>,
) {
  const { saveSessionID, saveRevision, characterID } = query;
  const identifier = saveSessionID ?? "";

  return useQuery({
    queryKey: queryKeys.worldView(
      identifier,
      characterID ?? noCharacter,
      view,
      saveRevision ?? "",
    ),
    queryFn:
      identifier === "" || saveRevision === undefined || characterID === undefined
        ? skipToken
        : async () =>
            requireCurrentSaveResponse(
              await call({ saveSessionID: identifier, characterID }),
              identifier,
              saveRevision,
            ),
    retry: false,
  });
}

export function useWorldRegions(query: WorldQuery) {
  const port = useWorldPort();
  return useWorldQuery<WorldRegions>(query, "regions", (request) => port.getRegions(request));
}

export function useWorldMapRegions(query: WorldQuery) {
  const port = useWorldPort();
  return useWorldQuery<WorldMapRegions>(query, "map-regions", (request) =>
    port.getMapRegions(request),
  );
}

export function useWorldGraces(query: WorldQuery) {
  const port = useWorldPort();
  return useWorldQuery<WorldGraces>(query, "graces", (request) => port.getGraces(request));
}

export function useWorldBosses(query: WorldQuery) {
  const port = useWorldPort();
  return useWorldQuery<WorldBosses>(query, "bosses", (request) => port.getBosses(request));
}

export function useWorldQuests(query: WorldQuery) {
  const port = useWorldPort();
  return useWorldQuery<WorldQuests>(query, "quests", (request) => port.getQuests(request));
}

export function useWorldGestures(query: WorldQuery) {
  const port = useWorldPort();
  return useWorldQuery<WorldGestures>(query, "gestures", (request) => port.getGestures(request));
}

export function useWorldCookbooks(query: WorldQuery) {
  const port = useWorldPort();
  return useWorldQuery<WorldCookbooks>(query, "cookbooks", (request) => port.getCookbooks(request));
}

export function useWorldBellBearings(query: WorldQuery) {
  const port = useWorldPort();
  return useWorldQuery<WorldBellBearings>(query, "bell-bearings", (request) =>
    port.getBellBearings(request),
  );
}

export function useWorldWhetblades(query: WorldQuery) {
  const port = useWorldPort();
  return useWorldQuery<WorldWhetblades>(query, "whetblades", (request) =>
    port.getWhetblades(request),
  );
}

export function useWorldTutorials(query: WorldQuery) {
  const port = useWorldPort();
  return useWorldQuery<WorldTutorials>(query, "tutorials", (request) => port.getTutorials(request));
}

export function useWorldSummoningPools(query: WorldQuery) {
  const port = useWorldPort();
  return useWorldQuery<WorldSummoningPools>(query, "summoning-pools", (request) =>
    port.getSummoningPools(request),
  );
}

export function useWorldColosseums(query: WorldQuery) {
  const port = useWorldPort();
  return useWorldQuery<WorldColosseums>(query, "colosseums", (request) =>
    port.getColosseums(request),
  );
}

export function useWorldSpectralSteedAttires(query: WorldQuery) {
  const port = useWorldPort();
  return useWorldQuery<WorldSpectralSteedAttires>(query, "spectral-steed-attires", (request) =>
    port.getSpectralSteedAttires(request),
  );
}
