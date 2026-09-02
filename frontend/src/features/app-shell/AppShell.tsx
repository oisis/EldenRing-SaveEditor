import { Trans, useLingui } from "@lingui/react/macro";
import { type ReactNode, useEffect, useState } from "react";
import appIconURL from "../../../../build/appicon.png";
import type { Locale } from "../../i18n/i18n";
import { locales } from "../../i18n/i18n";
import { Badge } from "../../ui/components/Badge/Badge";
import { Button } from "../../ui/components/Button/Button";
import { Card } from "../../ui/components/Card/Card";
import { Input } from "../../ui/components/Input/Input";
import { Select } from "../../ui/components/Select/Select";
import { message } from "../../ui/patterns/panel.css";
import type { ThemeName } from "../../ui/tokens/themes.css";
import { themeNames } from "../../ui/tokens/themes.css";
import { ApplicationInfoPanel } from "../application-info/ApplicationInfoPanel";
import { CharacterSidebar } from "../character/CharacterSidebar";
import { InventoryAndStoragePanel } from "../items/inventory-storage/InventoryAndStoragePanel";
import { ItemDatabasePanel } from "../items/item-database/ItemDatabasePanel";
import { PendingChangesDialog, ReviewChangesDialog } from "../review-changes/ReviewChangesDialog";
import { RecoveryJournalDialog } from "../save-session/RecoveryJournalDialog";
import { SaveSessionContent } from "../save-session/SaveSessionPanel";
import type { SaveSessionFlow } from "../save-session/useSaveSessionFlow";
import {
  brand,
  brandLogo,
  brandName,
  consoleBar,
  consoleBarButton,
  consoleLatest,
  consolePanel,
  consolePanelBody,
  consolePanelHeader,
  fileHeader,
  fileLine,
  fileName,
  itemSubnav,
  moduleNav,
  moduleTab,
  operationGlyph,
  operations,
  operationText,
  placeholder,
  screen,
  shell,
  sidebar,
  sidebarBody,
  sidebarEmpty,
  topbar,
  topbarSpacer,
  workspace,
} from "./AppShell.css";

export type AppSection =
  | "home"
  | "character"
  | "items"
  | "equipment"
  | "world"
  | "advanced"
  | "tools";

type ItemsSection = "inventory" | "database";

export type AppShellProps = {
  flow: SaveSessionFlow;
  theme: ThemeName;
  onThemeChange: (theme: ThemeName) => void;
  locale: Locale;
  onLocaleChange: (locale: Locale) => void;
};

const sections: readonly AppSection[] = [
  "home",
  "character",
  "items",
  "equipment",
  "world",
  "advanced",
  "tools",
];

/**
 * The production application chrome. It owns presentation state only: the
 * selected screen and console disclosure. The save, characters and validation
 * remain owned by the one SaveSessionFlow supplied by the composition root.
 */
