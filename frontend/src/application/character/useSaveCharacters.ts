import { skipToken, useQuery } from "@tanstack/react-query";
import { queryKeys } from "../queryKeys";
import { useCharacterPort } from "./characterClient";

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

  return useQuery({
    queryKey: queryKeys.saveCharacters(identifier),
    queryFn: identifier === "" ? skipToken : () => port.getSaveCharacters(identifier),
    retry: false,
  });
}
