import { useMutation, useQueryClient } from "@tanstack/react-query";
import { queryKeys } from "../queryKeys";
import { useSaveSessionPort } from "./saveSessionClient";
import type { SaveSession, SaveSourceKind } from "./saveSessionPort";

export type LoadSaveInput = {
  /** The source the host layer supplied, forwarded to the backend unchanged. */
  source: string;
  /** The expected platform, forwarded to the backend unchanged. */
  expectedPlatform: string;
  /**
   * What the source is. There is no default: the caller states it, because a
   * session must never claim an origin nobody stated.
   */
  sourceKind: SaveSourceKind;
};

/**
 * Creates a save session. The backend result is the only thing that reaches the
 * cache, and only once it has succeeded: a failed load leaves no partial
 * session behind, and no session at all for a source the backend refused.
 *
 * Retiring the session this one replaces belongs to the flow that owns the
 * session, not here: this hook does not know which session was open before it.
 */
export function useLoadSave() {
  const port = useSaveSessionPort();
  const queryClient = useQueryClient();

  return useMutation<SaveSession, Error, LoadSaveInput>({
    mutationFn: ({ source, expectedPlatform, sourceKind }) =>
      port.loadSave(source, expectedPlatform, sourceKind),
    onSuccess: (session) => {
      queryClient.setQueryData(queryKeys.loadedSave(session.saveSessionID), session);
    },
  });
}
