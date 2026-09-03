import { I18nProvider } from "@lingui/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { type RenderResult, render } from "@testing-library/react";
import type { ReactNode } from "react";
import { AppearancePortProvider } from "../application/appearance/appearanceClient";
import type {
  AppearancePort,
  AppearancePresetSummary,
} from "../application/appearance/appearancePort";
import { ApplicationInfoPortProvider } from "../application/application-info/applicationInfoClient";
import type {
  ApplicationInfo,
  ApplicationInfoPort,
} from "../application/application-info/applicationInfoPort";
import { CatalogPortProvider } from "../application/catalog/catalogClient";
import type {
  CatalogFact,
  CatalogItemDatabasePage,
  CatalogItemVariantsResult,
  CatalogPort,
  CatalogResourceDetail,
  CatalogResourcesPage,
} from "../application/catalog/catalogPort";
import { CharacterPortProvider } from "../application/character/characterClient";
import type {
  CharacterPort,
  CharacterProfile,
  CharacterStats,
  SaveCharacters,
} from "../application/character/characterPort";
import { DiagnosticsPortProvider } from "../application/diagnostics/diagnosticsClient";
import type {
  DiagnosticsPort,
  SaveValidationReport,
} from "../application/diagnostics/diagnosticsPort";
import { EquipmentPortProvider } from "../application/equipment/equipmentClient";
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
} from "../application/equipment/equipmentPort";
import { FavoritesPortProvider } from "../application/favorites/favoritesClient";
import type { FavoritesPort, SaveFavoritePresets } from "../application/favorites/favoritesPort";
import { ItemsPortProvider } from "../application/items/itemsClient";
import type {
  ItemMutationReceipt,
  ItemPage,
  ItemsPort,
  OwnedItemRow,
  OwnedItemsPage,
} from "../application/items/itemsPort";
import { ItemPreferencesProvider } from "../application/preferences/itemPreferences";
import { SaveSessionPortProvider } from "../application/save-session/saveSessionClient";
import type {
  MutationReceipt,
  SaveSession,
  SaveSessionPort,
} from "../application/save-session/saveSessionPort";
import { SettingsPortProvider } from "../application/settings/settingsClient";
import type { SafetyProfileSettings, SettingsPort } from "../application/settings/settingsPort";
import { NetworkPortProvider } from "../application/network/networkClient";
import type {
  NetworkPort,
  NetworkPresetsResult,
  NetworkSettingsSnapshot,
  SetNetworkSettingsResult,
} from "../application/network/networkPort";
import { WorldPortProvider } from "../application/world/worldClient";
import type {
  WorldMutationReceipt,
  WorldPort,
} from "../application/world/worldPort";
import { activateLocale, i18n, type Locale } from "../i18n/i18n";

/**
 * Components and hooks are exercised through the application ports, never
 * through a mock of the generated desktop bindings: such a test must fail when
 * the application contract breaks, not when the transport detail changes.
 */
export const stubApplicationInfo: ApplicationInfo = {
  version: "2.0.0-test",
  schemas: [{ name: "game_catalog", minimumVersion: 1, currentVersion: 16 }],
  capabilities: ["catalog_read"],
};

export const stubSaveSession: SaveSession = {
  saveSessionID: "session-1",
  platform: "pc",
  format: "sl2_v2",
  // Deliberately a path with spaces and mixed case: nothing above the port may
  // normalise, shorten or rebuild it.
  sourcePath: "/Users/Tarnished/Elden Ring/ER0000.sl2",
  sourceKind: "local",
  saveRevision: "0",
  unsavedChanges: false,
  eventSequence: "0",
};

export const stubSaveCharacters: SaveCharacters = {
  saveSessionID: "session-1",
  saveRevision: "0",
  characters: [
    { characterID: 0, active: true, name: "Tarnished", level: 150 },
    { characterID: 1, active: false, name: "", level: 0 },
  ],
};

export const stubCharacterProfile: CharacterProfile = {
  saveSessionID: "session-1",
  saveRevision: "0",
  characterID: 0,
  active: true,
  name: "Tarnished",
  level: 150,
  startingClassID: 3,
  gender: 1,
  secondsPlayed: 123456,
};

export const stubCharacterStats: CharacterStats = {
  saveSessionID: "session-1",
  saveRevision: "0",
  characterID: 0,
  active: true,
  vigor: 40,
  mind: 20,
  endurance: 25,
  strength: 50,
  dexterity: 18,
  intelligence: 9,
  faith: 12,
  arcane: 7,
  level: 150,
  hp: 1450,
  maxHP: 1900,
  baseMaxHP: 1900,
  fp: 200,
  maxFP: 220,
  baseMaxFP: 220,
  sp: 130,
  maxSP: 130,
  baseMaxSP: 130,
  runes: 250000,
  soulMemory: 1750000,
};

export const stubInventoryPage: ItemPage = {
  saveSessionID: "session-1",
  saveRevision: "0",
  characterID: 0,
  active: true,
  records: [
    {
      ownedItemID: "owned-1",
      kind: "item",
      key: "weapon/uchigatana",
      gameID: 0x00bb8000,
      containerSection: "common",
      physicalIndex: 3,
      gaItemHandle: 0x8000000a,
      quantity: 1,
      acquisitionIndex: 42,
    },
  ],
  total: 1,
  page: 1,
  pageSize: 30,
};

export const stubStoragePage: ItemPage = {
  ...stubInventoryPage,
  records: [{ ...stubInventoryPage.records[0], ownedItemID: "owned-2", physicalIndex: 7 }],
};

/**
 * The catalog stub carries a named row and a nameless one: an unknown name is
 * the empty string in the backend contract, and nothing above the port may turn
 * it into a key or a placeholder.
 */
export const stubCatalogPage: CatalogResourcesPage = {
  resources: [
    { kind: "item", key: "weapon/uchigatana", family: "weapon", name: "Uchigatana" },
    { kind: "item", key: "goods/unnamed", family: "", name: "" },
  ],
  total: 2,
  page: 1,
  pageSize: 50,
};

