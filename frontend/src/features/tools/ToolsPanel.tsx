import { Trans, useLingui } from "@lingui/react/macro";
import { useEffect, useState } from "react";
import { useItemPreferences } from "../../application/preferences/itemPreferences";
import type { MutationReceipt } from "../../application/save-session/saveSessionPort";
import { useSetSaveAccountID } from "../../application/save-session/useSetSaveAccountID";
import { useSafetyProfile, useSetSafetyProfile } from "../../application/settings/useSafetyProfile";
import {
  useExportDiagnosticReport,
  useHostSettings,
  useOpenHostLocation,
  useSetHostSettings,
} from "../../application/settings/useHostSettings";
import type { Locale } from "../../i18n/i18n";
import { locales } from "../../i18n/i18n";
import { Button } from "../../ui/components/Button/Button";
import { Card } from "../../ui/components/Card/Card";
import { Checkbox } from "../../ui/components/Checkbox/Checkbox";
import { Input } from "../../ui/components/Input/Input";
import { Select } from "../../ui/components/Select/Select";
import { alert, message } from "../../ui/patterns/panel.css";
import type { ThemeName } from "../../ui/tokens/themes.css";
import { themeNames } from "../../ui/tokens/themes.css";
import { AboutTab } from "./AboutTab";
import { DeploymentTab } from "./DeploymentTab";
import { SaveIntegrityCard } from "./SaveIntegrityCard";
import { SaveManagerTab } from "./SaveManagerTab";
import { TemplatesTab } from "./TemplatesTab";
import {
  fieldWithAction,
  sections,
  settingField,
  settingItem,
  settingList,
  settingsRow,
  subnav,
} from "./ToolsPanel.css";

/** The five subtabs of `Tools`, in the accepted order. */
const subtabs = ["settings", "templates", "deployment", "save-manager", "about"] as const;

export type ToolsSubtab = (typeof subtabs)[number];

export type ToolsPanelProps = {
  theme: ThemeName;
  onThemeChange: (theme: ThemeName) => void;
  locale: Locale;
  onLocaleChange: (locale: Locale) => void;
  saveSessionID?: string | undefined;
  saveRevision?: string | undefined;
  /** The backend platform of the open session, carried verbatim. */
  platform?: string | undefined;
  characterID?: number | undefined;
  backupRetention?: number | undefined;
  onBackupRetentionChange: (retention: number) => void;
  /**
   * Loads a file SaveForge itself staged, such as a save downloaded from a
   * deployment target, as a temporary session.
   */
  onOpenStagedFile: (path: string) => void;
  /** Loads a durable local file the user saved, such as a downloaded backup. */
  onOpenLocalFile: (path: string) => void;
  /**
   * The shared save-mutation path. It is required: no control of this panel may
   * commit a mutation the session is never told about.
   */
  applyMutationReceipt: (receipt: MutationReceipt) => Promise<unknown>;
  sessionBusy?: boolean | undefined;
};

/**
 * The `Tools` module.
 *
 * It owns the horizontal subtab selection and nothing else: switching a subtab
 * is presentation state, so it touches neither the open session nor any other
 * module. Each of the five subtabs is a module of its own below this file.
 */
export function ToolsPanel({
  theme,
  onThemeChange,
  locale,
  onLocaleChange,
  saveSessionID,
  saveRevision,
  platform,
  characterID,
  backupRetention,
  onBackupRetentionChange,
  onOpenStagedFile,
  onOpenLocalFile,
  applyMutationReceipt,
  sessionBusy = false,
}: ToolsPanelProps) {
  const { t } = useLingui();
  const [subtab, setSubtab] = useState<ToolsSubtab>("settings");

  const subtabLabels: Record<ToolsSubtab, string> = {
    settings: t`Settings`,
    templates: t`Templates`,
    deployment: t`Deployment`,
    "save-manager": t`Save Manager`,
    about: t`About & Updates`,
  };

  return (
    <div className={sections}>
      <nav aria-label={t`Tools sections`} className={subnav}>
        {subtabs.map((name) => (
          <Button
            key={name}
            size="sm"
            pressed={subtab === name}
            onClick={() => setSubtab(name)}
          >
            {subtabLabels[name]}
          </Button>
        ))}
      </nav>

      {subtab === "settings" && (
        <SettingsTab
          theme={theme}
          onThemeChange={onThemeChange}
          locale={locale}
          onLocaleChange={onLocaleChange}
          saveSessionID={saveSessionID}
          saveRevision={saveRevision}
          platform={platform}
          characterID={characterID}
          backupRetention={backupRetention}
          onBackupRetentionChange={onBackupRetentionChange}
          applyMutationReceipt={applyMutationReceipt}
          sessionBusy={sessionBusy}
        />
      )}

      {subtab === "templates" && (
        <TemplatesTab
          saveSessionID={saveSessionID}
          saveRevision={saveRevision}
          characterID={characterID}
          applyMutationReceipt={applyMutationReceipt}
          sessionBusy={sessionBusy}
        />
      )}

      {subtab === "deployment" && (
        <DeploymentTab
          saveSessionID={saveSessionID}
          saveRevision={saveRevision}
          sessionBusy={sessionBusy}
          onDownloadedSave={onOpenStagedFile}
        />
      )}

      {subtab === "save-manager" && (
        <SaveManagerTab onOpenDownloadedBackup={onOpenLocalFile} />
      )}

      {subtab === "about" && <AboutTab />}
    </div>
  );
}

