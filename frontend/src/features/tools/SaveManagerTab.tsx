import { Trans, useLingui } from "@lingui/react/macro";
import { useState } from "react";
import type { TargetBackup } from "../../application/deployment/deploymentPort";
import {
  useDeploymentTargets,
  useTargetBackupMutations,
  useTargetBackups,
} from "../../application/deployment/useDeployment";
import { Badge } from "../../ui/components/Badge/Badge";
import { Button } from "../../ui/components/Button/Button";
import { Card } from "../../ui/components/Card/Card";
import { Dialog } from "../../ui/components/Dialog/Dialog";
import { Input } from "../../ui/components/Input/Input";
import { Select } from "../../ui/components/Select/Select";
import { Table, TableFrame } from "../../ui/components/Table/Table";
import { actionCell, alert, message, tableFrame } from "../../ui/patterns/panel.css";
import { actionsBar, badgeRow, sections, settingField, stageList } from "./ToolsPanel.css";

export type SaveManagerTabProps = {
  /** Loads a downloaded backup into the editor, as an explicit user action. */
  onOpenDownloadedBackup: (localPath: string) => void;
};

/**
 * `Tools → Save Manager`.
 *
 * It reads the same targets and the same backups as Deployment, because the
 * backend has one model of each. The table deliberately has no Size column, and
 * activating a backup goes through the backend's own replacement path, which
 * backs the current target save up first.
 */
