import { skipToken, useQuery } from "@tanstack/react-query";
import { queryKeys } from "../queryKeys";
import { useCharacterPort } from "./characterClient";
import type { CharacterPort } from "./characterPort";

/**
 * The one description of the character-list query: its key, its call and its
 * retry rule. Both the hook below and the session flow that has to fetch the
 * list imperatively build on it, so a session can never be read through two
 * differently configured queries.
 */
export function saveCharactersQuery(port: CharacterPort, saveSessionID: string) {
  return {
    queryKey: queryKeys.saveCharacters(saveSessionID),
    queryFn: () => port.getSaveCharacters(saveSessionID),
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
export function useSaveCharacters(saveSessionID: string | undefined) {
  const port = useCharacterPort();
  // A key placeholder only: with no identifier there is no query function that
  // could pass it to the backend.
  const identifier = saveSessionID ?? "";
  const query = saveCharactersQuery(port, identifier);

  return useQuery({
    ...query,
    queryFn: identifier === "" ? skipToken : query.queryFn,
  });
}
