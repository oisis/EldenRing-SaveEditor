import { Trans } from "@lingui/react/macro";
import { Card } from "../../ui/components/Card/Card";
import { message } from "../../ui/patterns/panel.css";
import { placeholder } from "../app-shell/AppShell.css";

export function SuperMarchantPlaceholder() {
  return (
    <Card className={placeholder}>
      <h2>
        <Trans>Super marchant</Trans>
      </h2>
      <p className={message}>
        <Trans>This feature will be added in a future update.</Trans>
      </p>
    </Card>
  );
}
