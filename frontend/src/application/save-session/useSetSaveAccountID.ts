import { useMutation } from "@tanstack/react-query";
import { useSaveSessionPort } from "./saveSessionClient";
import type { MutationReceipt } from "./saveSessionPort";

export type SetSaveAccountIDInput = {
  saveSessionID: string;
  /** The canonical decimal identifier as the user typed it. Never a number. */
  accountID: string;
  expectedRevision: string;
};

/**
 * Stores the save owner identifier of one session.
 *
 * Nothing is written to the cache here: the receipt goes to the shared
 * save-mutation path of the session flow, which owns invalidation, the session
 * refresh and the operation history. The identifier itself is neither returned
 * nor cached — the backend exposes no getter for it, so no layer of this
 * application may hold a value it cannot confirm.
 */
export function useSetSaveAccountID() {
  const port = useSaveSessionPort();

  return useMutation<MutationReceipt, Error, SetSaveAccountIDInput>({
    mutationFn: ({ saveSessionID, accountID, expectedRevision }) =>
      port.setSaveAccountID(saveSessionID, accountID, expectedRevision),
  });
}
