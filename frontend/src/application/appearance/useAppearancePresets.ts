import { useQuery } from "@tanstack/react-query";
import { queryKeys } from "../queryKeys";
import { useAppearancePort } from "./appearanceClient";
import type { GetAppearancePresetsRequest } from "./appearancePort";

export function useAppearancePresets(request: GetAppearancePresetsRequest = {}) {
  const port = useAppearancePort();

  return useQuery({
    queryKey: queryKeys.appearancePresets(request.search, request.tags),
    queryFn: () => port.getAppearancePresets(request),
    retry: false,
  });
}