export function AppShell({ flow, theme, onThemeChange, locale, onLocaleChange }: AppShellProps) {
  const { t } = useLingui();
  const [section, setSection] = useState<AppSection>("home");
  const [consoleOpen, setConsoleOpen] = useState(false);
  const [itemsSection, setItemsSection] = useState<ItemsSection>("inventory");
  const [retentionDraft, setRetentionDraft] = useState("10");

  useEffect(() => {
    if (flow.lifecycleSettings !== undefined) {
      setRetentionDraft(String(flow.lifecycleSettings.backupRetention));
    }
  }, [flow.lifecycleSettings]);

  const labels: Record<AppSection, string> = {
    home: t`Home`,
    character: t`Character`,
    items: t`Items`,
    equipment: t`Equipment`,
    world: t`World`,
    advanced: t`Advanced`,
    tools: t`Tools`,
  };
  const descriptions: Record<AppSection, string> = {
    home: t`Session overview and file actions`,
    character: t`Character profile and appearance`,
    items: t`Inventory, Storage and Item Database`,
    equipment: t`Equipment and quick slots`,
    world: t`World progress and unlocks`,
    advanced: t`Advanced save features`,
    tools: t`Application settings and tools`,
  };
  const themeLabels: Record<ThemeName, string> = {
    light: t`Light`,
    dark: t`Dark`,
    "elden-ring": t`Elden Ring`,
  };
  const localeLabels: Record<Locale, string> = {
    en: t`English`,
    pl: t`Polish`,
  };

  const session = flow.session;
  const selectedCharacterID = flow.selection.selectedCharacterID;

  return (
    <div className={shell}>
      <div className={brand}>
        <img className={brandLogo} src={appIconURL} alt="" aria-hidden="true" />
        <span className={brandName}>SaveForge</span>
      </div>

      <header className={topbar}>
        <nav aria-label={t`Modules`} className={moduleNav}>
          {sections.map((name) => (
            <button
              key={name}
              type="button"
              className={moduleTab}
              aria-current={section === name ? "page" : undefined}
              title={descriptions[name]}
              onClick={() => setSection(name)}
            >
              {labels[name]}
            </button>
          ))}
        </nav>

        <span className={topbarSpacer} />

        {session !== undefined && (
          <Badge tone={session.unsavedChanges ? "accent" : "neutral"}>
            {session.unsavedChanges ? <Trans>Modified</Trans> : <Trans>Saved</Trans>}
          </Badge>
        )}

        <div className={operations} role="toolbar" aria-label={t`Session operations`}>
          <Button
            size="sm"
            disabled={session === undefined || flow.isBusy || (flow.history?.undoCount ?? 0) === 0}
            title={t`Undo the last operation`}
            onClick={flow.undo}
          >
            <span className={operationGlyph} aria-hidden="true">
              ↶
            </span>
            <span className={operationText}>
              <Trans>Undo</Trans>
            </span>
          </Button>
          <Button
            size="sm"
            disabled={session === undefined || flow.isBusy || (flow.history?.redoCount ?? 0) === 0}
            title={t`Redo the last undone operation`}
            onClick={flow.redo}
          >
            <span className={operationGlyph} aria-hidden="true">
              ↷
            </span>
            <span className={operationText}>
              <Trans>Redo</Trans>
            </span>
          </Button>
          <Button
            size="sm"
            disabled={session === undefined || flow.isBusy}
            title={t`Review changes and validate before saving`}
            onClick={flow.openReview}
          >
            <span className={operationGlyph} aria-hidden="true">
              Δ
            </span>
            <span className={operationText}>
              <Trans>Changes</Trans>
            </span>
          </Button>
        </div>

        <Select
          aria-label={t`Theme`}
          value={theme}
          onChange={(event) => onThemeChange(event.currentTarget.value as ThemeName)}
        >
          {themeNames.map((name) => (
            <option key={name} value={name}>
              {themeLabels[name]}
            </option>
          ))}
        </Select>
      </header>

      <div className={sidebar}>
        <div className={fileHeader}>
          <div className={fileLine}>
            <span className={fileName} title={session?.sourcePath}>
              {session === undefined ? (
                <Trans>No save loaded</Trans>
              ) : (
                fileNameFromPath(session.sourcePath)
              )}
            </span>
            {session !== undefined && <Badge mono>{session.platform}</Badge>}
          </div>
        </div>
        <div className={sidebarBody}>
          {session === undefined ? (
            <p className={sidebarEmpty}>
              <Trans>Open a save from Home to see its character slots.</Trans>
            </p>
          ) : (
            <CharacterSidebar model={flow.selection} />
          )}
        </div>
      </div>

      <main className={workspace} id="workspace" tabIndex={-1}>
        {section === "home" && <SaveSessionContent flow={flow} />}
        {section === "character" && (
          <UnavailableScreen>
            <Trans>The Character workspace will be implemented in a later frontend stage.</Trans>
          </UnavailableScreen>
        )}
        {section === "items" && (
          <section aria-label={t`Items`} className={screen}>
            <nav aria-label={t`Items sections`} className={itemSubnav}>
              <Button
                size="sm"
                pressed={itemsSection === "inventory"}
                onClick={() => setItemsSection("inventory")}
              >
                <Trans>Inventory &amp; Storage</Trans>
              </Button>
              <Button
                size="sm"
                pressed={itemsSection === "database"}
                onClick={() => setItemsSection("database")}
              >
                <Trans>Item Database</Trans>
              </Button>
            </nav>
            {itemsSection === "inventory" ? (
              <InventoryAndStoragePanel
                saveSessionID={session?.saveSessionID}
                saveRevision={session?.saveRevision}
                characterID={selectedCharacterID}
                containerSection="common"
              />
            ) : (
              <ItemDatabasePanel />
            )}
          </section>
        )}
        {section === "equipment" && (
          <UnavailableScreen>
            <Trans>The Equipment workspace will be implemented in a later frontend stage.</Trans>
          </UnavailableScreen>
        )}
        {section === "world" && (
          <UnavailableScreen>
            <Trans>The World workspace will be implemented in a later frontend stage.</Trans>
          </UnavailableScreen>
        )}
        {section === "advanced" && (
          <UnavailableScreen>
            <Trans>Advanced save features are not available yet.</Trans>
          </UnavailableScreen>
        )}
        {section === "tools" && (
          <section aria-label={t`Tools`} className={screen}>
            <Card aria-label={t`Application settings`}>
              <h2>
                <Trans>Application</Trans>
              </h2>
              <label htmlFor="tools-theme">
                <Trans>Theme</Trans>{" "}
                <Select
                  id="tools-theme"
                  value={theme}
                  onChange={(event) => onThemeChange(event.currentTarget.value as ThemeName)}
                >
                  {themeNames.map((name) => (
                    <option key={name} value={name}>
                      {themeLabels[name]}
                    </option>
                  ))}
                </Select>
              </label>
              <label htmlFor="tools-language">
                <Trans>Language</Trans>{" "}
                <Select
                  id="tools-language"
                  value={locale}
                  onChange={(event) => onLocaleChange(event.currentTarget.value as Locale)}
                >
                  {locales.map((name) => (
                    <option key={name} value={name}>
                      {localeLabels[name]}
                    </option>
                  ))}
                </Select>
              </label>
              <label htmlFor="tools-backup-retention">
                <Trans>Automatic backups kept</Trans>{" "}
                <Input
                  id="tools-backup-retention"
                  type="number"
                  min={1}
                  max={1000}
                  step={1}
                  value={retentionDraft}
                  onChange={(event) => setRetentionDraft(event.currentTarget.value)}
                  onBlur={() => {
                    const retention = Number(retentionDraft);
                    if (Number.isInteger(retention) && retention >= 1 && retention <= 1000) {
                      flow.setBackupRetention(retention);
                    } else {
                      setRetentionDraft(String(flow.lifecycleSettings?.backupRetention ?? 10));
                    }
                  }}
                  disabled={flow.isBusy}
                />
              </label>
            </Card>
            <ApplicationInfoPanel />
          </section>
        )}
      </main>

      <section
        id="diagnostic-console"
        className={consolePanel}
        aria-label={t`Diagnostic console`}
        hidden={!consoleOpen}
      >
        <header className={consolePanelHeader}>
          <strong>
            <Trans>Console</Trans>
          </strong>
          <Button size="sm" onClick={() => setConsoleOpen(false)}>
            <Trans>Close</Trans>
          </Button>
        </header>
        <div className={consolePanelBody}>
          <p className={message}>
            <Trans>Live diagnostic messages are not available in this build yet.</Trans>
          </p>
        </div>
      </section>

      <div className={consoleBar}>
        <button
          type="button"
          className={consoleBarButton}
          aria-expanded={consoleOpen}
          aria-controls="diagnostic-console"
          onClick={() => setConsoleOpen((open) => !open)}
        >
          <span>
            <Trans>Console</Trans>
          </span>
          <span className={consoleLatest}>
            <Trans>No live messages</Trans>
          </span>
        </button>
      </div>

      <ReviewChangesDialog flow={flow} />
      <PendingChangesDialog flow={flow} />
      <RecoveryJournalDialog flow={flow} />
    </div>
  );
}

function UnavailableScreen({ children }: { children: ReactNode }) {
  return (
    <Card className={placeholder}>
      <p className={message}>{children}</p>
    </Card>
  );
}

/** Returns only presentation text; the exact backend path stays untouched. */
export function fileNameFromPath(sourcePath: string): string {
  const parts = sourcePath.split(/[\\/]/);
  return parts.at(-1) || sourcePath;
}
