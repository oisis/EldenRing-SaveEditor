import { Trans, useLingui } from "@lingui/react/macro";
import { useCallback, useEffect, useMemo, useState } from "react";
import {
  useNetworkPresets,
  useNetworkSettings,
} from "../../application/network/useNetworkQueries";
import { useSetNetworkSettings } from "../../application/network/useNetworkMutations";
import type {
  NetworkParamValues,
  NetworkPreset,
  SetNetworkSettingsResult,
} from "../../application/network/networkPort";
import { Button } from "../../ui/components/Button/Button";
import { Card } from "../../ui/components/Card/Card";
import { alert, message, panel } from "../../ui/patterns/panel.css";
import {
  actionsBar,
  controlsGrid,
  groupDescription,
  groupHeader,
  groupSection,
  groupTitle,
  notice,
  presetButtons,
  presetMissingMessage,
  presetRoleItem,
  presetRolesCard,
  presetRoleTitle,
  subnav,
} from "./AdvancedPanel.css";
import { NetworkParamControl } from "./NetworkParamControl";
import {
  networkGroups,
  presetRoles,
  type PresetRoleDefinition,
} from "./networkMetadata";
import { SuperMarchantPlaceholder } from "./SuperMarchantPlaceholder";

export type AdvancedPanelProps = {
  saveSessionID?: string | undefined;
  saveRevision?: string | undefined;
  applyMutationReceipt?: ((receipt: SetNetworkSettingsResult) => Promise<unknown>) | undefined;
  sessionBusy?: boolean | undefined;
};

type AdvancedSubtab = "network" | "super-marchant";

type DraftState = {
  saveSessionID: string;
  saveRevision: string;
  values: NetworkParamValues;
};

function areParamsEqual(a: NetworkParamValues, b: NetworkParamValues): boolean {
  for (const key of Object.keys(a) as (keyof NetworkParamValues)[]) {
    if (a[key] !== b[key]) {
      return false;
    }
  }
  return true;
}

