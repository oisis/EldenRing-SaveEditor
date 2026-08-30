import { useMutation, useQueryClient } from "@tanstack/react-query";
import { queryKeys } from "../queryKeys";
import { useSaveSessionPort } from "./saveSessionClient";
import type { SaveSession } from "./saveSessionPort";

export type LoadSaveInput = {
  /** The source the host layer supplies, forwarded to the backend unchanged. */
  source: string;
  /** The expected platform, forwarded to the backend unchanged. */
  expectedPlatform: string;
};

/**
 * Creates a save session. The backend result is the only thing that reaches the
 * cache, and only once it has succeeded: a failed load leaves no partial
 * session behind.
 *
 * A previous session is deliberately left open. Which session is replaced, and
 * when, belongs to the session lifecycle that the interface does not implement
 * yet.
 */
export function useLoadSave() {
  const port = useSaveSessionPort();
  const queryClient = useQueryClient();

  return useMutation<SaveSession, Error, LoadSaveInput>({
    mutationFn: ({ source, expectedPlatform }) => port.loadSave(source, expectedPlatform),
    onSuccess: (session) => {
      queryClient.setQueryData(queryKeys.loadedSave(session.saveSessionID), session);
    },
  });
}
