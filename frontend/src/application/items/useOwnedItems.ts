import { skipToken, useQuery } from "@tanstack/react-query";
import { noCharacter, queryKeys } from "../queryKeys";
import { requireCurrentSaveResponse } from "../saveRevision";
import { useItemsPort } from "./itemsClient";
import type { OwnedItemsRequest } from "./itemsPort";

/**
 * One authoritative container page as a view asks for it: the session and the
 * slot may still be unknown, while the container, the filters, the sort order
 * and the paging are presentation state the view always holds.
 */
export type OwnedItemsQuery = Omit<OwnedItemsRequest, "saveSessionID" | "characterID"> & {
  saveSessionID: string | undefined;
  saveRevision: string | undefined;
  characterID: number | undefined;
};

/**
 * Reads one page of one container with the backend applying the search, the
 * category, the favourites and the sort order to the whole container.
 *
 * Without a session or a slot there is nothing to ask for, so no query function
 * is built at all. `skipToken` rather than `enabled` is what guards the port: a
 * manual `refetch()` runs the query function even while the query is disabled,
 * so a missing identifier has to make the call impossible, not merely inactive.
 * Every filter is forwarded unchecked; which values exist is the backend's
 * contract. The query is not retried: a failing desktop bridge call does not
 * become healthy by repeating it.
 */
export function useOwnedItems(query: OwnedItemsQuery) {
  const port = useItemsPort();
  const { saveSessionID, saveRevision, characterID, ...request } = query;
  const identifier = saveSessionID ?? "";

  return useQuery({
    queryKey: queryKeys.ownedItems(
      identifier,
      characterID ?? noCharacter,
      saveRevision ?? "",
      request,
    ),
    queryFn:
      identifier === "" || saveRevision === undefined || characterID === undefined
        ? skipToken
        : async () =>
            requireCurrentSaveResponse(
              await port.getOwnedItems({ ...request, saveSessionID: identifier, characterID }),
              identifier,
              saveRevision,
            ),
    retry: false,
  });
}
