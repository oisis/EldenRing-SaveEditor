import { Trans, useLingui } from "@lingui/react/macro";
import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { useDiagnosticsPort } from "../../application/diagnostics/diagnosticsClient";
import type { RepairPlan } from "../../application/diagnostics/diagnosticsPort";
import {
  aggregateValidationReports,
  reportAnswersFor,
  saveValidationReportQuery,
} from "../../application/diagnostics/useSaveValidationReports";
import {
  useApplyRepairs,
  useRepairPlan,
} from "../../application/diagnostics/useSaveRepairs";
import type { MutationReceipt } from "../../application/save-session/saveSessionPort";
import { Button } from "../../ui/components/Button/Button";
import { Card } from "../../ui/components/Card/Card";
import { Checkbox } from "../../ui/components/Checkbox/Checkbox";
import { alert, message } from "../../ui/patterns/panel.css";
import {
  actionsBar,
  finding,
  findingMeta,
  findings,
  findingText,
  sections,
} from "./ToolsPanel.css";

export type SaveIntegrityCardProps = {
  saveSessionID?: string | undefined;
  saveRevision?: string | undefined;
  characterID?: number | undefined;
  /**
   * The shared save-mutation path. It is required: a repair may not be
   * committed without the session refresh that follows it.
   */
  applyMutationReceipt: (receipt: MutationReceipt) => Promise<unknown>;
  sessionBusy?: boolean | undefined;
};

/** The plan together with the exact selection it was derived from. */
type PlanState = {
  plan: RepairPlan;
  issueIDs: readonly string[];
};

/**
 * The scan and repair section of `Tools → Settings`.
 *
 * Every verdict on screen is the backend's. This component runs no check of its
 * own, proposes no repair the plan does not contain and never turns an empty
 * finding list into "this save is clean" while a scope stayed unchecked. The
 * scan is started by the user, and the whole state — report, plan and selection
 * — belongs to one exact session, slot and revision. The parent remounts this
 * card whenever any of the three changes, which is what discards the previous
 * report, plan, selection and call state; the identity checks below are the
 * second line of that same rule, so a cached answer to another question can
 * never be read as this one.
 */
