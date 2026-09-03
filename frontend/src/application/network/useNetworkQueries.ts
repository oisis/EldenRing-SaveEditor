import { skipToken, useQuery } from "@tanstack/react-query";
import { queryKeys } from "../queryKeys";
import { requireCurrentSaveResponse } from "../saveRevision";
import { useNetworkPort } from "./networkClient";
import type { NetworkSettingsSnapshot } from "./networkPort";

export type NetworkSettingsQuery = {
  saveSessionID: string | undefined;
  saveRevision: string | undefined;
};

export function useNetworkSettings(query: NetworkSettingsQuery) {
  const { saveSessionID, saveRevision } = query;
  const port = useNetworkPort();
  const identifier = saveSessionID ?? "";

  return useQuery({
    queryKey: queryKeys.networkSettings(identifier, saveRevision ?? ""),
    queryFn:
      identifier === "" || saveRevision === undefined
        ? skipToken
        : async (): Promise<NetworkSettingsSnapshot> =>
            requireCurrentSaveResponse(
              await port.getNetworkSettings(identifier),
              identifier,
              saveRevision,
            ),
    retry: false,
  });
}

export function useNetworkPresets(presetID?: string) {
  const port = useNetworkPort();

  return useQuery({
    queryKey: queryKeys.networkPresets(presetID),
    queryFn: async () => port.getNetworkPresets(presetID),
    retry: false,
  });
}
