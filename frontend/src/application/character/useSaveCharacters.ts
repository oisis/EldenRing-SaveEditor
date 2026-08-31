import { skipToken, useQuery } from "@tanstack/react-query";
import { queryKeys } from "../queryKeys";
import { requireCurrentSaveResponse } from "../saveRevision";
import { useCharacterPort } from "./characterClient";
import type { CharacterPort } from "./characterPort";

/**
 * The one description of the character-list query: its key, its call and its
 * retry rule. Both the hook below and the session flow that has to fetch the
 * list imperatively build on it, so a session can never be read through two
 * differently configured queries.
 */
export function saveCharactersQuery(
  port: CharacterPort,
  saveSessionID: string,
  saveRevision: string,
) {
  return {
    queryKey: queryKeys.saveCharacters(saveSessionID, saveRevision),
    queryFn: async () =>
      requireCurrentSaveResponse(
        await port.getSaveCharacters(saveSessionID),
        saveSessionID,
        saveRevision,
      ),
    retry: false,
  };
}

/**
 * Reads every character slot of a session. Feature modules use this and never
 * the generated desktop bindings.
 *
 * Without an identifier there is nothing to ask for, so no query function is
 * built at all. `skipToken` rather than `enabled` is what guards the port: a
 * manual `refetch()` runs the query function even while the query is disabled,
 * so a missing identifier has to make the call impossible, not merely inactive.
 * The query is not retried: a failing desktop bridge call does not become
 * healthy by repeating it.
 */
export function useSaveCharacters(
  saveSessionID: string | undefined,
  saveRevision: string | undefined,
) {
  const port = useCharacterPort();
  // A key placeholder only: with no identifier there is no query function that
  // could pass it to the backend.
  const identifier = saveSessionID ?? "";
  const revision = saveRevision ?? "";
  const query = saveCharactersQuery(port, identifier, revision);

  return useQuery({
    ...query,
    queryFn: identifier === "" || saveRevision === undefined ? skipToken : query.queryFn,
  });
}