/** A fact the backend resolved, with a complete provenance record. */
function knownFact<T>(value: T): CatalogFact<T> {
  return {
    known: true,
    value,
    provenance: {
      source: "legacy_db_data",
      method: "regulation_row",
      table: "EquipParamWeapon",
      row: "1000000",
      field: "maxLevel",
    },
  };
}

/**
 * A fact the backend could not resolve. Its raw value is carried as reported,
 * and the provenance the backend still supplies stays complete: an unknown fact
 * is not a missing one.
 */
function unknownFact<T>(value: T): CatalogFact<T> {
  return {
    known: false,
    value,
    provenance: { source: "legacy_db_data", method: "unresolved", table: "", row: "", field: "" },
  };
}

/**
 * The detail stub deliberately mixes the shapes the mapping has to survive: a
 * resolved fact, an unknown one keeping its raw value, empty strings, zeros,
 * absent optional facts and a capability the backend reports without rules.
 */
export const stubCatalogResourceDetail: CatalogResourceDetail = {
  kind: "item",
  key: "000F4240",
  item: {
    gameID: knownFact(1000000),
    family: knownFact("weapon"),
    category: knownFact("dagger"),
    subcategory: unknownFact(""),
    presentation: {
      name: knownFact("Dagger"),
      caption: knownFact("A small dagger."),
      description: unknownFact(""),
      location: unknownFact(""),
      iconPath: knownFact("MENU_Knowledge_00100.png"),
    },
    storage: {
      recordMode: knownFact("separate_instances"),
      maxInventory: knownFact(600),
      safeModeMaxInventory: knownFact(99),
      maxInventorySFV: null,
      maxStorage: knownFact(600),
      safeModeMaxStorage: null,
      maxStorageSFV: null,
    },
    safety: {
      cutContent: knownFact(false),
      banRisk: knownFact(false),
      dlc: knownFact(false),
      noDatabase: unknownFact(false),
      scalesWithNG: knownFact(false),
      preOrder: knownFact(false),
    },
    capabilities: {
      upgrade: {
        known: true,
        enabled: true,
        rules: { model: "standard", maxLevel: 25, maxLevelSFV: null },
        provenance: knownFact(0).provenance,
      },
      infusion: {
        known: true,
        enabled: true,
        rules: { allowedAffinities: ["standard", "heavy"] },
        provenance: knownFact(0).provenance,
      },
      ashOfWarMount: {
        known: true,
        enabled: true,
        rules: { mode: "custom", weaponType: "dagger", compatibilityBit: 1 },
        provenance: knownFact(0).provenance,
      },
      stack: {
        known: true,
        enabled: false,
        rules: null,
        provenance: unknownFact(0).provenance,
      },
      equipment: {
        known: true,
        enabled: true,
        rules: { allowedSlots: ["left_hand", "right_hand"] },
        provenance: knownFact(0).provenance,
      },
    },
  },
};

/**
 * The variant stub keeps the catalog order it is written in and mixes the
 * shapes the mapping has to survive: a resolved fact, an unknown one keeping
 * its raw zero and empty string, and an affinity that is empty for a pure
 * upgrade variant.
 */
export const stubCatalogItemVariants: CatalogItemVariantsResult = {
  variants: [
    {
      gameID: knownFact(1000100),
      kind: knownFact("affinity"),
      affinity: knownFact("heavy"),
      upgradeLevel: knownFact(0),
      sourceRowID: knownFact(1000100),
    },
    {
      gameID: knownFact(1000001),
      kind: knownFact("upgrade"),
      affinity: unknownFact(""),
      upgradeLevel: knownFact(1),
      sourceRowID: unknownFact(0),
    },
  ],
};

/**
 * The Item Database stub mixes the shapes the mapping has to survive: a named
 * row with an icon and a known identifier, and a nameless one whose identifier
 * the catalog does not know. One row is marked ban risk, so the confirmation
 * path has something real to act on.
 */
export const stubItemDatabasePage: CatalogItemDatabasePage = {
  safetyProfile: "safe",
  resources: [
    {
      kind: "item",
      key: "weapon/uchigatana",
      gameID: 0x00bb8000,
      gameIDKnown: true,
      family: "weapon",
      category: "melee_armaments",
      subcategory: "katana",
      name: "Uchigatana",
      iconPath: "assets/icons/items/uchigatana.png",
      banRisk: false,
      cutContent: false,
      dlc: false,
      preOrder: false,
    },
    {
      kind: "item",
      key: "goods/unnamed",
      gameID: 0,
      gameIDKnown: false,
      family: "",
      category: "",
      subcategory: "",
      name: "",
      iconPath: "",
      banRisk: true,
      cutContent: false,
      dlc: false,
      preOrder: false,
    },
  ],
  categories: [{ category: "melee_armaments", count: 1 }],
  total: 2,
  page: 1,
  pageSize: 20,
};

export function makeCatalogPort(overrides: Partial<CatalogPort> = {}): CatalogPort {
  return {
    getItemDatabase: () => Promise.resolve(stubItemDatabasePage),
    getResources: () => Promise.resolve(stubCatalogPage),
    getResourcePresentationSummaries: (identities) =>
      Promise.resolve({
        resources: identities.map(({ kind, key }) => ({ kind, key, name: "", iconPath: "" })),
      }),
    getResource: () => Promise.resolve(stubCatalogResourceDetail),
    getItemVariants: () => Promise.resolve(stubCatalogItemVariants),
    ...overrides,
  };
}

/**
 * One authoritative Inventory row. Every action the backend can allow is true
 * here, so a test that expects an action to be hidden has to make the backend
 * say so rather than rely on a stub that never offered it.
 */
