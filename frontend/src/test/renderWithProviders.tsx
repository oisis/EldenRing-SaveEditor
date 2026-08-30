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

export function makeCatalogPort(overrides: Partial<CatalogPort> = {}): CatalogPort {
  return {
    getResources: () => Promise.resolve(stubCatalogPage),
    getResource: () => Promise.resolve(stubCatalogResourceDetail),
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

export function makePort(overrides: Partial<ApplicationInfoPort> = {}): ApplicationInfoPort {
  return {
    getApplicationInfo: () => Promise.resolve(stubApplicationInfo),
    ...overrides,
  };
}

export function makeSaveSessionPort(overrides: Partial<SaveSessionPort> = {}): SaveSessionPort {
  return {
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
  itemsPort,
  catalogPort,
}: {
  children: ReactNode;
  queryClient: QueryClient;
  port?: ApplicationInfoPort;
  saveSessionPort?: SaveSessionPort;
  characterPort?: CharacterPort;
  itemsPort?: ItemsPort;
  catalogPort?: CatalogPort;
}) {
  return (
    <QueryClientProvider client={queryClient}>
      <ApplicationInfoPortProvider port={port ?? makePort()}>
        <CatalogPortProvider port={catalogPort ?? makeCatalogPort()}>
          <SaveSessionPortProvider port={saveSessionPort ?? makeSaveSessionPort()}>
            <CharacterPortProvider port={characterPort ?? makeCharacterPort()}>
              <ItemsPortProvider port={itemsPort ?? makeItemsPort()}>{children}</ItemsPortProvider>
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
    itemsPort?: ItemsPort;
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
        itemsPort={options.itemsPort}
        catalogPort={options.catalogPort}
      >
        {ui}
      </TestProviders>
    </I18nProvider>,
  );
}
