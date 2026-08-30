// The ONLY module allowed to import the generated Wails bindings. Everything
// above this layer depends on the application ports, never on generated code or
// on `window.go`.

import {
  CloseSave,
  GetApplicationInfo,
  GetCharacterProfile,
  GetCharacterStats,
  GetEquipment,
  GetEquippedSpells,
  GetInventory,
  GetItemVariants,
  GetLoadedSave,
  GetPhysickMixture,
  GetPouchItems,
  GetQuickItems,
  GetResource,
  GetResourcePresentationSummaries,
  GetResources,
  GetSaveCharacters,
  GetStorage,
  LoadSave,
} from "../../../wailsjs/go/desktop/Bridge";
import type { schema } from "../../../wailsjs/go/models";
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
import type {
  CharacterPort,
  CharacterProfile,
  CharacterStats,
  SaveCharacters,
} from "../../application/character/characterPort";
import type {
  CharacterEquipment,
  CharacterEquippedSpells,
  CharacterPhysickMixture,
  CharacterPouchItems,
  CharacterQuickItems,
  EquipmentPort,
} from "../../application/equipment/equipmentPort";
import type { ItemPage, ItemsPort } from "../../application/items/itemsPort";
import type { SaveSession, SaveSessionPort } from "../../application/save-session/saveSessionPort";

/**
 * The stable code a failed bridge call is reported with. The transport error is
 * deliberately dropped here: a Wails rejection can carry a Go error string or a
 * runtime stack, and neither may reach the interface. The UI maps this code to
 * a localized message and never renders the code itself.
 *
 * The code says that the call failed and nothing more. Classifying a domain
 * failure would mean reading the rejection text, which is exactly what this
 * boundary refuses to do; a structured backend error contract is what a finer
 * distinction has to come from.
 */
export const bridgeFailureCode = "bridge_call_failed";

async function callBridge<T>(call: () => Promise<T>): Promise<T> {
  try {
    return await call();
  } catch {
    throw new Error(bridgeFailureCode);
  }
}

/** Projects the generated session result onto the application port shape. */
function toSaveSession(result: Awaited<ReturnType<typeof GetLoadedSave>>): SaveSession {
  return {
    saveSessionID: result.saveSessionID,
    platform: result.platform,
    format: result.format,
    unsavedChanges: result.unsavedChanges,
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
    characterID: result.characterID,
    active: result.active,
    slots: [...result.slots],
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
 * The single adapter behind every application port. A second, parallel
 * adaptation layer would give the generated bindings a second way into the
 * application, so all ports are fulfilled here.
 */
export const wailsDesktopBridge: ApplicationInfoPort &
  SaveSessionPort &
  CharacterPort &
  ItemsPort &
  EquipmentPort &
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

  loadSave: async (source, expectedPlatform) =>
    toSaveSession(await callBridge(() => LoadSave(source, expectedPlatform))),

  getLoadedSave: async (saveSessionID) =>
    toSaveSession(await callBridge(() => GetLoadedSave(saveSessionID))),

  closeSave: async (saveSessionID) => {
    await callBridge(() => CloseSave(saveSessionID));
  },

  getSaveCharacters: async (saveSessionID): Promise<SaveCharacters> => {
    const result = await callBridge(() => GetSaveCharacters(saveSessionID));

    return {
      saveSessionID: result.saveSessionID,
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

  // The pair reaches the bridge in the order the backend contract defines; the
  // grouped request only protects the caller from transposing them. Neither
  // value is trimmed, defaulted or clamped on the way.
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

  // The seven arguments reach the bridge in the order the backend contract
  // defines; the grouped request only protects the caller from transposing
  // them. No filter is trimmed, recased or dropped on the way.
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
