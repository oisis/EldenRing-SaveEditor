import { Trans, useLingui } from "@lingui/react/macro";
import { type ReactNode, useState } from "react";
import { type AppError, appErrorCodes } from "../../application/errors/appError";
import { Badge } from "../../ui/components/Badge/Badge";
import { Button } from "../../ui/components/Button/Button";
import { Card } from "../../ui/components/Card/Card";
import { alertPanel, factLabel, message } from "../../ui/patterns/panel.css";
import { CharacterSidebar } from "../character/CharacterSidebar";
import {
  actions,
  cardBody,
  cardHeader,
  cardTitle,
  fact,
  facts,
  factValue,
  homeCard,
  homeGrid,
  layout,
  layoutSingle,
  recentList,
  recentName,
  recentOpen,
  recentPath,
  recentRow,
  recentTime,
  reportItem,
  reportList,
  reportScope,
  stack,
  summaryTile,
  summaryTiles,
  summaryValue,
  warningBanner,
} from "./SaveSessionPanel.css";
import { type SaveSessionFlow, useSaveSessionFlow } from "./useSaveSessionFlow";

/**
 * The Home dashboard: session metadata, local backup policy and recent files.
 *
 * The screen renders backend answers and never forms its own. Every count, every
 * message and every state shown here was decided by the backend; nothing here
 * inspects a save.
 */
export function SaveSessionPanel() {
  const flow = useSaveSessionFlow();

  return <SaveSessionContent flow={flow} showCharacterSidebar />;
}

export type HomeDestination = "appearance" | "database" | "settings" | "about";

function recentFileName(path: string): string {
  return path.split(/[\\/]/).at(-1) || path;
}

function recentOpenedAt(value: string, locale: string): string {
  const date = new Date(value);
  return Number.isNaN(date.getTime())
    ? value
    : new Intl.DateTimeFormat(locale, { dateStyle: "medium", timeStyle: "short" }).format(date);
}

/**
 * The session workspace without ownership of the session controller. AppShell
 * uses this form so one flow can feed Home, the global sidebar and the global
 * status controls without copying session state into another store.
 */
