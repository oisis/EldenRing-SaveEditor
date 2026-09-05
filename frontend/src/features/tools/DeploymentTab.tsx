import { Trans, useLingui } from "@lingui/react/macro";
import { useEffect, useId, useState } from "react";
import type {
  DeploymentOperationResult,
  DeploymentProgress,
  DeploymentTarget,
  DeploymentTargetInput,
} from "../../application/deployment/deploymentPort";
import { useDeploymentPort } from "../../application/deployment/deploymentClient";
import {
  useDeploymentGameStatus,
  useDeploymentOperations,
  useDeploymentTargetMutations,
  useDeploymentTargets,
} from "../../application/deployment/useDeployment";
import { useSaveSessionPort } from "../../application/save-session/saveSessionClient";
import { useHostSettings, useSetHostSettings } from "../../application/settings/useHostSettings";
import { Badge } from "../../ui/components/Badge/Badge";
import { Button } from "../../ui/components/Button/Button";
import { Card } from "../../ui/components/Card/Card";
import { Dialog } from "../../ui/components/Dialog/Dialog";
import { Input } from "../../ui/components/Input/Input";
import { Select } from "../../ui/components/Select/Select";
import { Table, TableFrame } from "../../ui/components/Table/Table";
import { actionCell, alert, message, tableFrame } from "../../ui/patterns/panel.css";
import {
  actionsBar,
  formGrid,
  sections,
  settingField,
  stageList,
} from "./ToolsPanel.css";

/**
 * A fresh operation identifier. It only has to be unique inside this process
 * for as long as one operation runs, which is exactly what the backend uses it
 * for: correlating progress and answering a cancellation.
 */
