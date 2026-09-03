import { useQuery } from "@tanstack/react-query";
import { queryKeys } from "../queryKeys";
import { useCatalogPort } from "./catalogClient";
import type { CatalogItemDatabaseRequest } from "./catalogPort";

/**
 * One page of the Item Database.
 *
 * The catalog is global and belongs to no save session, so this query works
 * with no save loaded and its cache survives closing one. Visibility, order and
 * paging are the backend's answer under the active Safety Profile and are never
 * re-sorted, re-filtered or re-paged here. The query is not retried: a failing
 * desktop bridge call does not become healthy by repeating it.
 */
export function useCatalogItemDatabase(request: CatalogItemDatabaseRequest) {
  const port = useCatalogPort();

  return useQuery({
    queryKey: queryKeys.catalogItemDatabase(request),
    queryFn: () => port.getItemDatabase(request),
    retry: false,
  });
}