export function SaveSessionContent({
  flow,
  showCharacterSidebar = false,
  onNavigate,
}: {
  flow: SaveSessionFlow;
  showCharacterSidebar?: boolean;
  onNavigate?: (destination: HomeDestination) => void;
}) {
  const { t, i18n } = useLingui();
  const [reportVisible, setReportVisible] = useState(false);

  const { state, session, validation, selection, failure, appError, unclosedSessionID } = flow;
  // Every surface that acts on a session is tied to holding one, not to the
  // verdict on it. A session whose validation or character list failed is still
  // open in the backend, so it keeps its metadata and its Close action: hiding
  // it would leave a session nobody can reach or close.
  const hasSession = session !== undefined;
  // An unconfirmed close is unresolved state about a session that is still open
  // in the backend. Nothing else may be opened beside it, and no *other* session
  // may be closed; closing the unconfirmed one is the retry and stays offered.
  const cleanupPending = unclosedSessionID !== undefined;
  const closeBlocked = cleanupPending && unclosedSessionID !== session?.saveSessionID;
  const currentHistory =
    session !== undefined &&
    flow.history?.saveSessionID === session.saveSessionID &&
    flow.history.saveRevision === session.saveRevision
      ? flow.history
      : undefined;
  const pendingCount = currentHistory?.operations.length;
  const currentValidation =
    session !== undefined &&
    flow.reviewValidation?.saveSessionID === session.saveSessionID &&
    flow.reviewValidation.saveRevision === session.saveRevision
      ? flow.reviewValidation
      : undefined;
  const shortcuts: { destination: HomeDestination; label: string }[] = [
    { destination: "appearance", label: t`Appearance Presets` },
    { destination: "database", label: t`Item Database` },
    { destination: "settings", label: t`Settings` },
    { destination: "about", label: t`About & Updates` },
  ];

  return (
    <div className={stack}>
      <div className={showCharacterSidebar ? layout : layoutSingle}>
        <div className={homeGrid}>
          <Card aria-label={t`Save session`} className={homeCard}>
            <header className={cardHeader}>
              <h2 className={cardTitle}>
                <Trans>Current save</Trans>
              </h2>
            </header>
            <div className={cardBody}>
              <div className={stack}>
                <div className={actions}>
                  <Button
                    tone="accent"
                    onClick={flow.openSave}
                    disabled={flow.isBusy || cleanupPending}
                    aria-busy={flow.isBusy}
                  >
                    <Trans>Open Save</Trans>
                  </Button>
                  <Button
                    onClick={flow.save}
                    disabled={!hasSession || flow.isBusy || session?.sourceKind !== "local"}
                    title={t`Review and save changes`}
                  >
                    <Trans>Save</Trans>
                  </Button>
                  <Button
                    onClick={flow.saveAs}
                    disabled={!hasSession || flow.isBusy}
                    title={t`Review changes, then choose a new target`}
                  >
                    <Trans>Save As</Trans>
                  </Button>
                </div>

                {state === "opening" && (
                  <p role="status" className={message}>
                    <Trans>Opening save…</Trans>
                  </p>
                )}

                {state === "cancelled" && (
                  <p role="status" className={message}>
                    <Trans>No file was chosen.</Trans>
                  </p>
                )}

                {state === "empty" && (
                  <p className={message}>
                    <Trans>No save is open.</Trans>
                  </p>
                )}

                {failure === "dialog_failed" && (
                  // The dialog failed, so nothing was chosen and nothing was loaded.
                  // Any session already open is untouched and stays usable below.
                  <p role="alert" className={alertPanel}>
                    <Trans>
                      The file chooser could not be opened. No file was selected and nothing was
                      loaded.
                    </Trans>
                  </p>
                )}

                {failure === "load_failed" && (
                  // The operation-level message says only that opening failed. The
                  // structured error is presented separately below, without deriving
                  // a verdict about the file from its fallback sentence.
                  //
                  // It also claims nothing about the backend's own state. LoadSave may
                  // have created a session before failing to report it, so "no session
                  // was created" would be an assurance nothing here can give.
                  <p role="alert" className={alertPanel}>
                    <Trans>The save was not opened for editing.</Trans>
                  </p>
                )}

                {failure === "validation_failed" && (
                  // A report that could not be obtained, and one that answered about
                  // another save state, are the same outcome here: the save was not
                  // checked, so it may not be edited. Nothing is inferred about what
                  // the save contains.
                  <p role="alert" className={alertPanel}>
                    <Trans>This save could not be checked, so it was not opened for editing.</Trans>
                  </p>
                )}

                {failure === "no_active_character" && (
                  // Whether the session created for the check was closed again is a
                  // separate outcome with its own message below, so this one states
                  // only what is known: the save was not opened for editing.
                  <p role="alert" className={alertPanel}>
                    <Trans>
                      This save has no active character, so it cannot be edited. It was not opened
                      for editing.
                    </Trans>
                  </p>
                )}

                {appError !== undefined && <AppErrorDetails error={appError} />}
                {flow.lifecycleError !== undefined && (
                  <AppErrorDetails error={flow.lifecycleError} />
                )}
                {flow.lifecycleMessage !== undefined && (
                  <p role="status" className={message}>
                    {flow.lifecycleMessage}
                  </p>
                )}
                {flow.lastSaveResult !== undefined && (
                  <div role="status" className={message}>
                    <Trans>Saved to {flow.lastSaveResult.target}.</Trans>{" "}
                    {flow.lastSaveResult.backupPath !== undefined && (
                      <Trans>Backup: {flow.lastSaveResult.backupPath}.</Trans>
                    )}{" "}
                    {flow.lastSaveResult.retentionNoticeRequired && (
                      <Trans>
                        The automatic backup limit was reached. Future saves remove the oldest
                        backup.
                      </Trans>
                    )}
                  </div>
                )}

                {unclosedSessionID !== undefined && (
                  <div role="alert" className={alertPanel}>
                    <span>
                      <Trans>
                        A save session could not be closed and is still open. Nothing was changed on
                        disk. Opening and closing a save stay unavailable until it is closed.
                      </Trans>
                    </span>{" "}
                    <Button onClick={flow.retryClose} disabled={flow.isBusy}>
                      <Trans>Retry closing</Trans>
                    </Button>
                  </div>
                )}

                {state === "failed" && hasSession && (
                  <p role="alert" className={alertPanel}>
                    <Trans>
                      This save could not be checked, so it is not confirmed clean. The session is
                      still open.
                    </Trans>
                  </p>
                )}

                {session !== undefined && (
                  <dl className={facts}>
                    <div className={fact}>
                      <dt className={factLabel}>
                        <Trans>File</Trans>
                      </dt>
                      {/* The backend's exact path, rendered as reported. */}
                      <dd className={factValue}>{session.sourcePath}</dd>
                    </div>
                    <div className={fact}>
                      <dt className={factLabel}>
                        <Trans>Platform</Trans>
                      </dt>
                      <dd className={factValue}>
                        <Badge mono>{session.platform}</Badge>
                      </dd>
                    </div>
                    <div className={fact}>
                      <dt className={factLabel}>
                        <Trans>Format</Trans>
                      </dt>
                      <dd className={factValue}>
                        <Badge mono>{session.format}</Badge>
                      </dd>
                    </div>
                    <div className={fact}>
                      <dt className={factLabel}>
                        <Trans>Source</Trans>
                      </dt>
                      <dd className={factValue}>
                        <Badge mono>{session.sourceKind}</Badge>
                      </dd>
                    </div>
                    <div className={fact}>
                      <dt className={factLabel}>
                        <Trans>Revision</Trans>
                      </dt>
                      {/* A backend string, rendered as one; never parsed or counted. */}
                      <dd className={factValue}>
                        <Badge mono>{session.saveRevision}</Badge>
                      </dd>
                    </div>
                    <div className={fact}>
                      <dt className={factLabel}>
                        <Trans>Changes</Trans>
                      </dt>
                      <dd className={factValue}>
                        {session.unsavedChanges ? (
                          <Trans>Unsaved changes</Trans>
                        ) : (
                          <Trans>No unsaved changes</Trans>
                        )}
                      </dd>
                    </div>
                    <div className={fact}>
                      <dt className={factLabel}>
                        <Trans>Last pre-save validation</Trans>
                      </dt>
                      <dd className={factValue}>
                        {currentValidation === undefined ? (
                          <Trans>Not validated for the current revision.</Trans>
                        ) : currentValidation.valid ? (
                          <Trans>Validation passed.</Trans>
                        ) : (
                          <Trans>Validation did not pass.</Trans>
                        )}
                      </dd>
                    </div>
                  </dl>
                )}

                {state === "warnings" && (
                  <div role="alert" className={warningBanner}>
                    <span>
                      {/* Both counts are the backend's own totals, added up and
                    never recomputed from the findings. */}
                      <Trans>
                        This save opened with {validation.errorCount} error(s) and{" "}
                        {validation.warningCount} warning(s) reported by the backend.
                      </Trans>
                    </span>
                    <Button
                      pressed={reportVisible}
                      onClick={() => setReportVisible((visible) => !visible)}
                    >
                      <Trans>View report</Trans>
                    </Button>
                  </div>
                )}

                {state === "warnings" && reportVisible && (
                  <ul aria-label={t`Validation report`} className={reportList}>
                    {validation.reports.flatMap((report) =>
                      report.issues.map((issue) => (
                        <li key={`${report.characterID}-${issue.id}`} className={reportItem}>
                          <span className={reportScope}>
                            {issue.severity} · {issue.scope}
                          </span>{" "}
                          {/* The backend's own safe message, rendered verbatim. */}
                          {issue.message}
                        </li>
                      )),
                    )}
                    {validation.uncheckedScopes > 0 && (
                      <li className={reportItem}>
                        <Trans>
                          {validation.uncheckedScopes} scope(s) could not be checked, so this save
                          is not confirmed clean.
                        </Trans>
                      </li>
                    )}
                    {validation.unresolvedRecords > 0 && (
                      <li className={reportItem}>
                        <Trans>
                          {validation.unresolvedRecords} record(s) could not be resolved.
                        </Trans>
                      </li>
                    )}
                  </ul>
                )}

                {hasSession && (
                  <div>
                    <Button onClick={flow.closeSave} disabled={flow.isBusy || closeBlocked}>
                      <Trans>Close Save</Trans>
                    </Button>
                  </div>
                )}
              </div>
            </div>
          </Card>
          <div className={stack}>
            {hasSession ? (
              <>
                <div className={summaryTiles}>
                  <section className={summaryTile} aria-label={t`Pending changes`}>
                    <span className={factLabel}>
                      <Trans>Pending changes</Trans>
                    </span>
                    <strong className={summaryValue}>
                      {pendingCount === undefined ? (
                        <Trans>Unavailable</Trans>
                      ) : (
                        <Trans>{pendingCount} changes</Trans>
                      )}
                    </strong>
                  </section>
                  <section className={summaryTile} aria-label={t`Active characters`}>
                    <span className={factLabel}>
                      <Trans>Active characters</Trans>
                    </span>
                    <strong className={summaryValue}>
                      {selection.characters.isSuccess ? (
                        <Trans>{selection.activeCharacters.length} active</Trans>
                      ) : (
                        <Trans>Unavailable</Trans>
                      )}
                    </strong>
                  </section>
                </div>
                <Card aria-label={t`Local backups`} className={homeCard}>
                  <header className={cardHeader}>
                    <h2 className={cardTitle}>
                      <Trans>Local backups</Trans>
                    </h2>
                  </header>
                  <div className={cardBody}>
                    {flow.lifecycleSettings === undefined ? (
                      <p className={message}>
                        <Trans>Backup settings are unavailable.</Trans>
                      </p>
                    ) : (
                      <dl className={facts}>
                        <div className={fact}>
                          <dt className={factLabel}>
                            <Trans>Name pattern</Trans>
                          </dt>
                          <dd className={factValue}>{flow.lifecycleSettings.backupNamePattern}</dd>
                        </div>
                        <div className={fact}>
                          <dt className={factLabel}>
                            <Trans>Automatic backups kept</Trans>
                          </dt>
                          <dd className={factValue}>{flow.lifecycleSettings.backupRetention}</dd>
                        </div>
                      </dl>
                    )}
                  </div>
                </Card>
              </>
            ) : (
              <Card aria-label={t`Available without a save`} className={homeCard}>
                <header className={cardHeader}>
                  <h2 className={cardTitle}>
                    <Trans>Available without a save</Trans>
                  </h2>
                </header>
                <div className={cardBody}>
                  <div className={stack}>
                    {shortcuts.map(({ destination, label }) => (
                      <Button
                        key={destination}
                        disabled={onNavigate === undefined || flow.isBusy}
                        onClick={() => onNavigate?.(destination)}
                      >
                        {label}
                      </Button>
                    ))}
                  </div>
                </div>
              </Card>
            )}
          </div>
        </div>

        {showCharacterSidebar && hasSession && <CharacterSidebar model={selection} />}
      </div>

      <Card aria-label={t`Recent files`} className={homeCard}>
        <header className={cardHeader}>
          <h2 className={cardTitle}>
            <Trans>Recent Files</Trans>
          </h2>
          <Button
            size="sm"
            disabled={flow.isBusy || flow.recentFiles.length === 0}
            onClick={flow.clearRecent}
          >
            <Trans>Clear</Trans>
          </Button>
        </header>
        {flow.recentFiles.length === 0 ? (
          <p className={cardBody}>
            <Trans>No recent files.</Trans>
          </p>
        ) : (
          <ul className={recentList}>
            {flow.recentFiles.map((recent) => (
              <li key={recent.path} className={recentRow}>
                <button
                  type="button"
                  className={recentOpen}
                  disabled={flow.isBusy || cleanupPending}
                  onClick={() => flow.openRecent(recent.path)}
                  title={recent.path}
                >
                  <span className={recentName}>{recentFileName(recent.path)}</span>
                  <Badge mono>{recent.platform}</Badge>
                  <span className={recentPath}>{recent.path}</span>
                  <time className={recentTime} dateTime={recent.lastOpenedAt}>
                    {recentOpenedAt(recent.lastOpenedAt, i18n.locale)}
                  </time>
                </button>
                <Button
                  size="sm"
                  disabled={flow.isBusy}
                  onClick={() => flow.removeRecent(recent.path)}
                  aria-label={t`Remove recent file: ${recent.path}`}
                >
                  <Trans>Remove</Trans>
                </Button>
              </li>
            ))}
          </ul>
        )}
      </Card>

      {flow.recoveryJournals.length > 0 && (
        <Card aria-label={t`Recovery journals`}>
          <h2 className={cardTitle}>
            <Trans>Recovery</Trans>
          </h2>
          <ul className={reportList}>
            {flow.recoveryJournals.map((journal) => (
              <li key={journal.journalID} className={reportItem}>
                {journal.sourcePath || journal.journalID} <Badge mono>{journal.status}</Badge>{" "}
                <Button
                  size="sm"
                  disabled={flow.isBusy}
                  onClick={() => flow.exportRecovery(journal.journalID)}
                >
                  <Trans>Export</Trans>
                </Button>{" "}
                <Button
                  size="sm"
                  disabled={flow.isBusy}
                  onClick={() => flow.discardRecovery(journal.journalID)}
                >
                  <Trans>Discard</Trans>
                </Button>{" "}
                <Button
                  size="sm"
                  disabled={flow.isBusy || journal.status !== "compatible"}
                  onClick={() => flow.restoreRecovery(journal.journalID)}
                >
                  <Trans>Restore</Trans>
                </Button>
              </li>
            ))}
          </ul>
        </Card>
      )}
    </div>
  );
}

