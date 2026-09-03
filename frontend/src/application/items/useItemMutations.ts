import { useCallback, useRef, useState } from "react";
import { type AppError, toAppError } from "../errors/appError";
import { useItemsPort } from "./itemsClient";
import type { AddItemsRequestEntry, ItemMutationReceipt } from "./itemsPort";

/**
 * The single path every Items mutation takes.
 *
 * A feature component never calls the port directly and never invalidates a
 * query itself: it calls one of the functions below, and the receipt of the
 * committed mutation goes to `applyReceipt`, the one place that notes the
 * mutation against the `session.changed` stream, maps `changedScopes` onto
 * query keys and re-reads the session. There is deliberately no second mapping
 * from scopes to keys anywhere above the application layer.
 *
 * Three rules are enforced here rather than in each screen:
 *
 *   - one mutation at a time, because a session accepts one mutating operation
 *     at a time and a second call would race the revision;
 *   - the revision travels with the call and is never re-read or repaired: a
 *     conflict is reported to the user, and this layer never retries it;
 *   - a failed mutation changes no cache at all, because `applyReceipt` runs
 *     only after the backend committed.
 */
export type ItemMutations = {
  /** True while one mutation is in flight; every action is disabled meanwhile. */
  isBusy: boolean;
  /** The last failure, or undefined. It is never cleared by a later success. */
  error: AppError | undefined;
  clearError: () => void;
  addItems: (input: {
    saveSessionID: string;
    characterID: number;
    expectedRevision: string;
    items: readonly AddItemsRequestEntry[];
    confirmBanRisk: boolean;
  }) => Promise<boolean>;
  moveToStorage: (input: OwnedItemsMutationInput) => Promise<boolean>;
  moveToInventory: (input: OwnedItemsMutationInput) => Promise<boolean>;
  removeItems: (input: OwnedItemsMutationInput) => Promise<boolean>;
  reorderItems: (input: {
    saveSessionID: string;
    characterID: number;
    expectedRevision: string;
    anchorOwnedItemID: string;
    groupOwnedItemIDs: readonly string[];
    targetPosition: number;
  }) => Promise<boolean>;
  setQuantity: (input: {
    saveSessionID: string;
    characterID: number;
    expectedRevision: string;
    ownedItemID: string;
    quantity: number;
  }) => Promise<boolean>;
};

export type OwnedItemsMutationInput = {
  saveSessionID: string;
  characterID: number;
  expectedRevision: string;
  ownedItemIDs: readonly string[];
};

/**
 * `applyReceipt` is the session controller's own post-mutation step. It is
 * passed in rather than reached for, so the Items module cannot grow a second
 * copy of it and a test can observe exactly what a mutation published.
 */
export function useItemMutations(
  applyReceipt: (receipt: ItemMutationReceipt) => Promise<unknown>,
): ItemMutations {
  const port = useItemsPort();
  // The lock is a ref rather than the state flag because two calls started
  // before the next render both read the same stale `isBusy`. The ref is
  // written synchronously, so the second call sees the first one's claim and is
  // refused. `isBusy` stays a state value: it is what drives the interface.
  const running = useRef(false);
  const [isBusy, setBusy] = useState(false);
  const [error, setError] = useState<AppError | undefined>(undefined);

  const run = useCallback(
    async (call: () => Promise<ItemMutationReceipt>): Promise<boolean> => {
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
    addItems: ({ saveSessionID, characterID, expectedRevision, items, confirmBanRisk }) =>
      run(() =>
        port.addItemsToContainers({
          saveSessionID,
          characterID,
          items,
          confirmBanRisk,
          expectedRevision,
        }),
      ),
    moveToStorage: (input) => run(() => port.moveOwnedItemsToStorage(input)),
    moveToInventory: (input) => run(() => port.moveOwnedItemsToInventory(input)),
    removeItems: (input) => run(() => port.removeOwnedItems(input)),
    reorderItems: (input) => run(() => port.reorderInventoryItems(input)),
    setQuantity: (input) => run(() => port.setOwnedItemQuantity(input)),
  };
}
