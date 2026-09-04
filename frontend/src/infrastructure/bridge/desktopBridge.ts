// The ONLY module allowed to import the generated Wails bindings. Everything
// above this layer depends on the application ports, never on generated code or
// on `window.go`.

import {
  ActivateTargetBackup,
  AddItemsToContainers,
  ApplyAppearancePreset,
  ApplyBuildTemplate,
  ApplyFavoritePreset,
  ApplyRepairs,
  CancelDeploymentOperation,
  CheckForUpdates,
  ClearActiveTargetBackup,
  ClearRecentFiles,
  CloseSave,
  CloseTargetGame,
  CreateBuildTemplate,
  CreateDeploymentTarget,
  CreateTargetBackup,
  DeleteBuildTemplate,
  DeleteDeploymentTarget,
  DeleteFavoritePreset,
  DeleteTargetBackup,
  DeployToTarget,
  DiscardChanges,
  DiscardRecoveryJournal,
  DownloadFromTarget,
  DownloadTargetBackup,
  ExportDiagnosticReport,
  ExportRecoveryJournal,
  ForgetDeploymentHostKey,
  GetAppearancePresets,
  GetBellBearings,
  GetBuildTemplatePreview,
  GetBuildTemplates,
  GetBosses,
  GetApplicationInfo,
  GetCharacterLoadout,
  GetCharacterProfile,
  GetCharacterStats,
  GetColosseums,
  GetCookbooks,
  GetDeploymentGameStatus,
  GetDeploymentTargets,
  GetEquipment,
  GetEquipmentCandidates,
  GetEquippedSpells,
  GetFavoritePresets,
  GetGestures,
  GetGraces,
  GetHostSettings,
  GetInventory,
  GetItemDatabase,
  GetItemVariants,
  GetLoadedSave,
  GetMapRegions,
  GetNetworkPresets,
  GetNetworkSettings,
  GetOperationHistory,
  GetOwnedItems,
  GetPhysickMixture,
  GetPouchItems,
  GetProjectLinks,
  GetQuests,
  GetQuickItems,
  GetRecentFiles,
  GetRecoveryJournal,
  GetRecoveryJournals,
  GetRegions,
  GetRepairPlan,
  GetResource,
  GetResourcePresentationSummaries,
  GetResources,
  GetSafetyProfile,
  GetSaveCharacters,
  GetSaveLifecycleSettings,
  GetSaveValidationReport,
  GetSpectralSteedAttires,
  GetStorage,
  GetSummoningPools,
  GetTargetBackups,
  GetTutorials,
  GetWhetblades,
  GetWorldMutationCapabilities,
  ImportBuildTemplate,
  LaunchTargetGame,
  LoadSave,
  LockAllSpectralSteedAttires,
  MoveOwnedItemsToInventory,
  MoveOwnedItemsToStorage,
  OpenHostLocation,
  OpenProjectLink,
  QuitApplication,
  ReleaseDeploymentStaging,
  RecordRecentFile,
  RedoLastOperation,
  RemoveOwnedItems,
  RemoveRecentFile,
  ReorderInventoryItems,
  RestoreRecoveryJournal,
  RevertOperation,
  Save,
  SaveAs,
  SelectSaveFile,
  SelectSaveTarget,
  SetCharacterGender,
  SetCharacterName,
  SetCharacterRunes,
  SetCharacterStartingClass,
  SetCharacterStats,
  SetEquippedArmaments,
  SetEquippedArmor,
  SetEquippedSpells,
  SetEquippedTalismans,
  SetFavoritePreset,
  SetHostSettings,
  SetOwnedItemQuantity,
  SetPhysickMixture,
  SetPouchItems,
  SetQuickItems,
  SetBellBearingUnlocked,
  SetBossDefeated,
  SetColosseumUnlocked,
  SetCookbookUnlocked,
  SetFogOfWarRemoved,
  SetGestureUnlocked,
  SetGraceVisited,
  SetMapRegionRevealed,
  SetNetworkSettings,
  SetQuestStep,
  SetRegionUnlocked,
  SetSafetyProfile,
  SetSaveAccountID,
  SetSpectralSteedAttire,
  SetSummoningPoolActivated,
  SetTutorialUnlocked,
  SetWhetbladeUnlocked,
  SetSaveLifecycleSettings,
  TestDeploymentTarget,
  UndoLastOperation,
  UpdateBuildTemplate,
  UpdateDeploymentTarget,
  UpdateTargetBackup,
  ValidateReviewChanges,
} from "../../../wailsjs/go/desktop/Bridge";
import {
  type application,
  type deployment,
  saveengine,
  type schema,
  templates,
} from "../../../wailsjs/go/models";
import { EventsOn } from "../../../wailsjs/runtime/runtime";
import type {
  AppearancePort,
  AppearancePresetSummary,
} from "../../application/appearance/appearancePort";
import type {
  ApplicationInfo,
  ApplicationInfoPort,
} from "../../application/application-info/applicationInfoPort";
import type {
  CatalogAshOfWarMountRules,
  CatalogCapability,
  CatalogEquipmentRules,
  CatalogFact,
  CatalogInfusionRules,
  CatalogItemDatabasePage,
  CatalogItemDetail,
  CatalogItemVariantsResult,
  CatalogPort,
  CatalogProvenance,
  CatalogResourceDetail,
  CatalogResourcePresentationSummaries,
  CatalogResourcesPage,
  CatalogStackRules,
  CatalogUpgradeRules,
} from "../../application/catalog/catalogPort";
import { type ChangedScope, changedScopes } from "../../application/changedScopes";
import type {
  CharacterAttributes,
  CharacterPort,
  CharacterProfile,
  CharacterStats,
  SaveCharacters,
} from "../../application/character/characterPort";
import type {
  ApplyRepairsResult,
  DiagnosticsPort,
  RepairAction,
  RepairPlan,
  RepairRejection,
  SaveValidationReport,
} from "../../application/diagnostics/diagnosticsPort";
import type {
  CharacterEquipment,
  CharacterEquippedSpells,
  CharacterLoadout,
  CharacterPhysickMixture,
  CharacterPouchItems,
  CharacterQuickItems,
  EquipmentCandidatesPage,
  EquipmentMutationReceipt,
  EquipmentPort,
  LoadoutOwnedSlot,
  LoadoutSlot,
  LoadoutSlotState,
  LoadoutSpellSlot,
} from "../../application/equipment/equipmentPort";
import {
  AppErrorException,
  bridgeCallFailed,
  bridgeFailureCode,
} from "../../application/errors/appError";
import type { FavoritesPort, SaveFavoritePresets } from "../../application/favorites/favoritesPort";
import type {
  ItemMutationReceipt,
  ItemPage,
  ItemsPort,
  OwnedItemsPage,
} from "../../application/items/itemsPort";
import type {
  NetworkParamValues,
  NetworkPort,
  NetworkPresetsResult,
  NetworkSettingsSnapshot,
  SetNetworkSettingsResult,
} from "../../application/network/networkPort";
import type {
  HistoryMutationResult,
  MutationReceipt,
  OperationHistory,
  OperationRecord,
  OperationRisk,
  RecentFile,
  RecoveryJournal,
  ReviewValidationResult,
  SaveLifecycleResult,
  SaveLifecycleSettings,
  SaveSession,
  SaveSessionPort,
} from "../../application/save-session/saveSessionPort";
import type { AboutPort, ProjectLink, UpdateCheck } from "../../application/about/aboutPort";
import type {
  CommandOutcome,
  DeploymentOperationResult,
  DeploymentPort,
  DeploymentTarget,
  DeploymentProgress,
  DeploymentTargets,
  TargetBackup,
  TargetBackups,
  TargetTestResult,
} from "../../application/deployment/deploymentPort";
import type {
  HostSettings,
  HostSettingsPort,
  SafetyProfileSettings,
  SettingsPort,
} from "../../application/settings/settingsPort";
import type {
  BuildTemplateOverrides,
  BuildTemplatePage,
  BuildTemplatePlan,
  BuildTemplatePreview,
  TemplateMutationReceipt,
  TemplatePort,
} from "../../application/templates/templatePort";
import type {
  SpectralSteedAttireStatus,
  WorldBellBearings,
  WorldBosses,
  WorldColosseums,
  WorldCookbooks,
  WorldGestures,
  WorldGraces,
  WorldMapRegions,
  WorldMutationCapability,
  WorldMutationReceipt,
  WorldOperationKind,
  WorldPort,
  WorldQuests,
  WorldRegions,
  WorldResourceToggleRequest,
  WorldSpectralSteedAttires,
  WorldSummoningPools,
  WorldTutorials,
  WorldWhetblades,
} from "../../application/world/worldPort";
import { worldOperationKinds } from "../../application/world/worldPort";
import { parseBridgeError } from "./bridgeError";
import { parseSessionChangedEvent, sessionChangedEventName } from "./sessionChangedEvent";

/**
 * Re-exported so consumers of this adapter keep one import for the boundary.
 * The code itself is owned by the application error model.
 */
export { bridgeFailureCode };

/**
 * The single choke point of every bridge call.
 *
 * A rejection is never propagated as it arrived: Wails hands back one string,
 * which is either the backend's structured envelope or something the frontend
 * cannot vouch for. `parseBridgeError` validates the envelope strictly and
 * reduces everything else to `bridge_call_failed`, so no raw Go error, host
 * stack or truncated payload ever leaves this function.
 *
 * The thrown value stays an `Error` whose message is the stable code, so a
 * consumer that only knows the code keeps working and nothing is ever tempted
 * to parse a sentence.
 */
async function callBridge<T>(call: () => Promise<T>): Promise<T> {
  try {
    return await call();
  } catch (reason) {
    throw new AppErrorException(parseBridgeError(reason));
  }
}

/**
 * Projects the generated session result onto the application port shape. Every
 * field is carried verbatim: the source path is not resolved or shortened, the
 * source kind is not narrowed to a union, and the revision stays the string the
 * backend sent.
 */
function toSaveSession(result: Awaited<ReturnType<typeof GetLoadedSave>>): SaveSession {
  return {
    saveSessionID: result.saveSessionID,
    platform: result.platform,
    format: result.format,
    sourcePath: result.sourcePath,
    sourceKind: result.sourceKind,
    saveRevision: result.saveRevision,
    unsavedChanges: result.unsavedChanges,
    eventSequence: result.eventSequence,
  };
}

const operationRisks = ["normal", "warning", "ban risk", "critical"] as const;

function toChangedScopes(values: readonly string[]): readonly ChangedScope[] {
  if (
    !values.every((value): value is ChangedScope => changedScopes.includes(value as ChangedScope))
  ) {
    throw new AppErrorException(bridgeCallFailed());
  }
  return [...values];
}

const spectralSteedAttireStatuses = ["resolved", "legacy", "conflict"] as const;

/**
 * The Spectral Steed status is a closed backend contract. An unknown value is
 * an unknown contract, so it fails closed with the same stable code as any
 * other bridge failure and never reaches the screen verbatim.
 */
function toSpectralSteedAttireStatus(value: string): SpectralSteedAttireStatus {
  if (!spectralSteedAttireStatuses.includes(value as SpectralSteedAttireStatus)) {
    throw new AppErrorException(bridgeCallFailed());
  }
  return value as SpectralSteedAttireStatus;
}

function toOperationRisk(value: string): OperationRisk {
  if (!operationRisks.includes(value as OperationRisk)) {
    throw new AppErrorException(bridgeCallFailed());
  }
  return value as OperationRisk;
}

function toOperationRecord(
  result: Awaited<ReturnType<typeof GetOperationHistory>>["operations"][number],
): OperationRecord {
  return {
    operationID: result.operationID,
    operationKind: result.operationKind,
    saveSessionID: result.saveSessionID,
    saveRevision: result.saveRevision,
    order: result.order,
    characterID: result.characterID,
    area: result.area,
    description: result.description,
    relatedResource: result.relatedResource,
    beforeState: result.beforeState,
    afterState: result.afterState,
    risk: toOperationRisk(result.risk),
    riskReason: result.riskReason,
    changedByteCount: result.changedByteCount,
    changedScopes: toChangedScopes(result.changedScopes),
  };
}

