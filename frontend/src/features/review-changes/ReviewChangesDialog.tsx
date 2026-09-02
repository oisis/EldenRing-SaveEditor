import { Trans, useLingui } from "@lingui/react/macro";
import { useMemo, useState } from "react";
import type {
  OperationRecord,
  OperationRisk,
} from "../../application/save-session/saveSessionPort";
import { Badge } from "../../ui/components/Badge/Badge";
import { Button } from "../../ui/components/Button/Button";
import { Checkbox } from "../../ui/components/Checkbox/Checkbox";
import { Dialog } from "../../ui/components/Dialog/Dialog";
import { Select } from "../../ui/components/Select/Select";
import { message } from "../../ui/patterns/panel.css";
import type { SaveSessionFlow } from "../save-session/useSaveSessionFlow";
import {
  actions,
  group,
  groupHeading,
  item,
  itemBody,
  list,
  metadata,
  state,
  summary,
  toolbar,
  validation,
  validationList,
} from "./ReviewChangesDialog.css";

type RiskFilter = "all" | OperationRisk;
type GroupBy = "none" | "area" | "character";

export function ReviewChangesDialog({ flow }: { flow: SaveSessionFlow }) {
  const { t } = useLingui();
  const [risk, setRisk] = useState<RiskFilter>("all");
  const [groupBy, setGroupBy] = useState<GroupBy>("none");
  const [confirmations, setConfirmations] = useState({
    validationToken: undefined as string | undefined,
    warnings: false,
    banRisk: false,
  });
  const operations = useMemo(
    () =>
      (flow.history?.operations ?? []).filter(
        (operation) => risk === "all" || operation.risk === risk,
      ),
    [flow.history?.operations, risk],
  );
  const result = flow.reviewValidation;
  const groups = useMemo(() => {
    const grouped = new Map<string, OperationRecord[]>();
    for (const operation of operations) {
      const key =
        groupBy === "area"
          ? operation.area
          : groupBy === "character"
            ? operation.characterID === undefined
              ? t`Session`
              : `${t`Character`} ${operation.characterID}`
            : t`All operations`;
      grouped.set(key, [...(grouped.get(key) ?? []), operation]);
    }
    return [...grouped.entries()];
  }, [groupBy, operations, t]);
  const validationToken = result?.validationToken;
  const validationCurrent =
    result !== undefined &&
    result.saveSessionID === flow.session?.saveSessionID &&
    result.saveRevision === flow.session?.saveRevision;
  const warningsConfirmed =
    confirmations.validationToken === validationToken && confirmations.warnings;
  const banRiskConfirmed =
    confirmations.validationToken === validationToken && confirmations.banRisk;
  const saveEnabled =
    validationCurrent &&
    result.valid === true &&
    result.validationToken !== undefined &&
    (result.warningCount === 0 || warningsConfirmed) &&
    (result.banRiskCount === 0 || banRiskConfirmed) &&
    !flow.isBusy;

  return (
    <Dialog
      open={flow.reviewOpen}
      onOpenChange={(open) => {
        if (!open && !flow.isBusy) flow.closeReview();
      }}
      title={<Trans>Review Changes</Trans>}
      description={
        <Trans>Review the ordered operation journal and validation before writing.</Trans>
      }
      closeLabel={<Trans>Close</Trans>}
    >
      <div className={toolbar}>
        <label htmlFor="review-risk">
          <Trans>Risk</Trans>
        </label>
        <Select
          id="review-risk"
          value={risk}
          onChange={(event) => setRisk(event.currentTarget.value as RiskFilter)}
        >
          <option value="all">{t`All levels`}</option>
          <option value="normal">{t`Normal`}</option>
          <option value="warning">{t`Warning`}</option>
          <option value="ban risk">{t`Ban risk`}</option>
          <option value="critical">{t`Critical`}</option>
        </Select>
        <label htmlFor="review-group">
          <Trans>Group by</Trans>
        </label>
        <Select
          id="review-group"
          value={groupBy}
          onChange={(event) => setGroupBy(event.currentTarget.value as GroupBy)}
        >
          <option value="none">{t`No grouping`}</option>
          <option value="area">{t`Area`}</option>
          <option value="character">{t`Character`}</option>
        </Select>
      </div>

      <p className={summary}>
        <Badge>
          {flow.history?.operations.length ?? 0} <Trans>operation(s)</Trans>
        </Badge>
        <Badge>
          {flow.history?.undoCount ?? 0} <Trans>undo</Trans>
        </Badge>
        <Badge>
          {flow.history?.redoCount ?? 0} <Trans>redo</Trans>
        </Badge>
      </p>

      {operations.length === 0 ? (
        <p className={message}>
          <Trans>No operations match this view.</Trans>
        </p>
      ) : (
        <section className={list} aria-label={t`Operation history`}>
          {groups.map(([label, entries]) => (
            <section key={label} className={group}>
              {groupBy !== "none" && <h3 className={groupHeading}>{label}</h3>}
              <ol className={list}>
                {entries.map((operation) => (
                  <li key={operation.operationID} className={item}>
                    <div className={itemBody}>
                      <strong>{operation.description}</strong>
                      <span className={metadata}>
                        #{operation.order} · {operation.area} · {operation.risk} ·{" "}
                        {operation.changedByteCount} B
                      </span>
                      <p className={state}>
                        {operation.beforeState} → {operation.afterState}
                      </p>
                      {operation.riskReason !== "" && (
                        <p className={state}>{operation.riskReason}</p>
                      )}
                    </div>
                    <Button
                      size="sm"
                      disabled={flow.isBusy}
                      onClick={() => flow.revertOperation(operation.operationID)}
                    >
                      <Trans>Revert</Trans>
                    </Button>
                  </li>
                ))}
              </ol>
            </section>
          ))}
        </section>
      )}

      <section className={validation} aria-label={t`Save validation`}>
        <strong>
          <Trans>Validation</Trans>
        </strong>
        {flow.isBusy && result === undefined && (
          <p className={message}>
            <Trans>Validating…</Trans>
          </p>
        )}
        {result !== undefined && validationCurrent && (
          <>
            <p className={summary}>
              <Badge>
                {result.warningCount} <Trans>warning(s)</Trans>
              </Badge>
              <Badge>
                {result.banRiskCount} <Trans>ban risk</Trans>
              </Badge>
              <Badge>
                {result.criticalCount} <Trans>critical</Trans>
              </Badge>
            </p>
            <p className={message}>
              {result.valid ? (
                <Trans>Validation passed.</Trans>
              ) : (
                <Trans>Critical validation issues block saving.</Trans>
              )}
            </p>
            {result.issues.length > 0 && (
              <ul className={validationList}>
                {result.issues.map((issue) => (
                  <li key={`${issue.code}-${issue.operationID ?? "session"}`}>
                    {issue.severity}: {issue.message}
                  </li>
                ))}
              </ul>
            )}
            {result.warningCount > 0 && (
              <label htmlFor="review-confirm-warnings">
                <Checkbox
                  id="review-confirm-warnings"
                  checked={warningsConfirmed}
                  onChange={(event) =>
                    setConfirmations((current) => ({
                      validationToken,
                      warnings: event.currentTarget.checked,
                      banRisk:
                        current.validationToken === validationToken ? current.banRisk : false,
                    }))
                  }
                />{" "}
                <Trans>I reviewed and accept the warnings.</Trans>
              </label>
            )}
            {result.banRiskCount > 0 && (
              <label htmlFor="review-confirm-ban-risk">
                <Checkbox
                  id="review-confirm-ban-risk"
                  checked={banRiskConfirmed}
                  onChange={(event) =>
                    setConfirmations((current) => ({
                      validationToken,
                      warnings:
                        current.validationToken === validationToken ? current.warnings : false,
                      banRisk: event.currentTarget.checked,
                    }))
                  }
                />{" "}
                <Trans>I understand and accept the separate ban risk.</Trans>
              </label>
            )}
          </>
        )}
        {result !== undefined && !validationCurrent && (
          <div>
            <p className={message}>
              <Trans>
                The session changed after validation. Validate the current revision again.
              </Trans>
            </p>
            <Button disabled={flow.isBusy} onClick={flow.openReview}>
              <Trans>Validate again</Trans>
            </Button>
          </div>
        )}
      </section>

      <div className={actions}>
        <Button
          disabled={!saveEnabled || flow.session?.sourceKind !== "local"}
          onClick={() => flow.saveReviewed(false, warningsConfirmed, banRiskConfirmed)}
        >
          <Trans>Save</Trans>
        </Button>
        <Button
          tone="accent"
          disabled={!saveEnabled}
          onClick={() => flow.saveReviewed(true, warningsConfirmed, banRiskConfirmed)}
        >
          <Trans>Save As</Trans>
        </Button>
      </div>
    </Dialog>
  );
}

export function PendingChangesDialog({ flow }: { flow: SaveSessionFlow }) {
  return (
    <Dialog
      open={flow.pendingSessionAction !== undefined && !flow.reviewOpen}
      onOpenChange={(open) => {
        if (!open && !flow.isBusy) flow.cancelPendingAction();
      }}
      title={<Trans>Unsaved changes</Trans>}
      description={<Trans>Save or discard the current changes before continuing.</Trans>}
      closeLabel={<Trans>Cancel</Trans>}
    >
      <div className={actions}>
        <Button disabled={flow.isBusy} onClick={flow.discardPendingChanges}>
          <Trans>Discard</Trans>
        </Button>
        <Button tone="accent" disabled={flow.isBusy} onClick={flow.savePendingChanges}>
          <Trans>Save…</Trans>
        </Button>
      </div>
    </Dialog>
  );
}
