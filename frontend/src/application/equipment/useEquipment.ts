import { skipToken, useQuery } from "@tanstack/react-query";
import { noCharacter, queryKeys } from "../queryKeys";
import { requireCurrentSaveResponse, type SaveSnapshotIdentity } from "../saveRevision";
import { useEquipmentPort } from "./equipmentClient";
import type {
  CharacterEquipment,
  CharacterEquippedSpells,
  CharacterLoadout,
  CharacterPhysickMixture,
  CharacterPouchItems,
  CharacterQuickItems,
  EquipmentCandidatesPage,
  EquipmentCandidatesRequest,
  EquipmentRequest,
} from "./equipmentPort";

/**
 * An equipped view as a view asks for it: the session and the slot may still be
 * unknown while nothing has been selected.
 */
export type EquipmentQuery = {
  saveSessionID: string | undefined;
  saveRevision: string | undefined;
  characterID: number | undefined;
};

/**
 * The five getters share one lifecycle and one argument pair, so they share one
 * query builder. Only the key and the port call differ, and each getter keeps
 * its own cache entry.
 *
 * Without a session or a slot there is nothing to ask for, so no query function
 * is built at all. `skipToken` rather than `enabled` is what guards the port: a
 * manual `refetch()` runs the query function even while the query is disabled,
 * so a missing identifier has to make the call impossible, not merely inactive.
 * The identifier and the slot are forwarded unchecked; which session exists and
 * which slot is in range is the backend's contract. The query is not retried: a
 * failing desktop bridge call does not become healthy by repeating it, and the
 * failure stays the query's own state rather than becoming a local fallback.
 */
function useEquipmentQuery<T extends SaveSnapshotIdentity>(
  query: EquipmentQuery,
  queryKey: readonly unknown[],
  call: (request: EquipmentRequest) => Promise<T>,
) {
  const { saveSessionID, saveRevision, characterID } = query;
  const identifier = saveSessionID ?? "";

  return useQuery({
    queryKey,
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

/** The key arguments every equipped view is scoped by. */
function scope(query: EquipmentQuery): [string, number | typeof noCharacter, string] {
  return [query.saveSessionID ?? "", query.characterID ?? noCharacter, query.saveRevision ?? ""];
}

export function useEquipment(query: EquipmentQuery) {
  const port = useEquipmentPort();

  return useEquipmentQuery<CharacterEquipment>(
    query,
    queryKeys.equipment(...scope(query)),
    (request) => port.getEquipment(request),
  );
}

export function useCharacterLoadout(query: EquipmentQuery) {
  const port = useEquipmentPort();

  return useEquipmentQuery<CharacterLoadout>(
    query,
    queryKeys.characterLoadout(...scope(query)),
    (request) => port.getCharacterLoadout(request),
  );
}

export function useQuickItems(query: EquipmentQuery) {
  const port = useEquipmentPort();

  return useEquipmentQuery<CharacterQuickItems>(
    query,
    queryKeys.quickItems(...scope(query)),
    (request) => port.getQuickItems(request),
  );
}

export function usePouchItems(query: EquipmentQuery) {
  const port = useEquipmentPort();

  return useEquipmentQuery<CharacterPouchItems>(
    query,
    queryKeys.pouchItems(...scope(query)),
    (request) => port.getPouchItems(request),
  );
}

export function usePhysickMixture(query: EquipmentQuery) {
  const port = useEquipmentPort();

  return useEquipmentQuery<CharacterPhysickMixture>(
    query,
    queryKeys.physickMixture(...scope(query)),
    (request) => port.getPhysickMixture(request),
  );
}

export function useEquippedSpells(query: EquipmentQuery) {
  const port = useEquipmentPort();

  return useEquipmentQuery<CharacterEquippedSpells>(
    query,
    queryKeys.equippedSpells(...scope(query)),
    (request) => port.getEquippedSpells(request),
  );
}

/**
 * One picker page as a view asks for it: the session, the revision and the slot
 * may still be unknown, and the slot type is `undefined` while no picker is
 * open. The slot type is one value of the closed backend dictionary and is
 * forwarded unchecked.
 */
export type EquipmentCandidatesQuery = EquipmentQuery & {
  slotType: string | undefined;
  search: string;
  page: number;
  pageSize: number;
};

/**
 * The candidates of one slot type.
 *
 * Every argument that selects a different answer takes part in the key, so two
 * slot types, two searches and two pages never share an entry. Without a
 * session, a revision, a slot or an open picker there is nothing to ask for, so
 * `skipToken` keeps the port out of reach entirely rather than merely disabling
 * the query. The result is not retried and is accepted only when its own
 * session and revision match the ones it was requested for.
 */
export function useEquipmentCandidates(query: EquipmentCandidatesQuery) {
  const port = useEquipmentPort();
  const { saveSessionID, saveRevision, characterID, slotType, search, page, pageSize } = query;
  const identifier = saveSessionID ?? "";

  return useQuery({
    queryKey: queryKeys.equipmentCandidates(
      identifier,
      characterID ?? noCharacter,
      saveRevision ?? "",
      { slotType: slotType ?? "", search, page, pageSize },
    ),
    queryFn:
      identifier === "" ||
      saveRevision === undefined ||
      characterID === undefined ||
      slotType === undefined
        ? skipToken
        : async (): Promise<EquipmentCandidatesPage> => {
            const request: EquipmentCandidatesRequest = {
              saveSessionID: identifier,
              characterID,
              slotType,
              search,
              page,
              pageSize,
            };
            return requireCurrentSaveResponse(
              await port.getEquipmentCandidates(request),
              identifier,
              saveRevision,
            );
          },
    retry: false,
  });
}