function toOperationHistory(
  result: Awaited<ReturnType<typeof GetOperationHistory>>,
): OperationHistory {
  return {
    saveSessionID: result.saveSessionID,
    saveRevision: result.saveRevision,
    operations: result.operations.map(toOperationRecord),
    undoCount: result.undoCount,
    redoCount: result.redoCount,
  };
}

function toMutationReceipt(result: {
  operationID: string;
  operationKind: string;
  saveSessionID: string;
  saveRevision: string;
  changedScopes: string[];
}): MutationReceipt {
  return {
    operationID: result.operationID,
    operationKind: result.operationKind,
    saveSessionID: result.saveSessionID,
    saveRevision: result.saveRevision,
    changedScopes: toChangedScopes(result.changedScopes),
  };
}

function toHistoryMutationResult(
  result: Awaited<ReturnType<typeof UndoLastOperation>>,
): HistoryMutationResult {
  return {
    ...toMutationReceipt(result),
    affectedOperationID: result.affectedOperationID,
    affectedOperationKind: result.affectedOperationKind,
  };
}

function toReviewValidationResult(
  result: Awaited<ReturnType<typeof ValidateReviewChanges>>,
): ReviewValidationResult {
  return {
    saveSessionID: result.saveSessionID,
    saveRevision: result.saveRevision,
    validationToken: result.validationToken,
    valid: result.valid,
    warningCount: result.warningCount,
    banRiskCount: result.banRiskCount,
    criticalCount: result.criticalCount,
    stages: result.stages.map((stage) => ({ stage: stage.stage, percent: stage.percent })),
    issues: result.issues.map((issue) => ({
      code: issue.code,
      severity: toOperationRisk(issue.severity),
      message: issue.message,
      operationID: issue.operationID,
    })),
  };
}

function toSaveLifecycleResult(result: Awaited<ReturnType<typeof Save>>): SaveLifecycleResult {
  return {
    ...toMutationReceipt(result),
    target: result.target,
    backupPath: result.backupPath,
    warnings: [...result.warnings],
    retentionNoticeRequired: result.retentionNoticeRequired,
  };
}

function toRecentFile(result: Awaited<ReturnType<typeof GetRecentFiles>>[number]): RecentFile {
  return { ...result };
}

function toRecoveryJournal(
  result: Awaited<ReturnType<typeof GetRecoveryJournal>>,
): RecoveryJournal {
  if (!["compatible", "incompatible", "corrupt"].includes(result.status)) {
    throw new AppErrorException(bridgeCallFailed());
  }
  return {
    journalID: result.journalID,
    status: result.status as RecoveryJournal["status"],
    sourcePath: result.sourcePath,
    platform: result.platform,
    format: result.format,
    saveRevision: result.saveRevision,
    updatedAt: result.updatedAt,
    operationCount: result.operationCount,
    operations: result.operations.map(toOperationRecord),
    failureCode: result.failureCode,
  };
}

function toSaveLifecycleSettings(
  result: Awaited<ReturnType<typeof GetSaveLifecycleSettings>>,
): SaveLifecycleSettings {
  return {
    backupRetention: result.backupRetention,
    retentionNoticeShown: result.retentionNoticeShown,
  };
}

/**
 * Projects the generated validation report. The counters, the coverage and the
 * findings are copied into arrays this layer owns and are otherwise untouched:
 * nothing is recounted, reordered, filtered or reclassified here, because the
 * verdict is the backend's alone.
 */
function toSaveValidationReport(
  result: Awaited<ReturnType<typeof GetSaveValidationReport>>,
): SaveValidationReport {
  return {
    saveSessionID: result.saveSessionID,
    saveRevision: result.saveRevision,
    characterID: result.characterID,
    active: result.active,
    coverage: result.coverage.map((scope) => ({
      scope: scope.scope,
      checked: scope.checked,
      reason: scope.reason,
      recordsChecked: scope.recordsChecked,
      unresolvedRecords: scope.unresolvedRecords,
    })),
    issues: result.issues.map((issue) => ({
      id: issue.id,
      code: issue.code,
      severity: issue.severity,
      scope: issue.scope,
      message: issue.message,
      ownedItemID: issue.ownedItemID,
    })),
    errorCount: result.errorCount,
    warningCount: result.warningCount,
  };
}

/**
 * Projects one planned action. Every member is copied as the backend reported
 * it: no operation is renamed, no target value is recomputed and no description
 * is rewritten. `targetValue` and `ownedItemID` are omitted by the backend for
 * an operation that carries neither, so the absent number stays `undefined`
 * rather than becoming a zero the backend never sent.
 */
function toRepairAction(action: {
  issueIDs: string[];
  scope: string;
  operation: string;
  ownedItemID?: string;
  targetValue?: number;
  attributes?: CharacterAttributes;
  description: string;
}): RepairAction {
  return {
    issueIDs: [...action.issueIDs],
    scope: action.scope,
    operation: action.operation,
    ownedItemID: action.ownedItemID ?? "",
    targetValue: action.targetValue,
    // The statistics payload of the action is carried through untouched: it is
    // the backend's own derived block, never a value recomputed here.
    attributes: action.attributes,
    description: action.description,
  };
}

function toRepairRejection(rejection: {
  issueID: string;
  code: string;
  scope: string;
  reason: string;
}): RepairRejection {
  return {
    issueID: rejection.issueID,
    code: rejection.code,
    scope: rejection.scope,
    reason: rejection.reason,
  };
}

function toRepairPlan(result: Awaited<ReturnType<typeof GetRepairPlan>>): RepairPlan {
  return {
    saveSessionID: result.saveSessionID,
    saveRevision: result.saveRevision,
    characterID: result.characterID,
    planToken: result.planToken,
    actions: result.actions.map(toRepairAction),
    rejected: result.rejected.map(toRepairRejection),
  };
}

/**
 * Projects the two success variants of ApplyRepairs.
 *
 * `applied` is the backend's own discriminator and the only thing branched on:
 * a committed transaction must carry a whole receipt and goes through the same
 * strict receipt projection as any other mutation, while a result that applied
 * nothing describes no execution and therefore gains no operation identifier,
 * operation kind or changed scopes. A payload that claims a commit without a
 * complete receipt is an unknown contract and fails as a bridge failure.
 */
function toApplyRepairsResult(
  result: Awaited<ReturnType<typeof ApplyRepairs>>,
): ApplyRepairsResult {
  const outcome = {
    saveSessionID: result.saveSessionID,
    saveRevision: result.saveRevision,
    characterID: result.characterID,
    actions: result.actions.map(toRepairAction),
    rejected: result.rejected.map(toRepairRejection),
  };
  if (!result.applied) {
    return { ...outcome, applied: false };
  }
  const { operationID, operationKind, changedScopes } = result;
  if (operationID === undefined || operationKind === undefined || changedScopes === undefined) {
    throw new AppErrorException(bridgeCallFailed());
  }
  return {
    ...toMutationReceipt({
      operationID,
      operationKind,
      saveSessionID: result.saveSessionID,
      saveRevision: result.saveRevision,
      changedScopes,
    }),
    ...outcome,
    applied: true,
  };
}

/**
 * Projects a generated container page onto the application port shape. The two
 * generated result types carry the same fields, so one projection covers both
 * and neither container gets a second, drifting mapping.
 */
function toItemPage(
  result: Awaited<ReturnType<typeof GetInventory | typeof GetStorage>>,
): ItemPage {
  return {
    saveSessionID: result.saveSessionID,
    saveRevision: result.saveRevision,
    characterID: result.characterID,
    active: result.active,
    records: result.records.map((record) => ({
      ownedItemID: record.ownedItemID,
      kind: record.kind,
      key: record.key,
      gameID: record.gameID,
      containerSection: record.containerSection,
      physicalIndex: record.physicalIndex,
      gaItemHandle: record.gaItemHandle,
      quantity: record.quantity,
      acquisitionIndex: record.acquisitionIndex,
    })),
    total: result.total,
    page: result.page,
    pageSize: result.pageSize,
  };
}

/**
 * Projects the generated raw equipment result onto the application port shape.
 * The 22 values are copied into an array this layer owns, in the order the
 * backend reported them: the generated array itself never becomes application
 * state. No value is named, resolved, filtered or recognised as a sentinel.
 */
function toCharacterEquipment(
  result: Awaited<ReturnType<typeof GetEquipment>>,
): CharacterEquipment {
  return {
    saveSessionID: result.saveSessionID,
    saveRevision: result.saveRevision,
    characterID: result.characterID,
    active: result.active,
    slots: [...result.slots],
  };
}

function toLoadoutSlot(
  slot: Awaited<ReturnType<typeof GetCharacterLoadout>>["rightHand"][number],
): LoadoutSlot {
  return {
    slotType: slot.slotType,
    state: slot.state as LoadoutSlotState,
    ownedItemID: slot.ownedItemID,
    resource: slot.resource === undefined ? undefined : { ...slot.resource },
    name: slot.name,
    iconPath: slot.iconPath,
    rawValue: slot.rawValue,
  };
}

function toLoadoutOwnedSlot(
  slot: Awaited<ReturnType<typeof GetCharacterLoadout>>["quickItems"][number],
): LoadoutOwnedSlot {
  return {
    slotType: slot.slotType as LoadoutOwnedSlot["slotType"],
    state: slot.state as LoadoutSlotState,
    ownedItemID: slot.ownedItemID,
    resource: slot.resource === undefined ? undefined : { ...slot.resource },
    name: slot.name,
    iconPath: slot.iconPath,
    quantity: slot.quantity,
  };
}

function toLoadoutSpellSlot(
  slot: Awaited<ReturnType<typeof GetCharacterLoadout>>["spells"][number],
): LoadoutSpellSlot {
  return {
    state: slot.state as LoadoutSlotState,
    resource: slot.resource === undefined ? undefined : { ...slot.resource },
    name: slot.name,
    iconPath: slot.iconPath,
    memorySlots: slot.memorySlots,
  };
}

/**
 * Projects the complete backend loadout without interpreting any game ID,
 * sentinel, family, active index, capacity or locking rule locally.
 */
function toCharacterLoadout(
  result: Awaited<ReturnType<typeof GetCharacterLoadout>>,
): CharacterLoadout {
  return {
    saveSessionID: result.saveSessionID,
    saveRevision: result.saveRevision,
    characterID: result.characterID,
    active: result.active,
    rightHand: result.rightHand.map(toLoadoutSlot),
    leftHand: result.leftHand.map(toLoadoutSlot),
    arrows: result.arrows.map(toLoadoutSlot),
    bolts: result.bolts.map(toLoadoutSlot),
    armor: result.armor.map(toLoadoutSlot),
    talismans: result.talismans.map(toLoadoutSlot),
    quickItems: result.quickItems.map(toLoadoutOwnedSlot),
    pouch: result.pouch.map(toLoadoutOwnedSlot),
    activeQuickItem: result.activeQuickItem,
    physick: result.physick.map(toLoadoutSlot),
    spells: result.spells.map(toLoadoutSpellSlot),
    activeSpellIndex: result.activeSpellIndex,
    usedMemorySlots: result.usedMemorySlots,
    availableMemorySlots: result.availableMemorySlots,
    memoryStones: result.memoryStones,
    unlockedTalismanSlots: result.unlockedTalismanSlots,
  };
}

/**
 * Projects the generated quick-item result. `activeQuick` is signed in the
 * backend contract and is carried as reported: a negative value is not clamped,
 * zeroed or turned into an index.
 */
function toCharacterQuickItems(
  result: Awaited<ReturnType<typeof GetQuickItems>>,
): CharacterQuickItems {
  return {
    saveSessionID: result.saveSessionID,
    saveRevision: result.saveRevision,
    characterID: result.characterID,
    active: result.active,
    items: result.items.map((item) => ({ itemID: item.itemID, equipIndex: item.equipIndex })),
    activeQuick: result.activeQuick,
  };
}