export function AdvancedPanel({
  saveSessionID,
  saveRevision,
  applyMutationReceipt,
  sessionBusy = false,
}: AdvancedPanelProps) {
  const { t, i18n } = useLingui();
  const [subtab, setSubtab] = useState<AdvancedSubtab>("network");

  const settingsQuery = useNetworkSettings({ saveSessionID, saveRevision });
  const presetsQuery = useNetworkPresets("");

  const mutations = useSetNetworkSettings(async (receipt) => {
    if (applyMutationReceipt) {
      await applyMutationReceipt(receipt);
    }
  });

  // Local draft explicitly bound to saveSessionID and saveRevision
  const [draft, setDraft] = useState<DraftState | null>(null);

  // Stable editing context key derived from saveSessionID and saveRevision
  const contextKey = `${saveSessionID ?? ""}:${saveRevision ?? ""}`;

  // Reset token incremented on explicit Reset, preset application and context change
  const [resetToken, setResetToken] = useState(0);

  // Track invalid / incomplete control inputs: map of key -> isValid
  const [invalidKeys, setInvalidKeys] = useState<Set<keyof NetworkParamValues>>(new Set());

  // Clear invalidKeys and signal controls strictly when the editing context changes
  useEffect(() => {
    setInvalidKeys(new Set());
    setResetToken((c) => c + 1);
  }, [contextKey]);

  const handleValidityChange = useCallback((key: keyof NetworkParamValues, isValid: boolean) => {
    setInvalidKeys((prev) => {
      const hasKey = prev.has(key);
      if (isValid) {
        if (!hasKey) return prev;
        const next = new Set(prev);
        next.delete(key);
        return next;
      }
      if (hasKey) return prev;
      const next = new Set(prev);
      next.add(key);
      return next;
    });
  }, []);

  // Synchronize draft when settings query succeeds with a new session or revision
  useEffect(() => {
    if (
      settingsQuery.data &&
      saveSessionID &&
      saveRevision &&
      settingsQuery.data.saveSessionID === saveSessionID &&
      settingsQuery.data.saveRevision === saveRevision
    ) {
      setDraft((current) => {
        if (
          !current ||
          current.saveSessionID !== saveSessionID ||
          current.saveRevision !== saveRevision
        ) {
          return {
            saveSessionID,
            saveRevision,
            values: { ...settingsQuery.data.parameters },
          };
        }
        return current;
      });
    } else if (!saveSessionID || !saveRevision) {
      setDraft(null);
    }
  }, [settingsQuery.data, saveSessionID, saveRevision]);

  // Active draft values strictly verified to match current session and revision
  const currentValues: NetworkParamValues | null = useMemo(() => {
    if (
      draft &&
      saveSessionID &&
      saveRevision &&
      draft.saveSessionID === saveSessionID &&
      draft.saveRevision === saveRevision
    ) {
      return draft.values;
    }
    return null;
  }, [draft, saveSessionID, saveRevision]);

  const presetsMap = useMemo(() => {
    const map = new Map<string, NetworkPreset>();
    if (presetsQuery.data?.presets) {
      for (const p of presetsQuery.data.presets) {
        map.set(p.id, p);
      }
    }
    return map;
  }, [presetsQuery.data]);

  const vanillaPreset = presetsMap.get("vanilla");

  const isModified = useMemo(() => {
    if (!currentValues || !settingsQuery.data?.parameters) return false;
    return !areParamsEqual(currentValues, settingsQuery.data.parameters);
  }, [currentValues, settingsQuery.data]);

  const hasInvalidField = invalidKeys.size > 0;
  const isBusy = sessionBusy || mutations.isBusy;

  // Strict canMutate condition
  const canMutate =
    Boolean(applyMutationReceipt) &&
    Boolean(saveSessionID && saveRevision) &&
    settingsQuery.isSuccess &&
    Boolean(currentValues) &&
    isModified &&
    !isBusy &&
    !hasInvalidField;

  const handleFieldChange = useCallback(
    (key: keyof NetworkParamValues, value: number) => {
      if (!saveSessionID || !saveRevision) return;
      setDraft((prev) => {
        const base =
          prev &&
          prev.saveSessionID === saveSessionID &&
          prev.saveRevision === saveRevision
            ? prev.values
            : settingsQuery.data?.parameters;
        if (!base) return null;
        return {
          saveSessionID,
          saveRevision,
          values: {
            ...base,
            [key]: value,
          },
        };
      });
    },
    [saveSessionID, saveRevision, settingsQuery.data],
  );

  // Compute union of fields where Faster differs from Vanilla OR Aggressive differs from Vanilla
  const getRoleFields = useCallback(
    (role: PresetRoleDefinition): (keyof NetworkParamValues)[] | null => {
      if (!vanillaPreset) return null;
      const faster = presetsMap.get(role.fasterPresetID);
      const aggressive = presetsMap.get(role.aggressivePresetID);
      if (!faster || !aggressive) return null;

      const diffKeys = new Set<keyof NetworkParamValues>();
      const allKeys = Object.keys(vanillaPreset.parameters) as (keyof NetworkParamValues)[];

      for (const k of allKeys) {
        if (faster.parameters[k] !== vanillaPreset.parameters[k]) {
          diffKeys.add(k);
        }
        if (aggressive.parameters[k] !== vanillaPreset.parameters[k]) {
          diffKeys.add(k);
        }
      }

      return Array.from(diffKeys);
    },
    [vanillaPreset, presetsMap],
  );

  const handleApplyRolePreset = useCallback(
    (role: PresetRoleDefinition, variant: "vanilla" | "faster" | "aggressive") => {
      if (!saveSessionID || !saveRevision || !vanillaPreset) return;
      const targetPresetID =
        variant === "vanilla"
          ? "vanilla"
          : variant === "faster"
            ? role.fasterPresetID
            : role.aggressivePresetID;
      const targetPreset = presetsMap.get(targetPresetID);
      if (!targetPreset) return;

      const affectedFields = getRoleFields(role);
      if (!affectedFields || affectedFields.length === 0) return;

      setDraft((prev) => {
        const base =
          prev &&
          prev.saveSessionID === saveSessionID &&
          prev.saveRevision === saveRevision
            ? prev.values
            : settingsQuery.data?.parameters;
        if (!base) return null;

        const next = { ...base };
        for (const k of affectedFields) {
          next[k] = targetPreset.parameters[k];
        }

        return {
          saveSessionID,
          saveRevision,
          values: next,
        };
      });
      setResetToken((c) => c + 1);
      setInvalidKeys(new Set());
    },
    [saveSessionID, saveRevision, vanillaPreset, presetsMap, settingsQuery.data, getRoleFields],
  );

  const handleReset = useCallback(() => {
    if (!saveSessionID || !saveRevision || !settingsQuery.data?.parameters) return;
    setDraft({
      saveSessionID,
      saveRevision,
      values: { ...settingsQuery.data.parameters },
    });
    setResetToken((c) => c + 1);
    setInvalidKeys(new Set());
  }, [saveSessionID, saveRevision, settingsQuery.data]);

  const handleApplyChanges = useCallback(async () => {
    if (!canMutate || !saveSessionID || !saveRevision || !currentValues) return;
    await mutations.setNetworkSettings({
      saveSessionID,
      expectedRevision: saveRevision,
      networkSettings: currentValues,
    });
  }, [canMutate, saveSessionID, saveRevision, currentValues, mutations]);

  return (
    <div className={panel}>
      <nav aria-label={t`Advanced subcategories`} className={subnav}>
        <Button
          tone={subtab === "network" ? "accent" : "neutral"}
          onClick={() => setSubtab("network")}
        >
          <Trans>Network Tuning</Trans>
        </Button>
        <Button
          tone={subtab === "super-marchant" ? "accent" : "neutral"}
          onClick={() => setSubtab("super-marchant")}
        >
          <Trans>Super marchant</Trans>
        </Button>
      </nav>

      {subtab === "super-marchant" && <SuperMarchantPlaceholder />}

      {subtab === "network" && (
        <>
          {!saveSessionID && (
            <Card>
              <p className={message}>
                <Trans>Open a save to view and edit network tuning parameters.</Trans>
              </p>
            </Card>
          )}

          {saveSessionID && settingsQuery.isPending && (
            <Card>
              <p className={message}>
                <Trans>Loading network settings…</Trans>
              </p>
            </Card>
          )}

          {saveSessionID && settingsQuery.isError && (
            <Card>
              <p role="alert" className={alert}>
                <Trans>Unable to load network settings from the save file.</Trans>
              </p>
            </Card>
          )}

          {saveSessionID && currentValues && (
            <div style={{ display: "flex", flexDirection: "column", gap: "1rem", marginTop: "1rem" }}>
              {/* Preset Roles Section */}
              <section className={presetRolesCard} aria-label={t`Preset roles`}>
                <div className={groupHeader}>
                  <div>
                    <h3 className={groupTitle}>
                      <Trans>Preset roles</Trans>
                    </h3>
                    <p className={groupDescription}>
                      <Trans>
                        Apply curated tuning profiles to specific matchmaking roles without affecting other parameters.
                      </Trans>
                    </p>
                  </div>
                </div>

                {presetRoles.map((role) => {
                  const fasterPreset = presetsMap.get(role.fasterPresetID);
                  const aggressivePreset = presetsMap.get(role.aggressivePresetID);
                  const isPresetRoleAvailable = Boolean(
                    presetsQuery.isSuccess && vanillaPreset && fasterPreset && aggressivePreset,
                  );

                  return (
                    <div key={role.id} className={presetRoleItem}>
                      <div>
                        <h4 className={presetRoleTitle}>{i18n._(role.title)}</h4>
                        {!isPresetRoleAvailable && (
                          <p role="status" className={presetMissingMessage}>
                            <Trans>Presets for this role are unavailable or incomplete.</Trans>
                          </p>
                        )}
                      </div>
                      <div className={presetButtons}>
                        <Button
                          size="sm"
                          disabled={isBusy || !isPresetRoleAvailable}
                          onClick={() => handleApplyRolePreset(role, "vanilla")}
                        >
                          <Trans>Vanilla</Trans>
                        </Button>
                        <Button
                          size="sm"
                          disabled={isBusy || !isPresetRoleAvailable}
                          onClick={() => handleApplyRolePreset(role, "faster")}
                        >
                          <Trans>Faster</Trans>
                        </Button>
                        <Button
                          size="sm"
                          disabled={isBusy || !isPresetRoleAvailable}
                          onClick={() => handleApplyRolePreset(role, "aggressive")}
                        >
                          <Trans>Aggressive</Trans>
                        </Button>
                      </div>
                    </div>
                  );
                })}
              </section>

              {/* 5 Parameter Groups */}
              {networkGroups.map((group) => {
                const groupTitleText = i18n._(group.title);
                const groupDescText = i18n._(group.description);

                return (
                  <section key={group.id} className={groupSection} aria-label={groupTitleText}>
                    <div className={groupHeader}>
                      <div>
                        <h3 className={groupTitle}>{groupTitleText}</h3>
                        <p className={groupDescription}>{groupDescText}</p>
                      </div>
                    </div>

                    <div className={controlsGrid}>
                      {group.fields.map((field) => (
                        <NetworkParamControl
                          key={field.key}
                          metadata={field}
                          value={currentValues[field.key]}
                          resetToken={resetToken}
                          disabled={isBusy}
                          onChange={handleFieldChange}
                          onValidityChange={handleValidityChange}
                        />
                      ))}
                    </div>
                  </section>
                );
              })}

              {mutations.error && (
                <p role="alert" className={alert}>
                  <Trans>Failed to apply network settings.</Trans>
                </p>
              )}

              <div className={actionsBar}>
                <Button
                  tone="accent"
                  disabled={!canMutate}
                  onClick={handleApplyChanges}
                >
                  <Trans>Apply changes</Trans>
                </Button>
                <Button
                  tone="neutral"
                  disabled={isBusy || !isModified}
                  onClick={handleReset}
                >
                  <Trans>Reset</Trans>
                </Button>
                {hasInvalidField && (
                  <span role="alert" className={alert}>
                    <Trans>Fix invalid field values before applying changes.</Trans>
                  </span>
                )}
                {!hasInvalidField && isModified && (
                  <span className={message}>
                    <Trans>You have unsaved changes in your draft.</Trans>
                  </span>
                )}
              </div>

              <div className={notice}>
                <strong>
                  <Trans>Network tuning notice</Trans>
                </strong>
                <p style={{ margin: 0 }}>
                  <Trans>
                    Custom network settings modify matchmaking timers and buffer limits stored in
                    UserData11. Applying changes creates an operation that will be reviewed in Review
                    Changes before writing to disk.
                  </Trans>
                </p>
              </div>
            </div>
          )}
        </>
      )}
    </div>
  );
}