export function SaveManagerTab({ onOpenDownloadedBackup }: SaveManagerTabProps) {
  const { t } = useLingui();
  const targets = useDeploymentTargets();
  const [targetID, setTargetID] = useState<string>();
  const backups = useTargetBackups(targetID);
  const mutations = useTargetBackupMutations();

  const [creating, setCreating] = useState(false);
  const [editing, setEditing] = useState<TargetBackup>();
  const [pendingDelete, setPendingDelete] = useState<TargetBackup>();
  const [pendingActivate, setPendingActivate] = useState<TargetBackup>();
  const [downloaded, setDownloaded] = useState<string>();

  const transferReady = backups.data?.transferSupported === true;
  const activation = mutations.activate.data?.operation;

  return (
    <div className={sections}>
      <Card aria-label={t`Save Manager`} className={sections}>
        <h2>
          <Trans>Save Manager</Trans>
        </h2>
        <span className={settingField}>
          <label htmlFor="save-manager-target">
            <Trans>Target</Trans>
          </label>
          <Select
            id="save-manager-target"
            value={targetID ?? ""}
            onChange={(event) =>
              setTargetID(event.currentTarget.value === "" ? undefined : event.currentTarget.value)
            }
          >
            <option value="">{t`Select a target`}</option>
            {(targets.data?.targets ?? []).map((target) => (
              <option key={target.id} value={target.id}>
                {target.name}
              </option>
            ))}
          </Select>
        </span>

        {targetID === undefined ? (
          <p className={message}>
            <Trans>Select a target to see its backups.</Trans>
          </p>
        ) : null}
        {backups.isError ? (
          <p role="alert" className={alert}>
            <Trans>The backup library could not be read.</Trans>
          </p>
        ) : null}
        {backups.isSuccess && !transferReady ? (
          <p role="alert" className={alert}>
            {/* The backend's own sentence about what this build cannot do. */}
            {backups.data.unsupportedReason}
          </p>
        ) : null}

        {targetID !== undefined ? (
          <div className={actionsBar}>
            <Button
              size="sm"
              disabled={!transferReady || mutations.create.isPending}
              onClick={() => setCreating(true)}
            >
              <Trans>Create a manual backup</Trans>
            </Button>
            <Button
              size="sm"
              disabled={
                !transferReady ||
                mutations.clearActive.isPending ||
                !(backups.data?.backups ?? []).some((backup) => backup.active)
              }
              onClick={() => mutations.clearActive.mutate(targetID)}
            >
              <Trans>Clear the active mark</Trans>
            </Button>
          </div>
        ) : null}

        {targetID !== undefined ? (
          <TableFrame className={tableFrame}>
            <Table>
              <caption>{t`Target backups`}</caption>
              <thead>
                <tr>
                  <th scope="col">{t`File`}</th>
                  <th scope="col">{t`Created`}</th>
                  <th scope="col">{t`Kind`}</th>
                  <th scope="col">{t`Active`}</th>
                  <th scope="col">{t`Tags`}</th>
                  <th scope="col">{t`Description`}</th>
                  <th scope="col" className={actionCell}>
                    {t`Actions`}
                  </th>
                </tr>
              </thead>
              <tbody>
                {(backups.data?.backups ?? []).map((backup) => (
                  <tr key={backup.id}>
                    <th scope="row">{backup.fileName}</th>
                    <td>{backup.createdAt}</td>
                    <td>{backup.manual ? t`Manual` : t`Automatic`}</td>
                    <td>{backup.active ? <Badge tone="accent">{t`Active`}</Badge> : ""}</td>
                    <td>
                      <span className={badgeRow}>
                        {backup.tags.map((tag) => (
                          <Badge key={tag}>{tag}</Badge>
                        ))}
                      </span>
                    </td>
                    <td>{backup.description ?? ""}</td>
                    <td className={actionCell}>
                      <span className={actionsBar}>
                        <Button
                          size="sm"
                          disabled={!transferReady || mutations.activate.isPending}
                          onClick={() => setPendingActivate(backup)}
                        >
                          <Trans>Set active</Trans>
                        </Button>
                        <Button size="sm" onClick={() => setEditing(backup)}>
                          <Trans>Edit</Trans>
                        </Button>
                        <Button
                          size="sm"
                          disabled={!transferReady || mutations.download.isPending}
                          onClick={() =>
                            mutations.download.mutate(
                              { targetID, backupID: backup.id },
                              {
                                onSuccess: (result) => {
                                  // A cancelled Save As dialog wrote nothing, so
                                  // there is nothing to offer opening.
                                  if (result.target !== undefined) setDownloaded(result.target);
                                },
                              },
                            )
                          }
                        >
                          <Trans>Download</Trans>
                        </Button>
                        <Button
                          size="sm"
                          disabled={!transferReady}
                          onClick={() => setPendingDelete(backup)}
                        >
                          <Trans>Delete</Trans>
                        </Button>
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </Table>
          </TableFrame>
        ) : null}
        {backups.isSuccess && backups.data.backups.length === 0 ? (
          <p className={message}>
            <Trans>This target has no backups yet.</Trans>
          </p>
        ) : null}

        {activation !== undefined && !activation.completed ? (
          <>
            <p role="alert" className={alert}>
              {activation.targetState === "replaced_unverified" ? (
                <Trans>
                  The target was replaced with this backup, but final verification failed. Do not
                  use it until it has been inspected.
                </Trans>
              ) : activation.targetState === "replaced_verified" ? (
                <Trans>
                  The target was replaced and verified, but the active-backup metadata could not be
                  updated.
                </Trans>
              ) : (
                <Trans>The backup was not activated and the target was left unchanged.</Trans>
              )}
            </p>
            <ul className={stageList}>
              {activation.stages.map((stage) => (
                <li key={stage.stage}>
                  {stage.stage}: {stage.completed ? t`done` : t`not performed`}
                </li>
              ))}
            </ul>
          </>
        ) : null}
      </Card>

      <Dialog
        open={downloaded !== undefined}
        onOpenChange={(open) => {
          if (!open) setDownloaded(undefined);
        }}
        title={t`The backup was saved`}
        description={t`Opening it in the editor replaces the save you are working on.`}
        closeLabel={t`Close`}
      >
        <Button
          tone="accent"
          onClick={() => {
            const path = downloaded;
            setDownloaded(undefined);
            if (path !== undefined) onOpenDownloadedBackup(path);
          }}
        >
          <Trans>Open it in the editor</Trans>
        </Button>
      </Dialog>

      <Dialog
        open={pendingActivate !== undefined}
        onOpenChange={(open) => {
          if (!open) setPendingActivate(undefined);
        }}
        title={t`Make this backup the target save?`}
        description={t`The current target save is backed up first, then replaced with this backup.`}
        closeLabel={t`Cancel`}
      >
        <p className={message}>{pendingActivate?.fileName}</p>
        <Button
          tone="accent"
          disabled={mutations.activate.isPending}
          onClick={() => {
            if (pendingActivate === undefined || targetID === undefined) return;
            mutations.activate.mutate(
              {
                operationID: `activate-${pendingActivate.id}`,
                targetID,
                backupID: pendingActivate.id,
                // The game state cannot be confirmed on this target, so the user
                // is the one who states that continuing is safe.
                continueWithUnknownGameStatus: true,
                confirmRemoteBackup: true,
              },
              { onSuccess: () => setPendingActivate(undefined) },
            );
          }}
        >
          <Trans>Back up the target and activate</Trans>
        </Button>
      </Dialog>

      <Dialog
        open={pendingDelete !== undefined}
        onOpenChange={(open) => {
          if (!open) setPendingDelete(undefined);
        }}
        title={t`Delete this backup?`}
        description={t`The backup file is removed from the target. This cannot be undone.`}
        closeLabel={t`Cancel`}
      >
        <p className={message}>{pendingDelete?.fileName}</p>
        <Button
          disabled={mutations.remove.isPending}
          onClick={() => {
            if (pendingDelete === undefined || targetID === undefined) return;
            mutations.remove.mutate(
              { targetID, backupID: pendingDelete.id },
              { onSuccess: () => setPendingDelete(undefined) },
            );
          }}
        >
          <Trans>Delete the backup</Trans>
        </Button>
      </Dialog>

      <BackupMetadataDialog
        open={creating}
        title={t`Create a manual backup`}
        initialTags=""
        initialDescription=""
        busy={mutations.create.isPending}
        failed={mutations.create.isError}
        onSubmit={({ tags, description }) => {
          if (targetID === undefined) return;
          mutations.create.mutate(
            { targetID, tags, description },
            { onSuccess: () => setCreating(false) },
          );
        }}
        onClose={() => setCreating(false)}
      />

      <BackupMetadataDialog
        open={editing !== undefined}
        title={t`Edit the backup`}
        initialTags={(editing?.tags ?? []).join(", ")}
        initialDescription={editing?.description ?? ""}
        busy={mutations.update.isPending}
        failed={mutations.update.isError}
        onSubmit={({ tags, description }) => {
          if (editing === undefined || targetID === undefined) return;
          mutations.update.mutate(
            { targetID, backupID: editing.id, tags, description },
            { onSuccess: () => setEditing(undefined) },
          );
        }}
        onClose={() => setEditing(undefined)}
      />
    </div>
  );
}

/**
 * The tags and description form of creating and editing a backup. The draft is
 * remounted with the backup it belongs to, so a value typed for one backup can
 * never be submitted against another.
 */
function BackupMetadataDialog({
  open,
  title,
  initialTags,
  initialDescription,
  busy,
  failed,
  onSubmit,
  onClose,
}: {
  open: boolean;
  title: string;
  initialTags: string;
  initialDescription: string;
  busy: boolean;
  failed: boolean;
  onSubmit: (values: { tags: string[]; description: string }) => void;
  onClose: () => void;
}) {
  const { t } = useLingui();
  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (!next) onClose();
      }}
      title={title}
      closeLabel={t`Cancel`}
    >
      {open ? (
        <BackupMetadataForm
          key={`${title}:${initialTags}:${initialDescription}`}
          initialTags={initialTags}
          initialDescription={initialDescription}
          busy={busy}
          failed={failed}
          onSubmit={onSubmit}
        />
      ) : null}
    </Dialog>
  );
}

