import { useQuery } from "@tanstack/react-query";
import { queryKeys } from "../queryKeys";
import { useSaveSessionPort } from "./saveSessionClient";

/**
 * Reads the session the backend already holds. Feature modules use this and
 * never the generated desktop bindings.
 *
 * Without an identifier there is nothing to ask for, so the query stays
 * disabled instead of calling the backend with a placeholder. The query is not
 * retried: a failing desktop bridge call does not become healthy by repeating
 * it.
 */
export function useLoadedSave(saveSessionID: string | undefined) {
  const port = useSaveSessionPort();
  // Only ever passed to the backend while the query is enabled, so the empty
  // string is a key placeholder and never reaches a call.
  const identifier = saveSessionID ?? "";

  return useQuery({
    queryKey: queryKeys.loadedSave(identifier),
    queryFn: () => port.getLoadedSave(identifier),
    enabled: identifier !== "",
    retry: false,
  });
}
