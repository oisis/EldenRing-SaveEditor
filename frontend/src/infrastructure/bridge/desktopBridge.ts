// The ONLY module allowed to import the generated Wails bindings. Everything
// above this layer depends on the application ports, never on generated code or
// on `window.go`.

import {
  AddItemsToContainers,
  ApplyAppearancePreset,
  ApplyFavoritePreset,
  ClearRecentFiles,
  CloseSave,
  DeleteFavoritePreset,
  DiscardChanges,
  DiscardRecoveryJournal,
  ExportRecoveryJournal,
  GetAppearancePresets,
  GetApplicationInfo,
  GetCharacterLoadout,
  GetCharacterProfile,
  GetCharacterStats,
  GetEquipment,
  GetEquipmentCandidates,
  GetEquippedSpells,
  GetFavoritePresets,
  GetInventory,
  GetItemDatabase,
  GetItemVariants,
  GetLoadedSave,
  GetOperationHistory,
  GetOwnedItems,
  GetPhysickMixture,
  GetPouchItems,
  GetQuickItems,
  GetRecentFiles,
  GetRecoveryJournal,
  GetRecoveryJournals,
  GetResource,
  GetResourcePresentationSummaries,
  GetResources,
  GetSafetyProfile,
  GetSaveCharacters,
  GetSaveLifecycleSettings,
  GetSaveValidationReport,
  GetStorage,
  LoadSave,
  MoveOwnedItemsToInventory,
  MoveOwnedItemsToStorage,
  QuitApplication,
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
  SetCharacterStartingClass,
  SetCharacterStats,
  SetEquippedArmaments,
  SetEquippedArmor,
  SetEquippedSpells,
  SetEquippedTalismans,
  SetFavoritePreset,
  SetOwnedItemQuantity,
  SetPhysickMixture,
  SetPouchItems,
  SetQuickItems,
  SetSafetyProfile,
  SetSaveLifecycleSettings,
  UndoLastOperation,
  ValidateReviewChanges,
} from "../../../wailsjs/go/desktop/Bridge";
import { saveengine, type schema } from "../../../wailsjs/go/models";
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
  CharacterPort,
  CharacterProfile,
  CharacterStats,
  SaveCharacters,
} from "../../application/character/characterPort";
import type {
  DiagnosticsPort,
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
import type { SafetyProfileSettings, SettingsPort } from "../../application/settings/settingsPort";
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
  SettingsPort &
  CatalogPort = {
  getApplicationInfo: async (): Promise<ApplicationInfo> => {
    const result = await callBridge(GetApplicationInfo);

    return {
      version: result.applicationVersion,
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
};