export function SaveIntegrityCard({
  saveSessionID,
  saveRevision,
  characterID,
  applyMutationReceipt,
  sessionBusy = false,
}: SaveIntegrityCardProps) {
  const { t } = useLingui();
  const port = useDiagnosticsPort();
  const repairPlan = useRepairPlan();
  const applyRepairs = useApplyRepairs();

  const ready =
    saveSessionID !== undefined && saveRevision !== undefined && characterID !== undefined;
  const [scanRequested, setScanRequested] = useState(false);
  const [selected, setSelected] = useState<readonly string[]>([]);
  const [planState, setPlanState] = useState<PlanState | undefined>(undefined);
  const [applied, setApplied] = useState(false);
  const [nothingApplied, setNothingApplied] = useState(false);
  const [syncFailed, setSyncFailed] = useState(false);

  const reportQuery = useQuery({
    ...saveValidationReportQuery(port, saveSessionID ?? "", saveRevision ?? "", characterID ?? -1),
    enabled: ready && scanRequested,
  });

  // A report is shown only when it answers about the state now on screen. This
  // repeats the identity rule of the shared aggregation deliberately: a cached
  // entry that arrived for another question may never be read as this answer.
  const report =
    ready &&
    scanRequested &&
    reportQuery.data !== undefined &&
    reportAnswersFor(reportQuery.data, saveSessionID, saveRevision, characterID)
      ? reportQuery.data
      : undefined;

  const totals =
    report === undefined || !ready
      ? undefined
      : aggregateValidationReports([report], saveSessionID, saveRevision, [characterID]);

  const plan =
    planState !== undefined &&
    ready &&
    planState.plan.saveSessionID === saveSessionID &&
    planState.plan.saveRevision === saveRevision &&
    planState.plan.characterID === characterID
      ? planState
      : undefined;

  const busy =
    sessionBusy || reportQuery.isFetching || repairPlan.isPending || applyRepairs.isPending;

  function toggle(issueID: string) {
    setPlanState(undefined);
    setApplied(false);
    setNothingApplied(false);
    setSyncFailed(false);
    setSelected((current) =>
      current.includes(issueID)
        ? current.filter((id) => id !== issueID)
        : [...current, issueID],
    );
  }

  async function buildPlan() {
    if (!ready || selected.length === 0) return;
    setApplied(false);
    setNothingApplied(false);
    setSyncFailed(false);
    const issueIDs = [...selected];
    let result;
    try {
      result = await repairPlan.mutateAsync({
        saveSessionID,
        characterID,
        saveRevision,
        issueIDs,
      });
    } catch {
      // The rejection is the hook's own error state, which is already shown
      // below. It is caught here only so it never escapes as an unhandled one.
      return;
    }
    setPlanState({ plan: result, issueIDs });
  }

  async function applyPlan() {
    if (!ready || plan === undefined || plan.plan.actions.length === 0) return;
    let result;
    try {
      result = await applyRepairs.mutateAsync({
        saveSessionID,
        characterID,
        issueIDs: plan.issueIDs,
        planToken: plan.plan.planToken,
        expectedRevision: saveRevision,
      });
    } catch {
      // A rejected call changed nothing; the hook's error state says so.
      return;
    }
    if (!result.applied) {
      // The backend verified the selection and executed nothing. It named no
      // cause, so none is invented here.
      setNothingApplied(true);
      return;
    }
    // The mutation is committed from here on, so the screen says so even if the
    // refresh below fails. The receipt goes through the one shared
    // save-mutation path, exactly like any other committed mutation; it moves
    // the revision, which retires this report, this plan and this selection.
    setApplied(true);
    try {
      await applyMutationReceipt(result);
    } catch {
      setSyncFailed(true);
    }
  }

  return (
    <Card aria-label={t`Save integrity`} className={sections}>
      <h3>
        <Trans>Save integrity</Trans>
      </h3>

      {!ready ? (
        <p className={message}>
          <Trans>Open a save and select a character to scan it for problems.</Trans>
        </p>
      ) : null}

      <div className={actionsBar}>
        <Button
          tone="accent"
          disabled={!ready || busy}
          onClick={() => {
            setSelected([]);
            setPlanState(undefined);
            setApplied(false);
            setNothingApplied(false);
            setSyncFailed(false);
            setScanRequested(true);
          }}
        >
          <Trans>Scan for problems</Trans>
        </Button>
      </div>

      {reportQuery.isError ? (
        <p role="alert" className={alert}>
          <Trans>The save validation report is unavailable.</Trans>
        </p>
      ) : null}

      {scanRequested && report === undefined && !reportQuery.isError ? (
        <p className={message}>
          <Trans>Scanning…</Trans>
        </p>
      ) : null}

      {report !== undefined && totals !== undefined ? (
        <>
          <p className={message}>
            <Trans>
              Errors: {totals.errorCount}. Warnings: {totals.warningCount}.
            </Trans>
          </p>
          {totals.uncheckedScopes > 0 || totals.unresolvedRecords > 0 ? (
            <p className={message}>
              <Trans>
                {totals.uncheckedScopes} scope(s) could not be checked and{" "}
                {totals.unresolvedRecords} record(s) stayed unresolved, so this save is not
                confirmed clean.
              </Trans>
            </p>
          ) : null}
          <ul className={findings}>
            {report.coverage.map((scope) => (
              <li key={scope.scope} className={findingMeta}>
                {scope.checked ? (
                  <Trans>
                    {scope.scope}: {scope.recordsChecked} record(s) checked,{" "}
                    {scope.unresolvedRecords} unresolved.
                  </Trans>
                ) : (
                  <Trans>
                    {scope.scope}: not checked. {scope.reason}
                  </Trans>
                )}
              </li>
            ))}
          </ul>

          {report.issues.length === 0 ? (
            <p className={message}>
              {totals.verdict === "clean" ? (
                <Trans>The backend reported no problems and checked every scope.</Trans>
              ) : (
                <Trans>The backend reported no problems, but its coverage is incomplete.</Trans>
              )}
            </p>
          ) : (
            <ul className={findings} aria-label={t`Reported problems`}>
              {report.issues.map((issue) => (
                <li key={issue.id} className={finding}>
                  <Checkbox
                    id={`repair-${issue.id}`}
                    checked={selected.includes(issue.id)}
                    disabled={busy}
                    onChange={() => toggle(issue.id)}
                  />
                  <span className={findingText}>
                    <label htmlFor={`repair-${issue.id}`}>{issue.message}</label>
                    <span className={findingMeta}>
                      <Trans>
                        {issue.severity} · {issue.scope} · {issue.code}
                      </Trans>
                    </span>
                  </span>
                </li>
              ))}
            </ul>
          )}

          <div className={actionsBar}>
            <Button disabled={busy || selected.length === 0} onClick={() => void buildPlan()}>
              <Trans>Preview repair plan</Trans>
            </Button>
          </div>
        </>
      ) : null}

      {repairPlan.isError ? (
        <p role="alert" className={alert}>
          <Trans>The repair plan could not be built.</Trans>
        </p>
      ) : null}

      {planState !== undefined && plan === undefined ? (
        <p role="alert" className={alert}>
          <Trans>
            The repair plan describes another save state, so it was discarded. Scan the save
            again.
          </Trans>
        </p>
      ) : null}

      {plan !== undefined ? (
        <>
          <h4>
            <Trans>Repair plan</Trans>
          </h4>
          {plan.plan.actions.length === 0 ? (
            <p className={message}>
              <Trans>The backend planned no repair for this selection.</Trans>
            </p>
          ) : (
            <ul className={findings} aria-label={t`Planned repairs`}>
              {plan.plan.actions.map((action) => (
                <li key={action.issueIDs.join(",")} className={finding}>
                  <span className={findingText}>
                    <span>{action.description}</span>
                    <span className={findingMeta}>
                      <Trans>
                        {action.scope} · {action.operation}
                      </Trans>
                    </span>
                  </span>
                </li>
              ))}
            </ul>
          )}
          {plan.plan.rejected.length > 0 ? (
            <>
              <h4>
                <Trans>Not repaired</Trans>
              </h4>
              <ul className={findings} aria-label={t`Rejected repairs`}>
                {plan.plan.rejected.map((rejection) => (
                  <li key={rejection.issueID} className={finding}>
                    <span className={findingText}>
                      <span>{rejection.reason}</span>
                      <span className={findingMeta}>
                        <Trans>
                          {rejection.scope} · {rejection.code}
                        </Trans>
                      </span>
                    </span>
                  </li>
                ))}
              </ul>
            </>
          ) : null}
          <div className={actionsBar}>
            <Button
              tone="accent"
              disabled={busy || plan.plan.actions.length === 0}
              onClick={() => void applyPlan()}
            >
              <Trans>Apply planned repairs</Trans>
            </Button>
          </div>
        </>
      ) : null}

      {applyRepairs.isError ? (
        <p role="alert" className={alert}>
          <Trans>The planned repairs could not be applied.</Trans>
        </p>
      ) : null}
      {nothingApplied ? (
        <p role="alert" className={alert}>
          <Trans>The backend applied nothing, so this save is unchanged.</Trans>
        </p>
      ) : null}
      {applied ? (
        <p role="status" className={message}>
          <Trans>The planned repairs were applied.</Trans>
        </p>
      ) : null}
      {syncFailed ? (
        <p role="alert" className={alert}>
          <Trans>
            The repairs were applied, but this screen could not be refreshed. Reopen the save
            to see its current state.
          </Trans>
        </p>
      ) : null}
    </Card>
  );
}
