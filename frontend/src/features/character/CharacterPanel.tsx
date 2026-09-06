import { t } from "@lingui/core/macro";
import { Trans } from "@lingui/react/macro";
import { useMemo, useState } from "react";
import { useAppearancePresets } from "../../application/appearance/useAppearancePresets";
import { appearancePresetAssetURL } from "../../application/catalog/catalogAssetURL";
import { useCatalogResources } from "../../application/catalog/useCatalogResources";
import type {
  CharacterAttributes,
  CharacterStats,
} from "../../application/character/characterPort";
import { useCharacterMutations } from "../../application/character/useCharacterMutations";
import { useCharacterProfile } from "../../application/character/useCharacterProfile";
import { useCharacterStats } from "../../application/character/useCharacterStats";
import { useCharacterLoadout } from "../../application/equipment/useEquipment";
import { useFavoritePresets } from "../../application/favorites/useFavoritePresets";
import type { MutationReceipt } from "../../application/save-session/saveSessionPort";
import { Badge } from "../../ui/components/Badge/Badge";
import { Button } from "../../ui/components/Button/Button";
import { Card } from "../../ui/components/Card/Card";
import { Checkbox } from "../../ui/components/Checkbox/Checkbox";
import { Dialog } from "../../ui/components/Dialog/Dialog";
import { Input } from "../../ui/components/Input/Input";
import { Select } from "../../ui/components/Select/Select";
import { alert, message, panel } from "../../ui/patterns/panel.css";
import {
  disclosure,
  disclosureHeading,
  disclosureBody,
  workspaceStack,
} from "../../ui/patterns/workspace.css";
import {
  addSettingsAccordion,
  addSettingsBody,
  addSettingsGrid,
  addSettingsHeading,
  addSettingsSwitches,
  addSettingsTitle,
  addSettingsTitleGroup,
  cardBadges,
  cardBody,
  cardHeader,
  cardTitle,
  derivedGrid,
  derivedGroup,
  derivedGroupTitle,
  derivedStat,
  derivedStatLabel,
  derivedStatSub,
  derivedStatValue,
  favoriteSlotActions,
  favoriteSlotCard,
  favoriteSlotHeader,
  favoritesGrid,
  field,
  fieldGroup,
  fieldHint,
  fieldLabel,
  fieldLabelRow,
  fieldLabelSuffix,
  fieldRow,
  identityCard,
  identityGrid,
  presetContainer,
  presetControls,
  presetImage,
  presetImagePlaceholder,
  presetNeighbor,
  presetStage,
  presetTags,
  presetViewer,
  profileAdvanced,
  profileGrid,
  profileLower,
  profileNote,
  profilePanel,
  profileSection,
  profileSectionTitle,
  profileSections,
  rangeControl,
  rangeLabel,
  rangeNumberInput,
  rangeSlider,
  statBoxValue,
  subnav,
  summaryContent,
  summaryMeta,
  switchLabel,
  switchText,
} from "./CharacterPanel.css";

/** The confirmed backend maximum for held runes; the field never offers more. */
const runesHeldMaximum = 999_999_999;

export type CharacterPanelProps = {
  initialTab?: "profile" | "appearance";
  saveSessionID?: string | undefined;
  saveRevision?: string | undefined;
  characterID?: number | undefined;
  applyMutationReceipt?: ((receipt: MutationReceipt) => Promise<unknown>) | undefined;
  sessionBusy?: boolean;
};

const attributeKeys: (keyof CharacterAttributes)[] = [
  "vigor",
  "mind",
  "endurance",
  "strength",
  "dexterity",
  "intelligence",
  "faith",
  "arcane",
];

/**
 * The eight editable attributes as the backend reported them. The draft is only
 * ever seeded from a successful read, so the panel never offers a fabricated
 * default that a save could overwrite real statistics with.
 */
function statsAttributes(stats: CharacterStats): CharacterAttributes {
  return {
    vigor: stats.vigor,
    mind: stats.mind,
    endurance: stats.endurance,
    strength: stats.strength,
    dexterity: stats.dexterity,
    intelligence: stats.intelligence,
    faith: stats.faith,
    arcane: stats.arcane,
  };
}

const INFUSIONS = [
  "Standard",
  "Heavy",
  "Keen",
  "Quality",
  "Fire",
  "Flame Art",
  "Lightning",
  "Sacred",
  "Magic",
  "Cold",
  "Poison",
  "Blood",
  "Occult",
];

