import { useCallback, useRef, useState } from "react";
import { type AppError, toAppError } from "../errors/appError";
import { useEquipmentPort } from "./equipmentClient";
import type { EquipmentMutationReceipt, LoadoutResource } from "./equipmentPort";

/**
 * The single path every Equipment mutation takes.
 *
 * A feature component never calls the port directly and never invalidates a
 * query itself: it calls one of the functions below, and the receipt of the
 * committed mutation goes to `applyReceipt`, the one place that notes the
 * mutation against the `session.changed` stream, maps `changedScopes` onto
 * query keys and re-reads the session. There is deliberately no second mapping
 * from scopes to keys anywhere above the application layer.
 *
 * The same three rules the Items mutations enforce apply here:
 *
 *   - one mutation at a time, because a session accepts one mutating operation
 *     at a time and a second call would race the revision;
 *   - the revision travels with the call and is never re-read or repaired: a
 *     conflict is reported to the user, and this layer never retries it;
 *   - a failed mutation changes no cache at all, because `applyReceipt` runs
 *     only after the backend committed.
 *
 * Every function replaces a complete group. Sending only the touched position
 * is impossible by construction: the backend contract is the whole group in its
 * own order, and `null` is its own empty position.
 */
export type EquipmentMutations = {
  /** True while one mutation is in flight; every action is disabled meanwhile. */
  isBusy: boolean;
  /**
   * The last failure, or undefined. A new mutation clears it before it starts,
   * so a later success never leaves a stale failure behind; `clearError`
   * dismisses it without starting one.
   */
  error: AppError | undefined;
  clearError: () => void;
  setArmaments: (input: SlotAssignmentsInput) => Promise<boolean>;
  setArmor: (input: SlotAssignmentsInput) => Promise<boolean>;
  setTalismans: (
    input: EquipmentMutationScope & { orderedOwnedItemIDs: readonly string[] },
  ) => Promise<boolean>;
  setSpells: (
    input: EquipmentMutationScope & { orderedResources: readonly LoadoutResource[] },
  ) => Promise<boolean>;
  setPhysick: (
    input: EquipmentMutationScope & { crystalTearResources: readonly (LoadoutResource | null)[] },
  ) => Promise<boolean>;
  setPouch: (input: SlotAssignmentsInput) => Promise<boolean>;
  setQuickItems: (input: SlotAssignmentsInput) => Promise<boolean>;
};

/** The session, slot and revision every Equipment mutation is committed under. */
export type EquipmentMutationScope = {
  saveSessionID: string;
  characterID: number;
  expectedRevision: string;
};

/** One complete positional group, with `null` for a position the group clears. */
export type SlotAssignmentsInput = EquipmentMutationScope & {
  slotAssignments: readonly (string | null)[];
};

/**
 * `applyReceipt` is the session controller's own post-mutation step. It is
 * passed in rather than reached for, so the Equipment module cannot grow a
 * second copy of it and a test can observe exactly what a mutation published.
 */
export function useEquipmentMutations(
  applyReceipt: (receipt: EquipmentMutationReceipt) => Promise<unknown>,
): EquipmentMutations {
  const port = useEquipmentPort();
  // The lock is a ref rather than the state flag because two calls started
  // before the next render both read the same stale `isBusy`. The ref is
  // written synchronously, so the second call sees the first one's claim and is
  // refused. `isBusy` stays a state value: it is what drives the interface.
  const running = useRef(false);
  const [isBusy, setBusy] = useState(false);
  const [error, setError] = useState<AppError | undefined>(undefined);

  const run = useCallback(
    async (call: () => Promise<EquipmentMutationReceipt>): Promise<boolean> => {
      // A second mutation while one is in flight is refused rather than queued:
      // it would carry the revision of a snapshot that is about to be replaced.
      if (running.current) return false;
      running.current = true;
      setBusy(true);
      setError(undefined);
      try {
        await applyReceipt(await call());
        return true;
      } catch (reason) {
        // A revision conflict is reported and never retried automatically: the
        // user reviews and confirms the change again.
        setError(toAppError(reason));
        return false;
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
    clearError: () => setError(undefined),
    setArmaments: (input) => run(() => port.setEquippedArmaments(input)),
    setArmor: (input) => run(() => port.setEquippedArmor(input)),
    setTalismans: (input) => run(() => port.setEquippedTalismans(input)),
    setSpells: (input) => run(() => port.setEquippedSpells(input)),
    setPhysick: (input) => run(() => port.setPhysickMixture(input)),
    setPouch: (input) => run(() => port.setPouchItems(input)),
    setQuickItems: (input) => run(() => port.setQuickItems(input)),
  };
}