function SettingsTab({
  theme,
  onThemeChange,
  locale,
  onLocaleChange,
  saveSessionID,
  saveRevision,
  platform,
  characterID,
  backupRetention,
  onBackupRetentionChange,
  applyMutationReceipt,
  sessionBusy,
}: Omit<ToolsPanelProps, "sessionBusy" | "onOpenStagedFile" | "onOpenLocalFile"> & {
  sessionBusy: boolean;
}) {
  const { t } = useLingui();
  const preferences = useItemPreferences();
  const safetyProfile = useSafetyProfile();
  const setSafetyProfile = useSetSafetyProfile();
  const hostSettings = useHostSettings();
  const setHostSettings = useSetHostSettings();
  const openHostLocation = useOpenHostLocation();
  const exportReport = useExportDiagnosticReport();

  /**
   * Every host setting is stored as one complete value, so a change to one is
   * sent together with the other value the backend already holds. Sending a
   * patch would need a second, implicit source of truth for the fields it left
   * out.
   */
  const storeHostSettings = (patch: {
    skipReviewForNormalRisk?: boolean;
    remoteBackupPolicy?: string;
  }) => {
    const current = hostSettings.data;
    if (current === undefined) return;
    setHostSettings.mutate({
      skipReviewForNormalRisk: patch.skipReviewForNormalRisk ?? current.skipReviewForNormalRisk,
      remoteBackupPolicy: patch.remoteBackupPolicy ?? current.remoteBackupPolicy,
    });
  };
  const hostSettingsReady = hostSettings.data !== undefined && !setHostSettings.isPending;

  const [retentionDraft, setRetentionDraft] = useState(String(backupRetention ?? 10));
  useEffect(() => {
    if (backupRetention !== undefined) {
      setRetentionDraft(String(backupRetention));
    }
  }, [backupRetention]);

  const themeLabels: Record<ThemeName, string> = {
    light: t`Light`,
    dark: t`Dark`,
    "elden-ring": t`Elden Ring`,
  };
  // The three profile names are backend values; only their wording is local.
  const safetyProfileLabels: Record<string, string> = {
    safe: t`Safe`,
    expanded_limits: t`Expanded Limits`,
    chaos: t`Chaos Mode`,
  };
  const localeLabels: Record<Locale, string> = {
    en: t`English`,
    pl: t`Polish`,
  };

  return (
    <div className={sections}>
      <Card aria-label={t`Application`} className={sections}>
        <h2>
          <Trans>Application</Trans>
        </h2>
        <div className={settingsRow}>
          <span className={settingField}>
            <label htmlFor="tools-theme">
              <Trans>Theme</Trans>
            </label>
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
          </span>
          <span className={settingField}>
            <label htmlFor="tools-language">
              <Trans>Language</Trans>
            </label>
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
          </span>
          <SteamIDField
            key={`${saveSessionID ?? ""}:${saveRevision ?? ""}`}
            saveSessionID={saveSessionID}
            saveRevision={saveRevision}
            platform={platform}
            applyMutationReceipt={applyMutationReceipt}
            sessionBusy={sessionBusy}
          />
        </div>
      </Card>

      <Card aria-label={t`Safety profile`} className={sections}>
        <h2>
          <Trans>Safety profile</Trans>
        </h2>
        <span className={settingField}>
          <label htmlFor="tools-safety-profile">
            <Trans>Safety profile</Trans>
          </label>
          <Select
            id="tools-safety-profile"
            value={safetyProfile.data?.safetyProfile ?? ""}
            disabled={safetyProfile.data === undefined || setSafetyProfile.isPending}
            onChange={(event) => setSafetyProfile.mutate(event.currentTarget.value)}
          >
            {(safetyProfile.data?.availableProfiles ?? []).map((name) => (
              <option key={name} value={name}>
                {safetyProfileLabels[name] ?? name}
              </option>
            ))}
          </Select>
        </span>
        {safetyProfile.isError || setSafetyProfile.isError ? (
          <p role="alert" className={message}>
            {/* Reading and storing the setting fail for the same reasons
                and are reported the same way. The transport's own text
                never reaches the user: it carries bridge internals and
                host paths, and neither is actionable here. */}
            <Trans>The safety profile is unavailable.</Trans>
          </p>
        ) : null}
      </Card>

      <Card aria-label={t`Backups`} className={sections}>
        <h2>
          <Trans>Backups</Trans>
        </h2>
        <span className={settingField}>
          <label htmlFor="tools-backup-retention">
            <Trans>Automatic backups kept</Trans>
          </label>
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
                onBackupRetentionChange(retention);
              } else {
                setRetentionDraft(String(backupRetention ?? 10));
              }
            }}
            disabled={sessionBusy}
          />
        </span>
        {/* The backup name pattern is deliberately not offered. The backend
            states that the pattern stays fixed until a public grammar for it
            exists, and inventing tokens here would be a frontend rule replacing
            a contract that has not been agreed. */}
        <p className={message}>
          <Trans>
            The backup name pattern is fixed in this build: the backend defines no public
            grammar for it yet.
          </Trans>
        </p>
      </Card>

      <Card aria-label={t`Configuration`} className={sections}>
        <h2>
          <Trans>Configuration</Trans>
        </h2>
        <ul className={settingList}>
          <li className={settingItem}>
            {/* The directory is never named here: the frontend asks the backend
                to open a known location by identifier, so no path of the user's
                is rendered or sent. */}
            <Button
              disabled={
                hostSettings.data?.configurationDirectoryExists !== true ||
                openHostLocation.isPending
              }
              onClick={() => openHostLocation.mutate("configuration")}
            >
              <Trans>Open configuration directory</Trans>
            </Button>
          </li>
          <li className={settingItem}>
            <Checkbox
              id="tools-debug-mode"
              checked={false}
              disabled
              onChange={() => undefined}
            />
            <label htmlFor="tools-debug-mode">
              <Trans>Debug Mode</Trans>
            </label>
          </li>
          <li className={settingItem}>
            <Button disabled>
              <Trans>Open log directory</Trans>
            </Button>
          </li>
          <li className={settingItem}>
            <Button
              disabled={exportReport.isPending}
              onClick={() => exportReport.mutate(saveSessionID)}
            >
              <Trans>Export diagnostic report</Trans>
            </Button>
          </li>
          <li className={settingItem}>
            <Checkbox
              id="tools-show-item-id"
              checked={preferences.showItemID}
              onChange={(event) => preferences.setShowItemID(event.currentTarget.checked)}
            />
            <label htmlFor="tools-show-item-id">
              <Trans>Show Item ID</Trans>
            </label>
          </li>
        </ul>
        <p className={message}>
          <Trans>
            Debug Mode and local application logs remain unavailable until their diagnostic
            contract is defined.
          </Trans>
        </p>
        <p className={message}>
          {/* What the report may carry is stated plainly, because the user is
              the one who decides where it goes. */}
          <Trans>
            The diagnostic report carries the application version, the platform, the settings
            above and the current session's diagnostic records. It never carries save data,
            file paths or SSH keys.
          </Trans>
        </p>
        {openHostLocation.isError || exportReport.isError ? (
          <p role="alert" className={alert}>
            {/* The transport's own text never reaches the user: it carries
                bridge internals and host paths. */}
            <Trans>The action could not be completed.</Trans>
          </p>
        ) : null}
      </Card>

      <Card aria-label={t`Save behavior`} className={sections}>
        <h2>
          <Trans>Save behavior</Trans>
        </h2>
        <div className={settingsRow}>
          <span className={settingItem}>
            <Checkbox
              id="tools-skip-review"
              checked={hostSettings.data?.skipReviewForNormalRisk ?? false}
              disabled={!hostSettingsReady}
              onChange={(event) =>
                storeHostSettings({ skipReviewForNormalRisk: event.currentTarget.checked })
              }
            />
            <label htmlFor="tools-skip-review">
              <Trans>Skip Review Changes for normal operations</Trans>
            </label>
          </span>
          <span className={settingItem}>
            <Checkbox
              id="tools-remote-backup"
              checked={hostSettings.data?.remoteBackupPolicy === "always"}
              disabled={!hostSettingsReady}
              onChange={(event) =>
                storeHostSettings({
                  remoteBackupPolicy: event.currentTarget.checked ? "always" : "ask",
                })
              }
            />
            <label htmlFor="tools-remote-backup">
              <Trans>Always create a remote backup</Trans>
            </label>
          </span>
        </div>
        {/* Both settings are stored by the backend. Neither weakens the two
            rules they sit next to: validation always runs before a save or a
            deployment, and an existing target save is always backed up before
            it is replaced. */}
        <p className={message}>
          <Trans>
            Validation always runs. Skipping Review Changes only hides the modal for an
            operation the completed validation reported with no warning and no ban risk, and the
            remote backup setting only chooses whether the mandatory backup is confirmed each
            time.
          </Trans>
        </p>
        {hostSettings.isError || setHostSettings.isError ? (
          <p role="alert" className={alert}>
            <Trans>The save behavior settings are unavailable.</Trans>
          </p>
        ) : null}
      </Card>

      <SaveIntegrityCard
        key={`${saveSessionID ?? ""}:${saveRevision ?? ""}:${characterID ?? ""}`}
        saveSessionID={saveSessionID}
        saveRevision={saveRevision}
        characterID={characterID}
        applyMutationReceipt={applyMutationReceipt}
        sessionBusy={sessionBusy}
      />
    </div>
  );
}