function BackupMetadataForm({
  initialTags,
  initialDescription,
  busy,
  failed,
  onSubmit,
}: {
  initialTags: string;
  initialDescription: string;
  busy: boolean;
  failed: boolean;
  onSubmit: (values: { tags: string[]; description: string }) => void;
}) {
  const [tags, setTags] = useState(initialTags);
  const [description, setDescription] = useState(initialDescription);

  return (
    <div className={sections}>
      <span className={settingField}>
        <label htmlFor="backup-form-tags">
          <Trans>Tags, separated by commas</Trans>
        </label>
        <Input
          id="backup-form-tags"
          value={tags}
          onChange={(event) => setTags(event.currentTarget.value)}
        />
      </span>
      <span className={settingField}>
        <label htmlFor="backup-form-description">
          <Trans>Description</Trans>
        </label>
        <Input
          id="backup-form-description"
          value={description}
          onChange={(event) => setDescription(event.currentTarget.value)}
        />
      </span>
      <Button
        tone="accent"
        disabled={busy}
        onClick={() =>
          onSubmit({
            tags: tags
              .split(",")
              .map((tag) => tag.trim())
              .filter((tag) => tag !== ""),
            description: description.trim(),
          })
        }
      >
        <Trans>Store the backup details</Trans>
      </Button>
      {failed ? (
        <p role="alert" className={alert}>
          <Trans>The backup details were not stored.</Trans>
        </p>
      ) : null}
    </div>
  );
}
