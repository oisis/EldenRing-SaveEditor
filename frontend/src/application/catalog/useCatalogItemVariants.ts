import { skipToken, useQuery } from "@tanstack/react-query";
import { queryKeys } from "../queryKeys";
import { useCatalogPort } from "./catalogClient";

/**
 * The variants of one catalog item.
 *
 * `undefined` means nothing is selected, so no query function is built at all.
 * `skipToken` rather than `enabled` is what guards the port: a manual
 * `refetch()` runs the query function even while the query is disabled, so an
 * absent selection has to make the call impossible, not merely inactive. The
 * empty string is not an absent selection: it is a real identity the backend
 * rejects, and swallowing it here would hide that rejection.
 *
 * The pair is forwarded unchecked: that only the item kind carries variants,
 * which keys are well formed and that neither value is trimmed or aliased are
 * the backend's contract. An item without variants is a valid empty result, not
 * an error. The query depends on no save session and no character, and it is
 * not retried: a failing desktop bridge call does not become healthy by
 * repeating it.
 */
export function useCatalogItemVariants(kind: string | undefined, key: string | undefined) {
  const port = useCatalogPort();

  return useQuery({
    queryKey: queryKeys.catalogItemVariants(kind ?? null, key ?? null),
    queryFn:
      kind === undefined || key === undefined
        ? skipToken
        : () => port.getItemVariants({ kind, key }),
    retry: false,
  });
}