export const stubOwnedInventoryRow: OwnedItemRow = {
  ownedItemID: "owned-1",
  kind: "item",
  key: "weapon/uchigatana",
  gameID: 0x00bb8000,
  container: "inventory",
  containerSection: "common",
  physicalIndex: 3,
  acquisitionIndex: 42,
  orderPosition: 0,
  orderPositionKnown: true,
  quantity: 1,
  maxQuantity: 99,
  maxQuantityKnown: true,
  family: "weapon",
  category: "melee_armaments",
  subcategory: "katana",
  name: "Uchigatana",
  iconPath: "assets/icons/items/uchigatana.png",
  recordMode: "quantity_stack",
  banRisk: false,
  cutContent: false,
  dlc: false,
  preOrder: false,
  actions: {
    moveToStorage: true,
    moveToInventory: false,
    remove: true,
    setQuantity: true,
    reorder: true,
  },
};

export const stubOwnedStorageRow: OwnedItemRow = {
  ...stubOwnedInventoryRow,
  ownedItemID: "owned-2",
  container: "storage",
  physicalIndex: 7,
  orderPosition: 0,
  orderPositionKnown: false,
  actions: {
    moveToStorage: false,
    moveToInventory: true,
    remove: true,
    setQuantity: true,
    reorder: false,
  },
};

export const stubOwnedInventoryPage: OwnedItemsPage = {
  saveSessionID: "session-1",
  saveRevision: "0",
  characterID: 0,
  active: true,
  safetyProfile: "safe",
  container: "inventory",
  records: [stubOwnedInventoryRow],
  categories: [{ category: "melee_armaments", count: 1 }],
  total: 1,
  page: 1,
  pageSize: 30,
};

export const stubOwnedStoragePage: OwnedItemsPage = {
  ...stubOwnedInventoryPage,
  container: "storage",
  records: [stubOwnedStorageRow],
};

/** The receipt every mutation stub reports, in the shared backend shape. */
export function stubItemMutationReceipt(
  operationKind: string,
  changedScopes: ItemMutationReceipt["changedScopes"],
): ItemMutationReceipt {
  return {
    operationID: `operation-${operationKind}`,
    operationKind,
    saveSessionID: "session-1",
    saveRevision: "1",
    changedScopes,
  };
}

export function makeItemsPort(overrides: Partial<ItemsPort> = {}): ItemsPort {
  return {
    getInventory: () => Promise.resolve(stubInventoryPage),
    getStorage: () => Promise.resolve(stubStoragePage),
    getOwnedItems: ({ container }) =>
      Promise.resolve(container === "storage" ? stubOwnedStoragePage : stubOwnedInventoryPage),
    addItemsToContainers: () =>
      Promise.resolve(
        stubItemMutationReceipt("add_items_to_containers", [
          "save.session",
          "inventory",
          "diagnostics.report",
        ]),
      ),
    moveOwnedItemsToStorage: () =>
      Promise.resolve(
        stubItemMutationReceipt("move_owned_items_to_storage", [
          "save.session",
          "inventory",
          "storage",
          "diagnostics.report",
        ]),
      ),
    moveOwnedItemsToInventory: () =>
      Promise.resolve(
        stubItemMutationReceipt("move_owned_items_to_inventory", [
          "save.session",
          "inventory",
          "storage",
          "diagnostics.report",
        ]),
      ),
    removeOwnedItems: () =>
      Promise.resolve(
        stubItemMutationReceipt("remove_owned_items", [
          "save.session",
          "inventory",
          "storage",
          "diagnostics.report",
        ]),
      ),
    reorderInventoryItems: () =>
      Promise.resolve(
        stubItemMutationReceipt("reorder_inventory_items", [
          "save.session",
          "inventory",
          "diagnostics.report",
        ]),
      ),
    setOwnedItemQuantity: () =>
      Promise.resolve(
        stubItemMutationReceipt("set_owned_item_quantity", [
          "save.session",
          "inventory",
          "storage",
          "equipment.loadout",
          "diagnostics.report",
        ]),
      ),
    ...overrides,
  };
}

/** The product default: the safest profile, with the closed vocabulary. */
export const stubSafetyProfile: SafetyProfileSettings = {
  safetyProfile: "safe",
  availableProfiles: ["safe", "expanded_limits", "chaos"],
  defaultProfile: "safe",
};

export function makeSettingsPort(overrides: Partial<SettingsPort> = {}): SettingsPort {
  return {
    getSafetyProfile: () => Promise.resolve(stubSafetyProfile),
    setSafetyProfile: (safetyProfile) => Promise.resolve({ ...stubSafetyProfile, safetyProfile }),
    ...overrides,
  };
}

/**
 * The equipment stubs are deliberately full length and full of boundary values,
 * so a mapping that dropped, reordered, clamped or reinterpreted anything fails
 * instead of looking plausible: every array has its exact backend length, and
 * each one carries a zero, the maximum uint32 and ordinary values around them.
 */
export const stubCharacterEquipment: CharacterEquipment = {
  saveSessionID: "session-1",
  saveRevision: "0",
  characterID: 0,
  active: true,
  // Twenty-two raw fields in the backend's stored order. The unknown ones are
  // carried exactly like the known ones, because this stage names neither.
  slots: [
    0, 0xffffffff, 110, 120, 130, 140, 150, 160, 170, 180, 190, 200, 210, 220, 230, 240, 250, 260,
    270, 280, 290, 0xfffffffe,
  ],
};

const emptyLoadoutSlot = (slotType: string, rawValue = 0xffffffff) => ({
  slotType,
  state: "empty" as const,
  rawValue,
});

