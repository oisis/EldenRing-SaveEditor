import { skipToken, useQuery } from "@tanstack/react-query";
import { noCharacter, queryKeys } from "../queryKeys";
import { requireCurrentSaveResponse } from "../saveRevision";
import { useItemsPort } from "./itemsClient";
import type { ItemPage, ItemPageRequest } from "./itemsPort";

/**
 * A container page as a view asks for it: the session and the slot may still be
 * unknown, while the section and the paging are presentation state the view
 * always holds.
 */
export type ItemPageQuery = {
  saveSessionID: string | undefined;
  saveRevision: string | undefined;
  characterID: number | undefined;
  containerSection: string;
  page: number;
  pageSize: number;
};

/**
 * Both container getters share one lifecycle, so they share one query. Only the
 * key and the port call differ.
 *
 * Without a session or a slot there is nothing to ask for, so no query function
 * is built at all. `skipToken` rather than `enabled` is what guards the port: a
 * manual `refetch()` runs the query function even while the query is disabled,
 * so a missing identifier has to make the call impossible, not merely inactive.
 * The section and the paging are forwarded unchecked; which sections exist and
 * which page is in range is the backend's contract. The query is not retried: a
 * failing desktop bridge call does not become healthy by repeating it.
 */
function useItemPage(
  query: ItemPageQuery,
  queryKey: readonly unknown[],
  call: (request: ItemPageRequest) => Promise<ItemPage>,
) {
  const { saveSessionID, saveRevision, characterID, containerSection, page, pageSize } = query;
  const identifier = saveSessionID ?? "";

  return useQuery({
    queryKey,
    queryFn:
      identifier === "" || saveRevision === undefined || characterID === undefined
        ? skipToken
        : async () =>
            requireCurrentSaveResponse(
              await call({
                saveSessionID: identifier,
                characterID,
                containerSection,
                page,
                pageSize,
              }),
              identifier,
              saveRevision,
            ),
    retry: false,
  });
}

export function useInventory(query: ItemPageQuery) {
  const port = useItemsPort();

  return useItemPage(
    query,
    queryKeys.inventory(
      query.saveSessionID ?? "",
      query.characterID ?? noCharacter,
      query.containerSection,
      query.page,
      query.pageSize,
      query.saveRevision ?? "",
    ),
    (request) => port.getInventory(request),
  );
}

export function useStorage(query: ItemPageQuery) {
  const port = useItemsPort();

  return useItemPage(
    query,
    queryKeys.storage(
      query.saveSessionID ?? "",
      query.characterID ?? noCharacter,
      query.containerSection,
      query.page,
      query.pageSize,
      query.saveRevision ?? "",
    ),
    (request) => port.getStorage(request),
  );
}
