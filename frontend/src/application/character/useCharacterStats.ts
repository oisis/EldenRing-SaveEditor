import { skipToken, useQuery } from "@tanstack/react-query";
import { noCharacter, queryKeys } from "../queryKeys";
import { requireCurrentSaveResponse } from "../saveRevision";
import { useCharacterPort } from "./characterClient";

/**
 * Reads the raw statistics of one character slot. It follows the same rule as
 * `useCharacterProfile`: a session and a slot are required, the slot value
 * itself is the backend's to validate, and a missing parameter yields
 * `skipToken` so that no manual `refetch()` can reach the port either.
 */
export function useCharacterStats(
  saveSessionID: string | undefined,
  saveRevision: string | undefined,
  characterID: number | undefined,
) {
  const port = useCharacterPort();
  const identifier = saveSessionID ?? "";

  return useQuery({
    queryKey: queryKeys.characterStats(identifier, characterID ?? noCharacter, saveRevision ?? ""),
    queryFn:
      identifier === "" || saveRevision === undefined || characterID === undefined
        ? skipToken
        : async () =>
            requireCurrentSaveResponse(
              await port.getCharacterStats(identifier, characterID),
              identifier,
              saveRevision,
            ),
    retry: false,
  });
}