export const stubCharacterLoadout: CharacterLoadout = {
  saveSessionID: "session-1",
  saveRevision: "0",
  characterID: 0,
  active: true,
  rightHand: [
    {
      slotType: "right_hand",
      state: "occupied",
      // The backend names the exact Inventory record an occupied hand, armor or
      // talisman position references; the setters are called with it.
      ownedItemID: "owned-weapon-1",
      resource: { kind: "item", key: "000F4240" },
      name: "Dagger",
      iconPath: "icons/items/000F4240.png",
      rawValue: 0x000f4240,
    },
    emptyLoadoutSlot("right_hand", 0x0001adb0),
    emptyLoadoutSlot("right_hand", 0x0001adb0),
  ],
  leftHand: Array.from({ length: 3 }, () => emptyLoadoutSlot("left_hand", 0x0001adb0)),
  arrows: Array.from({ length: 2 }, () => emptyLoadoutSlot("arrow")),
  bolts: Array.from({ length: 2 }, () => emptyLoadoutSlot("bolt")),
  armor: ["head", "chest", "arms", "legs"].map((slotType) => emptyLoadoutSlot(slotType)),
  talismans: [
    {
      slotType: "talisman",
      state: "occupied",
      ownedItemID: "owned-talisman-1",
      resource: { kind: "item", key: "20000474" },
      name: "Moon of Nokstella",
      iconPath: "icons/items/20000474.png",
      rawValue: 0x20000474,
    },
    ...Array.from({ length: 3 }, () => ({
      slotType: "talisman",
      state: "locked" as const,
      rawValue: 0,
    })),
  ],
  quickItems: [
    {
      slotType: "quick_item",
      state: "occupied",
      ownedItemID: "owned-quick-1",
      resource: { kind: "item", key: "4000272E" },
      name: "Memory Stone",
      iconPath: "icons/items/4000272E.png",
      quantity: 3,
    },
    ...Array.from({ length: 9 }, () => ({
      slotType: "quick_item" as const,
      state: "empty" as const,
    })),
  ],
  pouch: Array.from({ length: 6 }, () => ({
    slotType: "pouch" as const,
    state: "empty" as const,
  })),
  activeQuickItem: 4,
  physick: [
    {
      slotType: "physick",
      state: "occupied",
      resource: { kind: "item", key: "40002AF9" },
      name: "Crimson Crystal Tear",
      iconPath: "icons/items/40002AF9.png",
      rawValue: 0x40002af9,
    },
    emptyLoadoutSlot("physick"),
  ],
  spells: [
    {
      state: "occupied",
      resource: { kind: "item", key: "40000FA0" },
      name: "Glintstone Pebble",
      iconPath: "icons/items/40000FA0.png",
      memorySlots: 1,
    },
    ...Array.from({ length: 11 }, () => ({ state: "empty" as const })),
  ],
  activeSpellIndex: 0,
  usedMemorySlots: 1,
  availableMemorySlots: 7,
  memoryStones: 3,
  unlockedTalismanSlots: 1,
};

export const stubCharacterQuickItems: CharacterQuickItems = {
  saveSessionID: "session-1",
  saveRevision: "0",
  characterID: 0,
  active: true,
  items: [
    { itemID: 0, equipIndex: 0 },
    { itemID: 0xffffffff, equipIndex: 1 },
    { itemID: 1010, equipIndex: 7 },
    { itemID: 1020, equipIndex: 2 },
    { itemID: 1030, equipIndex: 0xffffffff },
    { itemID: 1040, equipIndex: 4 },
    { itemID: 1050, equipIndex: 9 },
    { itemID: 1060, equipIndex: 3 },
    { itemID: 1070, equipIndex: 8 },
    { itemID: 1080, equipIndex: 5 },
  ],
  // Signed in the backend contract: a negative value is real reported state.
  activeQuick: -3,
};

export const stubCharacterPouchItems: CharacterPouchItems = {
  saveSessionID: "session-1",
  saveRevision: "0",
  characterID: 0,
  active: true,
  items: [
    { itemID: 0, equipIndex: 5 },
    { itemID: 0xffffffff, equipIndex: 0 },
    { itemID: 2010, equipIndex: 3 },
    { itemID: 2020, equipIndex: 1 },
    { itemID: 2030, equipIndex: 4 },
    { itemID: 2040, equipIndex: 2 },
  ],
};

export const stubCharacterPhysickMixture: CharacterPhysickMixture = {
  saveSessionID: "session-1",
  saveRevision: "0",
  characterID: 0,
  active: true,
  tears: [0xffffffff, 0],
};

/**
 * Twelve memory slots mixing the shapes the mapping has to survive: empty
 * records the backend resolved nothing for, occupied ones with different costs,
 * and two capacity counts that deliberately do not agree with each other or
 * with the records.
 */
export const stubCharacterEquippedSpells: CharacterEquippedSpells = {
  saveSessionID: "session-1",
  saveRevision: "0",
  characterID: 0,
  active: true,
  spells: [
    { rawMagicParamID: 0xffffffff, resourceKey: "", name: "", memorySlots: 0 },
    {
      rawMagicParamID: 3000,
      resourceKey: "item/glintstone-pebble",
      name: "Glintstone Pebble",
      memorySlots: 1,
    },
    { rawMagicParamID: 0xffffffff, resourceKey: "", name: "", memorySlots: 0 },
    {
      rawMagicParamID: 4010,
      resourceKey: "item/rotten-breath",
      name: "Rotten Breath",
      memorySlots: 2,
    },
    { rawMagicParamID: 5020, resourceKey: "item/comet-azur", name: "Comet Azur", memorySlots: 3 },
    { rawMagicParamID: 0xffffffff, resourceKey: "", name: "", memorySlots: 0 },
    { rawMagicParamID: 0xffffffff, resourceKey: "", name: "", memorySlots: 0 },
    { rawMagicParamID: 6030, resourceKey: "item/flame-sling", name: "Flame Sling", memorySlots: 1 },
    { rawMagicParamID: 0xffffffff, resourceKey: "", name: "", memorySlots: 0 },
    { rawMagicParamID: 0xffffffff, resourceKey: "", name: "", memorySlots: 0 },
    { rawMagicParamID: 0xffffffff, resourceKey: "", name: "", memorySlots: 0 },
    { rawMagicParamID: 0xffffffff, resourceKey: "", name: "", memorySlots: 0 },
  ],
  usedMemorySlots: 7,
  availableMemorySlots: 10,
};

/**
 * One candidate page in the shape the picker consumes. It carries a named
 * candidate, a nameless one and one flagged as ban risk, so nothing above the
 * port may invent a name or hide a row the backend already decided to serve.
 */
