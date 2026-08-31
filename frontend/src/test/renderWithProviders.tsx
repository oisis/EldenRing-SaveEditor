import { I18nProvider } from "@lingui/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { type RenderResult, render } from "@testing-library/react";
import type { ReactNode } from "react";
import { ApplicationInfoPortProvider } from "../application/application-info/applicationInfoClient";
import type {
  ApplicationInfo,
  ApplicationInfoPort,
} from "../application/application-info/applicationInfoPort";
import { CatalogPortProvider } from "../application/catalog/catalogClient";
import type {
  CatalogFact,
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
  EquipmentPort,
} from "../application/equipment/equipmentPort";
import { ItemsPortProvider } from "../application/items/itemsClient";
import type { ItemPage, ItemsPort } from "../application/items/itemsPort";
import { SaveSessionPortProvider } from "../application/save-session/saveSessionClient";
import type { SaveSession, SaveSessionPort } from "../application/save-session/saveSessionPort";
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
};

export const stubSaveCharacters: SaveCharacters = {
  saveSessionID: "session-1",
  characters: [
    { characterID: 0, active: true, name: "Tarnished", level: 150 },
    { characterID: 1, active: false, name: "", level: 0 },
  ],
};

export const stubCharacterProfile: CharacterProfile = {
  saveSessionID: "session-1",
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
};

export const stubInventoryPage: ItemPage = {
  saveSessionID: "session-1",
  saveRevision: "revision-1",
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

export function makeCatalogPort(overrides: Partial<CatalogPort> = {}): CatalogPort {
  return {
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

export function makeItemsPort(overrides: Partial<ItemsPort> = {}): ItemsPort {
  return {
    getInventory: () => Promise.resolve(stubInventoryPage),
    getStorage: () => Promise.resolve(stubStoragePage),
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
  saveRevision: "7",
  characterID: 0,
  active: true,
  rightHand: [
    {
      slotType: "right_hand",
      state: "occupied",
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
  unlockedTalismanSlots: 1,
};

export const stubCharacterQuickItems: CharacterQuickItems = {
  saveSessionID: "session-1",
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

export function makeEquipmentPort(overrides: Partial<EquipmentPort> = {}): EquipmentPort {
  return {
    getCharacterLoadout: () => Promise.resolve(stubCharacterLoadout),
    getEquipment: () => Promise.resolve(stubCharacterEquipment),
    getQuickItems: () => Promise.resolve(stubCharacterQuickItems),
    getPouchItems: () => Promise.resolve(stubCharacterPouchItems),
    getPhysickMixture: () => Promise.resolve(stubCharacterPhysickMixture),
    getEquippedSpells: () => Promise.resolve(stubCharacterEquippedSpells),
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
    loadSave: () => Promise.resolve(stubSaveSession),
    getLoadedSave: () => Promise.resolve(stubSaveSession),
    closeSave: () => Promise.resolve(),
    ...overrides,
  };
}

export function makeCharacterPort(overrides: Partial<CharacterPort> = {}): CharacterPort {
  return {
    getSaveCharacters: () => Promise.resolve(stubSaveCharacters),
    getCharacterProfile: () => Promise.resolve(stubCharacterProfile),
    getCharacterStats: () => Promise.resolve(stubCharacterStats),
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
  diagnosticsPort,
  itemsPort,
  equipmentPort,
  catalogPort,
}: {
  children: ReactNode;
  queryClient: QueryClient;
  port?: ApplicationInfoPort;
  saveSessionPort?: SaveSessionPort;
  characterPort?: CharacterPort;
  diagnosticsPort?: DiagnosticsPort;
  itemsPort?: ItemsPort;
  equipmentPort?: EquipmentPort;
  catalogPort?: CatalogPort;
}) {
  return (
    <QueryClientProvider client={queryClient}>
      <ApplicationInfoPortProvider port={port ?? makePort()}>
        <CatalogPortProvider port={catalogPort ?? makeCatalogPort()}>
          <SaveSessionPortProvider port={saveSessionPort ?? makeSaveSessionPort()}>
            <CharacterPortProvider port={characterPort ?? makeCharacterPort()}>
              <DiagnosticsPortProvider port={diagnosticsPort ?? makeDiagnosticsPort()}>
                <ItemsPortProvider port={itemsPort ?? makeItemsPort()}>
                  <EquipmentPortProvider port={equipmentPort ?? makeEquipmentPort()}>
                    {children}
                  </EquipmentPortProvider>
                </ItemsPortProvider>
              </DiagnosticsPortProvider>
            </CharacterPortProvider>
          </SaveSessionPortProvider>
        </CatalogPortProvider>
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
    diagnosticsPort?: DiagnosticsPort;
    itemsPort?: ItemsPort;
    equipmentPort?: EquipmentPort;
    catalogPort?: CatalogPort;
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
        diagnosticsPort={options.diagnosticsPort}
        itemsPort={options.itemsPort}
        equipmentPort={options.equipmentPort}
        catalogPort={options.catalogPort}
      >
        {ui}
      </TestProviders>
    </I18nProvider>,
  );
}
