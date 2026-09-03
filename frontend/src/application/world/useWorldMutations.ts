import { useQuery } from "@tanstack/react-query";
import { useCallback, useRef, useState } from "react";
import { type AppError, toAppError } from "../errors/appError";
import { queryKeys } from "../queryKeys";
import { useWorldPort } from "./worldClient";
import type {
  WorldMutationCapability,
  WorldMutationReceipt,
  WorldMutationScope,
  WorldOperationKind,
  WorldPort,
  WorldResourceToggleRequest,
} from "./worldPort";

/**
 * The World mutation contract of this build.
 *
 * It names no session, no slot and no revision, so it is read under one stable
 * key and is not refetched when a revision advances. It is not retried either:
 * a failing bridge call does not become healthy by repeating it, and a World
 * writer stays hidden while the contract is unknown rather than being shown
 * with an assumed risk.
 */
export function useWorldMutationCapabilities() {
  const port = useWorldPort();
  return useQuery({
    queryKey: queryKeys.worldMutationCapabilities(),
    queryFn: () => port.getWorldMutationCapabilities(),
    retry: false,
  });
}

/**
 * The single path every World mutation takes.
 *
 * It follows the Items and Equipment mutations exactly: one mutation at a time,
 * the revision travels with the call and is never re-read or repaired, no
 * automatic retry, and `applyReceipt` runs only after the backend committed. A
 * failure therefore changes no cache and no local view at all, and there are no
 * optimistic updates to roll back.
 *
 * Every function is one backend call. A bulk change is the backend's own atomic
 * operation, never a loop of single setters here.
 */
export type WorldMutations = {
  /** True while one mutation is in flight; every World action is disabled meanwhile. */
  isBusy: boolean;
  /**
   * The last failure, or undefined. A new mutation clears it before it starts,
   * so a later success never leaves a stale failure behind; `clearError`
   * dismisses it without starting one.
   */
  error: AppError | undefined;
  clearError: () => void;
  /**
   * Runs the toggle of one supported capability. The operation kind selects the
   * port method, so a screen states which backend operation it is asking for
   * instead of reaching for a method name of its own.
   */
  toggleResource: (
    operationKind: WorldResourceToggleOperationKind,
    request: WorldResourceToggleRequest,
  ) => Promise<boolean>;
  removeFogOfWar: (scope: WorldMutationScope) => Promise<boolean>;
  applyQuestStep: (
    request: WorldMutationScope & {
      questKind: string;
      questKey: string;
      stepKind: string;
      stepKey: string;
    },
  ) => Promise<boolean>;
  selectSpectralSteedAttire: (
    request: WorldMutationScope & { attireKey: string },
  ) => Promise<boolean>;
  lockAllSpectralSteedAttires: (scope: WorldMutationScope) => Promise<boolean>;
};

/** The eleven capabilities that assign one boolean to one catalog resource. */
export type WorldResourceToggleOperationKind = keyof typeof resourceTogglePortMethod;

/**
 * The one place a resource-toggle operation kind is resolved to its port
 * method. It is a lookup, not a rule: the operation kind is the backend's own,
 * and no risk, availability or default is derived from it here.
 */
const resourceTogglePortMethod = {
  set_region_unlocked: "setRegionUnlocked",
  set_map_region_revealed: "setMapRegionRevealed",
  set_grace_visited: "setGraceVisited",
  set_boss_defeated: "setBossDefeated",
  set_gesture_unlocked: "setGestureUnlocked",
  set_cookbook_unlocked: "setCookbookUnlocked",
  set_bell_bearing_unlocked: "setBellBearingUnlocked",
  set_whetblade_unlocked: "setWhetbladeUnlocked",
  set_tutorial_unlocked: "setTutorialUnlocked",
  set_summoning_pool_activated: "setSummoningPoolActivated",
  set_colosseum_unlocked: "setColosseumUnlocked",
} as const satisfies Record<string, keyof WorldPort>;

/**
 * `applyReceipt` is the session controller's own post-mutation step. It is
 * passed in rather than reached for, so the World module cannot grow a second
 * copy of it and a test can observe exactly what a mutation published.
 */
export function useWorldMutations(
  applyReceipt: (receipt: WorldMutationReceipt) => Promise<unknown>,
): WorldMutations {
  const port = useWorldPort();
  // The lock is a ref rather than the state flag because two calls started
  // before the next render both read the same stale `isBusy`. The ref is
  // written synchronously, so the second call sees the first one's claim and is
  // refused.
  const running = useRef(false);
  const [isBusy, setBusy] = useState(false);
  const [error, setError] = useState<AppError | undefined>(undefined);

  const run = useCallback(
    async (call: () => Promise<WorldMutationReceipt>): Promise<boolean> => {
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
        // user reviews the state and asks for the change again.
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
    toggleResource: (operationKind, request) =>
      run(() => port[resourceTogglePortMethod[operationKind]](request)),
    // Fog of War is one-way: the removal is the only state the backend accepts,
    // so `true` is the contract and not a value this layer chose.
    removeFogOfWar: (scope) => run(() => port.setFogOfWarRemoved({ ...scope, removed: true })),
    applyQuestStep: (request) => run(() => port.setQuestStep(request)),
    selectSpectralSteedAttire: (request) => run(() => port.setSpectralSteedAttire(request)),
    lockAllSpectralSteedAttires: (scope) => run(() => port.lockAllSpectralSteedAttires(scope)),
  };
}

/** Finds one capability in the backend's answer, or undefined while it is unknown. */
export function findWorldCapability(
  capabilities: readonly WorldMutationCapability[] | undefined,
  operationKind: WorldOperationKind,
): WorldMutationCapability | undefined {
  return capabilities?.find((capability) => capability.operationKind === operationKind);
}
