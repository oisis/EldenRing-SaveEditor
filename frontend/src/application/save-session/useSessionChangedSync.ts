import { useQueryClient } from "@tanstack/react-query";
import { useCallback, useEffect, useRef } from "react";
import {
  type ChangedScope,
  matchesQueryKeyPattern,
  queryKeyPatternsForScopes,
} from "../changedScopes";
import { queryKeys } from "../queryKeys";
import { isCanonicalDecimal } from "../saveRevision";
import { useSaveSessionPort } from "./saveSessionClient";
import type { SaveSession, SessionChangedEvent } from "./saveSessionPort";

/**
 * Keeps one session in step with the backend's committed mutations: both its
 * own state and the query cache built from it.
 *
 * The contract it implements:
 *
 *   - an event for another session is ignored;
 *   - a duplicate or an out-of-order event, meaning one whose sequence is not
 *     newer than the last processed one, is ignored;
 *   - the next event in sequence refreshes exactly the getters its scopes name
 *     and re-reads the session itself, because the event announces a change and
 *     is never the state document that describes it;
 *   - a gap in the sequence means events may have been lost, so the current
 *     session is read back and the baseline is reset from the sequence the
 *     backend confirmed. When that sequence moved past the baseline, the whole
 *     session prefix is refreshed once — the only session-wide invalidation in
 *     the application;
 *   - the same read happens when the listener starts, when the window becomes
 *     visible again, when the subscription is re-established and when a payload
 *     arrives that the adapter could not validate, because none of those may
 *     assume the earlier events arrived. A session that did not move meanwhile
 *     costs one getter call and no invalidation;
 *   - an event that only reports back the mutation this frontend just performed
 *     advances the sequence without invalidating a second time.
 *
 * There is exactly one place the session's own state is updated: the confirmed
 * answer of GetLoadedSave, handed to the owner through `onSession`. The revision,
 * the unsaved-changes flag and the sequence therefore always travel together and
 * always come from the backend.
 *
 * The sequence is a canonical decimal string end to end. It is compared, never
 * parsed: a JavaScript number could not represent every value the backend can
 * produce, and rounding one would silently reorder the stream.
 */
export type SessionChangedSync = {
  /**
   * Records a mutation this frontend performed and already refreshed from its
   * receipt, so the matching event does not repeat that work. The identity is
   * the operation, its revision and its scopes together, exactly as the receipt
   * and the event both report them.
   */
  noteAppliedMutation: (receipt: {
    operationID: string;
    saveRevision: string;
    changedScopes: readonly ChangedScope[];
  }) => void;
};

/** Compares two canonical decimal strings without turning either into a number. */
export function compareSequences(left: string, right: string): number {
  if (left.length !== right.length) {
    return left.length < right.length ? -1 : 1;
  }
  if (left === right) {
    return 0;
  }
  return left < right ? -1 : 1;
}

function mutationIdentity(
  operationID: string,
  saveRevision: string,
  changedScopes: readonly ChangedScope[],
): string {
  return [operationID, saveRevision, ...changedScopes].join(" ");
}

