import { Trans, useLingui } from "@lingui/react/macro";
import { useState } from "react";
import { Badge } from "../../ui/components/Badge/Badge";
import { Button } from "../../ui/components/Button/Button";
import { Dialog } from "../../ui/components/Dialog/Dialog";
import { message } from "../../ui/patterns/panel.css";
import { actions, item, itemBody, list, metadata } from "../review-changes/ReviewChangesDialog.css";
import type { SaveSessionFlow } from "./useSaveSessionFlow";

export function RecoveryJournalDialog({ flow }: { flow: SaveSessionFlow }) {
  const { t } = useLingui();
  const [dismissed, setDismissed] = useState(false);
  const open = flow.recoveryJournals.length > 0 && !dismissed;

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (!next && !flow.isBusy) setDismissed(true);
      }}
      title={<Trans>Recover unsaved work</Trans>}
      description={<Trans>SaveForge found operation journals from an earlier session.</Trans>}
      closeLabel={<Trans>Later</Trans>}
    >
      <ul className={list} aria-label={t`Recovery journals`}>
        {flow.recoveryJournals.map((journal) => (
          <li key={journal.journalID} className={item}>
            <div className={itemBody}>
              <strong>{journal.sourcePath || journal.journalID}</strong>
              <span className={metadata}>
                <Badge mono>{journal.status}</Badge> · {journal.operationCount}{" "}
                <Trans>operation(s)</Trans>
              </span>
              {journal.status !== "compatible" && (
                <p className={message}>
                  <Trans>
                    This journal cannot be replayed automatically. You can export or discard it.
                  </Trans>
                </p>
              )}
            </div>
            <div className={actions}>
              <Button
                size="sm"
                disabled={flow.isBusy}
                onClick={() => flow.exportRecovery(journal.journalID)}
              >
                <Trans>Export</Trans>
              </Button>
              <Button
                size="sm"
                disabled={flow.isBusy}
                onClick={() => flow.discardRecovery(journal.journalID)}
              >
                <Trans>Discard</Trans>
              </Button>
              <Button
                size="sm"
                tone="accent"
                disabled={flow.isBusy || journal.status !== "compatible"}
                onClick={() => {
                  setDismissed(true);
                  flow.restoreRecovery(journal.journalID);
                }}
              >
                <Trans>Restore</Trans>
              </Button>
            </div>
          </li>
        ))}
      </ul>
    </Dialog>
  );
}
