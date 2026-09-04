import { useMutation, useQuery } from "@tanstack/react-query";
import { queryKeys } from "../queryKeys";
import { useAboutPort } from "./aboutClient";
import type { UpdateCheck } from "./aboutPort";

/** The approved project links. They are a build constant, so they never expire. */
export function useProjectLinks() {
  const port = useAboutPort();

  return useQuery({
    queryKey: queryKeys.projectLinks(),
    queryFn: () => port.getProjectLinks(),
    staleTime: Number.POSITIVE_INFINITY,
    retry: false,
  });
}

/** Opens one approved address, as an explicit user action. */
export function useOpenProjectLink() {
  const port = useAboutPort();

  return useMutation<void, Error, string>({
    mutationFn: (linkID: string) => port.openProjectLink(linkID),
  });
}

/**
 * The manual update check.
 *
 * It is a mutation rather than a query on purpose: section 1.3 allows exactly
 * one check per explicit user action, and a query would invite a background
 * refetch, a retry or a mount-time run — none of which the specification
 * permits.
 */
export function useCheckForUpdates() {
  const port = useAboutPort();

  return useMutation<UpdateCheck, Error, void>({
    mutationFn: () => port.checkForUpdates(),
  });
}
