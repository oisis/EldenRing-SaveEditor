import { useMutation } from "@tanstack/react-query";
import { useDiagnosticsPort } from "./diagnosticsClient";
import type {
  ApplyRepairsRequest,
  ApplyRepairsResult,
  RepairPlan,
  RepairPlanRequest,
} from "./diagnosticsPort";

/**
 * Builds the repair plan for an explicit selection of findings.
 *
 * It is a mutation hook for a non-mutating call on purpose: a plan is requested
 * by an explicit user action for one exact selection, so it is never cached,
 * never refetched in the background and never revalidated behind the user. The
 * caller keeps the answer only for as long as the session, the slot and the
 * revision it was derived for are still the ones on screen.
 */
export function useRepairPlan() {
  const port = useDiagnosticsPort();

  return useMutation<RepairPlan, Error, RepairPlanRequest>({
    mutationFn: (request) => port.getRepairPlan(request),
  });
}

/**
 * Executes one sealed plan. The receipt is handed back to the caller, which
 * routes it through the shared save-mutation path; this hook invalidates
 * nothing itself, so repairs refresh the application exactly like every other
 * committed mutation.
 */
export function useApplyRepairs() {
  const port = useDiagnosticsPort();

  return useMutation<ApplyRepairsResult, Error, ApplyRepairsRequest>({
    mutationFn: (request) => port.applyRepairs(request),
  });
}
