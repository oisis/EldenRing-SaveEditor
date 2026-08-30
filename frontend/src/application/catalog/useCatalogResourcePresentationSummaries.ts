import { skipToken, useQuery } from "@tanstack/react-query";
import { queryKeys } from "../queryKeys";
import { useCatalogPort } from "./catalogClient";
import type { CatalogResourcePresentationIdentity } from "./catalogPort";

/**
 * Reads lightweight presentation metadata for an ordered identity batch.
 * `undefined` means no batch has been assembled and cannot reach the port;
 * an empty array is a valid request whose backend result is `resources: []`.
 * No identity is normalised and no rejected bridge call is retried.
 */
export function useCatalogResourcePresentationSummaries(
  identities: readonly CatalogResourcePresentationIdentity[] | undefined,
) {
  const port = useCatalogPort();

  return useQuery({
    queryKey: queryKeys.catalogResourcePresentationSummaries(identities ?? null),
    queryFn:
      identities === undefined
        ? skipToken
        : () => port.getResourcePresentationSummaries(identities),
    retry: false,
  });
}