export const stubEquipmentCandidatesPage: EquipmentCandidatesPage = {
  saveSessionID: "session-1",
  saveRevision: "0",
  characterID: 0,
  active: true,
  safetyProfile: "safe",
  slotType: "right_hand",
  candidates: [
    {
      resource: { kind: "item", key: "weapon/uchigatana" },
      ownedItemID: "owned-weapon-1",
      name: "Uchigatana",
      iconPath: "assets/icons/items/uchigatana.png",
      quantity: 1,
      banRisk: false,
      cutContent: false,
    },
    {
      resource: { kind: "item", key: "weapon/unnamed" },
      ownedItemID: "owned-weapon-2",
      name: "",
      iconPath: "",
      quantity: 1,
      banRisk: false,
      cutContent: false,
    },
  ],
  total: 2,
  page: 1,
  pageSize: 30,
};

/** The receipt every Equipment mutation stub returns unless a test overrides it. */
export const stubEquipmentMutationReceipt: EquipmentMutationReceipt = {
  operationID: "operation-equipment-1",
  operationKind: "set_equipped_armaments",
  saveSessionID: "session-1",
  saveRevision: "1",
  changedScopes: ["save.session", "equipment.loadout", "diagnostics.report"],
};

export function makeEquipmentPort(overrides: Partial<EquipmentPort> = {}): EquipmentPort {
  return {
    getCharacterLoadout: () => Promise.resolve(stubCharacterLoadout),
    getEquipmentCandidates: () => Promise.resolve(stubEquipmentCandidatesPage),
    setEquippedArmaments: () => Promise.resolve(stubEquipmentMutationReceipt),
    setEquippedArmor: () => Promise.resolve(stubEquipmentMutationReceipt),
    setEquippedTalismans: () => Promise.resolve(stubEquipmentMutationReceipt),
    setEquippedSpells: () => Promise.resolve(stubEquipmentMutationReceipt),
    setPhysickMixture: () => Promise.resolve(stubEquipmentMutationReceipt),
    setPouchItems: () => Promise.resolve(stubEquipmentMutationReceipt),
    setQuickItems: () => Promise.resolve(stubEquipmentMutationReceipt),
    getEquipment: () => Promise.resolve(stubCharacterEquipment),
    getQuickItems: () => Promise.resolve(stubCharacterQuickItems),
    getPouchItems: () => Promise.resolve(stubCharacterPouchItems),
    getPhysickMixture: () => Promise.resolve(stubCharacterPhysickMixture),
    getEquippedSpells: () => Promise.resolve(stubCharacterEquippedSpells),
    ...overrides,
  };
}

/**
 * The World identity every stub answer carries. World is read-only, so the stub
 * is a plain set of getters: there is no World mutation to fake.
 */
const stubWorldIdentity = {
  saveSessionID: "session-1",
  saveRevision: "3",
  characterID: 0,
  active: true,
};

/**
 * The receipt every stubbed World mutation returns. It carries the World scope,
 * so a panel test can assert exactly what the mutation published.
 */
export const stubWorldMutationReceipt: WorldMutationReceipt = {
  operationID: "operation-world-1",
  operationKind: "set_region_unlocked",
  saveSessionID: "session-1",
  saveRevision: "4",
  changedScopes: ["save.session", "world.flags", "diagnostics.report"],
};

export function makeWorldPort(overrides: Partial<WorldPort> = {}): WorldPort {
  return {
    getRegions: () => Promise.resolve({ ...stubWorldIdentity, regions: [] }),
    getMapRegions: () => Promise.resolve({ ...stubWorldIdentity, mapRegions: [] }),
    getGraces: () => Promise.resolve({ ...stubWorldIdentity, graces: [] }),
    getBosses: () => Promise.resolve({ ...stubWorldIdentity, bosses: [] }),
    getQuests: () => Promise.resolve({ ...stubWorldIdentity, quests: [] }),
    getGestures: () => Promise.resolve({ ...stubWorldIdentity, gestures: [] }),
    getCookbooks: () => Promise.resolve({ ...stubWorldIdentity, cookbooks: [] }),
    getBellBearings: () => Promise.resolve({ ...stubWorldIdentity, bellBearings: [] }),
    getWhetblades: () => Promise.resolve({ ...stubWorldIdentity, whetblades: [] }),
    getTutorials: () => Promise.resolve({ ...stubWorldIdentity, tutorials: [] }),
    getSummoningPools: () => Promise.resolve({ ...stubWorldIdentity, summoningPools: [] }),
    getColosseums: () => Promise.resolve({ ...stubWorldIdentity, colosseums: [] }),
    getSpectralSteedAttires: () =>
      Promise.resolve({
        ...stubWorldIdentity,
        status: "legacy",
        activeAttireKey: "",
        attires: [],
      }),
    // No capability by default: a World writer exists only where the backend
    // published one, so a test that does not state a contract gets the
    // read-only workspace rather than an assumed set of actions.
    getWorldMutationCapabilities: () => Promise.resolve([]),
    setRegionUnlocked: () => Promise.resolve(stubWorldMutationReceipt),
    setMapRegionRevealed: () => Promise.resolve(stubWorldMutationReceipt),
    setGraceVisited: () => Promise.resolve(stubWorldMutationReceipt),
    setBossDefeated: () => Promise.resolve(stubWorldMutationReceipt),
    setGestureUnlocked: () => Promise.resolve(stubWorldMutationReceipt),
    setCookbookUnlocked: () => Promise.resolve(stubWorldMutationReceipt),
    setBellBearingUnlocked: () => Promise.resolve(stubWorldMutationReceipt),
    setWhetbladeUnlocked: () => Promise.resolve(stubWorldMutationReceipt),
    setTutorialUnlocked: () => Promise.resolve(stubWorldMutationReceipt),
    setSummoningPoolActivated: () => Promise.resolve(stubWorldMutationReceipt),
    setColosseumUnlocked: () => Promise.resolve(stubWorldMutationReceipt),
    setFogOfWarRemoved: () => Promise.resolve(stubWorldMutationReceipt),
    setQuestStep: () => Promise.resolve(stubWorldMutationReceipt),
    setSpectralSteedAttire: () => Promise.resolve(stubWorldMutationReceipt),
    lockAllSpectralSteedAttires: () => Promise.resolve(stubWorldMutationReceipt),
    ...overrides,
  };
}