/**
 * Renders only text the frontend owns for codes it understands. A future
 * backend code falls back to the backend's already-safe message and always
 * carries its diagnostic identifier, so support can correlate it with the
 * internal log without exposing the raw transport failure.
 */
function AppErrorDetails({ error }: { error: AppError }) {
  let summary: ReactNode;
  switch (error.code) {
    case appErrorCodes.invalidRequest:
      summary = <Trans>The request was rejected because some input was invalid.</Trans>;
      break;
    case appErrorCodes.invalidRevision:
      summary = <Trans>The save revision supplied by the request was invalid.</Trans>;
      break;
    case appErrorCodes.revisionConflict:
      summary = <Trans>The save changed before the operation could be applied.</Trans>;
      break;
    case appErrorCodes.unknownSaveSession:
      summary = <Trans>The save session is no longer available.</Trans>;
      break;
    case appErrorCodes.operationFailed:
      summary = <Trans>The requested operation could not be completed.</Trans>;
      break;
    case appErrorCodes.internalError:
      summary = <Trans>The application backend encountered an internal error.</Trans>;
      break;
    case appErrorCodes.bridgeCallFailed:
      summary = <Trans>The application backend could not be reached.</Trans>;
      break;
    default:
      summary = error.message;
  }

  return (
    <p className={message} data-testid="app-error-details">
      <span>{summary}</span>
      {error.diagnosticID !== "" && (
        <>
          {" "}
          <Trans>Diagnostic ID:</Trans> <code>{error.diagnosticID}</code>
        </>
      )}
    </p>
  );
}