function newOperationID(): string {
  return `operation-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
}

/** The five deployment actions that need the prepared save or the target. */
type PendingOperation = {
  kind: "upload" | "deploy-and-launch" | "download" | "close-and-download";
};

/** Deploy & Launch is Upload plus the launch step; the kind states which. */
const launchesAfterUpload = (operation: PendingOperation) =>
  operation.kind === "deploy-and-launch";
/** Close & Download is Download plus the stop step. */
const closesBeforeDownload = (operation: PendingOperation) =>
  operation.kind === "close-and-download";

export type DeploymentTabProps = {
  saveSessionID?: string | undefined;
  saveRevision?: string | undefined;
  sessionBusy: boolean;
  /** Replaces the active session with a downloaded staging file. */
  onDownloadedSave: (localPath: string) => void;
};

/**
 * `Tools → Deployment`.
 *
 * Every operation is an explicit user action: nothing here contacts a target on
 * mount, on focus or on an interval. The blocked outcomes the backend reports
 * are answered with an explicit confirmation and retried; none of them is
 * defaulted to yes.
 */
export function DeploymentTab({
  saveSessionID,
  saveRevision,
  sessionBusy,
  onDownloadedSave,
}: DeploymentTabProps) {
  const { t } = useLingui();
  const port = useDeploymentPort();
  const sessionPort = useSaveSessionPort();
  const hostSettings = useHostSettings();
  const setHostSettings = useSetHostSettings();
  const targets = useDeploymentTargets();
  const targetMutations = useDeploymentTargetMutations();
  const operations = useDeploymentOperations();

  const [selectedID, setSelectedID] = useState<string>();
  const [editing, setEditing] = useState<DeploymentTargetInput>();
  const [pendingDelete, setPendingDelete] = useState<DeploymentTarget>();
  const [pending, setPending] = useState<PendingOperation>();
  const [operationID, setOperationID] = useState<string>();
  const [progress, setProgress] = useState<DeploymentProgress>();
  const [result, setResult] = useState<DeploymentOperationResult>();
  const [failure, setFailure] = useState<string>();
  const [review, setReview] = useState<{
    validation: { warningCount: number; banRiskCount: number; criticalCount: number };
    operation: PendingOperation;
    confirmations: {
      continueWithUnknownGameStatus?: boolean | undefined;
      confirmRemoteBackup?: boolean | undefined;
      confirmStopGame?: boolean | undefined;
    };
  }>();

  const selected = (targets.data?.targets ?? []).find((target) => target.id === selectedID);
  const gameStatus = useDeploymentGameStatus(selected?.id);

  // The backend emits only while an operation this screen started is running,
  // so the subscription costs nothing when the tab is idle.
  useEffect(() => {
    return port.subscribeDeploymentProgress((next) => {
      setProgress((current) =>
        current !== undefined && current.operationID !== next.operationID && !next.finished
          ? current
          : next,
      );
    });
  }, [port]);

  const sessionReady = saveSessionID !== undefined && saveRevision !== undefined;
  const transferReady = selected?.transferSupported === true;
  const busy = sessionBusy || operations.deploy.isPending || operations.download.isPending;

  /**
   * Runs one operation, having first completed the validation the deployment
   * specification requires.
   *
   * The validation is never skipped. What the host setting can skip is the
   * Review Changes modal, and only when the completed validation reported no
   * warning, no ban risk and no critical finding.
   */
  async function run(
    operation: PendingOperation,
    confirmations: {
      continueWithUnknownGameStatus?: boolean;
      confirmRemoteBackup?: boolean;
      confirmStopGame?: boolean;
    },
    reviewed = false,
  ) {
    if (selected === undefined || busy) return;
    setFailure(undefined);
    const id = operationID ?? newOperationID();
    setOperationID(id);

    if (operation.kind === "download" || operation.kind === "close-and-download") {
      try {
        const outcome = await operations.download.mutateAsync({
          operationID: id,
          targetID: selected.id,
          closeGameFirst: closesBeforeDownload(operation),
          ...confirmations,
        });
        setResult(outcome);
        if (outcome.completed && outcome.localPath !== undefined) {
          onDownloadedSave(outcome.localPath);
        }
      } catch {
        setFailure(t`The download failed and the target was left unchanged.`);
      }
      return;
    }

    if (!sessionReady) return;
    let validation;
    try {
      validation = await sessionPort.validateReviewChanges(saveSessionID, saveRevision);
    } catch {
      setFailure(t`The save could not be validated, so nothing was deployed.`);
      return;
    }
    if (!validation.valid || validation.validationToken === undefined) {
      setFailure(t`The save did not pass validation, so nothing was deployed.`);
      return;
    }
    const normalRisk =
      validation.warningCount === 0 &&
      validation.banRiskCount === 0 &&
      validation.criticalCount === 0;
    const maySkipReview =
      normalRisk && hostSettings.data?.skipReviewForNormalRisk === true;
    if (!reviewed && !maySkipReview) {
      // Warnings and ban risk always reach the user, whatever the setting says.
      setPending(operation);
      setReview({ validation, operation, confirmations });
      return;
    }

    try {
      const outcome = await operations.deploy.mutateAsync({
        operationID: id,
        targetID: selected.id,
        saveSessionID,
        expectedRevision: saveRevision,
        validationToken: validation.validationToken,
        confirmWarnings: validation.warningCount > 0,
        confirmBanRisk: validation.banRiskCount > 0,
        launchAfter: launchesAfterUpload(operation),
        ...confirmations,
      });
      setResult(outcome);
    } catch {
      setFailure(t`The deployment failed and the target was left unchanged.`);
    }
  }

  function retryWith(confirmation: Record<string, boolean>) {
    if (pending === undefined) return;
    const operation = pending;
    setResult(undefined);
    void run(operation, { ...confirmation }, true);
  }

  return (
    <div className={sections}>
      <Card aria-label={t`Deployment targets`} className={sections}>
        <h2>
          <Trans>Deployment targets</Trans>
        </h2>
        {targets.isError ? (
          <p role="alert" className={alert}>
            <Trans>The deployment configuration could not be read.</Trans>
          </p>
        ) : null}
        <TableFrame className={tableFrame}>
          <Table>
            <caption>{t`Deployment targets`}</caption>
            <thead>
              <tr>
                <th scope="col">{t`Name`}</th>
                <th scope="col">{t`Type`}</th>
                <th scope="col">{t`Save path`}</th>
                <th scope="col">{t`Host key`}</th>
                <th scope="col" className={actionCell}>
                  {t`Actions`}
                </th>
              </tr>
            </thead>
            <tbody>
              {(targets.data?.targets ?? []).map((target) => (
                <tr key={target.id}>
                  <th scope="row">{target.name}</th>
                  <td>{target.kind}</td>
                  <td>{target.savePath}</td>
                  <td>
                    {target.kind === "ssh" ? (
                      target.hostKeyTrusted ? (
                        <Badge mono>{target.hostKeyFingerprint ?? ""}</Badge>
                      ) : (
                        <Trans>Not approved yet</Trans>
                      )
                    ) : (
                      "—"
                    )}
                  </td>
                  <td className={actionCell}>
                    <span className={actionsBar}>
                      <Button
                        size="sm"
                        pressed={selectedID === target.id}
                        onClick={() => {
                          setSelectedID(target.id);
                          setResult(undefined);
                          setFailure(undefined);
                        }}
                      >
                        <Trans>Select</Trans>
                      </Button>
                      <Button size="sm" onClick={() => setEditing({ ...target, targetID: target.id })}>
                        <Trans>Edit</Trans>
                      </Button>
                      <Button size="sm" onClick={() => setPendingDelete(target)}>
                        <Trans>Delete</Trans>
                      </Button>
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </Table>
        </TableFrame>
        <div className={actionsBar}>
          <Button
            size="sm"
            onClick={() =>
              setEditing({ name: "", kind: "local", savePath: "", port: 22 })
            }
          >
            <Trans>Add a target</Trans>
          </Button>
        </div>
      </Card>

      {selected !== undefined ? (
        <Card aria-label={t`Target operations`} className={sections}>
          <h2>
            <Trans>Operations</Trans>: {selected.name}
          </h2>

          <p className={message}>
            <Trans>Game state</Trans>:{" "}
            <Badge mono>{gameStatus.data ?? t`unknown`}</Badge>
          </p>
          {gameStatus.data === "unknown" ? (
            <p role="status" className={message}>
              {/* Section 4 of the deployment specification: an unknown state is
                  warned about and may be continued explicitly. It is never
                  silently treated as stopped. */}
              <Trans>
                SaveForge cannot confirm whether the game is running on this target. Continuing
                while it runs can corrupt the save, so every operation asks first.
              </Trans>
            </p>
          ) : null}
          {!selected.transferSupported ? (
            <p role="alert" className={alert}>
              {/* The backend's own sentence about what is missing. The actions
                  below stay disabled rather than offering a call that must
                  fail. */}
              {selected.unsupportedReason}
            </p>
          ) : null}

          <div className={actionsBar}>
            <Button
              size="sm"
              disabled={targetMutations.test.isPending}
              onClick={() => targetMutations.test.mutate(selected.id)}
            >
              <Trans>Test the target</Trans>
            </Button>
            <Button
              size="sm"
              disabled={busy || !transferReady}
              onClick={() => operations.launch.mutate(selected.id)}
            >
              <Trans>Launch</Trans>
            </Button>
            <Button
              size="sm"
              disabled={busy || !transferReady}
              onClick={() => operations.closeGame.mutate(selected.id)}
            >
              <Trans>Close Game</Trans>
            </Button>
          </div>

          <div className={actionsBar}>
            <Button
              tone="accent"
              disabled={busy || !transferReady || !sessionReady}
              onClick={() => {
                const operation = { kind: "upload" } as const;
                setPending(operation);
                setResult(undefined);
                void run(operation, {});
              }}
            >
              <Trans>Upload</Trans>
            </Button>
            <Button
              tone="accent"
              disabled={busy || !transferReady || !sessionReady}
              onClick={() => {
                const operation = { kind: "deploy-and-launch" } as const;
                setPending(operation);
                setResult(undefined);
                void run(operation, {});
              }}
            >
              <Trans>Deploy &amp; Launch</Trans>
            </Button>
            <Button
              disabled={busy || !transferReady}
              onClick={() => {
                const operation = { kind: "download" } as const;
                setPending(operation);
                setResult(undefined);
                void run(operation, {});
              }}
            >
              <Trans>Download</Trans>
            </Button>
            <Button
              disabled={busy || !transferReady}
              onClick={() => {
                const operation = { kind: "close-and-download" } as const;
                setPending(operation);
                setResult(undefined);
                void run(operation, {});
              }}
            >
              <Trans>Close &amp; Download</Trans>
            </Button>
          </div>

          {!sessionReady ? (
            <p className={message}>
              <Trans>Open a save to upload it to this target.</Trans>
            </p>
          ) : null}

          {selected.kind === "ssh" ? (
            <div className={actionsBar}>
              <Button
                size="sm"
                disabled={!selected.hostKeyTrusted}
                onClick={() => targetMutations.forgetHostKey.mutate(selected.id)}
              >
                <Trans>Forget the approved host key</Trans>
              </Button>
            </div>
          ) : null}

          {busy && progress !== undefined && !progress.finished ? (
            <div className={actionsBar}>
              <p role="status" className={message}>
                <Trans>Stage</Trans>: {progress.stage} — {progress.percent}% (
                {Math.round(progress.elapsedMS / 1000)}s)
              </p>
              <Button
                size="sm"
                onClick={() => {
                  if (operationID !== undefined) operations.cancel.mutate(operationID);
                }}
              >
                <Trans>Cancel</Trans>
              </Button>
            </div>
          ) : null}

          {failure !== undefined ? (
            <p role="alert" className={alert}>
              {failure}
            </p>
          ) : null}

          {result !== undefined ? (
            <>
              {result.targetState === "replacement_undetermined" ? (
                <p role="alert" className={alert}>
                  {/* Neither claim may be made here. The operation is not
                      retried and the game is not started from this state. */}
                  <Trans>
                    The replacement was sent to the target and its result could not be
                    established. The target save may or may not have been replaced. Inspect the
                    target before you deploy again or start the game.
                  </Trans>
                </p>
              ) : result.targetState === "replaced_unverified" ? (
                <p role="alert" className={alert}>
                  <Trans>
                    The target save was replaced, but its final verification failed. Do not use
                    this target until it has been inspected.
                  </Trans>
                </p>
              ) : result.targetState === "replaced_verified" && !result.completed ? (
                <p role="alert" className={alert}>
                  {result.failure === "launch_failed" ? (
                    <Trans>
                      The target save was replaced and verified, but the game could not be
                      launched.
                    </Trans>
                  ) : (
                    <Trans>
                      The target save was replaced and verified, but a later operation step failed.
                    </Trans>
                  )}
                </p>
              ) : (
                <p role="status" className={message}>
                  {result.completed ? (
                    <Trans>The operation finished.</Trans>
                  ) : (
                    <Trans>The operation stopped and the target was left unchanged.</Trans>
                  )}
                </p>
              )}
              <ul className={stageList}>
                {result.stages.map((stage) => (
                  <li key={stage.stage}>
                    {stage.stage}: {stage.completed ? t`done` : t`not performed`}
                    {stage.detail === undefined ? "" : ` — ${stage.detail}`}
                  </li>
                ))}
              </ul>
            </>
          ) : null}
        </Card>
      ) : null}

      {/* The blocked outcomes. Each one is answered by one explicit decision and
          retried with exactly that decision; refusing simply closes the
          dialog and leaves the target as it is. */}
      <Dialog
        open={result?.blocked === "game_status_unknown"}
        onOpenChange={(open) => {
          if (!open) setResult(undefined);
        }}
        title={t`The game state cannot be confirmed`}
        description={t`Continuing while the game is running can corrupt the target save.`}
        closeLabel={t`Cancel`}
      >
        <Button
          tone="accent"
          onClick={() => retryWith({ continueWithUnknownGameStatus: true })}
        >
          <Trans>Continue anyway</Trans>
        </Button>
      </Dialog>

      <Dialog
        open={result?.blocked === "remote_backup_confirmation_required"}
        onOpenChange={(open) => {
          if (!open) setResult(undefined);
        }}
        title={t`Create a backup of the target save?`}
        description={t`The target already has a save. It is always backed up before it is replaced; this only asks whether to go on.`}
        closeLabel={t`Cancel`}
      >
        <div className={actionsBar}>
          <Button
            tone="accent"
            onClick={() =>
              retryWith({ confirmRemoteBackup: true, continueWithUnknownGameStatus: true })
            }
          >
            <Trans>Back up and continue</Trans>
          </Button>
          <Button
            disabled={hostSettings.data === undefined || setHostSettings.isPending}
            onClick={() => {
              // "Don't ask again" switches the policy to always creating the
              // backup. It never turns backups off: no such mode exists, so the
              // only thing this changes is whether the question is asked.
              const settings = hostSettings.data;
              if (settings === undefined) return;
              setHostSettings.mutate(
                {
                  skipReviewForNormalRisk: settings.skipReviewForNormalRisk,
                  remoteBackupPolicy: "always",
                },
                {
                  onSuccess: () =>
                    retryWith({
                      confirmRemoteBackup: true,
                      continueWithUnknownGameStatus: true,
                    }),
                },
              );
            }}
          >
            <Trans>Always back up without asking</Trans>
          </Button>
        </div>
      </Dialog>

      <Dialog
        open={result?.blocked === "stop_game_confirmation_required"}
        onOpenChange={(open) => {
          if (!open) setResult(undefined);
        }}
        title={t`Stop the game on the target?`}
        description={t`The operation cannot continue while the game is running. Refusing cancels it.`}
        closeLabel={t`Cancel`}
      >
        <Button tone="accent" onClick={() => retryWith({ confirmStopGame: true })}>
          <Trans>Stop the game and continue</Trans>
        </Button>
      </Dialog>

      <Dialog
        open={result?.blocked === "game_running"}
        onOpenChange={(open) => {
          if (!open) setResult(undefined);
        }}
        title={t`The game is running on this target`}
        description={t`Upload and Download are blocked while the game runs. Use Close Game, Deploy & Launch or Close & Download instead.`}
        closeLabel={t`Close`}
      >
        {/* There is deliberately no Continue Anyway here: section 4 states the
            block cannot be overridden. */}
        <p className={message}>
          <Trans>Nothing on the target was changed.</Trans>
        </p>
      </Dialog>

      <Dialog
        open={review !== undefined}
        onOpenChange={(open) => {
          if (!open) setReview(undefined);
        }}
        title={t`Review the changes before deploying`}
        description={t`The save passed validation. Deploying replaces the target save.`}
        closeLabel={t`Cancel`}
      >
        <p className={message}>
          <Trans>Warnings</Trans>: {review?.validation.warningCount ?? 0} · <Trans>Ban risk</Trans>
          : {review?.validation.banRiskCount ?? 0}
        </p>
        <Button
          tone="accent"
          onClick={() => {
            const current = review;
            setReview(undefined);
            if (current === undefined) return;
            void run(current.operation, current.confirmations, true);
          }}
        >
          <Trans>Deploy these changes</Trans>
        </Button>
      </Dialog>

      <TargetForm
        draft={editing}
        busy={targetMutations.create.isPending || targetMutations.update.isPending}
        failed={targetMutations.create.isError || targetMutations.update.isError}
        onClose={() => setEditing(undefined)}
        onSubmit={(input) => {
          const mutation =
            input.targetID === undefined ? targetMutations.create : targetMutations.update;
          mutation.mutate(input, { onSuccess: () => setEditing(undefined) });
        }}
      />

      {/* Trust On First Use. The fingerprint shown is the one the backend
          observed during the handshake it refused; approving sends exactly that
          value back, and the backend accepts nothing else. */}
      <Dialog
        open={targetMutations.test.data?.hostKeyPending === true}
        onOpenChange={(open) => {
          if (!open) targetMutations.test.reset();
        }}
        title={t`Approve the SSH host key?`}
        description={t`SaveForge has never connected to this host before. The connection was refused until you approve the key it presented.`}
        closeLabel={t`Cancel`}
      >
        <p className={message}>
          <Badge mono>{targetMutations.test.data?.observedFingerprint ?? ""}</Badge>
        </p>
        <Button
          tone="accent"
          disabled={
            targetMutations.trustHostKey.isPending ||
            targetMutations.test.data?.observedFingerprint === undefined
          }
          onClick={() => {
            const observation = targetMutations.test.data;
            if (observation?.observedFingerprint === undefined) return;
            targetMutations.trustHostKey.mutate(
              {
                targetID: observation.targetID,
                fingerprint: observation.observedFingerprint,
              },
              { onSuccess: () => targetMutations.test.reset() },
            );
          }}
        >
          <Trans>Approve this host key</Trans>
        </Button>
      </Dialog>

      <Dialog
        open={targetMutations.test.data?.hostKeyChanged === true}
        onOpenChange={(open) => {
          if (!open) targetMutations.test.reset();
        }}
        title={t`The SSH host key changed`}
        description={t`This host presented a different key than the one approved for it. The connection was refused and nothing on the target was touched.`}
        closeLabel={t`Close`}
      >
        {/* There is deliberately no approve button here: a changed key is
            forgotten as a separate, explicit decision first. */}
        <p className={message}>
          <Trans>
            If you replaced the machine or reinstalled its operating system, forget the approved
            host key first and then test the target again.
          </Trans>
        </p>
      </Dialog>

      <Dialog
        open={pendingDelete !== undefined}
        onOpenChange={(open) => {
          if (!open) setPendingDelete(undefined);
        }}
        title={t`Delete this target?`}
        description={t`Only the configuration is removed. No file on the target is touched.`}
        closeLabel={t`Cancel`}
      >
        <p className={message}>{pendingDelete?.name}</p>
        <Button
          disabled={targetMutations.remove.isPending}
          onClick={() => {
            if (pendingDelete === undefined) return;
            targetMutations.remove.mutate(pendingDelete.id, {
              onSuccess: () => {
                if (selectedID === pendingDelete.id) setSelectedID(undefined);
                setPendingDelete(undefined);
              },
            });
          }}
        >
          <Trans>Delete the target</Trans>
        </Button>
      </Dialog>
    </div>
  );
}

/**
 * The target configuration form.
 *
 * The draft is keyed by the target it belongs to, so a value typed for one
 * target can never be submitted against another. The SSH key is a path: the
 * form never reads, uploads or stores key material, and there is no password
 * field because password authentication is not supported.
 */
function TargetForm({
  draft,
  busy,
  failed,
  onSubmit,
  onClose,
}: {
  draft: DeploymentTargetInput | undefined;
  busy: boolean;
  failed: boolean;
  onSubmit: (input: DeploymentTargetInput) => void;
  onClose: () => void;
}) {
  const { t } = useLingui();
  return (
    <Dialog
      open={draft !== undefined}
      onOpenChange={(open) => {
        if (!open) onClose();
      }}
      title={draft?.targetID === undefined ? t`Add a target` : t`Edit the target`}
      closeLabel={t`Cancel`}
    >
      {draft !== undefined ? (
        <TargetFormFields
          key={draft.targetID ?? "new"}
          initial={draft}
          busy={busy}
          failed={failed}
          onSubmit={onSubmit}
        />
      ) : null}
    </Dialog>
  );
}

function TargetFormFields({
  initial,
  busy,
  failed,
  onSubmit,
}: {
  initial: DeploymentTargetInput;
  busy: boolean;
  failed: boolean;
  onSubmit: (input: DeploymentTargetInput) => void;
}) {
  const { t } = useLingui();
  const prefix = useId();
  const [draft, setDraft] = useState<DeploymentTargetInput>(initial);
  const update = (patch: Partial<DeploymentTargetInput>) =>
    setDraft((current) => ({ ...current, ...patch }));

  return (
    <div className={sections}>
      <div className={formGrid}>
        <span className={settingField}>
          <label htmlFor={`${prefix}-name`}>
            <Trans>Name</Trans>
          </label>
          <Input
            id={`${prefix}-name`}
            value={draft.name}
            onChange={(event) => update({ name: event.currentTarget.value })}
          />
        </span>
        <span className={settingField}>
          <label htmlFor={`${prefix}-kind`}>
            <Trans>Type</Trans>
          </label>
          <Select
            id={`${prefix}-kind`}
            value={draft.kind}
            onChange={(event) => update({ kind: event.currentTarget.value })}
          >
            <option value="local">{t`Local filesystem`}</option>
            <option value="ssh">{t`SSH`}</option>
          </Select>
        </span>
        <span className={settingField}>
          <label htmlFor={`${prefix}-save-path`}>
            <Trans>Save path on the target</Trans>
          </label>
          <Input
            id={`${prefix}-save-path`}
            value={draft.savePath}
            onChange={(event) => update({ savePath: event.currentTarget.value })}
          />
        </span>
        <span className={settingField}>
          <label htmlFor={`${prefix}-start`}>
            <Trans>Start command</Trans>
          </label>
          <Input
            id={`${prefix}-start`}
            value={draft.startCommand ?? ""}
            onChange={(event) => update({ startCommand: event.currentTarget.value })}
          />
        </span>
        <span className={settingField}>
          <label htmlFor={`${prefix}-stop`}>
            <Trans>Stop command</Trans>
          </label>
          <Input
            id={`${prefix}-stop`}
            value={draft.stopCommand ?? ""}
            onChange={(event) => update({ stopCommand: event.currentTarget.value })}
          />
        </span>
        <span className={settingField}>
          <label htmlFor={`${prefix}-status`}>
            <Trans>Status command</Trans>
          </label>
          <Input
            id={`${prefix}-status`}
            value={draft.statusCommand ?? ""}
            onChange={(event) => update({ statusCommand: event.currentTarget.value })}
          />
        </span>
        {draft.kind === "ssh" ? (
          <>
            <span className={settingField}>
              <label htmlFor={`${prefix}-host`}>
                <Trans>Host</Trans>
              </label>
              <Input
                id={`${prefix}-host`}
                value={draft.host ?? ""}
                onChange={(event) => update({ host: event.currentTarget.value })}
              />
            </span>
            <span className={settingField}>
              <label htmlFor={`${prefix}-port`}>
                <Trans>Port</Trans>
              </label>
              <Input
                id={`${prefix}-port`}
                type="number"
                min={1}
                max={65535}
                value={String(draft.port ?? 22)}
                onChange={(event) => update({ port: Number(event.currentTarget.value) })}
              />
            </span>
            <span className={settingField}>
              <label htmlFor={`${prefix}-user`}>
                <Trans>User</Trans>
              </label>
              <Input
                id={`${prefix}-user`}
                value={draft.user ?? ""}
                onChange={(event) => update({ user: event.currentTarget.value })}
              />
            </span>
            <span className={settingField}>
              <label htmlFor={`${prefix}-key`}>
                <Trans>Private key path</Trans>
              </label>
              <Input
                id={`${prefix}-key`}
                value={draft.keyPath ?? ""}
                onChange={(event) => update({ keyPath: event.currentTarget.value })}
              />
            </span>
          </>
        ) : null}
      </div>
      <p className={message}>
        {/* The convention is the contract, so it is stated where the command is
            typed rather than only in the documentation. */}
        <Trans>
          The status command is the only way SaveForge learns whether the game runs on this
          target. Its exit code is the whole answer: 0 means the game is running, 1 means it is
          not, and anything else — no command, another exit code, a timeout or a connection
          problem — leaves the state unknown. SaveForge never guesses it from a process name or
          from the start command.
        </Trans>
      </p>
      {draft.kind === "ssh" ? (
        <p className={message}>
          <Trans>
            SSH uses key authentication only. SaveForge stores the path of the key and never its
            contents, and it never puts that path into a log or a diagnostic report.
          </Trans>
        </p>
      ) : null}
      <Button tone="accent" disabled={busy} onClick={() => onSubmit(draft)}>
        <Trans>Save the target</Trans>
      </Button>
      {failed ? (
        <p role="alert" className={alert}>
          {/* The backend's validation refused the target. Its own wording is not
              rendered: it can carry a host path. */}
          <Trans>The target was rejected. Check the required fields and try again.</Trans>
        </p>
      ) : null}
    </div>
  );
}