export function makePort(overrides: Partial<ApplicationInfoPort> = {}): ApplicationInfoPort {
  return {
    getApplicationInfo: () => Promise.resolve(stubApplicationInfo),
    ...overrides,
  };
}

/**
 * A clean report: every scope checked, nothing unresolved, no finding. It is
 * the shape the flow must read as `clean`, and the base every warning stub is
 * built from so a single differing counter is what changes the outcome.
 */
export const stubCleanValidationReport: SaveValidationReport = {
  saveSessionID: "session-1",
  saveRevision: "0",
  characterID: 0,
  active: true,
  coverage: ["inventory", "storage", "stats", "equipment", "spells"].map((scope) => ({
    scope,
    checked: true,
    reason: "",
    recordsChecked: 12,
    unresolvedRecords: 0,
  })),
  issues: [],
  errorCount: 0,
  warningCount: 0,
};

/**
 * The default port answers about the session and the slot it was asked about,
 * because a real backend does: a stub that always named one session would make
 * a second session's report look stale and hide what the flow really does.
 */
export function makeDiagnosticsPort(overrides: Partial<DiagnosticsPort> = {}): DiagnosticsPort {
  return {
    getSaveValidationReport: ({ saveSessionID, characterID }) =>
      Promise.resolve({ ...stubCleanValidationReport, saveSessionID, characterID }),
    ...overrides,
  };
}

export function makeSaveSessionPort(overrides: Partial<SaveSessionPort> = {}): SaveSessionPort {
  return {
    selectSaveFile: () => Promise.resolve(stubSaveSession.sourcePath),
    selectSaveTarget: () => Promise.resolve("/Users/Tarnished/Elden Ring/ER0000-copy.sl2"),
    subscribeApplicationCloseRequested: () => () => {},
    quitApplication: () => Promise.resolve(),
    loadSave: () => Promise.resolve(stubSaveSession),
    getLoadedSave: () => Promise.resolve(stubSaveSession),
    closeSave: () => Promise.resolve(),
    getOperationHistory: (saveSessionID) =>
      Promise.resolve({
        saveSessionID,
        saveRevision: stubSaveSession.saveRevision,
        operations: [],
        undoCount: 0,
        redoCount: 0,
      }),
    undoLastOperation: () => Promise.reject(new Error("no operation to undo")),
    redoLastOperation: () => Promise.reject(new Error("no operation to redo")),
    revertOperation: () => Promise.reject(new Error("no operation to revert")),
    discardChanges: (saveSessionID) =>
      Promise.resolve({
        operationID: "operation-discard",
        operationKind: "discard_changes",
        saveSessionID,
        saveRevision: "1",
        changedScopes: ["save.session"],
        discardedOperations: 0,
      }),
    validateReviewChanges: (saveSessionID, expectedRevision) =>
      Promise.resolve({
        saveSessionID,
        saveRevision: expectedRevision,
        validationToken: "validation-test",
        valid: true,
        warningCount: 0,
        banRiskCount: 0,
        criticalCount: 0,
        stages: [{ stage: "validation", percent: 100 }],
        issues: [],
      }),
    save: (saveSessionID) =>
      Promise.resolve({
        operationID: "operation-save",
        operationKind: "save",
        saveSessionID,
        saveRevision: "1",
        changedScopes: ["save.session"],
        target: stubSaveSession.sourcePath,
        warnings: [],
        retentionNoticeRequired: false,
      }),
    saveAs: (saveSessionID, _revision, _token, _warnings, _banRisk, target) =>
      Promise.resolve({
        operationID: "operation-save-as",
        operationKind: "save_as",
        saveSessionID,
        saveRevision: "1",
        changedScopes: ["save.session"],
        target,
        warnings: [],
        retentionNoticeRequired: false,
      }),
    getRecentFiles: () => Promise.resolve([]),
    recordRecentFile: () => Promise.resolve([]),
    removeRecentFile: () => Promise.resolve([]),
    clearRecentFiles: () => Promise.resolve(),
    getRecoveryJournals: () => Promise.resolve([]),
    getRecoveryJournal: () => Promise.reject(new Error("no recovery journal")),
    restoreRecoveryJournal: () => Promise.reject(new Error("no recovery journal")),
    discardRecoveryJournal: () => Promise.resolve(),
    exportRecoveryJournal: () => Promise.resolve(),
    getSaveLifecycleSettings: () =>
      Promise.resolve({ backupRetention: 10, retentionNoticeShown: false }),
    setSaveLifecycleSettings: (backupRetention) =>
      Promise.resolve({ backupRetention, retentionNoticeShown: false }),
    // No event ever arrives through the default stub, and unsubscribing is a
    // no-op: a test that cares about events injects its own subscription.
    subscribeSessionChanged: () => () => {},
    ...overrides,
  };
}

export const stubCharacterMutationReceipt: MutationReceipt = {
  operationID: "op-1",
  operationKind: "set_character_name",
  saveSessionID: "session-1",
  saveRevision: "1",
  changedScopes: ["save.session", "character.list", "character.profile", "diagnostics.report"],
};

export const stubAppearancePresetSummaries: readonly AppearancePresetSummary[] = [
  {
    id: "geralt-of-rivia-the-witcher",
    name: "Geralt of Rivia, the Witcher",
    image: "geralt-of-rivia-the-witcher.jpg",
    bodyType: "Type A",
    tags: ["witcher"],
  },
  {
    id: "ciri-the-princess-of-cintra-from-witcher",
    name: "Ciri, the Princess of Cintra",
    image: "ciri-the-princess-of-cintra-from-witcher.jpg",
    bodyType: "Type B",
    tags: ["witcher"],
  },
];

export const stubFavoritePresets: SaveFavoritePresets = {
  saveSessionID: "session-1",
  presets: Array.from({ length: 15 }, (_, i) => ({
    favoriteSlotID: i,
    active: i === 0,
  })),
};