export function useSessionChangedSync(
  saveSessionID: string | undefined,
  onSession: (session: SaveSession) => void = () => {},
): SessionChangedSync {
  const port = useSaveSessionPort();
  const queryClient = useQueryClient();
  // The last sequence this listener processed, and the identities of the
  // mutations this frontend already refreshed itself. Both are refs: they are
  // listener bookkeeping and must never re-render a component or restart the
  // subscription.
  const lastSequence = useRef("0");
  // The sequence of the session state last handed to the owner. It is tracked
  // separately from the event baseline because an ordinary event advances the
  // baseline before the confirmed session arrives, and the answer that confirms
  // that very position still has to be applied.
  const appliedSequence = useRef("0");
  const appliedMutations = useRef(new Set<string>());
  // The owner's state update, held in a ref so an inline callback does not
  // restart the subscription on every render of the component that owns it.
  const publishSession = useRef(onSession);
  publishSession.current = onSession;

  const noteAppliedMutation = useCallback(
    (receipt: {
      operationID: string;
      saveRevision: string;
      changedScopes: readonly ChangedScope[];
    }) => {
      if (appliedMutations.current.size >= 128) {
        const oldest = appliedMutations.current.values().next().value;
        if (oldest !== undefined) {
          appliedMutations.current.delete(oldest);
        }
      }
      appliedMutations.current.add(
        mutationIdentity(receipt.operationID, receipt.saveRevision, receipt.changedScopes),
      );
    },
    [],
  );

  useEffect(() => {
    if (saveSessionID === undefined || saveSessionID === "") {
      return;
    }
    let active = true;

    // A hook instance survives replacement of one save by another. Listener
    // bookkeeping does not: sequences and operation identities are scoped to a
    // session and must never leak across that boundary.
    lastSequence.current = "0";
    appliedSequence.current = "0";
    appliedMutations.current.clear();

    const invalidateScopes = (scopes: readonly ChangedScope[]) => {
      const patterns = queryKeyPatternsForScopes(saveSessionID, scopes);
      if (patterns.length === 0) {
        return;
      }
      void queryClient.invalidateQueries({
        predicate: (query) =>
          patterns.some((pattern) => matchesQueryKeyPattern(query.queryKey, pattern)),
      });
    };

    // Reading the session back is what makes a lost event recoverable: the
    // backend reports the authoritative sequence, so the listener never has to
    // guess how many events it missed. It is also the single point at which the
    // active session's own state changes, so its revision, its unsaved-changes
    // flag and its sequence always come from one confirmed answer.
    //
    // An answer is applied only when it is strictly newer than the state already
    // published. Resynchronisations overlap by design — a start, a visible
    // window, a gap, an unreadable payload, an ordinary event — and the older
    // reply may land last; accepting it would replace confirmed newer session
    // state with stale state and move the event baseline backwards. An equal
    // sequence is not newer either, so a start on a session that changed nothing
    // costs one getter call, no state update and no invalidation at all.
    //
    // The session-wide refresh is the narrower case: it happens only when the
    // confirmed sequence moved past the event baseline, which means events were
    // missed. An answer that merely confirms the position an ordinary event
    // already reported updates the session and invalidates nothing extra.
    const resynchronise = async () => {
      try {
        const session = await port.getLoadedSave(saveSessionID);
        if (!active || session.saveSessionID !== saveSessionID) {
          return;
        }
        if (
          !isCanonicalDecimal(session.eventSequence) ||
          compareSequences(session.eventSequence, appliedSequence.current) <= 0
        ) {
          return;
        }
        appliedSequence.current = session.eventSequence;
        publishSession.current(session);
        if (compareSequences(session.eventSequence, lastSequence.current) > 0) {
          lastSequence.current = session.eventSequence;
          void queryClient.invalidateQueries({ queryKey: queryKeys.saveSession(saveSessionID) });
        }
        // LoadSave seeds this cache entry, so every later authoritative read
        // keeps it coherent even though the active flow owns the render state.
        queryClient.setQueryData(queryKeys.loadedSave(saveSessionID), session);
      } catch {
        // A failed resynchronisation leaves the baseline alone. The next event
        // still reads as a gap, so the attempt simply repeats; it never silently
        // accepts a position the backend did not confirm.
      }
    };

    const handle = (event: SessionChangedEvent | null) => {
      if (!active) {
        return;
      }
      // A payload the adapter could not validate is a notification that cannot
      // be read: something committed and what it touched is unknown. It is
      // treated exactly like a gap and never applied in part.
      if (event === null) {
        void resynchronise();
        return;
      }
      if (event.saveSessionID !== saveSessionID) {
        return;
      }
      // A duplicate and an out-of-order event are both "not newer than what has
      // already been processed", so one comparison covers them.
      if (compareSequences(event.sequence, lastSequence.current) <= 0) {
        return;
      }
      // A gap means events may have been lost, so the scopes of this one event
      // are not enough to describe what changed. The baseline is deliberately
      // left alone here: the resynchronisation is what moves it, and only after
      // the backend confirmed the position.
      const expected = nextSequence(lastSequence.current);
      if (expected === null || event.sequence !== expected) {
        void resynchronise();
        return;
      }
      lastSequence.current = event.sequence;

      // The session itself is always re-read: the event names the new revision
      // but is not a state document, and the revision, the unsaved-changes flag
      // and the sequence have to stay one consistent answer of the backend.
      void resynchronise();

      const identity = mutationIdentity(event.operationID, event.saveRevision, event.changedScopes);
      if (appliedMutations.current.delete(identity)) {
        // The initiator of this mutation already refreshed from its receipt, so
        // the event does not invalidate its scopes a second time.
        return;
      }
      invalidateScopes(event.changedScopes);
    };

    const unsubscribe = port.subscribeSessionChanged(handle);
    void resynchronise();

    const onVisible = () => {
      if (document.visibilityState === "visible") {
        void resynchronise();
      }
    };
    document.addEventListener("visibilitychange", onVisible);

    return () => {
      active = false;
      document.removeEventListener("visibilitychange", onVisible);
      unsubscribe();
    };
  }, [port, queryClient, saveSessionID]);

  return { noteAppliedMutation };
}

/**
 * The sequence that must follow `value`, or `null` when it cannot be computed
 * exactly. Incrementing a canonical decimal string is done digit by digit, so
 * no value is ever routed through a JavaScript number.
 */
export function nextSequence(value: string): string | null {
  if (!isCanonicalDecimal(value)) {
    return null;
  }
  const digits = value.split("");
  let index = digits.length - 1;
  while (index >= 0) {
    const digit = Number(digits[index]);
    if (digit < 9) {
      digits[index] = String(digit + 1);
      return digits.join("");
    }
    digits[index] = "0";
    index -= 1;
  }
  return `1${digits.join("")}`;
}