/** Projects the generated pouch result under the same rules. */
function toCharacterPouchItems(
  result: Awaited<ReturnType<typeof GetPouchItems>>,
): CharacterPouchItems {
  return {
    saveSessionID: result.saveSessionID,
    saveRevision: result.saveRevision,
    characterID: result.characterID,
    active: result.active,
    items: result.items.map((item) => ({ itemID: item.itemID, equipIndex: item.equipIndex })),
  };
}

/** Projects both raw Crystal Tear identifiers into an array this layer owns. */
function toCharacterPhysickMixture(
  result: Awaited<ReturnType<typeof GetPhysickMixture>>,
): CharacterPhysickMixture {
  return {
    saveSessionID: result.saveSessionID,
    saveRevision: result.saveRevision,
    characterID: result.characterID,
    active: result.active,
    tears: [...result.tears],
  };
}

/**
 * Projects the generated equipped-spells result. The three resolved fields are
 * the backend's own answer and are carried verbatim, empty values included: an
 * empty record keeps its raw identifier and its empty key, name and cost, and
 * neither count is recomputed from the records.
 */
function toCharacterEquippedSpells(
  result: Awaited<ReturnType<typeof GetEquippedSpells>>,
): CharacterEquippedSpells {
  return {
    saveSessionID: result.saveSessionID,
    saveRevision: result.saveRevision,
    characterID: result.characterID,
    active: result.active,
    spells: result.spells.map((spell) => ({
      rawMagicParamID: spell.rawMagicParamID,
      resourceKey: spell.resourceKey,
      name: spell.name,
      memorySlots: spell.memorySlots,
    })),
    usedMemorySlots: result.usedMemorySlots,
    availableMemorySlots: result.availableMemorySlots,
  };
}

/** Projects the generated catalog page onto the application port shape. */
function toCatalogResourcesPage(
  result: Awaited<ReturnType<typeof GetResources>>,
): CatalogResourcesPage {
  return {
    resources: result.resources.map((resource) => ({
      kind: resource.kind,
      key: resource.key,
      family: resource.family,
      name: resource.name,
    })),
    total: result.total,
    // The served page and page size are the backend's answer, not an echo of
    // the request: zero paging resolves to the backend default there.
    page: result.page,
    pageSize: result.pageSize,
  };
}

/** Projects the generated lightweight batch onto transport-free port values. */
function toCatalogResourcePresentationSummaries(
  result: Awaited<ReturnType<typeof GetResourcePresentationSummaries>>,
): CatalogResourcePresentationSummaries {
  return {
    resources: result.resources.map(({ kind, key, name, iconPath }) => ({
      kind,
      key,
      name,
      iconPath,
    })),
  };
}

/**
 * The generated `Fact[T]` and `Capability[T]` classes are one per instantiated
 * type parameter, so they share a shape but no common base. These two structural
 * types are what lets one projection cover all of them instead of one copy per
 * instantiation.
 */
type GeneratedFact<T> = { known: boolean; value: T; provenance: schema.Provenance };
type GeneratedCapability<R> = {
  known: boolean;
  enabled: boolean;
  rules?: R;
  provenance: schema.Provenance;
};

/**
 * The three optional provenance parts are omitted by the backend exactly when
 * they are empty, so restoring the empty string is the encoding, not a
 * fallback: no origin is invented for a fact that has none.
 */
function toProvenance(provenance: schema.Provenance): CatalogProvenance {
  return {
    source: provenance.source,
    method: provenance.method,
    table: provenance.table ?? "",
    row: provenance.row ?? "",
    field: provenance.field ?? "",
  };
}

/** Carries one fact over unchanged, including the raw value of an unknown one. */
function toFact<T>(fact: GeneratedFact<T>): CatalogFact<T> {
  return { known: fact.known, value: fact.value, provenance: toProvenance(fact.provenance) };
}

/** An absent optional fact stays absent: null, never a zero-valued fact. */
function toOptionalFact<T>(fact: GeneratedFact<T> | undefined | null): CatalogFact<T> | null {
  return fact ? toFact(fact) : null;
}

/**
 * Carries one capability over unchanged. Absent rules stay null and are never
 * derived from `enabled`, nor `enabled` from them. `rulesEvidence` is dropped:
 * it is not part of the application contract of this step.
 */
function toCapability<R, M>(
  capability: GeneratedCapability<R>,
  toRules: (rules: R) => M,
): CatalogCapability<M> {
  return {
    known: capability.known,
    enabled: capability.enabled,
    rules: capability.rules ? toRules(capability.rules) : null,
    provenance: toProvenance(capability.provenance),
  };
}

function toUpgradeRules(rules: schema.UpgradeRules): CatalogUpgradeRules {
  return {
    model: rules.model,
    maxLevel: rules.maxLevel,
    maxLevelSFV: toOptionalFact(rules["maxLevel-sfv"]),
  };
}

function toInfusionRules(rules: schema.InfusionRules): CatalogInfusionRules {
  return { allowedAffinities: [...rules.allowedAffinities] };
}

function toAshOfWarMountRules(rules: schema.AshOfWarMountRules): CatalogAshOfWarMountRules {
  return {
    mode: rules.mode,
    weaponType: rules.weaponType,
    compatibilityBit: rules.compatibilityBit,
  };
}

function toStackRules(rules: schema.StackRules): CatalogStackRules {
  return { maxPerStack: rules.maxPerStack };
}

function toEquipmentRules(rules: schema.EquipmentRules): CatalogEquipmentRules {
  return { allowedSlots: [...rules.allowedSlots] };
}

/**
 * Projects the common part of a generated item document onto the application
 * port shape. Only the fields the port declares are read: acquisition,
 * modifiers, links, variants, aliases, unlocks, technical and source records
 * and the family-specific blocks stay in the transport result and reach no
 * layer above this one.
 */
function toCatalogItemDetail(item: schema.ItemDocument): CatalogItemDetail {
  return {
    gameID: toFact(item.gameID),
    family: toFact(item.family),
    category: toFact(item.category),
    subcategory: toFact(item.subcategory),
    presentation: {
      name: toFact(item.presentation.name),
      caption: toFact(item.presentation.caption),
      description: toFact(item.presentation.description),
      location: toFact(item.presentation.location),
      // Metadata only: the adapter never turns this path into an icon.
      iconPath: toFact(item.presentation.iconPath),
    },
    storage: {
      recordMode: toFact(item.storage.recordMode),
      // No effective limit is computed here: the raw limits are carried as they
      // are reported, and which one applies is a later contract.
      maxInventory: toFact(item.storage.maxInventory),
      safeModeMaxInventory: toOptionalFact(item.storage.safeModeMaxInventory),
      maxInventorySFV: toOptionalFact(item.storage["maxInventory-sfv"]),
      maxStorage: toFact(item.storage.maxStorage),
      safeModeMaxStorage: toOptionalFact(item.storage.safeModeMaxStorage),
      maxStorageSFV: toOptionalFact(item.storage["maxStorage-sfv"]),
    },
    safety: {
      // Six independent facts, never folded into a risk level here.
      cutContent: toFact(item.safety.cutContent),
      banRisk: toFact(item.safety.banRisk),
      dlc: toFact(item.safety.dlc),
      noDatabase: toFact(item.safety.noDatabase),
      scalesWithNG: toFact(item.safety.scalesWithNG),
      preOrder: toFact(item.safety.preOrder),
    },
    capabilities: {
      upgrade: toCapability(item.capabilities.upgrade, toUpgradeRules),
      infusion: toCapability(item.capabilities.infusion, toInfusionRules),
      ashOfWarMount: toCapability(item.capabilities.ashOfWarMount, toAshOfWarMountRules),
      stack: toCapability(item.capabilities.stack, toStackRules),
      equipment: toCapability(item.capabilities.equipment, toEquipmentRules),
    },
  };
}

/**
 * Projects the generated resource union onto the application port shape. The
 * identity is carried verbatim, and a resource of any other kind keeps `item`
 * null rather than having its own document mapped onto the item shape.
 */
function toCatalogResourceDetail(
  result: Awaited<ReturnType<typeof GetResource>>,
): CatalogResourceDetail {
  return {
    kind: result.resource.kind,
    key: result.resource.key,
    item: result.resource.item ? toCatalogItemDetail(result.resource.item) : null,
  };
}

/**
 * Projects the generated variant list onto the application port shape. Only the
 * five facts the port declares are read: `data` and `sourceRecords` stay in the
 * transport result and reach no layer above this one, and no variant is
 * materialised into an item document. The catalog order is the backend's, so
 * the list is mapped in place and never sorted, filtered or deduplicated; an
 * item without variants stays an empty list.
 */
function toCatalogItemVariants(
  result: Awaited<ReturnType<typeof GetItemVariants>>,
): CatalogItemVariantsResult {
  return {
    variants: result.variants.map((variant) => ({
      gameID: toFact(variant.gameID),
      kind: toFact(variant.kind),
      affinity: toFact(variant.affinity),
      upgradeLevel: toFact(variant.upgradeLevel),
      sourceRowID: toFact(variant.sourceRowID),
    })),
  };
}

/**
 * Projects the generated authoritative container page onto the application port
 * shape. Every value is copied into an object this layer owns, exactly as the
 * backend reported it: nothing is renamed, defaulted, clamped or recomputed,
 * and the action flags stay the backend's own decisions.
 */
function toOwnedItemsPage(result: Awaited<ReturnType<typeof GetOwnedItems>>): OwnedItemsPage {
  return {
    saveSessionID: result.saveSessionID,
    saveRevision: result.saveRevision,
    characterID: result.characterID,
    active: result.active,
    safetyProfile: result.safetyProfile,
    container: result.container,
    records: result.records.map((record) => ({
      ownedItemID: record.ownedItemID,
      kind: record.kind,
      key: record.key,
      gameID: record.gameID,
      container: record.container,
      containerSection: record.containerSection,
      physicalIndex: record.physicalIndex,
      acquisitionIndex: record.acquisitionIndex,
      orderPosition: record.orderPosition,
      orderPositionKnown: record.orderPositionKnown,
      quantity: record.quantity,
      maxQuantity: record.maxQuantity,
      maxQuantityKnown: record.maxQuantityKnown,
      family: record.family,
      category: record.category,
      subcategory: record.subcategory,
      name: record.name,
      iconPath: record.iconPath,
      recordMode: record.recordMode,
      banRisk: record.banRisk,
      cutContent: record.cutContent,
      dlc: record.dlc,
      preOrder: record.preOrder,
      actions: {
        moveToStorage: record.actions.moveToStorage,
        moveToInventory: record.actions.moveToInventory,
        remove: record.actions.remove,
        setQuantity: record.actions.setQuantity,
        reorder: record.actions.reorder,
      },
    })),
    categories: result.categories.map((entry) => ({
      category: entry.category,
      count: entry.count,
    })),
    total: result.total,
    page: result.page,
    pageSize: result.pageSize,
  };
}

/** Projects the generated Item Database page onto the application port shape. */
function toCatalogItemDatabasePage(
  result: Awaited<ReturnType<typeof GetItemDatabase>>,
): CatalogItemDatabasePage {
  return {
    safetyProfile: result.safetyProfile,
    resources: result.resources.map((entry) => ({
      kind: entry.kind,
      key: entry.key,
      gameID: entry.gameID,
      gameIDKnown: entry.gameIDKnown,
      family: entry.family,
      category: entry.category,
      subcategory: entry.subcategory,
      name: entry.name,
      iconPath: entry.iconPath,
      banRisk: entry.banRisk,
      cutContent: entry.cutContent,
      dlc: entry.dlc,
      preOrder: entry.preOrder,
    })),
    categories: result.categories.map((entry) => ({
      category: entry.category,
      count: entry.count,
    })),
    total: result.total,
    page: result.page,
    pageSize: result.pageSize,
  };
}