export function makeCharacterPort(overrides: Partial<CharacterPort> = {}): CharacterPort {
  return {
    getSaveCharacters: (saveSessionID) => Promise.resolve({ ...stubSaveCharacters, saveSessionID }),
    getCharacterProfile: (saveSessionID, characterID) =>
      Promise.resolve({ ...stubCharacterProfile, saveSessionID, characterID }),
    getCharacterStats: (saveSessionID, characterID) =>
      Promise.resolve({ ...stubCharacterStats, saveSessionID, characterID }),
    setCharacterName: () => Promise.resolve(stubCharacterMutationReceipt),
    setCharacterStats: () => Promise.resolve(stubCharacterMutationReceipt),
    setCharacterStartingClass: () => Promise.resolve(stubCharacterMutationReceipt),
    setCharacterGender: () => Promise.resolve(stubCharacterMutationReceipt),
    setCharacterRunes: () => Promise.resolve(stubCharacterMutationReceipt),
    ...overrides,
  };
}

export function makeAppearancePort(overrides: Partial<AppearancePort> = {}): AppearancePort {
  return {
    getAppearancePresets: () => Promise.resolve(stubAppearancePresetSummaries),
    applyAppearancePreset: () => Promise.resolve(stubCharacterMutationReceipt),
    ...overrides,
  };
}

export function makeFavoritesPort(overrides: Partial<FavoritesPort> = {}): FavoritesPort {
  return {
    getFavoritePresets: (saveSessionID) =>
      Promise.resolve({ ...stubFavoritePresets, saveSessionID }),
    setFavoritePreset: () => Promise.resolve(stubCharacterMutationReceipt),
    applyFavoritePreset: () => Promise.resolve(stubCharacterMutationReceipt),
    deleteFavoritePreset: () => Promise.resolve(stubCharacterMutationReceipt),
    ...overrides,
  };
}

export const stubNetworkParamValues = {
  maxBreakInTargetListCount: 5,
  breakInRequestIntervalTimeSec: 30,
  breakInRequestTimeOutSec: 20,
  breakInRequestAreaCount: 5,
  summonTimeoutTime: 45,
  reloadSignIntervalTime2: 60,
  reloadSignTotalCount: 20,
  reloadSignCellCount: 10,
  updateSignIntervalTime: 30,
  singGetMax: 32,
  signDownloadSpan: 30,
  signUpdateSpan: 60,
  reloadVisitListCoolTime: 20,
  maxCoopBlueSummonCount: 2,
  maxVisitListCount: 5,
  reloadSearchCoopBlueMin: 30,
  reloadSearchCoopBlueMax: 180,
  allAreaSearchRateCoopBlue: 30,
  allAreaSearchRateVsBlue: 30,
  visitorListMax: 10,
  visitorTimeOutTime: 60,
  visitorDownloadSpan: 60,
};

export const stubNetworkSettingsSnapshot: NetworkSettingsSnapshot = {
  saveSessionID: "session-1",
  saveRevision: "3",
  parameters: stubNetworkParamValues,
};

export const stubNetworkPresetsResult: NetworkPresetsResult = {
  presets: [
    { id: "vanilla", parameters: stubNetworkParamValues },
    {
      id: "faster-reds",
      parameters: {
        ...stubNetworkParamValues,
        maxBreakInTargetListCount: 8,
        breakInRequestIntervalTimeSec: 12,
        breakInRequestTimeOutSec: 8,
        breakInRequestAreaCount: 8,
      },
    },
    {
      id: "aggressive-reds",
      parameters: {
        ...stubNetworkParamValues,
        maxBreakInTargetListCount: 12,
        breakInRequestIntervalTimeSec: 10,
        breakInRequestTimeOutSec: 7,
        breakInRequestAreaCount: 12,
      },
    },
    {
      id: "faster-summons",
      parameters: {
        ...stubNetworkParamValues,
        reloadSignIntervalTime2: 20,
        reloadSignTotalCount: 40,
        reloadSignCellCount: 20,
        updateSignIntervalTime: 15,
        singGetMax: 64,
        signDownloadSpan: 15,
        signUpdateSpan: 20,
      },
    },
    {
      id: "aggressive-summons",
      parameters: {
        ...stubNetworkParamValues,
        reloadSignIntervalTime2: 10,
        reloadSignTotalCount: 64,
        reloadSignCellCount: 32,
        updateSignIntervalTime: 10,
        singGetMax: 96,
        signDownloadSpan: 10,
        signUpdateSpan: 10,
      },
    },
    {
      id: "faster-blue",
      parameters: {
        ...stubNetworkParamValues,
        reloadVisitListCoolTime: 8,
        maxVisitListCount: 10,
        reloadSearchCoopBlueMin: 10,
        reloadSearchCoopBlueMax: 40,
        allAreaSearchRateCoopBlue: 60,
      },
    },
    {
      id: "aggressive-blue",
      parameters: {
        ...stubNetworkParamValues,
        reloadVisitListCoolTime: 5,
        maxVisitListCount: 15,
        reloadSearchCoopBlueMin: 5,
        reloadSearchCoopBlueMax: 20,
        allAreaSearchRateCoopBlue: 100,
      },
    },
    {
      id: "faster-summon-host",
      parameters: {
        ...stubNetworkParamValues,
        summonTimeoutTime: 10,
        reloadSignIntervalTime2: 20,
        reloadSignTotalCount: 24,
        singGetMax: 40,
        signDownloadSpan: 15,
      },
    },
    {
      id: "aggressive-summon-host",
      parameters: {
        ...stubNetworkParamValues,
        summonTimeoutTime: 7,
        reloadSignIntervalTime2: 12,
        singGetMax: 48,
        signDownloadSpan: 10,
      },
    },
    {
      id: "faster-summon-guest",
      parameters: {
        ...stubNetworkParamValues,
        updateSignIntervalTime: 15,
        signUpdateSpan: 20,
      },
    },
    {
      id: "aggressive-summon-guest",
      parameters: {
        ...stubNetworkParamValues,
        updateSignIntervalTime: 10,
        signUpdateSpan: 12,
      },
    },
    {
      id: "faster-hunter",
      parameters: {
        ...stubNetworkParamValues,
        reloadVisitListCoolTime: 10,
        maxVisitListCount: 8,
        reloadSearchCoopBlueMin: 12,
        reloadSearchCoopBlueMax: 72,
      },
    },
    {
      id: "aggressive-hunter",
      parameters: {
        ...stubNetworkParamValues,
        reloadVisitListCoolTime: 6,
        maxVisitListCount: 12,
        reloadSearchCoopBlueMin: 8,
        reloadSearchCoopBlueMax: 48,
      },
    },
  ],
};

