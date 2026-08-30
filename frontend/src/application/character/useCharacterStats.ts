import { skipToken, useQuery } from "@tanstack/react-query";
import { noCharacter, queryKeys } from "../queryKeys";
import { useCharacterPort } from "./characterClient";

/**
 * Reads the raw statistics of one character slot. It follows the same rule as
 * `useCharacterProfile`: a session and a slot are required, the slot value
 * itself is the backend's to validate, and a missing parameter yields
 * `skipToken` so that no manual `refetch()` can reach the port either.
 */
export function useCharacterStats(
  saveSessionID: string | undefined,
  characterID: number | undefined,
) {
  const port = useCharacterPort();
  const identifier = saveSessionID ?? "";

  return useQuery({
    queryKey: queryKeys.characterStats(identifier, characterID ?? noCharacter),
    queryFn:
      identifier === "" || characterID === undefined
        ? skipToken
        : () => port.getCharacterStats(identifier, characterID),
    retry: false,
  });
}
