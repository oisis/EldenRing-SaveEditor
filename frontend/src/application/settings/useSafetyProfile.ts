import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { queryKeys } from "../queryKeys";
import { useSettingsPort } from "./settingsClient";
import type { SafetyProfileSettings } from "./settingsPort";

/**
 * The global Safety Profile as the interface reads it.
 *
 * The value is a host setting rather than save state, so it lives outside every
 * session key and survives closing a save. The query is not retried: a failing
 * desktop bridge call does not become healthy by repeating it.
 */
export function useSafetyProfile() {
  const port = useSettingsPort();

  return useQuery({
    queryKey: queryKeys.safetyProfile(),
    queryFn: () => port.getSafetyProfile(),
    retry: false,
  });
}

/**
 * Stores one profile. The answer of the call that changed the setting is what
 * updates the cache, so the interface never renders a profile the backend did
 * not confirm.
 *
 * Changing the profile changes what the backend reports for the Item Database
 * and for both containers, so every catalog and owned-item view is invalidated.
 * The save session itself does not change: no revision moves and no mutation
 * receipt exists, which is exactly why this is not routed through the shared
 * save-mutation path.
 */
export function useSetSafetyProfile() {
  const port = useSettingsPort();
  const queryClient = useQueryClient();

  return useMutation<SafetyProfileSettings, Error, string>({
    mutationFn: (safetyProfile) => port.setSafetyProfile(safetyProfile),
    onSuccess: async (settings) => {
      queryClient.setQueryData(queryKeys.safetyProfile(), settings);
      await queryClient.invalidateQueries({ queryKey: ["catalog"] });
      await queryClient.invalidateQueries({
        predicate: (query) => query.queryKey.includes("owned-items"),
      });
    },
  });
}
