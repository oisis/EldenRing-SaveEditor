import { useCallback, useEffect, useRef, useState } from "react";
import { type AppError, toAppError } from "../errors/appError";
import type { MutationReceipt } from "../save-session/saveSessionPort";
import { useCharacterPort } from "./characterClient";
import type {
  CloneCharacterInput,
  DeleteCharacterInput,
  SetCharacterActiveInput,
} from "./characterPort";

/**
 * What one slot request ended as. The commit is the boundary: everything before
 * it is a refusal that changed nothing, everything after it is a change the
 * session already carries, whether or not the interface managed to catch up.
 *
 *   - `rejected`   — refused before any commit; the session is untouched.
 *   - `unchanged`  — accepted and idempotent; no receipt, no history entry.
 *   - `applied`    — committed and the receipt is reflected on screen.
 *   - `desynced`   — committed, but the receipt could not be applied. The
 *                    mutation must never be offered again for this reason.
 */
export type SlotMutationOutcome = "rejected" | "unchanged" | "applied" | "desynced";

/**
 * What the slots on screen were read for, as the caller currently knows it. It
 * is the only evidence the hook accepts that a committed mutation is reflected
 * again: the session the operations run against, and the revision the list on
 * screen was read for while it is unread.
 */
export type SlotSyncState = {
  saveSessionID: string;
  /** `undefined` while no list has been read for this session. */
  readRevision: string | undefined;
};

/**
 * Whether the list on screen states the committed mutation recorded in
 * `desync`: it was read for that session and for exactly the revision the
 * mutation committed. That is what the session synchronisation produces once it
 * catches up, and it settles the incident for good. Revisions stay opaque
 * backend strings and are compared for equality only.
 */
function isReflected(
  desync: { saveSessionID: string; saveRevision: string },
  sync: SlotSyncState | undefined,
): boolean {
  return desync.saveSessionID === sync?.saveSessionID && desync.saveRevision === sync.readRevision;
}

/**
 * Whether the committed mutation recorded in `desync` is still unreflected.
 *
 * Another session is not this session, so a late answer of a retired one can
 * never block the new one, and a settled incident is dropped rather than kept:
 * a later revision is not a new desynchronisation of an old operation.
 */
function stillDesynced(
  desync: { saveSessionID: string; saveRevision: string } | undefined,
  sync: SlotSyncState | undefined,
): boolean {
  if (desync === undefined || sync === undefined) return false;
  return desync.saveSessionID === sync.saveSessionID && !isReflected(desync, sync);
}

/**
 * The three slot-wide mutations of one save session. They are kept apart from
 * `useCharacterMutations` because the slot management is the only caller that
 * needs them and because `setActive` has a second success variant the receipt
 * pipeline must not see.
 *
 * Every call stays a session change: nothing is written to the save file here.
 *
 * The hook owns the in-flight lock, so it must be mounted where the operation
 * lives — above the dialog that starts it. A hook mounted inside the dialog
 * would hand a reopened dialog a fresh, unlocked instance.
 */
export type CharacterSlotMutations = {
  isBusy: boolean;
  /** Set only for a refusal before the commit; a committed change never sets it. */
  error: AppError | undefined;
  /**
   * Set once a committed change could not be reflected on screen, and cleared
   * only by the confirmed synchronisation of that change or by another session.
   * Closing the dialog, opening it again and picking another slot leave it.
   */
  isDesynced: boolean;
  /** Clears the refusal of an uncommitted request; never the desync block. */
  clearError: () => void;
  setActive: (input: SetCharacterActiveInput) => Promise<SlotMutationOutcome>;
  clone: (input: CloneCharacterInput) => Promise<SlotMutationOutcome>;
  /** Deletes an active character or clears a residual slot; one backend writer. */
  remove: (input: DeleteCharacterInput) => Promise<SlotMutationOutcome>;
};

export function useCharacterSlotMutations(
  applyReceipt: (receipt: MutationReceipt) => Promise<unknown>,
  sync?: SlotSyncState,
): CharacterSlotMutations {
  const port = useCharacterPort();

  // A ref and not the busy state: two clicks in one render pass would both read
  // the same state value, and the second one must not reach the backend.
  const running = useRef(false);
  const [isBusy, setBusy] = useState(false);
  // The commit that could not be reflected, kept as the receipt's own identity
  // so the block belongs to the session it happened in. Mirrored in a ref
  // because `run` has to refuse before the port call, in the same pass.
  const [desync, setDesync] = useState<MutationReceipt | undefined>(undefined);
  const desynced = useRef<MutationReceipt | undefined>(undefined);
  const [error, setError] = useState<AppError | undefined>(undefined);
  const synced = useRef(sync);
  synced.current = sync;

  // One confirmed reading of the committed revision settles that incident for
  // good; without dropping it, the next revision would present the very same
  // receipt as a fresh desynchronisation. Only the settled receipt is dropped,
  // so a newer incident recorded meanwhile survives its predecessor.
  useEffect(() => {
    if (desync === undefined || !isReflected(desync, sync)) return;
    if (desynced.current === desync) desynced.current = undefined;
    setDesync((current) => (current === desync ? undefined : current));
  }, [desync, sync]);

  const run = useCallback(
    async (call: () => Promise<MutationReceipt | undefined>): Promise<SlotMutationOutcome> => {
      if (running.current) return "rejected";
      // A committed change that is still unreflected may not be followed by
      // another one, whichever control asks for it.
      if (stillDesynced(desynced.current, synced.current)) return "rejected";
      running.current = true;
      setBusy(true);
      setError(undefined);
      try {
        let receipt: MutationReceipt | undefined;
        try {
          receipt = await call();
        } catch (reason) {
          // Refused before the commit: the session is exactly where it was.
          setError(toAppError(reason));
          return "rejected";
        }
        // An idempotent request — the slot is already in the wanted state —
        // carries no receipt, so no receipt is applied, no history entry is
        // claimed and the revision on screen stays exactly where it was.
        if (receipt === undefined) return "unchanged";
        try {
          await applyReceipt(receipt);
        } catch {
          // The backend already committed. Reporting this as a rejection would
          // invite the user to repeat a mutation that has happened.
          desynced.current = receipt;
          setDesync(receipt);
          return "desynced";
        }
        return "applied";
      } finally {
        running.current = false;
        setBusy(false);
      }
    },
    [applyReceipt],
  );

  return {
    isBusy,
    error,
    isDesynced: stillDesynced(desync, sync),
    clearError: () => setError(undefined),
    setActive: (input) =>
      run(async () => {
        const result = await port.setCharacterActive(input);
        return result.changed ? result.receipt : undefined;
      }),
    clone: (input) => run(() => port.cloneCharacter(input)),
    remove: (input) => run(() => port.deleteCharacter(input)),
  };
}
