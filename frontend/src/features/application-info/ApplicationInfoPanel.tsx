import { Trans, useLingui } from "@lingui/react/macro";
import { useApplicationInfo } from "../../application/application-info/useApplicationInfo";
import { Badge } from "../../ui/components/Badge/Badge";
import { Button } from "../../ui/components/Button/Button";
import { Card } from "../../ui/components/Card/Card";
import { alert, definitionList, description, term } from "./ApplicationInfoPanel.css";

/**
 * The foundation screen of SaveForge 2.0. It renders exactly what the
 * GetApplicationInfo endpoint reports and nothing else: no save, no character
 * and no placeholder data.
 */
export function ApplicationInfoPanel() {
  const { t } = useLingui();
  const info = useApplicationInfo();

  return (
    <Card aria-label={t`Application information`}>
      <h2>
        <Trans>Backend</Trans>
      </h2>

      {info.isPending && (
        <p role="status">
          <Trans>Loading application information…</Trans>
        </p>
      )}

      {info.isError && (
        // The transport error is never rendered. The adapter replaces it with a
        // stable code and the user sees a safe, localized message instead.
        // Retry belongs to this state only: there is nothing to retry while the
        // call is in flight or after it succeeded.
        <>
          <p role="alert" className={alert}>
            <Trans>Could not read application information from the backend.</Trans>
          </p>
          <div>
            <Button tone="accent" size="sm" onClick={() => void info.refetch()}>
              <Trans>Retry</Trans>
            </Button>
          </div>
        </>
      )}

      {info.isSuccess && (
        <dl className={definitionList}>
          <dt className={term}>
            <Trans>Application</Trans>
          </dt>
          <dd className={description}>SaveForge 2.0</dd>

          <dt className={term}>
            <Trans>Version</Trans>
          </dt>
          <dd className={description}>
            <Badge mono>{info.data.version}</Badge>
          </dd>

          <dt className={term}>
            <Trans>Supported schemas</Trans>
          </dt>
          <dd className={description}>
            {info.data.schemas.map((schema) => (
              <Badge key={schema.name} mono>
                {/* Schema name and range are backend data, not interface text. */}
                {`${schema.name} ${schema.minimumVersion}–${schema.currentVersion}`}
              </Badge>
            ))}
          </dd>

          <dt className={term}>
            <Trans>Capabilities</Trans>
          </dt>
          <dd className={description}>
            {info.data.capabilities.map((capability) => (
              <Badge key={capability} tone="accent" mono>
                {capability}
              </Badge>
            ))}
          </dd>
        </dl>
      )}
    </Card>
  );
}
