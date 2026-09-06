import { Plural, Trans, useLingui } from "@lingui/react/macro";
import { useMemo, useState } from "react";
import { useDiagnosticEvents } from "../../application/settings/useHostSettings";
import appIconURL from "../../../../build/appicon.png";
import type { Locale } from "../../i18n/i18n";
import { Badge } from "../../ui/components/Badge/Badge";
import { Button } from "../../ui/components/Button/Button";
import { Select } from "../../ui/components/Select/Select";
import { message } from "../../ui/patterns/panel.css";
import type { ThemeName } from "../../ui/tokens/themes.css";
import { themeNames } from "../../ui/tokens/themes.css";
import { AdvancedPanel } from "../advanced/AdvancedPanel";
import { CharacterPanel } from "../character/CharacterPanel";
import { CharacterSidebar } from "../character/CharacterSidebar";
import { EquipmentPanel } from "../equipment/EquipmentPanel";
import { InventoryAndStoragePanel } from "../items/inventory-storage/InventoryAndStoragePanel";
import { ItemDatabasePanel } from "../items/item-database/ItemDatabasePanel";
import { PendingChangesDialog, ReviewChangesDialog } from "../review-changes/ReviewChangesDialog";
import { RecoveryJournalDialog } from "../save-session/RecoveryJournalDialog";
import { SaveSessionContent } from "../save-session/SaveSessionPanel";
import type { SaveSessionFlow } from "../save-session/useSaveSessionFlow";
import { ToolsPanel } from "../tools/ToolsPanel";
import { WorldPanel } from "../world/WorldPanel";
import {
  brand,
  brandLogo,
  brandName,
  changesCounter,
  consoleBar,
  consoleBarButton,
  consoleEmpty,
  consoleFilter,
  consoleIndicator,
  consoleLatest,
  consoleLevel,
  consoleList,
  consoleRow,
  consoleTime,
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
  screen,
  shell,
  sidebar,
  sidebarBody,
  sidebarEmpty,
  topbar,
  topbarSeparator,
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

/** The console's own filter vocabulary: the backend severities plus "all". */
type ConsoleLevel = "all" | "debug" | "info" | "warning" | "error";

const consoleLevels: readonly ConsoleLevel[] = ["all", "debug", "info", "warning", "error"];

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
  // The console polls only while it is expanded and mounted; collapsing it
  // stops the refresh rather than leaving a timer running behind the interface.
  const [levelFilter, setConsoleLevel] = useState<ConsoleLevel>("all");
  const diagnosticEvents = useDiagnosticEvents(consoleOpen);
  const consoleRecords = useMemo(
    () =>
      levelFilter === "all"
        ? diagnosticEvents.records
        : diagnosticEvents.records.filter((record) => record.severity === levelFilter),
    [diagnosticEvents.records, levelFilter],
  );
  const latestRecord = diagnosticEvents.records.at(-1);
  const [itemsSection, setItemsSection] = useState<ItemsSection>("inventory");
  const [characterEntryTab, setCharacterEntryTab] = useState<"profile" | "appearance">("profile");
  const [toolsEntryTab, setToolsEntryTab] = useState<"settings" | "about">("settings");

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
  const consoleLevelLabels: Record<ConsoleLevel, string> = {
    all: t`All`,
    debug: t`Debug`,
    info: t`Info`,
    warning: t`Warning`,
    error: t`Error`,
  };
  const themeLabels: Record<ThemeName, string> = {
    light: t`Light`,
    dark: t`Dark`,
    "elden-ring": t`Elden Ring`,
  };

  const session = flow.session;
  const selectedCharacterID = flow.selection.selectedCharacterID;

  /**
   * The pending changes of the loaded save are exactly the operations the
   * backend still holds for it: an undone operation leaves that list for the
   * redo stack and a successful Save empties it. The count is only claimed
   * while the fetched history belongs to the session revision on screen, so a
   * session whose history is not read yet shows no number instead of a zero
   * it cannot confirm.
   */
  const history = flow.history;
  const pendingChangeCount =
    session !== undefined &&
    history !== undefined &&
    history.saveSessionID === session.saveSessionID &&
    history.saveRevision === session.saveRevision
      ? history.operations.length
      : undefined;

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
              onClick={() => {
                setCharacterEntryTab("profile");
                setToolsEntryTab("settings");
                setSection(name);
              }}
            >
              {labels[name]}
            </button>
          ))}
        </nav>

        <span className={topbarSpacer} />

        {session !== undefined && (
          <Badge dot tone={session.unsavedChanges ? "warning" : "accent"}>
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
            className={changesCounter}
            data-dirty={pendingChangeCount !== undefined && pendingChangeCount > 0}
            disabled={session === undefined || flow.isBusy}
            title={t`Review changes and validate before saving`}
            onClick={flow.openReview}
          >
            <span className={operationGlyph} aria-hidden="true">
              Δ
            </span>
            {/* The count itself is never hidden by width: it is the label. */}
            {pendingChangeCount === undefined || pendingChangeCount === 0 ? (
              <Trans>Changes</Trans>
            ) : (
              <Plural value={pendingChangeCount} one="# change" other="# changes" />
            )}
          </Button>
        </div>

        <div className={topbarSeparator} aria-hidden="true" />

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
        {section === "home" && (
          <SaveSessionContent
            flow={flow}
            onNavigate={(destination) => {
              if (destination === "appearance") {
                setCharacterEntryTab("appearance");
                setSection("character");
              } else if (destination === "database") {
                setItemsSection("database");
                setSection("items");
              } else {
                setToolsEntryTab(destination);
                setSection("tools");
              }
            }}
          />
        )}
        {section === "character" && (
          <section aria-label={t`Character`} className={screen}>
            <CharacterPanel
              initialTab={characterEntryTab}
              saveSessionID={session?.saveSessionID}
              saveRevision={session?.saveRevision}
              characterID={selectedCharacterID}
              applyMutationReceipt={flow.applyMutationReceipt}
              sessionBusy={flow.isBusy}
            />
          </section>
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
                applyMutationReceipt={flow.applyMutationReceipt}
                sessionBusy={flow.isBusy}
              />
            ) : (
              <ItemDatabasePanel
                saveSessionID={session?.saveSessionID}
                saveRevision={session?.saveRevision}
                characterID={selectedCharacterID}
                applyMutationReceipt={flow.applyMutationReceipt}
                sessionBusy={flow.isBusy}
              />
            )}
          </section>
        )}
        {section === "equipment" && (
          <section aria-label={t`Equipment`} className={screen}>
            <EquipmentPanel
              saveSessionID={session?.saveSessionID}
              saveRevision={session?.saveRevision}
              characterID={selectedCharacterID}
              applyMutationReceipt={flow.applyMutationReceipt}
              sessionBusy={flow.isBusy}
            />
          </section>
        )}
        {section === "world" && (
          <section aria-label={t`World`} className={screen}>
            <WorldPanel
              saveSessionID={session?.saveSessionID}
              saveRevision={session?.saveRevision}
              characterID={selectedCharacterID}
              applyMutationReceipt={flow.applyMutationReceipt}
              sessionBusy={flow.isBusy}
            />
          </section>
        )}
        {section === "advanced" && (
          <section aria-label={t`Advanced`} className={screen}>
            <AdvancedPanel
              saveSessionID={session?.saveSessionID}
              saveRevision={session?.saveRevision}
              applyMutationReceipt={flow.applyMutationReceipt}
              sessionBusy={flow.isBusy}
            />
          </section>
        )}
        {section === "tools" && (
          <section aria-label={t`Tools`} className={screen}>
            <ToolsPanel
              initialSubtab={toolsEntryTab}
              theme={theme}
              onThemeChange={onThemeChange}
              locale={locale}
              onLocaleChange={onLocaleChange}
              saveSessionID={session?.saveSessionID}
              saveRevision={session?.saveRevision}
              platform={session?.platform}
              characterID={selectedCharacterID}
              backupRetention={flow.lifecycleSettings?.backupRetention}
              backupNamePattern={flow.lifecycleSettings?.backupNamePattern}
              backupNameExample={flow.lifecycleSettings?.backupNameExample}
              backupSettingsStatus={flow.backupSettingsStatus}
              onBackupSettingsChange={flow.setBackupSettings}
              onOpenStagedFile={flow.openStagedFile}
              onOpenLocalFile={flow.openRecent}
              applyMutationReceipt={flow.applyMutationReceipt}
              sessionBusy={flow.isBusy}
            />
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
          <div className={consoleFilter}>
            <label htmlFor="console-level">
              <Trans>Level</Trans>
            </label>
            <Select
              id="console-level"
              value={levelFilter}
              onChange={(event) => setConsoleLevel(event.currentTarget.value as ConsoleLevel)}
            >
              {consoleLevels.map((level) => (
                <option key={level} value={level}>
                  {consoleLevelLabels[level]}
                </option>
              ))}
            </Select>
            <Button size="sm" onClick={() => setConsoleOpen(false)}>
              <Trans>Close</Trans>
            </Button>
          </div>
        </header>
        <div className={consolePanelBody}>
          {diagnosticEvents.failed ? (
            <p role="alert" className={message}>
              <Trans>The diagnostic messages could not be read.</Trans>
            </p>
          ) : consoleRecords.length === 0 ? (
            <p className={consoleEmpty}>
              <Trans>No diagnostic messages yet.</Trans>
            </p>
          ) : (
            <ul className={consoleList}>
              {consoleRecords.map((record) => (
                <li key={record.seq} className={consoleRow}>
                  <time
                    className={consoleTime}
                    dateTime={record.timestamp}
                    title={record.timestamp}
                  >
                    {consoleTimeLabel(record.timestamp)}
                  </time>
                  <span className={consoleLevel} data-level={record.severity}>
                    {record.severity}
                  </span>
                  {/* The message is the backend's own safe wording, rendered
                      unchanged: the frontend composes no diagnostic text. */}
                  <span>{[record.message, record.operation, record.stage, record.status, record.code, record.targetState].filter(Boolean).join(" · ")}</span>
                </li>
              ))}
            </ul>
          )}
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
          <span className={consoleIndicator} aria-hidden="true" />
          <strong>
            <Trans>Console</Trans>
          </strong>
          <span className={consoleLatest}>
            {latestRecord ? latestRecord.message : <Trans>No live messages</Trans>}
          </span>
          <Badge>{diagnosticEvents.records.length}</Badge>
        </button>
      </div>

      <ReviewChangesDialog flow={flow} />
      <PendingChangesDialog flow={flow} />
      <RecoveryJournalDialog flow={flow} />
    </div>
  );
}

/**
 * Shortens a diagnostic record's RFC 3339 timestamp to the console column's
 * `HH:mm:ssZ`. The trailing `Z` keeps the displayed time explicitly UTC, the
 * zone the backend records in, so it is never read as local time. The record's
 * own value is never rewritten: the caller keeps it in `dateTime` and `title`,
 * and a timestamp this function cannot parse falls back to a placeholder
 * instead of an invented hour.
 */
export function consoleTimeLabel(timestamp: string): string {
  const parsed = new Date(timestamp);
  return Number.isNaN(parsed.getTime()) ? "--:--:--" : `${parsed.toISOString().slice(11, 19)}Z`;
}

/** Returns only presentation text; the exact backend path stays untouched. */
export function fileNameFromPath(sourcePath: string): string {
  const parts = sourcePath.split(/[\\/]/);
  return parts.at(-1) || sourcePath;
}
