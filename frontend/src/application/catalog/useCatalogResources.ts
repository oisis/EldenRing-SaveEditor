import { useQuery } from "@tanstack/react-query";
import { queryKeys } from "../queryKeys";
import { useCatalogPort } from "./catalogClient";
import type { CatalogResourcesRequest } from "./catalogPort";

/**
 * One page of the catalog. The request is forwarded unchecked: which filter
 * values exist, what an empty filter means and which page and page size zero
 * resolve to are the backend's contract, and the served page and page size come
 * back from it rather than from the request.
 *
 * There is no `skipToken` guard here, unlike the container hooks: every empty
 * filter and zero paging value is a valid backend argument, so no request shape
 * means "nothing to ask for". The query neither depends on a save session nor
 * on a character, and is not retried: a failing desktop bridge call does not
 * become healthy by repeating it.
 */
export function useCatalogResources(request: CatalogResourcesRequest) {
  const port = useCatalogPort();

  return useQuery({
    queryKey: queryKeys.catalogResources(request),
    queryFn: () => port.getResources(request),
    retry: false,
  });
}