/**
 * Projects one committed item mutation onto the shared receipt. The scopes pass
 * through the same validation every other receipt uses, so an unknown scope is
 * rejected at this boundary instead of reaching the invalidation map.
 */
function toItemMutationReceipt(result: {
  operationID: string;
  operationKind: string;
  saveSessionID: string;
  saveRevision: string;
  changedScopes: string[];
}): ItemMutationReceipt {
  return {
    operationID: result.operationID,
    operationKind: result.operationKind,
    saveSessionID: result.saveSessionID,
    saveRevision: result.saveRevision,
    changedScopes: toChangedScopes(result.changedScopes),
  };
}

/**
 * Projects one served candidate page. The candidates are copied into arrays and
 * objects this layer owns, in the backend's own order: nothing is re-filtered,
 * re-sorted, de-duplicated or checked against a local compatibility rule.
 */
function toEquipmentCandidatesPage(
  result: Awaited<ReturnType<typeof GetEquipmentCandidates>>,
): EquipmentCandidatesPage {
  return {
    saveSessionID: result.saveSessionID,
    saveRevision: result.saveRevision,
    characterID: result.characterID,
    active: result.active,
    safetyProfile: result.safetyProfile,
    slotType: result.slotType,
    candidates: result.candidates.map((candidate) => ({
      resource: { kind: candidate.resource.kind, key: candidate.resource.key },
      ownedItemID: candidate.ownedItemID,
      name: candidate.name,
      iconPath: candidate.iconPath,
      quantity: candidate.quantity,
      memorySlots: candidate.memorySlots,
      banRisk: candidate.banRisk,
      cutContent: candidate.cutContent,
    })),
    total: result.total,
    // The served page and page size are the backend's answer, not an echo of
    // the request: zero paging resolves to the backend default there.
    page: result.page,
    pageSize: result.pageSize,
  };
}

/**
 * Projects one committed Equipment mutation onto the shared receipt. The scopes
 * pass through the same validation every other receipt uses, so an unknown
 * scope is rejected at this boundary instead of reaching the invalidation map.
 */
function toEquipmentMutationReceipt(result: {
  operationID: string;
  operationKind: string;
  saveSessionID: string;
  saveRevision: string;
  changedScopes: string[];
}): EquipmentMutationReceipt {
  return {
    operationID: result.operationID,
    operationKind: result.operationKind,
    saveSessionID: result.saveSessionID,
    saveRevision: result.saveRevision,
    changedScopes: toChangedScopes(result.changedScopes),
  };
}

/**
 * The identity and slot state every World getter reports. It is copied field by
 * field, so the generated transport object never becomes application state.
 */
function toWorldIdentity(result: {
  saveSessionID: string;
  saveRevision: string;
  characterID: number;
  active: boolean;
}) {
  return {
    saveSessionID: result.saveSessionID,
    saveRevision: result.saveRevision,
    characterID: result.characterID,
    active: result.active,
  };
}

/**
 * The positional arguments of one World resource toggle. Every one of the
 * eleven toggle endpoints takes the same six values in the same order, so the
 * request is unpacked once here instead of eleven times, and no name, kind, key
 * or value changes meaning on the way.
 */
function toggleArguments({
  saveSessionID,
  characterID,
  resourceKind,
  resourceKey,
  value,
  expectedRevision,
}: WorldResourceToggleRequest): [string, number, string, string, boolean, string] {
  return [saveSessionID, characterID, resourceKind, resourceKey, value, expectedRevision];
}

/**
 * The World mutation contract, validated as the closed vocabulary it is. An
 * operation kind this build does not know, or a risk level outside the shared
 * backend vocabulary, fails the whole answer with the same stable code as any
 * other bridge failure: an unknown capability must never reach the screen as an
 * enabled action, and a missing risk must never be replaced by a default.
 */
function toWorldMutationCapabilities(
  result: Awaited<ReturnType<typeof GetWorldMutationCapabilities>>,
): readonly WorldMutationCapability[] {
  return result.capabilities.map((capability) => {
    if (!worldOperationKinds.includes(capability.operationKind as WorldOperationKind)) {
      throw new AppErrorException(bridgeCallFailed());
    }
    return {
      operationKind: capability.operationKind as WorldOperationKind,
      risk: toOperationRisk(capability.risk),
      riskReason: capability.riskReason,
      supportsBulk: capability.supportsBulk,
    };
  });
}

/**
 * Projects one committed World mutation onto the shared receipt, through the
 * same scope validation every other receipt uses.
 */
function toWorldMutationReceipt(result: {
  operationID: string;
  operationKind: string;
  saveSessionID: string;
  saveRevision: string;
  changedScopes: string[];
}): WorldMutationReceipt {
  return toMutationReceipt(result);
}

/**
 * The thirteen World projections below copy the arrays and the entries into
 * objects this layer owns, in the backend's own order. No entry is filtered,
 * re-sorted, grouped, renamed or completed with a locally derived fact: the
 * resource kind stays the backend's own string, an empty label stays empty, and
 * the Spectral Steed status is carried exactly as classified.
 */
function toWorldRegions(result: Awaited<ReturnType<typeof GetRegions>>): WorldRegions {
  return {
    ...toWorldIdentity(result),
    regions: result.regions.map((entry) => ({
      kind: entry.kind,
      key: entry.key,
      name: entry.name,
      area: entry.area,
      unlocked: entry.unlocked,
    })),
  };
}

function toWorldMapRegions(result: Awaited<ReturnType<typeof GetMapRegions>>): WorldMapRegions {
  return {
    ...toWorldIdentity(result),
    mapRegions: result.mapRegions.map((entry) => ({
      kind: entry.kind,
      key: entry.key,
      name: entry.name,
      areaLabel: entry.areaLabel,
      visible: entry.visible,
    })),
  };
}

function toWorldGraces(result: Awaited<ReturnType<typeof GetGraces>>): WorldGraces {
  return {
    ...toWorldIdentity(result),
    graces: result.graces.map((entry) => ({
      kind: entry.kind,
      key: entry.key,
      name: entry.name,
      regionLabel: entry.regionLabel,
      bossArena: entry.bossArena,
      dungeonType: entry.dungeonType,
      visited: entry.visited,
    })),
  };
}

function toWorldBosses(result: Awaited<ReturnType<typeof GetBosses>>): WorldBosses {
  return {
    ...toWorldIdentity(result),
    bosses: result.bosses.map((entry) => ({
      kind: entry.kind,
      key: entry.key,
      name: entry.name,
      regionLabel: entry.regionLabel,
      encounterType: entry.encounterType,
      remembrance: entry.remembrance,
      defeated: entry.defeated,
    })),
  };
}

function toWorldQuests(result: Awaited<ReturnType<typeof GetQuests>>): WorldQuests {
  return {
    ...toWorldIdentity(result),
    quests: result.quests.map((quest) => ({
      kind: quest.kind,
      key: quest.key,
      name: quest.name,
      steps: quest.steps.map((step) => ({
        stepKind: step.stepKind,
        stepKey: step.stepKey,
        description: step.description,
        location: step.location,
        matched: step.matched,
      })),
    })),
  };
}

function toWorldGestures(result: Awaited<ReturnType<typeof GetGestures>>): WorldGestures {
  return {
    ...toWorldIdentity(result),
    gestures: result.gestures.map((entry) => ({
      kind: entry.kind,
      key: entry.key,
      slotID: entry.slotID,
      name: entry.name,
      category: entry.category,
      unlocked: entry.unlocked,
    })),
  };
}

function toWorldCookbooks(result: Awaited<ReturnType<typeof GetCookbooks>>): WorldCookbooks {
  return {
    ...toWorldIdentity(result),
    cookbooks: result.cookbooks.map((entry) => ({
      kind: entry.kind,
      key: entry.key,
      name: entry.name,
      category: entry.category,
      unlocked: entry.unlocked,
    })),
  };
}

function toWorldBellBearings(
  result: Awaited<ReturnType<typeof GetBellBearings>>,
): WorldBellBearings {
  return {
    ...toWorldIdentity(result),
    bellBearings: result.bellBearings.map((entry) => ({
      kind: entry.kind,
      key: entry.key,
      name: entry.name,
      category: entry.category,
      unlocked: entry.unlocked,
    })),
  };
}

function toWorldWhetblades(result: Awaited<ReturnType<typeof GetWhetblades>>): WorldWhetblades {
  return {
    ...toWorldIdentity(result),
    whetblades: result.whetblades.map((entry) => ({
      kind: entry.kind,
      key: entry.key,
      name: entry.name,
      unlocked: entry.unlocked,
    })),
  };
}

function toWorldTutorials(result: Awaited<ReturnType<typeof GetTutorials>>): WorldTutorials {
  return {
    ...toWorldIdentity(result),
    tutorials: result.tutorials.map((entry) => ({
      kind: entry.kind,
      key: entry.key,
      title: entry.title,
      unlocked: entry.unlocked,
    })),
  };
}

function toWorldSummoningPools(
  result: Awaited<ReturnType<typeof GetSummoningPools>>,
): WorldSummoningPools {
  return {
    ...toWorldIdentity(result),
    summoningPools: result.summoningPools.map((entry) => ({
      kind: entry.kind,
      key: entry.key,
      name: entry.name,
      regionLabel: entry.regionLabel,
      activated: entry.activated,
    })),
  };
}

function toWorldColosseums(result: Awaited<ReturnType<typeof GetColosseums>>): WorldColosseums {
  return {
    ...toWorldIdentity(result),
    colosseums: result.colosseums.map((entry) => ({
      kind: entry.kind,
      key: entry.key,
      name: entry.name,
      unlocked: entry.unlocked,
    })),
  };
}

function toWorldSpectralSteedAttires(
  result: Awaited<ReturnType<typeof GetSpectralSteedAttires>>,
): WorldSpectralSteedAttires {
  return {
    ...toWorldIdentity(result),
    // `legacy` and `conflict` are answers, not failures: the status and the
    // active key are carried as reported and never resolved into one attire.
    // Only a value outside the closed contract is rejected.
    status: toSpectralSteedAttireStatus(result.status),
    activeAttireKey: result.activeAttireKey,
    attires: result.attires.map((entry) => ({
      attireKey: entry.attireKey,
      name: entry.name,
      owned: entry.owned,
      requiredResourceKind: entry.requiredResourceKind,
      requiredResourceKey: entry.requiredResourceKey,
      iconPath: entry.iconPath,
    })),
  };
}

const networkParamKeys: readonly (keyof NetworkParamValues)[] = [
  "maxBreakInTargetListCount",
  "breakInRequestIntervalTimeSec",
  "breakInRequestTimeOutSec",
  "breakInRequestAreaCount",
  "summonTimeoutTime",
  "reloadSignIntervalTime2",
  "reloadSignTotalCount",
  "reloadSignCellCount",
  "updateSignIntervalTime",
  "singGetMax",
  "signDownloadSpan",
  "signUpdateSpan",
  "reloadVisitListCoolTime",
  "maxCoopBlueSummonCount",
  "maxVisitListCount",
  "reloadSearchCoopBlueMin",
  "reloadSearchCoopBlueMax",
  "allAreaSearchRateCoopBlue",
  "allAreaSearchRateVsBlue",
  "visitorListMax",
  "visitorTimeOutTime",
  "visitorDownloadSpan",
];

function toNetworkParamValues(raw: unknown): NetworkParamValues {
  if (typeof raw !== "object" || raw === null) {
    throw new AppErrorException(bridgeCallFailed());
  }
  const record = raw as Record<string, unknown>;
  const values = {} as NetworkParamValues;
  for (const key of networkParamKeys) {
    const val = record[key];
    if (typeof val !== "number" || !Number.isFinite(val)) {
      throw new AppErrorException(bridgeCallFailed());
    }
    values[key] = val;
  }
  return values;
}

