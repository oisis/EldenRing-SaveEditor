import { skipToken, useQuery } from "@tanstack/react-query";
import { noCharacter, queryKeys } from "../queryKeys";
import { requireCurrentSaveResponse } from "../saveRevision";
import { useCharacterPort } from "./characterClient";

/**
 * Reads the profile of one character slot. The query runs as soon as a session
 * and a slot are known; which slot indices exist is the backend's contract, so
 * `0` is an ordinary slot here and a value outside the backend range is passed
 * on to be rejected there instead of being filtered out silently.
 *
 * A missing session or slot yields `skipToken` instead of a query function, so
 * neither the first render nor a manual `refetch()` can reach the port. That
 * also narrows `characterID` to a number in the branch that does call it, so no
 * cast is needed to satisfy the port contract.
 */
export function useCharacterProfile(
  saveSessionID: string | undefined,
  saveRevision: string | undefined,
  characterID: number | undefined,
) {
  const port = useCharacterPort();
  const identifier = saveSessionID ?? "";

  return useQuery({
    queryKey: queryKeys.characterProfile(
      identifier,
      characterID ?? noCharacter,
      saveRevision ?? "",
    ),
    queryFn:
      identifier === "" || saveRevision === undefined || characterID === undefined
        ? skipToken
        : async () =>
            requireCurrentSaveResponse(
              await port.getCharacterProfile(identifier, characterID),
              identifier,
              saveRevision,
            ),
    retry: false,
  });
}