/**
 * The Steam ID control.
 *
 * The backend accepts the identifier for a PC session only and exposes no
 * getter for the stored value, so the field is deliberately empty: it is a way
 * to state a new identifier, never a display of the current one. The draft is
 * remounted with the session and the revision it belongs to, so a value typed
 * for one save can never be submitted against another. The identifier itself is
 * never stored, echoed into a message or written anywhere but the backend call.
 */
function SteamIDField({
  saveSessionID,
  saveRevision,
  platform,
  applyMutationReceipt,
  sessionBusy,
}: {
  saveSessionID: string | undefined;
  saveRevision: string | undefined;
  platform: string | undefined;
  applyMutationReceipt: (receipt: MutationReceipt) => Promise<unknown>;
  sessionBusy: boolean;
}) {
  const setSaveAccountID = useSetSaveAccountID();
  const [draft, setDraft] = useState("");
  const [stored, setStored] = useState(false);
  const [syncFailed, setSyncFailed] = useState(false);

  // "pc" is the backend's own platform value, compared and never normalised.
  const supported = saveSessionID !== undefined && saveRevision !== undefined && platform === "pc";
  const disabled = !supported || sessionBusy || setSaveAccountID.isPending;

  async function submit() {
    if (!supported || draft === "") return;
    setStored(false);
    setSyncFailed(false);
    let receipt;
    try {
      receipt = await setSaveAccountID.mutateAsync({
        saveSessionID,
        accountID: draft,
        expectedRevision: saveRevision,
      });
    } catch {
      // A rejected call stored nothing, which the hook's error state below
      // already states. It is caught only so it never escapes unhandled, and
      // the draft is kept so the user can correct it.
      return;
    }
    // The identifier is stored from here on, so a failing refresh below may not
    // be reported as a rejected value.
    setDraft("");
    setStored(true);
    try {
      await applyMutationReceipt(receipt);
    } catch {
      setSyncFailed(true);
    }
  }

  return (
    <span className={settingField}>
      <label htmlFor="tools-steam-id">
        <Trans>Steam ID</Trans>
      </label>
      <span className={fieldWithAction}>
        <Input
          id="tools-steam-id"
          type="text"
          inputMode="numeric"
          autoComplete="off"
          value={draft}
          disabled={disabled}
          onChange={(event) => {
            setStored(false);
            setSyncFailed(false);
            setDraft(event.currentTarget.value);
          }}
        />
        <Button
          size="sm"
          disabled={disabled || draft === ""}
          onClick={() => void submit()}
        >
          <Trans>Set Steam ID</Trans>
        </Button>
      </span>
      {saveSessionID === undefined ? (
        <span className={message}>
          <Trans>Open a PC save to set its Steam ID.</Trans>
        </span>
      ) : platform !== "pc" ? (
        <span className={message}>
          <Trans>The backend sets the Steam ID for PC saves only.</Trans>
        </span>
      ) : (
        <span className={message}>
          <Trans>The backend does not report the stored Steam ID, so this field stays empty.</Trans>
        </span>
      )}
      {setSaveAccountID.isError ? (
        <span role="alert" className={alert}>
          {/* The rejected identifier is never repeated back, here or in a log. */}
          <Trans>The Steam ID was rejected and nothing was changed.</Trans>
        </span>
      ) : null}
      {stored ? (
        <span role="status" className={message}>
          <Trans>The Steam ID was stored.</Trans>
        </span>
      ) : null}
      {syncFailed ? (
        <span role="alert" className={alert}>
          <Trans>
            The Steam ID was stored, but this screen could not be refreshed. Reopen the save to
            see its current state.
          </Trans>
        </span>
      ) : null}
    </span>
  );
}