export function CharacterPanel({
  initialTab = "profile",
  saveSessionID,
  saveRevision,
  characterID,
  applyMutationReceipt,
  sessionBusy = false,
}: CharacterPanelProps) {
  const [activeTab, setActiveTab] = useState<"profile" | "appearance">(initialTab);

  const mutations = useCharacterMutations(applyMutationReceipt ?? (async () => {}));
  const isBusy = sessionBusy || mutations.isBusy;
  const hasActiveCharacter =
    saveSessionID !== undefined && characterID !== undefined && saveRevision !== undefined;

  // Profile queries
  const profileQuery = useCharacterProfile(saveSessionID, saveRevision, characterID);
  const statsQuery = useCharacterStats(saveSessionID, saveRevision, characterID);
  // Memory Stones and Talisman Slots are owned by the Equipment loadout contract.
  // Character reads that one existing snapshot instead of restating its rules.
  const loadoutQuery = useCharacterLoadout({ saveSessionID, saveRevision, characterID });
  const classesQuery = useCatalogResources({
    resourceType: "class",
    family: "",
    capability: "",
    endpointID: "",
    search: "",
    page: 1,
    pageSize: 50,
  });

  // Appearance queries
  const [searchDraft, setSearchDraft] = useState("");
  const [bodyTypeFilter, setBodyTypeFilter] = useState<"all" | "Type A" | "Type B">("all");
  const presetsQuery = useAppearancePresets({ search: searchDraft });
  const favoritesQuery = useFavoritePresets(saveSessionID, saveRevision);

  // Every editable field stays closed until its own read succeeded. A pending or
  // failed read never renders a placeholder as if it were backend data, and it
  // never enables the mutation that would write that placeholder back.
  const profileReady = profileQuery.isSuccess;
  const statsReady = statsQuery.isSuccess;
  const loadoutReady = loadoutQuery.isSuccess;
  const classesReady = classesQuery.isSuccess;
  const presetsReady = presetsQuery.isSuccess;
  const favoritesReady = favoritesQuery.isSuccess;

  // Profile form drafts. Each draft carries the exact edit identity it was typed
  // for, so a draft belonging to another session, character or revision can
  // never be rendered for — or written to — the character currently in view.
  const editContextKey = `${saveSessionID ?? ""}|${characterID ?? ""}|${saveRevision ?? ""}`;

  const [nameDraft, setNameDraft] = useState<{ key: string; value: string } | undefined>(undefined);
  const nameValue =
    nameDraft?.key === editContextKey ? nameDraft.value : (profileQuery.data?.name ?? "");

  const [attributesDraft, setAttributesDraft] = useState<
    { key: string; value: CharacterAttributes } | undefined
  >(undefined);
  const attributeValues =
    attributesDraft?.key === editContextKey
      ? attributesDraft.value
      : statsQuery.isSuccess
        ? statsAttributes(statsQuery.data)
        : undefined;

  // Held runes are edited as raw text so a partially typed value is never
  // rounded, clamped or silently written back; the save action validates it.
  const [runesDraft, setRunesDraft] = useState<{ key: string; value: string } | undefined>(
    undefined,
  );
  const runesValue =
    runesDraft?.key === editContextKey
      ? runesDraft.value
      : statsQuery.isSuccess
        ? String(statsQuery.data.runes)
        : undefined;
  const runesParsed = runesValue === undefined ? Number.NaN : Number(runesValue);
  const runesValid =
    runesValue !== undefined &&
    runesValue.trim() !== "" &&
    Number.isInteger(runesParsed) &&
    runesParsed >= 0 &&
    runesParsed <= runesHeldMaximum;

  // Dialog states
  const [classPickerOpen, setClassPickerOpen] = useState(false);
  const [pendingClassID, setPendingClassID] = useState<number>(0);
  const [confirmClassResetOpen, setConfirmClassResetOpen] = useState(false);

  const [confirmGenderOpen, setConfirmGenderOpen] = useState(false);
  const [pendingGender, setPendingGender] = useState<number>(0);

  const [confirmReplaceFavSlot, setConfirmReplaceFavSlot] = useState<number | null>(null);
  const [confirmDeleteFavSlot, setConfirmDeleteFavSlot] = useState<number | null>(null);

  // Resolved class name. It stays empty until both reads succeeded, so an
  // unresolved identifier is never presented as the character's class.
  const currentClassName = useMemo(() => {
    if (!profileQuery.isSuccess || !classesQuery.isSuccess) return "";
    const classRes = classesQuery.data.resources.find(
      (r) => r.key === String(profileQuery.data.startingClassID),
    );
    return classRes?.name ?? `Class ${profileQuery.data.startingClassID}`;
  }, [profileQuery.isSuccess, profileQuery.data, classesQuery.isSuccess, classesQuery.data]);

  // Filtered appearance presets
  const filteredPresets = useMemo(() => {
    const list = presetsQuery.data ?? [];
    if (bodyTypeFilter === "all") return list;
    return list.filter((p) => p.bodyType === bodyTypeFilter);
  }, [presetsQuery.data, bodyTypeFilter]);

  // A search or filter change shrinks the list without touching the stored
  // index, so every consumer reads the clamped index instead. That is what keeps
  // the counter, the arrows and the applied preset on the same entry.
  const [selectedPresetIndex, setSelectedPresetIndex] = useState(0);
  const safePresetIndex = selectedPresetIndex >= filteredPresets.length ? 0 : selectedPresetIndex;
  const activePreset = filteredPresets[safePresetIndex];
  const activePresetImageURL =
    activePreset === undefined ? undefined : appearancePresetAssetURL(activePreset.image);

  // Actions
  const handleSaveName = async (event: React.FormEvent) => {
    event.preventDefault();
    if (!hasActiveCharacter || !profileReady || !nameValue.trim() || isBusy) return;
    await mutations.setName({
      saveSessionID,
      characterID,
      name: nameValue,
      expectedRevision: saveRevision,
    });
  };

  const handleSaveAttributes = async () => {
    if (!hasActiveCharacter || !statsReady || attributeValues === undefined || isBusy) return;
    await mutations.setStats({
      saveSessionID,
      characterID,
      attributes: attributeValues,
      levelPolicy: "recalculate",
      expectedRevision: saveRevision,
    });
  };

  const handleSaveRunes = async () => {
    if (!hasActiveCharacter || !statsReady || !runesValid || isBusy) return;
    await mutations.setRunes({
      saveSessionID,
      characterID,
      runes: runesParsed,
      expectedRevision: saveRevision,
    });
  };

  const handleResetAttributesDraft = () => {
    setAttributesDraft(undefined);
  };

  const handleConfirmClassChange = async () => {
    if (!hasActiveCharacter || !profileReady || !classesReady || isBusy) return;
    setConfirmClassResetOpen(false);
    await mutations.setStartingClass({
      saveSessionID,
      characterID,
      startingClassID: pendingClassID,
      confirmReset: true,
      expectedRevision: saveRevision,
    });
  };

  const handleConfirmGenderChange = async () => {
    if (!hasActiveCharacter || !profileReady || isBusy) return;
    setConfirmGenderOpen(false);
    await mutations.setGender({
      saveSessionID,
      characterID,
      gender: pendingGender,
      expectedRevision: saveRevision,
    });
  };

  const handleApplyPreset = async () => {
    if (!hasActiveCharacter || !presetsReady || !activePreset || isBusy) return;
    await mutations.applyAppearancePreset({
      saveSessionID,
      characterID,
      presetID: activePreset.id,
      expectedRevision: saveRevision,
    });
  };

  const handleApplyFavorite = async (slotID: number) => {
    if (!hasActiveCharacter || !favoritesReady || isBusy) return;
    await mutations.applyFavoritePreset({
      saveSessionID,
      characterID,
      favoriteSlotID: slotID,
      expectedRevision: saveRevision,
    });
  };

  const handleSaveFavorite = async (slotID: number) => {
    if (!hasActiveCharacter || !favoritesReady || isBusy) return;
    await mutations.setFavoritePreset({
      saveSessionID,
      favoriteSlotID: slotID,
      sourceCharacterID: characterID,
      expectedRevision: saveRevision,
    });
  };

  const handleConfirmReplaceFavorite = async () => {
    if (!hasActiveCharacter || !favoritesReady || confirmReplaceFavSlot === null || isBusy) return;
    const slotID = confirmReplaceFavSlot;
    setConfirmReplaceFavSlot(null);
    await mutations.setFavoritePreset({
      saveSessionID,
      favoriteSlotID: slotID,
      sourceCharacterID: characterID,
      expectedRevision: saveRevision,
    });
  };

  const handleConfirmDeleteFavorite = async () => {
    if (
      saveSessionID === undefined ||
      saveRevision === undefined ||
      !favoritesReady ||
      confirmDeleteFavSlot === null ||
      isBusy
    ) {
      return;
    }
    const slotID = confirmDeleteFavSlot;
    setConfirmDeleteFavSlot(null);
    await mutations.deleteFavoritePreset({
      saveSessionID,
      favoriteSlotID: slotID,
      expectedRevision: saveRevision,
    });
  };

  const attributeLabels: Record<keyof CharacterAttributes, string> = {
    vigor: t`Vigor`,
    mind: t`Mind`,
    endurance: t`Endurance`,
    strength: t`Strength`,
    dexterity: t`Dexterity`,
    intelligence: t`Intelligence`,
    faith: t`Faith`,
    arcane: t`Arcane`,
  };

  const attributeShortNames: Record<keyof CharacterAttributes, string> = {
    vigor: t`Vig`,
    mind: t`Min`,
    endurance: t`End`,
    strength: t`Str`,
    dexterity: t`Dex`,
    intelligence: t`Int`,
    faith: t`Fai`,
    arcane: t`Arc`,
  };

  const infusionLabels: Record<string, string> = {
    Standard: t`Standard`,
    Heavy: t`Heavy`,
    Keen: t`Keen`,
    Quality: t`Quality`,
    Fire: t`Fire`,
    "Flame Art": t`Flame Art`,
    Lightning: t`Lightning`,
    Sacred: t`Sacred`,
    Magic: t`Magic`,
    Cold: t`Cold`,
    Poison: t`Poison`,
    Blood: t`Blood`,
    Occult: t`Occult`,
  };

  const attrSummary = attributeValues
    ? attributeKeys
        .map((attr) => `${attributeShortNames[attr]} ${attributeValues[attr]}`)
        .join(" · ")
    : "";

  const resourcesSummary = statsQuery.isSuccess
    ? `HP ${statsQuery.data.hp}/${statsQuery.data.maxHP} · FP ${statsQuery.data.fp}/${statsQuery.data.maxFP} · SP ${statsQuery.data.sp}/${statsQuery.data.maxSP}`
    : "";

  return (
    <div className={`${panel} ${workspaceStack}`}>
      <nav aria-label={t`Character navigation`} className={subnav}>
        <Button size="sm" pressed={activeTab === "profile"} onClick={() => setActiveTab("profile")}>
          <Trans>Profile</Trans>
        </Button>
        <Button
          size="sm"
          pressed={activeTab === "appearance"}
          onClick={() => setActiveTab("appearance")}
        >
          <Trans>Appearance Presets</Trans>
        </Button>
      </nav>

      {/* The one rendering of a failed mutation. The interface owns the wording:
          the backend message may name a host path or an internal identifier and
          is never shown. Only the stable code is surfaced, separately, so a user
          can quote it in a report. */}
      {mutations.error !== undefined ? (
        <div role="alert">
          <p className={alert}>
            <Trans>The change was not applied.</Trans>
          </p>
          <p className={message}>
            <Trans>Error code: {mutations.error.code}</Trans>
          </p>
        </div>
      ) : null}

      {activeTab === "profile" && (
        <>
          {!hasActiveCharacter ? (
            <Card>
              <p className={message}>
                <Trans>Select an active character to view and edit profile data.</Trans>
              </p>
            </Card>
          ) : (
            <>
              <Card aria-label={t`Identity and Progression`} className={identityCard}>
                <header className={cardHeader}>
                  <h2 className={cardTitle}>
                    <Trans>Identity &amp; Progression</Trans>
                  </h2>
                  <div className={cardBadges}>
                    {profileReady ? (
                      <Badge tone="neutral">RL {profileQuery.data.level}</Badge>
                    ) : null}
                  </div>
                </header>
                <div className={cardBody}>
                  {profileQuery.isPending ? (
                    <p role="status" className={message}>
                      <Trans>Loading character profile…</Trans>
                    </p>
                  ) : null}
                  {profileQuery.isError ? (
                    <p role="alert" className={alert}>
                      <Trans>Unable to load the profile of this character slot.</Trans>
                    </p>
                  ) : null}
                  <div className={profileSections}>
                    <section className={profileSection}>
                      <h3 className={profileSectionTitle}>
                        <Trans>Identity</Trans>
                      </h3>
                      <div className={profileGrid}>
                        <form className={field} onSubmit={handleSaveName}>
                          <label htmlFor="character-name-input" className={fieldLabel}>
                            <Trans>Name</Trans>
                          </label>
                          <div className={fieldRow}>
                            <Input
                              id="character-name-input"
                              value={nameValue}
                              disabled={isBusy || !profileReady}
                              onChange={(e) =>
                                setNameDraft({ key: editContextKey, value: e.currentTarget.value })
                              }
                            />
                            <Button
                              type="submit"
                              size="sm"
                              disabled={
                                isBusy ||
                                !profileReady ||
                                !nameValue.trim() ||
                                nameValue === (profileQuery.data?.name ?? "")
                              }
                            >
                              <Trans>Save</Trans>
                            </Button>
                          </div>
                        </form>

                        <div className={field}>
                          <label htmlFor="character-starting-class-input" className={fieldLabel}>
                            <Trans>Starting Class</Trans>
                          </label>
                          <div className={fieldRow}>
                            <Input
                              id="character-starting-class-input"
                              aria-label={t`Starting Class`}
                              readOnly
                              value={
                                profileReady && classesReady
                                  ? currentClassName
                                  : profileQuery.isError || classesQuery.isError
                                    ? t`Unavailable`
                                    : t`Loading…`
                              }
                              disabled={isBusy || !profileReady}
                            />
                            <Button
                              type="button"
                              size="sm"
                              disabled={isBusy || !profileReady || !classesReady}
                              onClick={() => {
                                setPendingClassID(profileQuery.data?.startingClassID ?? 0);
                                setClassPickerOpen(true);
                              }}
                            >
                              <Trans>Change Class</Trans>
                            </Button>
                          </div>
                        </div>

                        <div className={field}>
                          <span className={fieldLabel}>
                            <Trans>Body Type</Trans>
                          </span>
                          <div className={fieldRow}>
                            {!profileReady ? (
                              <Badge tone="neutral">
                                {profileQuery.isError ? (
                                  <Trans>Unavailable</Trans>
                                ) : (
                                  <Trans>Loading…</Trans>
                                )}
                              </Badge>
                            ) : profileQuery.data.gender === 0 || profileQuery.data.gender === 1 ? (
                              <>
                                <Badge>
                                  {profileQuery.data.gender === 1 ? "Type A" : "Type B"}
                                </Badge>
                                <Button
                                  type="button"
                                  size="sm"
                                  disabled={isBusy}
                                  onClick={() => {
                                    if (!profileQuery.data) return;
                                    setPendingGender(profileQuery.data.gender === 1 ? 0 : 1);
                                    setConfirmGenderOpen(true);
                                  }}
                                >
                                  <Trans>
                                    Switch to{" "}
                                    {profileQuery.data.gender === 1 ? "Type B" : "Type A"}
                                  </Trans>
                                </Button>
                              </>
                            ) : null}
                          </div>
                        </div>
                      </div>
                    </section>

                    <section aria-label={t`Progression`} className={profileSection}>
                      <h3 className={profileSectionTitle}>
                        <Trans>Progression</Trans>
                      </h3>
                      {statsQuery.isPending ? (
                        <p role="status" className={message}>
                          <Trans>Loading progression…</Trans>
                        </p>
                      ) : null}
                      {statsQuery.isError ? (
                        <p role="alert" className={alert}>
                          <Trans>Unable to load the progression of this character slot.</Trans>
                        </p>
                      ) : null}
                      <div className={profileGrid}>
                        <div className={field}>
                          <div className={fieldLabelRow}>
                            <label htmlFor="character-rl-input" className={fieldLabel}>
                              <Trans>Rune Level</Trans>
                            </label>
                            {profileReady ? (
                              <span className={statBoxValue}>{profileQuery.data.level}</span>
                            ) : null}
                          </div>
                          <Input
                            id="character-rl-input"
                            type="number"
                            readOnly
                            value={profileReady ? String(profileQuery.data.level) : ""}
                            disabled={isBusy || !profileReady}
                            aria-label={t`Rune Level`}
                          />
                          <span className={fieldHint}>
                            <Trans>Determined by attribute total.</Trans>
                          </span>
                        </div>

                        <div className={field}>
                          <div className={fieldLabelRow}>
                            <label htmlFor="character-ng-input" className={fieldLabel}>
                              <Trans>NG+ Cycle</Trans>
                            </label>
                            <span className={fieldLabelSuffix}>
                              <Trans>Unavailable</Trans>
                            </span>
                          </div>
                          <Input
                            id="character-ng-input"
                            readOnly
                            value="--"
                            disabled
                            aria-label={t`NG+ Cycle`}
                          />
                          <span className={fieldHint}>
                            <Trans>Unavailable (save contract deferred).</Trans>
                          </span>
                        </div>

                        {statsReady && runesValue !== undefined ? (
                          <div className={field}>
                            <label htmlFor="character-runes-input" className={fieldLabel}>
                              <Trans>Runes Held</Trans>
                            </label>
                            <div className={fieldRow}>
                              <Input
                                id="character-runes-input"
                                type="number"
                                min={0}
                                max={runesHeldMaximum}
                                value={runesValue}
                                disabled={isBusy}
                                aria-label={t`Runes Held`}
                                onChange={(e) =>
                                  setRunesDraft({
                                    key: editContextKey,
                                    value: e.currentTarget.value,
                                  })
                                }
                              />
                              <Button
                                type="button"
                                size="sm"
                                disabled={isBusy || !runesValid}
                                onClick={handleSaveRunes}
                              >
                                <Trans>Save Runes</Trans>
                              </Button>
                            </div>
                          </div>
                        ) : null}

                        {statsReady ? (
                          <div className={field}>
                            <div className={fieldLabelRow}>
                              <label htmlFor="character-sm-input" className={fieldLabel}>
                                <Trans>Soul Memory</Trans>
                              </label>
                              <span className={statBoxValue}>{statsQuery.data.soulMemory}</span>
                            </div>
                            <Input
                              id="character-sm-input"
                              type="number"
                              readOnly
                              value={String(statsQuery.data.soulMemory)}
                              disabled={isBusy}
                              aria-label={t`Soul Memory`}
                            />
                            <span className={fieldHint}>
                              <Trans>Total runes acquired. Read-only.</Trans>
                            </span>
                          </div>
                        ) : null}

                        {loadoutQuery.isPending ? (
                          <p role="status" className={message}>
                            <Trans>Loading loadout capacity…</Trans>
                          </p>
                        ) : null}
                        {loadoutQuery.isError ? (
                          <p role="alert" className={alert}>
                            <Trans>Unable to load the loadout capacity of this character slot.</Trans>
                          </p>
                        ) : null}
                        {loadoutReady ? (
                          <>
                            <div className={field}>
                              <div className={fieldLabelRow}>
                                <label htmlFor="character-memory-stones-input" className={fieldLabel}>
                                  <Trans>Memory Stones</Trans>
                                </label>
                                <span className={fieldLabelSuffix}>
                                  <span className={statBoxValue}>{loadoutQuery.data.memoryStones}</span>
                                  /8
                                </span>
                              </div>
                              <Input
                                id="character-memory-stones-input"
                                type="number"
                                readOnly
                                value={String(loadoutQuery.data.memoryStones)}
                                disabled={isBusy}
                                aria-label={t`Memory Stones`}
                              />
                              <span className={fieldHint}>
                                <Trans>Read-only loadout capacity.</Trans>
                              </span>
                            </div>
                            <div className={field}>
                              <div className={fieldLabelRow}>
                                <label htmlFor="character-talisman-slots-input" className={fieldLabel}>
                                  <Trans>Talisman Slots</Trans>
                                </label>
                                <span className={fieldLabelSuffix}>
                                  <span className={statBoxValue}>
                                    {loadoutQuery.data.unlockedTalismanSlots}
                                  </span>
                                  /4
                                </span>
                              </div>
                              <Input
                                id="character-talisman-slots-input"
                                type="number"
                                readOnly
                                value={String(loadoutQuery.data.unlockedTalismanSlots)}
                                disabled={isBusy}
                                aria-label={t`Talisman Slots`}
                              />
                              <span className={fieldHint}>
                                <Trans>Read-only loadout capacity.</Trans>
                              </span>
                            </div>
                          </>
                        ) : null}
                      </div>
                    </section>
                  </div>
                </div>
              </Card>

              <div className={profileAdvanced}>
                <details className={addSettingsAccordion}>
                  <summary className={addSettingsHeading}>
                    <div className={addSettingsTitleGroup}>
                      <h3 className={addSettingsTitle}>
                        <Trans>Add Settings</Trans>
                      </h3>
                      <span className={fieldLabelSuffix}>
                        <Trans>Unavailable</Trans>
                      </span>
                    </div>
                  </summary>
                  <div className={addSettingsBody}>
                    <p className={fieldHint}>
                      <Trans>Add Settings are deferred and not connected to item operations.</Trans>
                    </p>
                    <div className={addSettingsGrid}>
                      <div className={rangeControl}>
                        <label htmlFor="add-settings-weapon25-slider" className={rangeLabel}>
                          <Trans>Weapon +25</Trans>
                        </label>
                        <input
                          id="add-settings-weapon25-slider"
                          type="range"
                          min={0}
                          max={25}
                          value={0}
                          disabled
                          readOnly
                          aria-label={t`Weapon +25 level`}
                          className={rangeSlider}
                        />
                        <Input
                          id="add-settings-weapon25-number"
                          type="number"
                          aria-label={t`Weapon +25 level input`}
                          min={0}
                          max={25}
                          value="0"
                          disabled
                          readOnly
                          className={rangeNumberInput}
                        />
                      </div>

                      <div className={rangeControl}>
                        <label htmlFor="add-settings-weapon10-slider" className={rangeLabel}>
                          <Trans>Weapon +10</Trans>
                        </label>
                        <input
                          id="add-settings-weapon10-slider"
                          type="range"
                          min={0}
                          max={10}
                          value={0}
                          disabled
                          readOnly
                          aria-label={t`Weapon +10 level`}
                          className={rangeSlider}
                        />
                        <Input
                          id="add-settings-weapon10-number"
                          type="number"
                          aria-label={t`Weapon +10 level input`}
                          min={0}
                          max={10}
                          value="0"
                          disabled
                          readOnly
                          className={rangeNumberInput}
                        />
                      </div>

                      <div className={rangeControl}>
                        <label htmlFor="add-settings-infusion-select" className={rangeLabel}>
                          <Trans>Infusion</Trans>
                        </label>
                        <Select
                          id="add-settings-infusion-select"
                          aria-label={t`Infusion`}
                          value="Standard"
                          disabled
                        >
                          {INFUSIONS.map((inf) => (
                            <option key={inf} value={inf}>
                              {infusionLabels[inf] ?? inf}
                            </option>
                          ))}
                        </Select>
                        <span />
                      </div>

                      <div className={rangeControl}>
                        <label htmlFor="add-settings-ash-slider" className={rangeLabel}>
                          <Trans>Spirit Ash</Trans>
                        </label>
                        <input
                          id="add-settings-ash-slider"
                          type="range"
                          min={0}
                          max={10}
                          value={0}
                          disabled
                          readOnly
                          aria-label={t`Spirit Ash level`}
                          className={rangeSlider}
                        />
                        <Input
                          id="add-settings-ash-number"
                          type="number"
                          aria-label={t`Spirit Ash level input`}
                          min={0}
                          max={10}
                          value="0"
                          disabled
                          readOnly
                          className={rangeNumberInput}
                        />
                      </div>
                    </div>

                    <div className={addSettingsSwitches}>
                      <label className={switchLabel}>
                        <Checkbox
                          disabled
                          checked={false}
                          aria-label={t`Set all weapons`}
                        />
                        <div className={switchText}>
                          <strong>
                            <Trans>Set all weapons</Trans>
                          </strong>
                          <span className={fieldHint}>
                            <Trans>Mass upgrade of owned weapons requires a dedicated writer (deferred).</Trans>
                          </span>
                        </div>
                      </label>
                      <label className={switchLabel}>
                        <Checkbox
                          disabled
                          checked={false}
                          aria-label={t`Highest talismans only`}
                        />
                        <div className={switchText}>
                          <strong>
                            <Trans>Highest talismans only</Trans>
                          </strong>
                          <span className={fieldHint}>
                            <Trans>Talisman tier hierarchy is not yet mapped in catalog.</Trans>
                          </span>
                        </div>
                      </label>
                      <label className={switchLabel}>
                        <Checkbox
                          disabled
                          checked={false}
                          aria-label={t`Include Ashen Capital`}
                        />
                        <div className={switchText}>
                          <strong>
                            <Trans>Include Ashen Capital</Trans>
                          </strong>
                          <span className={fieldHint}>
                            <Trans>World grace unlock is deferred to World tab.</Trans>
                          </span>
                        </div>
                      </label>
                    </div>
                  </div>
                </details>
              </div>

              <div className={profileLower}>
                <div className={profilePanel}>
                  <section aria-label={t`Attributes`}>
                    <details open className={disclosure}>
                      <summary className={disclosureHeading}>
                        <div className={summaryContent}>
                          <h2>
                            <Trans>Attributes</Trans>
                          </h2>
                          {attrSummary ? (
                            <span className={summaryMeta}>{attrSummary}</span>
                          ) : null}
                        </div>
                      </summary>
                      <div className={disclosureBody}>
                        {statsQuery.isPending ? (
                          <p role="status" className={message}>
                            <Trans>Loading attributes…</Trans>
                          </p>
                        ) : null}
                        {statsQuery.isError ? (
                          <p role="alert" className={alert}>
                            <Trans>Unable to load the attributes of this character slot.</Trans>
                          </p>
                        ) : null}
                        {statsReady && attributeValues !== undefined ? (
                          <>
                            {attributeKeys.map((attr) => (
                              <div key={attr} className={rangeControl}>
                                <label
                                  htmlFor={`character-attr-slider-${attr}`}
                                  className={rangeLabel}
                                >
                                  {attributeLabels[attr]}
                                </label>
                                <input
                                  id={`character-attr-slider-${attr}`}
                                  type="range"
                                  min={1}
                                  max={99}
                                  value={attributeValues[attr]}
                                  disabled={isBusy}
                                  onChange={(e) =>
                                    setAttributesDraft({
                                      key: editContextKey,
                                      value: {
                                        ...attributeValues,
                                        [attr]: Number(e.currentTarget.value),
                                      },
                                    })
                                  }
                                  className={rangeSlider}
                                />
                                <Input
                                  id={`character-attr-number-${attr}`}
                                  type="number"
                                  aria-label={t`${attributeLabels[attr]} value`}
                                  min={1}
                                  max={99}
                                  value={String(attributeValues[attr])}
                                  disabled={isBusy}
                                  onChange={(e) => {
                                    const val = Number(e.currentTarget.value);
                                    if (val >= 1 && val <= 99) {
                                      setAttributesDraft({
                                        key: editContextKey,
                                        value: { ...attributeValues, [attr]: val },
                                      });
                                    }
                                  }}
                                  className={rangeNumberInput}
                                />
                              </div>
                            ))}
                            <div className={favoriteSlotActions}>
                              <Button size="sm" disabled={isBusy} onClick={handleSaveAttributes}>
                                <Trans>Save Attributes</Trans>
                              </Button>
                              <Button
                                size="sm"
                                disabled={isBusy}
                                onClick={handleResetAttributesDraft}
                              >
                                <Trans>Reset</Trans>
                              </Button>
                            </div>
                          </>
                        ) : null}
                      </div>
                    </details>
                  </section>
                </div>

                <div className={profilePanel}>
                  <section aria-label={t`Base Resources`}>
                    <details open className={disclosure}>
                      <summary className={disclosureHeading}>
                        <div className={summaryContent}>
                          <h2>
                            <Trans>Base Resources</Trans>
                          </h2>
                          {resourcesSummary ? (
                            <span className={summaryMeta}>{resourcesSummary}</span>
                          ) : null}
                        </div>
                      </summary>
                      <div className={disclosureBody}>
                        {statsQuery.isPending ? (
                          <p role="status" className={message}>
                            <Trans>Loading base resources…</Trans>
                          </p>
                        ) : null}
                        {statsQuery.isError ? (
                          <p role="alert" className={alert}>
                            <Trans>Unable to load the base resources of this character slot.</Trans>
                          </p>
                        ) : null}
                        {statsQuery.isSuccess ? (
                          <>
                            <div className={derivedGroup}>
                              <h3 className={derivedGroupTitle}>
                                <Trans>Resources</Trans>
                              </h3>
                              <div className={derivedGrid}>
                                <div className={derivedStat}>
                                  <span className={derivedStatLabel}>
                                    <Trans>HP</Trans>
                                  </span>
                                  <span className={derivedStatValue}>
                                    {statsQuery.data.hp} / {statsQuery.data.maxHP}
                                  </span>
                                  <span className={derivedStatSub}>
                                    <Trans>Base: {statsQuery.data.baseMaxHP}</Trans>
                                  </span>
                                </div>
                                <div className={derivedStat}>
                                  <span className={derivedStatLabel}>
                                    <Trans>FP</Trans>
                                  </span>
                                  <span className={derivedStatValue}>
                                    {statsQuery.data.fp} / {statsQuery.data.maxFP}
                                  </span>
                                  <span className={derivedStatSub}>
                                    <Trans>Base: {statsQuery.data.baseMaxFP}</Trans>
                                  </span>
                                </div>
                                <div className={derivedStat}>
                                  <span className={derivedStatLabel}>
                                    <Trans>Stamina (SP)</Trans>
                                  </span>
                                  <span className={derivedStatValue}>
                                    {statsQuery.data.sp} / {statsQuery.data.maxSP}
                                  </span>
                                  <span className={derivedStatSub}>
                                    <Trans>Base: {statsQuery.data.baseMaxSP}</Trans>
                                  </span>
                                </div>
                                <div className={derivedStat}>
                                  <span className={derivedStatLabel}>
                                    <Trans>Equip Load</Trans>
                                  </span>
                                  <span className={derivedStatValue}>--</span>
                                </div>
                              </div>
                            </div>

                            <div className={derivedGroup}>
                              <h3 className={derivedGroupTitle}>
                                <Trans>Secondary</Trans>
                              </h3>
                              <div className={derivedGrid}>
                                <div className={derivedStat}>
                                  <span className={derivedStatLabel}>
                                    <Trans>Poise</Trans>
                                  </span>
                                  <span className={derivedStatValue}>--</span>
                                </div>
                                <div className={derivedStat}>
                                  <span className={derivedStatLabel}>
                                    <Trans>Discovery</Trans>
                                  </span>
                                  <span className={derivedStatValue}>--</span>
                                </div>
                              </div>
                            </div>

                            <div className={derivedGroup}>
                              <h3 className={derivedGroupTitle}>
                                <Trans>Defense</Trans>
                              </h3>
                              <div className={derivedGrid}>
                                <div className={derivedStat}>
                                  <span className={derivedStatLabel}>
                                    <Trans>Physical</Trans>
                                  </span>
                                  <span className={derivedStatValue}>-- / --%</span>
                                </div>
                                <div className={derivedStat}>
                                  <span className={derivedStatLabel}>
                                    <Trans>vs Strike</Trans>
                                  </span>
                                  <span className={derivedStatValue}>-- / --%</span>
                                </div>
                                <div className={derivedStat}>
                                  <span className={derivedStatLabel}>
                                    <Trans>vs Slash</Trans>
                                  </span>
                                  <span className={derivedStatValue}>-- / --%</span>
                                </div>
                                <div className={derivedStat}>
                                  <span className={derivedStatLabel}>
                                    <Trans>vs Pierce</Trans>
                                  </span>
                                  <span className={derivedStatValue}>-- / --%</span>
                                </div>
                                <div className={derivedStat}>
                                  <span className={derivedStatLabel}>
                                    <Trans>Magic</Trans>
                                  </span>
                                  <span className={derivedStatValue}>-- / --%</span>
                                </div>
                                <div className={derivedStat}>
                                  <span className={derivedStatLabel}>
                                    <Trans>Fire</Trans>
                                  </span>
                                  <span className={derivedStatValue}>-- / --%</span>
                                </div>
                                <div className={derivedStat}>
                                  <span className={derivedStatLabel}>
                                    <Trans>Lightning</Trans>
                                  </span>
                                  <span className={derivedStatValue}>-- / --%</span>
                                </div>
                                <div className={derivedStat}>
                                  <span className={derivedStatLabel}>
                                    <Trans>Holy</Trans>
                                  </span>
                                  <span className={derivedStatValue}>-- / --%</span>
                                </div>
                              </div>
                            </div>

                            <div className={derivedGroup}>
                              <h3 className={derivedGroupTitle}>
                                <Trans>Resistances</Trans>
                              </h3>
                              <div className={derivedGrid}>
                                <div className={derivedStat}>
                                  <span className={derivedStatLabel}>
                                    <Trans>Immunity</Trans>
                                  </span>
                                  <span className={derivedStatValue}>--</span>
                                </div>
                                <div className={derivedStat}>
                                  <span className={derivedStatLabel}>
                                    <Trans>Robustness</Trans>
                                  </span>
                                  <span className={derivedStatValue}>--</span>
                                </div>
                                <div className={derivedStat}>
                                  <span className={derivedStatLabel}>
                                    <Trans>Focus</Trans>
                                  </span>
                                  <span className={derivedStatValue}>--</span>
                                </div>
                                <div className={derivedStat}>
                                  <span className={derivedStatLabel}>
                                    <Trans>Vitality</Trans>
                                  </span>
                                  <span className={derivedStatValue}>--</span>
                                </div>
                              </div>
                            </div>
                          </>
                        ) : null}
                      </div>
                    </details>
                  </section>
                </div>
              </div>

              <p className={profileNote}>
                <Trans>Profile changes are staged until you save the file.</Trans>
              </p>
            </>
          )}

          {/* Starting Class Dialog */}
          <Dialog
            open={classPickerOpen}
            onOpenChange={setClassPickerOpen}
            title={t`Select Starting Class`}
            closeLabel={t`Close`}
          >
            <div className={fieldGroup}>
              <label htmlFor="select-starting-class" className={fieldLabel}>
                <Trans>Target Class</Trans>
              </label>
              <Select
                id="select-starting-class"
                value={String(pendingClassID)}
                onChange={(e) => setPendingClassID(Number(e.currentTarget.value))}
              >
                {(classesQuery.data?.resources ?? []).map((r) => (
                  <option key={r.key} value={r.key}>
                    {r.name}
                  </option>
                ))}
              </Select>
              <div className={favoriteSlotActions}>
                {/* Only one modal is open at a time: the picker closes before the
                    confirmation opens, so two Radix dialogs never stack. */}
                <Button
                  size="sm"
                  onClick={() => {
                    setClassPickerOpen(false);
                    setConfirmClassResetOpen(true);
                  }}
                >
                  <Trans>Change Class...</Trans>
                </Button>
                <Button size="sm" onClick={() => setClassPickerOpen(false)}>
                  <Trans>Cancel</Trans>
                </Button>
              </div>
            </div>
          </Dialog>

          {/* Confirm Class Reset Dialog */}
          <Dialog
            open={confirmClassResetOpen}
            onOpenChange={setConfirmClassResetOpen}
            title={t`Confirm Class Reset`}
            closeLabel={t`Cancel`}
          >
            <p>
              <Trans>
                Changing the starting class is a destructive reset: all eight attributes and Rune
                Level will be reset to the base values of the selected class.
              </Trans>
            </p>
            <p className={message}>
              <Trans>SoulMemory and held runes will remain unchanged.</Trans>
            </p>
            <div className={favoriteSlotActions}>
              <Button
                size="sm"
                disabled={isBusy || !profileReady || !classesReady}
                onClick={handleConfirmClassChange}
              >
                <Trans>Confirm Reset</Trans>
              </Button>
              <Button size="sm" onClick={() => setConfirmClassResetOpen(false)}>
                <Trans>Cancel</Trans>
              </Button>
            </div>
          </Dialog>

          {/* Confirm Gender / Body Type Dialog */}
          <Dialog
            open={confirmGenderOpen}
            onOpenChange={setConfirmGenderOpen}
            title={t`Confirm Body Type Change`}
            closeLabel={t`Cancel`}
          >
            <p>
              <Trans>
                Switching body type will apply the complete default appearance preset for{" "}
                {pendingGender === 1 ? "Type A" : "Type B"}. Any custom facial or body appearance
                changes will be replaced with the default preset.
              </Trans>
            </p>
            <div className={favoriteSlotActions}>
              <Button
                size="sm"
                disabled={isBusy || !profileReady}
                onClick={handleConfirmGenderChange}
              >
                <Trans>Confirm Change</Trans>
              </Button>
              <Button size="sm" onClick={() => setConfirmGenderOpen(false)}>
                <Trans>Cancel</Trans>
              </Button>
            </div>
          </Dialog>
        </>
      )}

      {activeTab === "appearance" && (
        <div className={presetContainer}>
          <Card aria-label={t`Appearance Presets`}>
            <h2>
              <Trans>Preset Catalog</Trans>
            </h2>
            <div className={identityGrid}>
              <div className={fieldGroup}>
                <label htmlFor="appearance-search-input" className={fieldLabel}>
                  <Trans>Search</Trans>
                </label>
                <Input
                  id="appearance-search-input"
                  placeholder={t`Search presets...`}
                  value={searchDraft}
                  onChange={(e) => setSearchDraft(e.currentTarget.value)}
                />
              </div>
              <div className={fieldGroup}>
                <label htmlFor="appearance-body-type-filter" className={fieldLabel}>
                  <Trans>Body Type Filter</Trans>
                </label>
                <Select
                  id="appearance-body-type-filter"
                  value={bodyTypeFilter}
                  onChange={(e) =>
                    setBodyTypeFilter(e.currentTarget.value as "all" | "Type A" | "Type B")
                  }
                >
                  <option value="all">{t`All Body Types`}</option>
                  <option value="Type A">Type A</option>
                  <option value="Type B">Type B</option>
                </Select>
              </div>
            </div>

            {presetsQuery.isPending ? (
              <p role="status" className={message}>
                <Trans>Loading appearance presets…</Trans>
              </p>
            ) : null}
            {presetsQuery.isError ? (
              <p role="alert" className={alert}>
                <Trans>Unable to load the appearance presets.</Trans>
              </p>
            ) : null}
            {presetsReady && filteredPresets.length === 0 ? (
              <p className={message}>
                <Trans>No appearance presets matched your search.</Trans>
              </p>
            ) : null}
            {presetsReady && activePreset !== undefined ? (
              <div className={presetStage}>
                {filteredPresets[safePresetIndex - 1] ? (
                  <img
                    className={presetNeighbor}
                    alt=""
                    aria-hidden="true"
                    src={appearancePresetAssetURL(filteredPresets[safePresetIndex - 1].image)}
                  />
                ) : (
                  <div className={presetNeighbor} aria-hidden="true" />
                )}
                <div className={presetViewer}>
                  {activePresetImageURL !== undefined ? (
                    <img
                      src={activePresetImageURL}
                      alt={activePreset.name}
                      className={presetImage}
                    />
                  ) : (
                    <div className={presetImagePlaceholder}>
                      <Trans>No preview image</Trans>
                    </div>
                  )}
                  <h3>{activePreset.name}</h3>
                  <Badge>{activePreset.bodyType}</Badge>
                  {activePreset.tags.length > 0 && (
                    <div className={presetTags}>
                      {activePreset.tags.map((tag) => (
                        <Badge key={tag} tone="neutral">
                          {tag}
                        </Badge>
                      ))}
                    </div>
                  )}
                  <div className={presetControls}>
                    <Button
                      size="sm"
                      disabled={safePresetIndex === 0}
                      onClick={() => setSelectedPresetIndex(Math.max(0, safePresetIndex - 1))}
                    >
                      <Trans>Previous</Trans>
                    </Button>
                    <span>
                      {safePresetIndex + 1} / {filteredPresets.length}
                    </span>
                    <Button
                      size="sm"
                      disabled={safePresetIndex === filteredPresets.length - 1}
                      onClick={() =>
                        setSelectedPresetIndex(
                          Math.min(filteredPresets.length - 1, safePresetIndex + 1),
                        )
                      }
                    >
                      <Trans>Next</Trans>
                    </Button>
                  </div>
                  <div>
                    <Button
                      size="sm"
                      disabled={!hasActiveCharacter || isBusy}
                      onClick={handleApplyPreset}
                    >
                      <Trans>Apply to Character</Trans>
                    </Button>
                  </div>
                </div>
                {filteredPresets[safePresetIndex + 1] ? (
                  <img
                    className={presetNeighbor}
                    alt=""
                    aria-hidden="true"
                    src={appearancePresetAssetURL(filteredPresets[safePresetIndex + 1].image)}
                  />
                ) : (
                  <div className={presetNeighbor} aria-hidden="true" />
                )}
              </div>
            ) : null}
          </Card>

          <Card aria-label={t`Mirror Favorites`}>
            <h2>
              <Trans>Mirror Favorites</Trans>
            </h2>
            {!saveSessionID ? (
              <p className={message}>
                <Trans>Mirror Favorites require an active save session.</Trans>
              </p>
            ) : (
              <>
                {favoritesQuery.isPending ? (
                  <p role="status" className={message}>
                    <Trans>Loading Mirror Favorites…</Trans>
                  </p>
                ) : null}
                {favoritesQuery.isError ? (
                  <p role="alert" className={alert}>
                    <Trans>Unable to load the Mirror Favorites of this save.</Trans>
                  </p>
                ) : null}
                {favoritesReady ? (
                  <div className={favoritesGrid}>
                    {favoritesQuery.data.presets.map((slot) => (
                      <div key={slot.favoriteSlotID} className={favoriteSlotCard}>
                        <div className={favoriteSlotHeader}>
                          <strong>
                            <Trans>Slot {slot.favoriteSlotID + 1}</Trans>
                          </strong>
                          <Badge tone={slot.active ? "accent" : "neutral"}>
                            {slot.active ? <Trans>Active</Trans> : <Trans>Empty</Trans>}
                          </Badge>
                        </div>
                        <div className={favoriteSlotActions}>
                          {slot.active ? (
                            <>
                              <Button
                                size="sm"
                                disabled={!hasActiveCharacter || isBusy}
                                onClick={() => handleApplyFavorite(slot.favoriteSlotID)}
                              >
                                <Trans>Apply</Trans>
                              </Button>
                              <Button
                                size="sm"
                                disabled={!hasActiveCharacter || isBusy}
                                onClick={() => setConfirmReplaceFavSlot(slot.favoriteSlotID)}
                              >
                                <Trans>Overwrite</Trans>
                              </Button>
                              <Button
                                size="sm"
                                disabled={isBusy}
                                onClick={() => setConfirmDeleteFavSlot(slot.favoriteSlotID)}
                              >
                                <Trans>Delete</Trans>
                              </Button>
                            </>
                          ) : (
                            <Button
                              size="sm"
                              disabled={!hasActiveCharacter || isBusy}
                              onClick={() => handleSaveFavorite(slot.favoriteSlotID)}
                            >
                              <Trans>Save Appearance</Trans>
                            </Button>
                          )}
                        </div>
                      </div>
                    ))}
                  </div>
                ) : null}
              </>
            )}
          </Card>

          {/* Confirm Replace Favorite Dialog */}
          <Dialog
            open={confirmReplaceFavSlot !== null}
            onOpenChange={(open) => {
              if (!open) setConfirmReplaceFavSlot(null);
            }}
            title={t`Confirm Overwrite`}
            closeLabel={t`Cancel`}
          >
            <p>
              <Trans>
                Are you sure you want to overwrite Mirror Favorite Slot{" "}
                {(confirmReplaceFavSlot ?? 0) + 1} with the current character&apos;s appearance?
              </Trans>
            </p>
            <div className={favoriteSlotActions}>
              <Button size="sm" disabled={isBusy} onClick={handleConfirmReplaceFavorite}>
                <Trans>Overwrite</Trans>
              </Button>
              <Button size="sm" onClick={() => setConfirmReplaceFavSlot(null)}>
                <Trans>Cancel</Trans>
              </Button>
            </div>
          </Dialog>

          {/* Confirm Delete Favorite Dialog */}
          <Dialog
            open={confirmDeleteFavSlot !== null}
            onOpenChange={(open) => {
              if (!open) setConfirmDeleteFavSlot(null);
            }}
            title={t`Confirm Delete`}
            closeLabel={t`Cancel`}
          >
            <p>
              <Trans>
                Are you sure you want to delete Mirror Favorite Slot{" "}
                {(confirmDeleteFavSlot ?? 0) + 1}?
              </Trans>
            </p>
            <div className={favoriteSlotActions}>
              <Button size="sm" disabled={isBusy} onClick={handleConfirmDeleteFavorite}>
                <Trans>Delete</Trans>
              </Button>
              <Button size="sm" onClick={() => setConfirmDeleteFavSlot(null)}>
                <Trans>Cancel</Trans>
              </Button>
            </div>
          </Dialog>
        </div>
      )}
    </div>
  );
}
