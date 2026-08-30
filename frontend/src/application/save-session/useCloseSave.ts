import { useMutation, useQueryClient } from "@tanstack/react-query";
import { queryKeys } from "../queryKeys";
import { useSaveSessionPort } from "./saveSessionClient";

/**
 * Closes a save session. The cached session data is dropped only after the
 * backend confirms the close, so a rejected call leaves the view intact and the
 * failure stays visible to the caller.
 */
export function useCloseSave() {
  const port = useSaveSessionPort();
  const queryClient = useQueryClient();

  return useMutation<void, Error, string>({
    mutationFn: (saveSessionID) => port.closeSave(saveSessionID),
    onSuccess: (_result, saveSessionID) => {
      queryClient.removeQueries({ queryKey: queryKeys.saveSession(saveSessionID) });
    },
  });
}
