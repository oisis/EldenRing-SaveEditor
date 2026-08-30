import { skipToken, useQuery } from "@tanstack/react-query";
import { noCharacter, queryKeys } from "../queryKeys";
import { useEquipmentPort } from "./equipmentClient";
import type {
  CharacterEquipment,
  CharacterEquippedSpells,
  CharacterLoadout,
  CharacterPhysickMixture,
  CharacterPouchItems,
  CharacterQuickItems,
  EquipmentRequest,
} from "./equipmentPort";

/**
 * An equipped view as a view asks for it: the session and the slot may still be
 * unknown while nothing has been selected.
 */
export type EquipmentQuery = {
  saveSessionID: string | undefined;
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
function useEquipmentQuery<T>(
  query: EquipmentQuery,
  queryKey: readonly unknown[],
  call: (request: EquipmentRequest) => Promise<T>,
) {
  const { saveSessionID, characterID } = query;
  const identifier = saveSessionID ?? "";

  return useQuery({
    queryKey,
    queryFn:
      identifier === "" || characterID === undefined
        ? skipToken
        : () => call({ saveSessionID: identifier, characterID }),
    retry: false,
  });
}

/** The key arguments every equipped view is scoped by. */
function scope(query: EquipmentQuery): [string, number | typeof noCharacter] {
  return [query.saveSessionID ?? "", query.characterID ?? noCharacter];
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