function toNetworkSettingsSnapshot(
  result: Awaited<ReturnType<typeof GetNetworkSettings>>,
): NetworkSettingsSnapshot {
  if (typeof result.saveSessionID !== "string" || typeof result.saveRevision !== "string") {
    throw new AppErrorException(bridgeCallFailed());
  }
  return {
    saveSessionID: result.saveSessionID,
    saveRevision: result.saveRevision,
    parameters: toNetworkParamValues(result.parameters),
  };
}

function toNetworkPresetsResult(
  result: Awaited<ReturnType<typeof GetNetworkPresets>>,
): NetworkPresetsResult {
  if (!Array.isArray(result?.presets)) {
    throw new AppErrorException(bridgeCallFailed());
  }
  return {
    presets: result.presets.map((preset) => {
      if (typeof preset?.id !== "string" || preset.id === "") {
        throw new AppErrorException(bridgeCallFailed());
      }
      return {
        id: preset.id,
        parameters: toNetworkParamValues(preset.parameters),
      };
    }),
  };
}

function toSetNetworkSettingsResult(
  result: Awaited<ReturnType<typeof SetNetworkSettings>>,
): SetNetworkSettingsResult {
  const receipt = toMutationReceipt(result);
  return {
    ...receipt,
    networkSettings: toNetworkParamValues(result.networkSettings),
  };
}

/** Projects the generated settings result onto the application port shape. */
function toSafetyProfileSettings(
  result: Awaited<ReturnType<typeof GetSafetyProfile>>,
): SafetyProfileSettings {
  return {
    safetyProfile: result.safetyProfile,
    availableProfiles: [...result.availableProfiles],
    defaultProfile: result.defaultProfile,
  };
}


/**
 * Projects the frontend's override shape onto the backend apply options.
 *
 * Only the four confirmed option groups exist here. An override the backend
 * does not define is not invented, and an unset value stays unset rather than
 * being sent as a default the user never chose.
 */
function toApplyOptions(overrides: BuildTemplateOverrides | undefined) {
  if (overrides === undefined) return undefined;
  const options: Record<string, unknown> = {};
  if (overrides.itemsMode !== undefined) {
    options.items = {
      mode: overrides.itemsMode,
      preserveExtraItems: overrides.preserveExtraItems ?? false,
    };
  }
  if (overrides.inventoryLayoutMode !== undefined) {
    options.inventoryLayout = { mode: overrides.inventoryLayoutMode };
  }
  if (overrides.storageLayoutMode !== undefined) {
    options.storageLayout = { mode: overrides.storageLayoutMode };
  }
  if (
    overrides.useTemplateWeaponLevels !== undefined ||
    overrides.standardUpgradeOverride !== undefined ||
    overrides.somberUpgradeOverride !== undefined
  ) {
    options.weaponLevelOverride = {
      useTemplateLevels: overrides.useTemplateWeaponLevels ?? true,
      standardOverride: overrides.standardUpgradeOverride,
      somberOverride: overrides.somberUpgradeOverride,
    };
  }
  return Object.keys(options).length === 0 ? undefined : options;
}

/** Projects the generated template index onto the application port shape. */
function toBuildTemplatePage(
  result: Awaited<ReturnType<typeof GetBuildTemplates>>,
): BuildTemplatePage {
  return {
    templates: result.templates.map((template) => ({
      templateID: template.templateID,
      name: template.name,
      description: template.description,
      tags: [...(template.tags ?? [])],
      createdAt: template.createdAt,
      updatedAt: template.updatedAt,
      schemaVersion: template.schemaVersion,
      selectedSections: [...(template.selectedSections ?? [])],
      inventoryItems: template.inventoryItems,
      storageItems: template.storageItems,
      warnings: template.warnings,
      templateRevision: template.templateRevision,
    })),
    total: result.total,
    page: result.page,
    pageSize: result.pageSize,
  };
}

/**
 * Projects the generated preview plan onto the application port shape.
 *
 * The eight statistics are flattened into an ordered list of named changes, so
 * the interface renders exactly the fields the backend reported and never
 * assumes a fixed set of its own.
 */
function toBuildTemplatePlan(
  plan: Awaited<ReturnType<typeof GetBuildTemplatePreview>>["plan"],
): BuildTemplatePlan {
  const statistics = plan.stats;
  const fields =
    statistics === undefined
      ? []
      : (
          [
            ["vigor", statistics.vigor],
            ["mind", statistics.mind],
            ["endurance", statistics.endurance],
            ["strength", statistics.strength],
            ["dexterity", statistics.dexterity],
            ["intelligence", statistics.intelligence],
            ["faith", statistics.faith],
            ["arcane", statistics.arcane],
          ] as const
        )
          .filter(([, change]) => change !== undefined)
          .map(([field, change]) => ({
            field,
            change: {
              current: (change as { current: number }).current,
              target: (change as { target: number }).target,
              changed: (change as { changed: boolean }).changed,
            },
          }));

  return {
    profile:
      plan.profile === undefined
        ? undefined
        : {
            name: plan.profile.name,
            level: plan.profile.level,
          },
    stats:
      statistics === undefined
        ? undefined
        : {
            fields,
            resultLevel: statistics.resultLevel,
            resultSoulMemory: statistics.resultSoulMemory,
          },
    spells:
      plan.spells === undefined
        ? undefined
        : {
            changedSlots: (plan.spells.slots ?? []).filter((slot) => slot.changed).length,
            usedMemorySlots: plan.spells.usedMemorySlots,
            availableMemorySlots: plan.spells.availableMemorySlots,
          },
  };
}

function toBuildTemplatePreview(
  result: Awaited<ReturnType<typeof GetBuildTemplatePreview>>,
): BuildTemplatePreview {
  return {
    templateID: result.templateID,
    templateRevision: result.templateRevision,
    characterID: result.characterID,
    saveSessionID: result.saveSessionID,
    saveRevision: result.saveRevision,
    executable: result.executable,
    plan: toBuildTemplatePlan(result.plan),
    blockingIssues: (result.blockingIssues ?? []).map((issue) => ({
      code: issue.code,
      section: issue.section,
      field: issue.field,
      message: issue.message,
    })),
  };
}

/** Projects the generated host settings onto the application port shape. */
function toHostSettings(result: application.HostSettingsResult): HostSettings {
  return {
    skipReviewForNormalRisk: result.skipReviewForNormalRisk,
    remoteBackupPolicy: result.remoteBackupPolicy,
    availableRemoteBackupPolicies: [...result.availableRemoteBackupPolicies],
    defaultRemoteBackupPolicy: result.defaultRemoteBackupPolicy,
    configurationDirectoryExists: result.configurationDirectoryExists,
    logDirectoryExists: result.logDirectoryExists,
  };
}

function toDeploymentTarget(target: deployment.TargetEntry): DeploymentTarget {
  return {
    id: target.id,
    name: target.name,
    kind: target.kind,
    savePath: target.savePath,
    startCommand: target.startCommand,
    stopCommand: target.stopCommand,
    host: target.host,
    port: target.port,
    user: target.user,
    keyPath: target.keyPath,
    hostKeyTrusted: target.hostKeyTrusted,
    hostKeyFingerprint: target.hostKeyFingerprint,
    transferSupported: target.transferSupported,
    unsupportedReason: target.unsupportedReason,
  };
}

function toDeploymentTargets(result: deployment.GetDeploymentTargetsResult): DeploymentTargets {
  return {
    targets: result.targets.map(toDeploymentTarget),
    availableKinds: [...result.availableKinds],
  };
}

function toCommandOutcome(outcome: deployment.CommandOutcome): CommandOutcome {
  return {
    configured: outcome.configured,
    executed: outcome.executed,
    exitCode: outcome.exitCode,
    detail: outcome.detail,
  };
}

function toDeploymentTargetState(
  value: string,
): DeploymentOperationResult["targetState"] {
  switch (value) {
    case "unchanged":
    case "replaced_verified":
    case "replaced_unverified":
      return value;
    default:
      throw new AppErrorException(bridgeCallFailed());
  }
}

function toOperationResult(result: deployment.OperationResult): DeploymentOperationResult {
  return {
    operationID: result.operationID,
    targetID: result.targetID,
    completed: result.completed,
    blocked: result.blocked,
    failure: result.failure,
    targetState: toDeploymentTargetState(result.targetState),
    gameStatus: result.gameStatus,
    stages: (result.stages ?? []).map((stage) => ({
      stage: stage.stage,
      completed: stage.completed,
      detail: stage.detail,
    })),
    backupID: result.backupID,
    localPath: result.localPath,
    stop: result.stop === undefined ? undefined : toCommandOutcome(result.stop),
    launch: result.launch === undefined ? undefined : toCommandOutcome(result.launch),
  };
}

function toTargetBackup(backup: deployment.BackupRecord): TargetBackup {
  return {
    id: backup.id,
    targetID: backup.targetID,
    fileName: backup.fileName,
    createdAt: backup.createdAt,
    manual: backup.manual,
    active: backup.active,
    tags: [...(backup.tags ?? [])],
    description: backup.description,
  };
}

function toTargetBackups(result: deployment.GetTargetBackupsResult): TargetBackups {
  return {
    targetID: result.targetID,
    backups: (result.backups ?? []).map(toTargetBackup),
    transferSupported: result.transferSupported,
    unsupportedReason: result.unsupportedReason,
  };
}

/**
 * The single adapter behind every application port. A second, parallel
 * adaptation layer would give the generated bindings a second way into the
 * application, so all ports are fulfilled here.
 */