export const stubSetNetworkSettingsResult: SetNetworkSettingsResult = {
  operationID: "op-network-1",
  operationKind: "set_network_settings",
  saveSessionID: "session-1",
  saveRevision: "4",
  changedScopes: ["network"],
  networkSettings: stubNetworkParamValues,
};

export function makeNetworkPort(overrides: Partial<NetworkPort> = {}): NetworkPort {
  return {
    getNetworkSettings: (saveSessionID) =>
      Promise.resolve({ ...stubNetworkSettingsSnapshot, saveSessionID }),
    getNetworkPresets: () => Promise.resolve(stubNetworkPresetsResult),
    setNetworkSettings: (saveSessionID, networkSettings, expectedRevision) =>
      Promise.resolve({
        ...stubSetNetworkSettingsResult,
        saveSessionID,
        saveRevision: String(Number(expectedRevision) + 1),
        networkSettings,
      }),
    ...overrides,
  };
}

export const failingPort: ApplicationInfoPort = {
  getApplicationInfo: () => Promise.reject(new Error("bridge_call_failed")),
};

/**
 * The component-test client. `gcTime: 0` keeps rendered views from leaking
 * between tests; a test that asserts on cache contents passes its own client.
 */
export function createTestQueryClient(): QueryClient {
  return new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } });
}

export function TestProviders({
  children,
  queryClient,
  port,
  saveSessionPort,
  characterPort,
  appearancePort,
  favoritesPort,
  diagnosticsPort,
  itemsPort,
  equipmentPort,
  worldPort,
  networkPort,
  catalogPort,
  settingsPort,
  showItemID,
}: {
  children: ReactNode;
  queryClient: QueryClient;
  port?: ApplicationInfoPort;
  saveSessionPort?: SaveSessionPort;
  characterPort?: CharacterPort;
  appearancePort?: AppearancePort;
  favoritesPort?: FavoritesPort;
  diagnosticsPort?: DiagnosticsPort;
  itemsPort?: ItemsPort;
  equipmentPort?: EquipmentPort;
  worldPort?: WorldPort;
  networkPort?: NetworkPort;
  catalogPort?: CatalogPort;
  settingsPort?: SettingsPort;
  showItemID?: boolean;
}) {
  return (
    <QueryClientProvider client={queryClient}>
      <ApplicationInfoPortProvider port={port ?? makePort()}>
        <SettingsPortProvider port={settingsPort ?? makeSettingsPort()}>
          <ItemPreferencesProvider initialShowItemID={showItemID}>
            <CatalogPortProvider port={catalogPort ?? makeCatalogPort()}>
              <AppearancePortProvider port={appearancePort ?? makeAppearancePort()}>
                <FavoritesPortProvider port={favoritesPort ?? makeFavoritesPort()}>
                  <SaveSessionPortProvider port={saveSessionPort ?? makeSaveSessionPort()}>
                    <CharacterPortProvider port={characterPort ?? makeCharacterPort()}>
                      <DiagnosticsPortProvider port={diagnosticsPort ?? makeDiagnosticsPort()}>
                        <ItemsPortProvider port={itemsPort ?? makeItemsPort()}>
                          <EquipmentPortProvider port={equipmentPort ?? makeEquipmentPort()}>
                            <WorldPortProvider port={worldPort ?? makeWorldPort()}>
                              <NetworkPortProvider port={networkPort ?? makeNetworkPort()}>
                                {children}
                              </NetworkPortProvider>
                            </WorldPortProvider>
                          </EquipmentPortProvider>
                        </ItemsPortProvider>
                      </DiagnosticsPortProvider>
                    </CharacterPortProvider>
                  </SaveSessionPortProvider>
                </FavoritesPortProvider>
              </AppearancePortProvider>
            </CatalogPortProvider>
          </ItemPreferencesProvider>
        </SettingsPortProvider>
      </ApplicationInfoPortProvider>
    </QueryClientProvider>
  );
}

export async function renderApp(
  ui: ReactNode,
  options: {
    port?: ApplicationInfoPort;
    saveSessionPort?: SaveSessionPort;
    characterPort?: CharacterPort;
    appearancePort?: AppearancePort;
    favoritesPort?: FavoritesPort;
    diagnosticsPort?: DiagnosticsPort;
    itemsPort?: ItemsPort;
    equipmentPort?: EquipmentPort;
    worldPort?: WorldPort;
    networkPort?: NetworkPort;
    catalogPort?: CatalogPort;
    settingsPort?: SettingsPort;
    showItemID?: boolean;
    locale?: Locale;
    queryClient?: QueryClient;
  } = {},
): Promise<RenderResult> {
  await activateLocale(options.locale ?? "en");

  return render(
    <I18nProvider i18n={i18n}>
      <TestProviders
        queryClient={options.queryClient ?? createTestQueryClient()}
        port={options.port}
        saveSessionPort={options.saveSessionPort}
        characterPort={options.characterPort}
        appearancePort={options.appearancePort}
        favoritesPort={options.favoritesPort}
        diagnosticsPort={options.diagnosticsPort}
        itemsPort={options.itemsPort}
        equipmentPort={options.equipmentPort}
        worldPort={options.worldPort}
        networkPort={options.networkPort}
        catalogPort={options.catalogPort}
        settingsPort={options.settingsPort}
        showItemID={options.showItemID}
      >
        {ui}
      </TestProviders>
    </I18nProvider>,
  );
}
