import { Trans, useLingui } from "@lingui/react/macro";
import { useApplicationInfo } from "../../application/application-info/useApplicationInfo";
import { useCheckForUpdates, useOpenProjectLink } from "../../application/about/useAbout";
import { Badge } from "../../ui/components/Badge/Badge";
import { Button } from "../../ui/components/Button/Button";
import { Card } from "../../ui/components/Card/Card";
import { alert, message } from "../../ui/patterns/panel.css";
import { cardActions, cardGrid, gridCard } from "./ToolsPanel.css";

/**
 * `About & Updates`.
 *
 * Four cards, two per row, every card in a row the same height. No address is
 * ever typed here: the backend owns the approved link table and the frontend
 * asks for one by identifier, so no defect on this side can make the host open
 * something else.
 */
export function AboutTab() {
  const { t } = useLingui();
  const info = useApplicationInfo();
  const openLink = useOpenProjectLink();
  const updates = useCheckForUpdates();

  return (
    <div className={cardGrid}>
      <Card aria-label={t`Application Info`} className={gridCard}>
        <h2>
          <Trans>Application Info</Trans>
        </h2>
        {info.isPending && (
          <p role="status" className={message}>
            <Trans>Loading application information…</Trans>
          </p>
        )}
        {info.isError && (
          <p role="alert" className={alert}>
            {/* The transport error is never rendered: it carries bridge
                internals and host paths, and neither is actionable here. */}
            <Trans>Could not read application information from the backend.</Trans>
          </p>
        )}
        {info.isSuccess && (
          <>
            <p className={message}>
              <Trans>Version</Trans> <Badge mono>{info.data.version}</Badge>
            </p>
            <p className={message}>
              <Trans>Build</Trans> <Badge mono>{info.data.build}</Badge>
            </p>
            <p className={message}>
              <Trans>Platform</Trans> <Badge mono>{info.data.platform}</Badge>
            </p>
            <p className={message}>
              <Trans>Supported schemas</Trans>{" "}
              {info.data.schemas.map((schema) => (
                <Badge key={schema.name} mono>
                  {/* Schema name and range are backend data, not interface text. */}
                  {`${schema.name} ${schema.minimumVersion}–${schema.currentVersion}`}
                </Badge>
              ))}
            </p>
            <p className={message}>
              <Trans>Capabilities</Trans>{" "}
              {info.data.capabilities.map((capability) => (
                <Badge key={capability} tone="accent" mono>
                  {capability}
                </Badge>
              ))}
            </p>
          </>
        )}
      </Card>

      <Card aria-label={t`Sponsor this project`} className={gridCard}>
        <h2>
          <Trans>Sponsor this project</Trans>
        </h2>
        <p className={message}>
          <Trans>Both destinations are opened in your default browser.</Trans>
        </p>
        <div className={cardActions}>
          <Button size="sm" onClick={() => openLink.mutate("sponsor_coffee")}>
            <Trans>Buy me a coffee</Trans>
          </Button>
          <Button size="sm" onClick={() => openLink.mutate("sponsor_bitcoin")}>
            <Trans>Bitcoin</Trans>
          </Button>
        </div>
        {openLink.isError ? (
          <p role="alert" className={alert}>
            <Trans>The link could not be opened.</Trans>
          </p>
        ) : null}
      </Card>

      <Card aria-label={t`Updates`} className={gridCard}>
        <h2>
          <Trans>Updates</Trans>
        </h2>
        <p className={message}>
          {/* Nothing checks in the background and nothing is ever downloaded or
              installed: the check is one manual request that reads the
              project's stable releases and reports what it found. */}
          <Trans>
            SaveForge never checks for updates on its own and never downloads or installs one.
          </Trans>
        </p>
        {updates.isSuccess && updates.data.status === "current" ? (
          <p role="status" className={message}>
            <Trans>You are running the latest release.</Trans>
          </p>
        ) : null}
        {updates.isSuccess && updates.data.status === "available" ? (
          <p role="status" className={message}>
            <Trans>A newer release is available:</Trans>{" "}
            <Badge mono>{updates.data.latestVersion ?? ""}</Badge>
          </p>
        ) : null}
        {updates.isSuccess && updates.data.status === "unknown" ? (
          <p role="status" className={message}>
            <Trans>
              This build carries no comparable version, so the newest release cannot be compared
              with it.
            </Trans>
          </p>
        ) : null}
        {updates.isSuccess && updates.data.status === "unavailable" ? (
          <p role="status" className={message}>
            <Trans>The project has published no stable release yet.</Trans>
          </p>
        ) : null}
        {updates.isError ? (
          <p role="alert" className={alert}>
            {/* The upstream failure is never repeated: a rate limit or a network
                fault is not actionable and reads as an application defect. */}
            <Trans>The update check could not be completed.</Trans>
          </p>
        ) : null}
        <div className={cardActions}>
          <Button
            tone="accent"
            size="sm"
            disabled={updates.isPending}
            onClick={() => updates.mutate()}
          >
            <Trans>Check for updates</Trans>
          </Button>
          {updates.isSuccess && updates.data.status === "available" ? (
            <Button size="sm" onClick={() => openLink.mutate("releases")}>
              <Trans>Open releases</Trans>
            </Button>
          ) : null}
        </div>
      </Card>

      <Card aria-label={t`Author`} className={gridCard}>
        <h2>
          <Trans>Author</Trans>
        </h2>
        <p className={message}>oisis</p>
        <div className={cardActions}>
          <Button size="sm" onClick={() => openLink.mutate("repository")}>
            <Trans>Open the repository</Trans>
          </Button>
        </div>
      </Card>
    </div>
  );
}