export const wailsDesktopBridge: ApplicationInfoPort &
  SaveSessionPort &
  DiagnosticsPort &
  CharacterPort &
  AppearancePort &
  FavoritesPort &
  ItemsPort &
  EquipmentPort &
  WorldPort &
  NetworkPort &
  SettingsPort &
  HostSettingsPort &
  AboutPort &
  TemplatePort &
  DeploymentPort &
  CatalogPort = {
  getApplicationInfo: async (): Promise<ApplicationInfo> => {
    const result = await callBridge(GetApplicationInfo);

    return {
      version: result.applicationVersion,
      build: result.build,
      platform: result.platform,
      schemas: result.supportedSchemas.map((schema) => ({
        name: schema.name,
        minimumVersion: schema.minimumVersion,
        currentVersion: schema.currentVersion,
      })),
      capabilities: [...result.capabilities],
    };
  },

  // Cancelling is the host's empty path, so it must survive the boundary as an
  // empty string rather than becoming an error: only a dialog that actually
  // failed rejects here.
  selectSaveFile: async () => callBridge(SelectSaveFile),

  selectSaveTarget: async (suggestedName) => callBridge(() => SelectSaveTarget(suggestedName)),

  subscribeApplicationCloseRequested: (listener) =>
    EventsOn("application.close-requested", listener),

  quitApplication: async () => {
    await callBridge(QuitApplication);
  },

  // All three values reach the bridge in the order the backend contract
  // defines and exactly as received. The adapter supplies no default source
  // kind: a session must never claim an origin nobody stated.
  loadSave: async (source, expectedPlatform, sourceKind) =>
    toSaveSession(await callBridge(() => LoadSave(source, expectedPlatform, sourceKind))),

  releaseDeploymentStaging: async (localPath) => {
    await callBridge(() => ReleaseDeploymentStaging(localPath));
  },

  getLoadedSave: async (saveSessionID) =>
    toSaveSession(await callBridge(() => GetLoadedSave(saveSessionID))),

  closeSave: async (saveSessionID) => {
    await callBridge(() => CloseSave(saveSessionID));
  },

  getOperationHistory: async (saveSessionID) =>
    toOperationHistory(await callBridge(() => GetOperationHistory(saveSessionID))),

  undoLastOperation: async (saveSessionID, expectedRevision) =>
    toHistoryMutationResult(
      await callBridge(() => UndoLastOperation(saveSessionID, expectedRevision)),
    ),

  redoLastOperation: async (saveSessionID, expectedRevision) =>
    toHistoryMutationResult(
      await callBridge(() => RedoLastOperation(saveSessionID, expectedRevision)),
    ),

  revertOperation: async (saveSessionID, operationID, expectedRevision) =>
    toHistoryMutationResult(
      await callBridge(() => RevertOperation(saveSessionID, operationID, expectedRevision)),
    ),

  discardChanges: async (saveSessionID, expectedRevision) => {
    const result = await callBridge(() => DiscardChanges(saveSessionID, expectedRevision));
    return { ...toMutationReceipt(result), discardedOperations: result.discardedOperations };
  },

  validateReviewChanges: async (saveSessionID, expectedRevision) =>
    toReviewValidationResult(
      await callBridge(() => ValidateReviewChanges(saveSessionID, expectedRevision)),
    ),

  save: async (saveSessionID, expectedRevision, validationToken, confirmWarnings, confirmBanRisk) =>
    toSaveLifecycleResult(
      await callBridge(() =>
        Save(saveSessionID, expectedRevision, validationToken, confirmWarnings, confirmBanRisk),
      ),
    ),

  saveAs: async (
    saveSessionID,
    expectedRevision,
    validationToken,
    confirmWarnings,
    confirmBanRisk,
    target,
  ) =>
    toSaveLifecycleResult(
      await callBridge(() =>
        SaveAs(
          saveSessionID,
          expectedRevision,
          validationToken,
          confirmWarnings,
          confirmBanRisk,
          target,
        ),
      ),
    ),

  getRecentFiles: async () => (await callBridge(GetRecentFiles)).map(toRecentFile),

  recordRecentFile: async (saveSessionID) =>
    (await callBridge(() => RecordRecentFile(saveSessionID))).map(toRecentFile),

  removeRecentFile: async (path) =>
    (await callBridge(() => RemoveRecentFile(path))).map(toRecentFile),

  clearRecentFiles: async () => {
    await callBridge(ClearRecentFiles);
  },

  getRecoveryJournals: async () => (await callBridge(GetRecoveryJournals)).map(toRecoveryJournal),

  getRecoveryJournal: async (journalID) =>
    toRecoveryJournal(await callBridge(() => GetRecoveryJournal(journalID))),

  restoreRecoveryJournal: async (journalID) =>
    toSaveSession(await callBridge(() => RestoreRecoveryJournal(journalID))),

  discardRecoveryJournal: async (journalID) => {
    await callBridge(() => DiscardRecoveryJournal(journalID));
  },

  exportRecoveryJournal: async (journalID, target) => {
    await callBridge(() => ExportRecoveryJournal(journalID, target));
  },

  getSaveLifecycleSettings: async () =>
    toSaveLifecycleSettings(await callBridge(GetSaveLifecycleSettings)),

  setSaveLifecycleSettings: async (backupRetention) =>
    toSaveLifecycleSettings(await callBridge(() => SetSaveLifecycleSettings(backupRetention))),

  // The identifier is forwarded as the string it is and is never echoed back:
  // the result of the call is the mutation receipt alone.
  setSaveAccountID: async (saveSessionID, accountID, expectedRevision) =>
    toMutationReceipt(
      await callBridge(() => SetSaveAccountID(saveSessionID, accountID, expectedRevision)),
    ),

  /**
   * Subscribes to the backend's committed mutations.
   *
   * The Wails event bus hands the payload over as an untyped value, so it is
   * validated here exactly like an error envelope: an event that does not carry
   * the complete contract is dropped rather than delivered half-built. A
   * dropped event is safe — the listener notices the sequence gap and
   * resynchronises — while a half-built one would invalidate the wrong scopes.
   */
  // A payload that fails validation is reported as null rather than swallowed:
  // the listener owns what to do about a notification it cannot read, and its
  // answer is a resynchronisation, not silence.
  subscribeSessionChanged: (listener) =>
    EventsOn(sessionChangedEventName, (...data: unknown[]) => {
      listener(parseSessionChangedEvent(data[0]));
    }),

  // The scope reaches the bridge exactly as received, the empty string
  // included: which scopes exist and what an empty one means is the backend's
  // contract, and this layer neither narrows nor expands it.
  getSaveValidationReport: async ({ saveSessionID, characterID, scope }) =>
    toSaveValidationReport(
      await callBridge(() => GetSaveValidationReport(saveSessionID, characterID, scope)),
    ),

  // The selected findings reach the backend in the caller's order and the plan
  // comes back untouched: which of them earns an action and which one is
  // rejected is decided there and nowhere else.
  getRepairPlan: async ({ saveSessionID, characterID, saveRevision, issueIDs }) =>
    toRepairPlan(
      await callBridge(() =>
        GetRepairPlan(saveSessionID, characterID, saveRevision, [...issueIDs]),
      ),
    ),

  applyRepairs: async ({ saveSessionID, characterID, issueIDs, planToken, expectedRevision }) =>
    toApplyRepairsResult(
      await callBridge(() =>
        ApplyRepairs(saveSessionID, characterID, [...issueIDs], planToken, expectedRevision),
      ),
    ),

  getSaveCharacters: async (saveSessionID): Promise<SaveCharacters> => {
    const result = await callBridge(() => GetSaveCharacters(saveSessionID));

    return {
      saveSessionID: result.saveSessionID,
      saveRevision: result.saveRevision,
      characters: result.characters.map((summary) => ({
        characterID: summary.characterID,
        active: summary.active,
        name: summary.name,
        level: summary.level,
      })),
    };
  },

  getCharacterProfile: async (saveSessionID, characterID): Promise<CharacterProfile> => {
    const result = await callBridge(() => GetCharacterProfile(saveSessionID, characterID));

    return {
      saveSessionID: result.saveSessionID,
      saveRevision: result.saveRevision,
      characterID: result.characterID,
      active: result.active,
      name: result.name,
      level: result.level,
      startingClassID: result.startingClassID,
      gender: result.gender,
      secondsPlayed: result.secondsPlayed,
    };
  },

  getCharacterStats: async (saveSessionID, characterID): Promise<CharacterStats> => {
    const result = await callBridge(() => GetCharacterStats(saveSessionID, characterID));

    return {
      saveSessionID: result.saveSessionID,
      saveRevision: result.saveRevision,
      characterID: result.characterID,
      active: result.active,
      vigor: result.vigor,
      mind: result.mind,
      endurance: result.endurance,
      strength: result.strength,
      dexterity: result.dexterity,
      intelligence: result.intelligence,
      faith: result.faith,
      arcane: result.arcane,
      level: result.level,
      hp: result.hp,
      maxHP: result.maxHP,
      baseMaxHP: result.baseMaxHP,
      fp: result.fp,
      maxFP: result.maxFP,
      baseMaxFP: result.baseMaxFP,
      sp: result.sp,
      maxSP: result.maxSP,
      baseMaxSP: result.baseMaxSP,
      runes: result.runes,
      soulMemory: result.soulMemory,
    };
  },

  setCharacterName: async ({ saveSessionID, characterID, name, expectedRevision }) =>
    toMutationReceipt(
      await callBridge(() => SetCharacterName(saveSessionID, characterID, name, expectedRevision)),
    ),

  setCharacterStats: async ({
    saveSessionID,
    characterID,
    attributes,
    levelPolicy,
    expectedRevision,
  }) =>
    toMutationReceipt(
      await callBridge(() =>
        SetCharacterStats(
          saveSessionID,
          characterID,
          new saveengine.CharacterAttributes(attributes),
          levelPolicy,
          expectedRevision,
        ),
      ),
    ),

  setCharacterStartingClass: async ({
    saveSessionID,
    characterID,
    startingClassID,
    confirmReset,
    expectedRevision,
  }) =>
    toMutationReceipt(
      await callBridge(() =>
        SetCharacterStartingClass(
          saveSessionID,
          characterID,
          startingClassID,
          confirmReset,
          expectedRevision,
        ),
      ),
    ),

  setCharacterGender: async ({ saveSessionID, characterID, gender, expectedRevision }) =>
    toMutationReceipt(
      await callBridge(() =>
        SetCharacterGender(saveSessionID, characterID, gender, expectedRevision),
      ),
    ),

  setCharacterRunes: async ({ saveSessionID, characterID, runes, expectedRevision }) =>
    toMutationReceipt(
      await callBridge(() =>
        SetCharacterRunes(saveSessionID, characterID, runes, expectedRevision),
      ),
    ),

  getAppearancePresets: async ({
    search = "",
    tags = [],
  } = {}): Promise<readonly AppearancePresetSummary[]> => {
    const result = await callBridge(() => GetAppearancePresets(search, [...tags]));
    return result.presets.map((preset) => ({
      id: preset.id,
      name: preset.name,
      image: preset.image,
      bodyType: preset.bodyType,
      tags: [...preset.tags],
    }));
  },

  applyAppearancePreset: async ({ saveSessionID, characterID, presetID, expectedRevision }) =>
    toMutationReceipt(
      await callBridge(() =>
        ApplyAppearancePreset(saveSessionID, characterID, presetID, expectedRevision),
      ),
    ),

  getFavoritePresets: async (saveSessionID, favoriteSlotID): Promise<SaveFavoritePresets> => {
    const result = await callBridge(() =>
      GetFavoritePresets(saveSessionID, favoriteSlotID !== undefined ? favoriteSlotID : null),
    );
    return {
      saveSessionID: result.saveSessionID,
      presets: result.presets.map((preset) => ({
        favoriteSlotID: preset.favoriteSlotID,
        active: preset.active,
      })),
    };
  },

  setFavoritePreset: async ({
    saveSessionID,
    favoriteSlotID,
    sourceCharacterID,
    expectedRevision,
  }) =>
    toMutationReceipt(
      await callBridge(() =>
        SetFavoritePreset(saveSessionID, favoriteSlotID, sourceCharacterID, expectedRevision),
      ),
    ),

  applyFavoritePreset: async ({ saveSessionID, characterID, favoriteSlotID, expectedRevision }) =>
    toMutationReceipt(
      await callBridge(() =>
        ApplyFavoritePreset(saveSessionID, characterID, favoriteSlotID, expectedRevision),
      ),
    ),

  deleteFavoritePreset: async ({ saveSessionID, favoriteSlotID, expectedRevision }) =>
    toMutationReceipt(
      await callBridge(() => DeleteFavoritePreset(saveSessionID, favoriteSlotID, expectedRevision)),
    ),

  // The five arguments reach the bridge in the order the backend contract
  // defines; the grouped request only protects the caller from transposing them.
  getInventory: async ({ saveSessionID, characterID, containerSection, page, pageSize }) =>
    toItemPage(
      await callBridge(() =>
        GetInventory(saveSessionID, characterID, containerSection, page, pageSize),
      ),
    ),

  getStorage: async ({ saveSessionID, characterID, containerSection, page, pageSize }) =>
    toItemPage(
      await callBridge(() =>
        GetStorage(saveSessionID, characterID, containerSection, page, pageSize),
      ),
    ),

  // The eleven arguments reach the bridge in the backend's own order. No safety
  // profile is sent: the backend reads the host setting itself, so a call from
  // here can never widen a limit or reveal a hidden resource.
  getOwnedItems: async ({
    saveSessionID,
    characterID,
    container,
    containerSection,
    search,
    category,
    favoritesOnly,
    favorites,
    sort,
    page,
    pageSize,
  }) =>
    toOwnedItemsPage(
      await callBridge(() =>
        GetOwnedItems(
          saveSessionID,
          characterID,
          container,
          containerSection,
          search,
          category,
          favoritesOnly,
          favorites.map(({ kind, key }) => ({ kind, key })),
          sort,
          page,
          pageSize,
        ),
      ),
    ),

  addItemsToContainers: async ({
    saveSessionID,
    characterID,
    items,
    confirmBanRisk,
    expectedRevision,
  }) =>
    toItemMutationReceipt(
      await callBridge(() =>
        AddItemsToContainers(
          saveSessionID,
          characterID,
          items.map((entry) => ({
            kind: entry.kind,
            key: entry.key,
            variantID: entry.variantID,
            inventoryQuantity: entry.inventoryQuantity,
            storageQuantity: entry.storageQuantity,
          })),
          confirmBanRisk,
          expectedRevision,
        ),
      ),
    ),

  moveOwnedItemsToStorage: async ({ saveSessionID, characterID, ownedItemIDs, expectedRevision }) =>
    toItemMutationReceipt(
      await callBridge(() =>
        MoveOwnedItemsToStorage(saveSessionID, characterID, [...ownedItemIDs], expectedRevision),
      ),
    ),

  moveOwnedItemsToInventory: async ({
    saveSessionID,
    characterID,
    ownedItemIDs,
    expectedRevision,
  }) =>
    toItemMutationReceipt(
      await callBridge(() =>
        MoveOwnedItemsToInventory(saveSessionID, characterID, [...ownedItemIDs], expectedRevision),
      ),
    ),

  removeOwnedItems: async ({ saveSessionID, characterID, ownedItemIDs, expectedRevision }) =>
    toItemMutationReceipt(
      await callBridge(() =>
        RemoveOwnedItems(saveSessionID, characterID, [...ownedItemIDs], expectedRevision),
      ),
    ),

  reorderInventoryItems: async ({
    saveSessionID,
    characterID,
    anchorOwnedItemID,
    groupOwnedItemIDs,
    targetPosition,
    expectedRevision,
  }) =>
    toItemMutationReceipt(
      await callBridge(() =>
        ReorderInventoryItems(
          saveSessionID,
          characterID,
          anchorOwnedItemID,
          [...groupOwnedItemIDs],
          targetPosition,
          expectedRevision,
        ),
      ),
    ),

  setOwnedItemQuantity: async ({
    saveSessionID,
    characterID,
    ownedItemID,
    quantity,
    expectedRevision,
  }) =>
    toItemMutationReceipt(
      await callBridge(() =>
        SetOwnedItemQuantity(saveSessionID, characterID, ownedItemID, quantity, expectedRevision),
      ),
    ),

  getSafetyProfile: async () => toSafetyProfileSettings(await callBridge(GetSafetyProfile)),

  // The thirteen World getters are read-only. The pair reaches the bridge in
  // the order the backend contract defines; nothing is trimmed or defaulted on
  // the way, and no World mutation is exposed here — the operation risk level,
  // the risk reason and the per-action capabilities a write needs are not part
  // of the current contract, so stage 9A offers no writer at all.
  getRegions: async ({ saveSessionID, characterID }) =>
    toWorldRegions(await callBridge(() => GetRegions(saveSessionID, characterID))),

  getMapRegions: async ({ saveSessionID, characterID }) =>
    toWorldMapRegions(await callBridge(() => GetMapRegions(saveSessionID, characterID))),

  getGraces: async ({ saveSessionID, characterID }) =>
    toWorldGraces(await callBridge(() => GetGraces(saveSessionID, characterID))),

  getBosses: async ({ saveSessionID, characterID }) =>
    toWorldBosses(await callBridge(() => GetBosses(saveSessionID, characterID))),

  // The questline selector is the endpoint's own input: the resource kind it
  // accepts, and the empty key that asks for every declared questline. Neither
  // value is a local interpretation of the answer.
  getQuests: async ({ saveSessionID, characterID }) =>
    toWorldQuests(await callBridge(() => GetQuests(saveSessionID, characterID, "quest", ""))),

  // The empty availability filter is the endpoint's own "every entry" input.
  // The workspace shows unlocked and locked entries together, so it never asks
  // for a pre-filtered subset and never derives one locally.
  getGestures: async ({ saveSessionID, characterID }) =>
    toWorldGestures(await callBridge(() => GetGestures(saveSessionID, characterID, ""))),

  getCookbooks: async ({ saveSessionID, characterID }) =>
    toWorldCookbooks(await callBridge(() => GetCookbooks(saveSessionID, characterID, ""))),

  getBellBearings: async ({ saveSessionID, characterID }) =>
    toWorldBellBearings(await callBridge(() => GetBellBearings(saveSessionID, characterID, ""))),

  getWhetblades: async ({ saveSessionID, characterID }) =>
    toWorldWhetblades(await callBridge(() => GetWhetblades(saveSessionID, characterID, ""))),

  getTutorials: async ({ saveSessionID, characterID }) =>
    toWorldTutorials(await callBridge(() => GetTutorials(saveSessionID, characterID, ""))),

  getSummoningPools: async ({ saveSessionID, characterID }) =>
    toWorldSummoningPools(await callBridge(() => GetSummoningPools(saveSessionID, characterID))),

  getColosseums: async ({ saveSessionID, characterID }) =>
    toWorldColosseums(await callBridge(() => GetColosseums(saveSessionID, characterID))),

  getSpectralSteedAttires: async ({ saveSessionID, characterID }) =>
    toWorldSpectralSteedAttires(
      await callBridge(() => GetSpectralSteedAttires(saveSessionID, characterID)),
    ),


  // The capability contract is read as reported and validated as a closed
  // vocabulary: an operation kind or a risk level this build does not know is an
  // unknown contract, so the whole answer is refused rather than partially
  // trusted. Nothing here fills in a missing capability or a missing risk.
  getWorldMutationCapabilities: async () =>
    toWorldMutationCapabilities(await callBridge(GetWorldMutationCapabilities)),

  // The eleven resource toggles below place the caller's own pair and value in
  // the positions their endpoint declares and add the expected revision. No
  // value is derived from the current view, and nothing is retried.
  setRegionUnlocked: async (request) =>
    toWorldMutationReceipt(await callBridge(() => SetRegionUnlocked(...toggleArguments(request)))),

  setMapRegionRevealed: async (request) =>
    toWorldMutationReceipt(
      await callBridge(() => SetMapRegionRevealed(...toggleArguments(request))),
    ),

  setGraceVisited: async (request) =>
    toWorldMutationReceipt(await callBridge(() => SetGraceVisited(...toggleArguments(request)))),

  setBossDefeated: async (request) =>
    toWorldMutationReceipt(await callBridge(() => SetBossDefeated(...toggleArguments(request)))),

  setGestureUnlocked: async (request) =>
    toWorldMutationReceipt(await callBridge(() => SetGestureUnlocked(...toggleArguments(request)))),

  setCookbookUnlocked: async (request) =>
    toWorldMutationReceipt(
      await callBridge(() => SetCookbookUnlocked(...toggleArguments(request))),
    ),

  setBellBearingUnlocked: async (request) =>
    toWorldMutationReceipt(
      await callBridge(() => SetBellBearingUnlocked(...toggleArguments(request))),
    ),

  setWhetbladeUnlocked: async (request) =>
    toWorldMutationReceipt(
      await callBridge(() => SetWhetbladeUnlocked(...toggleArguments(request))),
    ),

  setTutorialUnlocked: async (request) =>
    toWorldMutationReceipt(
      await callBridge(() => SetTutorialUnlocked(...toggleArguments(request))),
    ),

  setSummoningPoolActivated: async (request) =>
    toWorldMutationReceipt(
      await callBridge(() => SetSummoningPoolActivated(...toggleArguments(request))),
    ),

  setColosseumUnlocked: async (request) =>
    toWorldMutationReceipt(
      await callBridge(() => SetColosseumUnlocked(...toggleArguments(request))),
    ),

  // `removed` is the literal `true` of the port: the backend accepts no other
  // value, so the adapter has nothing to decide and passes what it received.
  setFogOfWarRemoved: async ({ saveSessionID, characterID, removed, expectedRevision }) =>
    toWorldMutationReceipt(
      await callBridge(() =>
        SetFogOfWarRemoved(saveSessionID, characterID, removed, expectedRevision),
      ),
    ),

  setQuestStep: async ({
    saveSessionID,
    characterID,
    questKind,
    questKey,
    stepKind,
    stepKey,
    expectedRevision,
  }) =>
    toWorldMutationReceipt(
      await callBridge(() =>
        SetQuestStep(
          saveSessionID,
          characterID,
          questKind,
          questKey,
          stepKind,
          stepKey,
          expectedRevision,
        ),
      ),
    ),

  setSpectralSteedAttire: async ({ saveSessionID, characterID, attireKey, expectedRevision }) =>
    toWorldMutationReceipt(
      await callBridge(() =>
        SetSpectralSteedAttire(saveSessionID, characterID, attireKey, expectedRevision),
      ),
    ),

  // One atomic call. The adapter never composes this out of a removal and a
  // selection, because a failure between them would leave the save wearing an
  // appearance whose item is gone.
  lockAllSpectralSteedAttires: async ({ saveSessionID, characterID, expectedRevision }) =>
    toWorldMutationReceipt(
      await callBridge(() =>
        LockAllSpectralSteedAttires(saveSessionID, characterID, expectedRevision),
      ),
    ),


  // The value reaches the bridge exactly as received: which profiles exist and
  // how an unknown one is rejected are the backend's contract.
  setSafetyProfile: async (safetyProfile) =>
    toSafetyProfileSettings(await callBridge(() => SetSafetyProfile(safetyProfile))),

  // The pair reaches the bridge in the order the backend contract defines; the
  // grouped request only protects the caller from transposing them. Neither
  // value is trimmed, defaulted or clamped on the way.
  getCharacterLoadout: async ({ saveSessionID, characterID }) =>
    toCharacterLoadout(await callBridge(() => GetCharacterLoadout(saveSessionID, characterID))),

  getEquipment: async ({ saveSessionID, characterID }) =>
    toCharacterEquipment(await callBridge(() => GetEquipment(saveSessionID, characterID))),

  getQuickItems: async ({ saveSessionID, characterID }) =>
    toCharacterQuickItems(await callBridge(() => GetQuickItems(saveSessionID, characterID))),

  getPouchItems: async ({ saveSessionID, characterID }) =>
    toCharacterPouchItems(await callBridge(() => GetPouchItems(saveSessionID, characterID))),

  getPhysickMixture: async ({ saveSessionID, characterID }) =>
    toCharacterPhysickMixture(
      await callBridge(() => GetPhysickMixture(saveSessionID, characterID)),
    ),

  getEquippedSpells: async ({ saveSessionID, characterID }) =>
    toCharacterEquippedSpells(
      await callBridge(() => GetEquippedSpells(saveSessionID, characterID)),
    ),

  // The six arguments reach the bridge in the order the backend contract
  // defines; the grouped request only protects the caller from transposing
  // them. No safety profile is sent: the backend reads the host setting itself.
  getEquipmentCandidates: async ({
    saveSessionID,
    characterID,
    slotType,
    search,
    page,
    pageSize,
  }) =>
    toEquipmentCandidatesPage(
      await callBridge(() =>
        GetEquipmentCandidates(saveSessionID, characterID, slotType, search, page, pageSize),
      ),
    ),

  // Every Equipment mutation forwards the complete group in the backend's own
  // order. The arrays are copied so a generated transport object never becomes
  // application state, and `null` stays the backend's own empty position.
  setEquippedArmaments: async ({ saveSessionID, characterID, slotAssignments, expectedRevision }) =>
    toEquipmentMutationReceipt(
      await callBridge(() =>
        SetEquippedArmaments(saveSessionID, characterID, [...slotAssignments], expectedRevision),
      ),
    ),

  setEquippedArmor: async ({ saveSessionID, characterID, slotAssignments, expectedRevision }) =>
    toEquipmentMutationReceipt(
      await callBridge(() =>
        SetEquippedArmor(saveSessionID, characterID, [...slotAssignments], expectedRevision),
      ),
    ),

  setEquippedTalismans: async ({
    saveSessionID,
    characterID,
    orderedOwnedItemIDs,
    expectedRevision,
  }) =>
    toEquipmentMutationReceipt(
      await callBridge(() =>
        SetEquippedTalismans(
          saveSessionID,
          characterID,
          [...orderedOwnedItemIDs],
          expectedRevision,
        ),
      ),
    ),

  setEquippedSpells: async ({ saveSessionID, characterID, orderedResources, expectedRevision }) =>
    toEquipmentMutationReceipt(
      await callBridge(() =>
        SetEquippedSpells(
          saveSessionID,
          characterID,
          orderedResources.map(({ kind, key }) => ({ kind, key })),
          expectedRevision,
        ),
      ),
    ),

  setPhysickMixture: async ({
    saveSessionID,
    characterID,
    crystalTearResources,
    expectedRevision,
  }) =>
    toEquipmentMutationReceipt(
      await callBridge(() =>
        SetPhysickMixture(
          saveSessionID,
          characterID,
          // The backend position is nullable and the generated signature is not,
          // so the empty position is asserted here rather than replaced by a
          // placeholder: `null` is what clears exactly that Physick position.
          crystalTearResources.map((entry) =>
            entry === null ? null : { kind: entry.kind, key: entry.key },
          ) as schema.ResourceRef[],
          expectedRevision,
        ),
      ),
    ),

  setPouchItems: async ({ saveSessionID, characterID, slotAssignments, expectedRevision }) =>
    toEquipmentMutationReceipt(
      await callBridge(() =>
        SetPouchItems(saveSessionID, characterID, [...slotAssignments], expectedRevision),
      ),
    ),

  setQuickItems: async ({ saveSessionID, characterID, slotAssignments, expectedRevision }) =>
    toEquipmentMutationReceipt(
      await callBridge(() =>
        SetQuickItems(saveSessionID, characterID, [...slotAssignments], expectedRevision),
      ),
    ),

  // The seven arguments reach the bridge in the order the backend contract
  // defines; the grouped request only protects the caller from transposing
  // them. No filter is trimmed, recased or dropped on the way.
  getItemDatabase: async ({
    family,
    category,
    search,
    favoritesOnly,
    favorites,
    sort,
    page,
    pageSize,
  }) =>
    toCatalogItemDatabasePage(
      await callBridge(() =>
        GetItemDatabase(
          family,
          category,
          search,
          favoritesOnly,
          favorites.map(({ kind, key }) => ({ kind, key })),
          sort,
          page,
          pageSize,
        ),
      ),
    ),

  getResources: async ({ resourceType, family, capability, endpointID, search, page, pageSize }) =>
    toCatalogResourcesPage(
      await callBridge(() =>
        GetResources(resourceType, family, capability, endpointID, search, page, pageSize),
      ),
    ),

  getResourcePresentationSummaries: async (identities) =>
    toCatalogResourcePresentationSummaries(
      await callBridge(() =>
        GetResourcePresentationSummaries(identities.map(({ kind, key }) => ({ kind, key }))),
      ),
    ),

  // The pair reaches the bridge exactly as received. Nothing is trimmed,
  // recased or retried under another kind: the identity contract is the
  // backend's, and every rejection of it is its own.
  getResource: async ({ kind, key }) =>
    toCatalogResourceDetail(await callBridge(() => GetResource(kind, key))),

  // The same pair, forwarded in the same order and just as unchanged. That only
  // the item kind carries variants, and that an unknown or non-item identity is
  // a rejection, is the backend's contract and is never anticipated here.
  getItemVariants: async ({ kind, key }) =>
    toCatalogItemVariants(await callBridge(() => GetItemVariants(kind, key))),

  getNetworkSettings: async (saveSessionID) =>
    toNetworkSettingsSnapshot(await callBridge(() => GetNetworkSettings(saveSessionID))),

  getNetworkPresets: async (presetID) =>
    toNetworkPresetsResult(await callBridge(() => GetNetworkPresets(presetID ?? ""))),

  setNetworkSettings: async (saveSessionID, networkSettings, expectedRevision) =>
    toSetNetworkSettingsResult(
      await callBridge(() =>
        SetNetworkSettings(saveSessionID, networkSettings, expectedRevision),
      ),
    ),

  getHostSettings: async () => toHostSettings(await callBridge(GetHostSettings)),

  setHostSettings: async ({ skipReviewForNormalRisk, remoteBackupPolicy }) =>
    toHostSettings(
      await callBridge(() =>
        SetHostSettings(skipReviewForNormalRisk, remoteBackupPolicy),
      ),
    ),

  // The identifier is the whole argument. There is deliberately no way to state
  // a path here: the backend resolves the location from its own state directory.
  openHostLocation: async (location) => {
    await callBridge(() => OpenHostLocation(location));
  },

  // A cancelled Save As dialog is an ordinary outcome: the backend writes
  // nothing and reports no records, which is what `exported: false` states.
  exportDiagnosticReport: async (saveSessionID) => {
    const result = await callBridge(() => ExportDiagnosticReport(saveSessionID ?? ""));
    return { exported: result.exported, recordCount: result.recordCount };
  },

  getProjectLinks: async () => {
    const result = await callBridge(GetProjectLinks);
    return result.links.map((link): ProjectLink => ({ id: link.id, url: link.url }));
  },

  openProjectLink: async (linkID) => {
    await callBridge(() => OpenProjectLink(linkID));
  },

  checkForUpdates: async (): Promise<UpdateCheck> => {
    const result = await callBridge(CheckForUpdates);
    return {
      status: result.status,
      currentVersion: result.currentVersion,
      latestVersion: result.latestVersion,
      releaseURL: result.releaseURL,
      publishedAt: result.publishedAt,
      comparisonPossible: result.comparisonPossible,
    };
  },

  getBuildTemplates: async ({ search, tags, page, pageSize }) =>
    toBuildTemplatePage(
      await callBridge(() => GetBuildTemplates(search, [...tags], page, pageSize)),
    ),

  getBuildTemplatePreview: async ({ saveSessionID, characterID, templateID, overrides }) =>
    toBuildTemplatePreview(
      await callBridge(() =>
        GetBuildTemplatePreview(
          new templates.GetBuildTemplatePreviewRequest({
            saveSessionID,
            characterID,
            templateID,
            options: toApplyOptions(overrides),
          }),
        ),
      ),
    ),

  // The result is the shared save mutation receipt, carried through unchanged
  // so applying a template refreshes the session exactly like every other save
  // mutation.
  applyBuildTemplate: async ({
    saveSessionID,
    characterID,
    templateID,
    expectedRevision,
    overrides,
  }): Promise<TemplateMutationReceipt> => {
    const result = await callBridge(() =>
      ApplyBuildTemplate(
        new templates.ApplyBuildTemplateRequest({
          saveSessionID,
          characterID,
          templateID,
          expectedRevision,
          options: toApplyOptions(overrides),
        }),
      ),
    );
    return toMutationReceipt(result);
  },

  createBuildTemplate: async ({ saveSessionID, sourceCharacterID, name, description, tags }) => {
    const result = await callBridge(() =>
      CreateBuildTemplate(
        saveSessionID,
        sourceCharacterID,
        name,
        description ?? "",
        tags === undefined ? [] : [...tags],
      ),
    );
    return { templateID: result.templateID };
  },

  updateBuildTemplate: async ({ templateID, templateRevision, name, description, tags }) => {
    const result = await callBridge(() =>
      UpdateBuildTemplate(
        templateID,
        new templates.UpdateBuildTemplateRequest({
          templateRevision,
          metadata: { name, description, tags: tags === undefined ? undefined : [...tags] },
        }),
      ),
    );
    return { templateID: result.templateID };
  },

  deleteBuildTemplate: async ({ templateID, templateRevision }) => {
    const result = await callBridge(() => DeleteBuildTemplate(templateID, templateRevision));
    return { templateID: result.templateID };
  },

  // Cancelling the document dialog is the backend's empty identifier, so it
  // survives the boundary as an undefined value rather than becoming an error.
  importBuildTemplate: async () => {
    const result = await callBridge(ImportBuildTemplate);
    return { templateID: result.templateID === "" ? undefined : result.templateID };
  },

  // The host event carries the backend's own progress shape. It is projected
  // field by field, so a malformed emission cannot leak an unexpected object
  // into the interface.
  subscribeDeploymentProgress: (listener) =>
    EventsOn("deployment.progress", (progress: DeploymentProgress) =>
      listener({
        operationID: progress.operationID,
        targetID: progress.targetID,
        stage: progress.stage,
        percent: progress.percent,
        elapsedMS: progress.elapsedMS,
        finished: progress.finished,
      }),
    ),

  getDeploymentTargets: async () => toDeploymentTargets(await callBridge(GetDeploymentTargets)),

  createDeploymentTarget: async (input) =>
    toDeploymentTargets(
      await callBridge(() => CreateDeploymentTarget(input as deployment.TargetInput)),
    ),

  updateDeploymentTarget: async (input) =>
    toDeploymentTargets(
      await callBridge(() => UpdateDeploymentTarget(input as deployment.TargetInput)),
    ),

  deleteDeploymentTarget: async (targetID) =>
    toDeploymentTargets(await callBridge(() => DeleteDeploymentTarget(targetID))),

  testDeploymentTarget: async (targetID): Promise<TargetTestResult> => {
    const result = await callBridge(() => TestDeploymentTarget(targetID));
    return {
      targetID: result.targetID,
      reachable: result.reachable,
      hostKeyTrusted: result.hostKeyTrusted,
      gameStatus: result.gameStatus,
      saveExists: result.saveExists,
    };
  },

  forgetDeploymentHostKey: async (targetID) =>
    toDeploymentTargets(await callBridge(() => ForgetDeploymentHostKey(targetID))),

  getDeploymentGameStatus: async (targetID) => {
    const result = await callBridge(() => GetDeploymentGameStatus(targetID));
    return result.gameStatus;
  },

  launchTargetGame: async (targetID) =>
    toCommandOutcome(await callBridge(() => LaunchTargetGame(targetID))),

  closeTargetGame: async (targetID) =>
    toCommandOutcome(await callBridge(() => CloseTargetGame(targetID))),

  deployToTarget: async (request) =>
    toOperationResult(
      await callBridge(() => DeployToTarget(request as unknown as deployment.DeployRequest)),
    ),

  downloadFromTarget: async (request) =>
    toOperationResult(
      await callBridge(() => DownloadFromTarget(request as unknown as deployment.DownloadRequest)),
    ),

  cancelDeploymentOperation: async (operationID) => {
    await callBridge(() => CancelDeploymentOperation(operationID));
  },

  getTargetBackups: async (targetID) =>
    toTargetBackups(await callBridge(() => GetTargetBackups(targetID))),

  createTargetBackup: async ({ targetID, tags, description }) =>
    toTargetBackups(await callBridge(() => CreateTargetBackup(targetID, [...tags], description))),

  activateTargetBackup: async ({
    operationID,
    targetID,
    backupID,
    continueWithUnknownGameStatus,
    confirmRemoteBackup,
  }) => {
    const result = await callBridge(() =>
      ActivateTargetBackup(
        operationID,
        targetID,
        backupID,
        continueWithUnknownGameStatus ?? false,
        confirmRemoteBackup ?? false,
      ),
    );
    return {
      operation: toOperationResult(result.operation),
      backups: toTargetBackups(result.backups),
    };
  },

  clearActiveTargetBackup: async (targetID) =>
    toTargetBackups(await callBridge(() => ClearActiveTargetBackup(targetID))),

  updateTargetBackup: async ({ targetID, backupID, tags, description }) =>
    toTargetBackups(
      await callBridge(() => UpdateTargetBackup(targetID, backupID, [...tags], description)),
    ),

  deleteTargetBackup: async ({ targetID, backupID }) =>
    toTargetBackups(await callBridge(() => DeleteTargetBackup(targetID, backupID))),

  // Cancelling the Save As dialog writes nothing and reports an empty target,
  // which the port carries as an undefined destination rather than an error.
  downloadTargetBackup: async ({ targetID, backupID }) => {
    const result = await callBridge(() => DownloadTargetBackup(targetID, backupID, ""));
    return { target: result.target === "" ? undefined : result.target };
  },
};
