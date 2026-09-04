import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { queryKeys } from "../queryKeys";
import { useSettingsPort } from "./settingsClient";
import type { DiagnosticReportResult, HostLocation, HostSettings } from "./settingsPort";

/**
 * The persistent host settings as the interface reads them.
 *
 * They are backend state and therefore live outside every session key. The
 * query is not retried: a failing desktop bridge call does not become healthy
 * by repeating it.
 */
export function useHostSettings() {
  const port = useSettingsPort();

  return useQuery({
    queryKey: queryKeys.hostSettings(),
    queryFn: () => port.getHostSettings(),
    retry: false,
  });
}

/**
 * Stores the complete host settings value.
 *
 * The answer of the call that changed the setting is what updates the cache, so
 * the interface never renders a setting the backend did not confirm. This is a
 * host-settings mutation and deliberately not a save mutation: no revision
 * moves and no mutation receipt exists, so nothing here touches a session key.
 */
export function useSetHostSettings() {
  const port = useSettingsPort();
  const queryClient = useQueryClient();

  return useMutation<
    HostSettings,
    Error,
    { skipReviewForNormalRisk: boolean; remoteBackupPolicy: string }
  >({
    mutationFn: (settings) => port.setHostSettings(settings),
    onSuccess: (settings) => queryClient.setQueryData(queryKeys.hostSettings(), settings),
  });
}

/** Opens one known host directory, as an explicit user action. */
export function useOpenHostLocation() {
  const port = useSettingsPort();

  return useMutation<void, Error, HostLocation>({
    mutationFn: (location) => port.openHostLocation(location),
  });
}

/** Exports the redacted diagnostic report through the native Save As dialog. */
export function useExportDiagnosticReport() {
  const port = useSettingsPort();

  return useMutation<DiagnosticReportResult, Error, string | undefined>({
    mutationFn: (saveSessionID) => port.exportDiagnosticReport(saveSessionID),
  });
}
