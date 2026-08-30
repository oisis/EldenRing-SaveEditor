import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  CloseSave,
  GetApplicationInfo,
  GetCharacterProfile,
  GetCharacterStats,
  GetInventory,
  GetItemVariants,
  GetLoadedSave,
  GetResource,
  GetResources,
  GetSaveCharacters,
  GetStorage,
  LoadSave,
} from "../../../wailsjs/go/desktop/Bridge";
import { application, catalog, inventory, saveengine } from "../../../wailsjs/go/models";
import { bridgeFailureCode, wailsDesktopBridge } from "./desktopBridge";

vi.mock("../../../wailsjs/go/desktop/Bridge", () => ({
  CloseSave: vi.fn(),
  GetApplicationInfo: vi.fn(),
  GetCharacterProfile: vi.fn(),
  GetCharacterStats: vi.fn(),
  GetInventory: vi.fn(),
  GetItemVariants: vi.fn(),
  GetLoadedSave: vi.fn(),
  GetResource: vi.fn(),
  GetResources: vi.fn(),
  GetSaveCharacters: vi.fn(),
  GetStorage: vi.fn(),
  LoadSave: vi.fn(),
}));

const getApplicationInfo = vi.mocked(GetApplicationInfo);
const getLoadedSave = vi.mocked(GetLoadedSave);
const loadSave = vi.mocked(LoadSave);
const closeSave = vi.mocked(CloseSave);
const getSaveCharacters = vi.mocked(GetSaveCharacters);
const getCharacterProfile = vi.mocked(GetCharacterProfile);
const getCharacterStats = vi.mocked(GetCharacterStats);
const getInventory = vi.mocked(GetInventory);
const getStorage = vi.mocked(GetStorage);
const getResources = vi.mocked(GetResources);
const getResource = vi.mocked(GetResource);
const getItemVariants = vi.mocked(GetItemVariants);

beforeEach(() => {
  getApplicationInfo.mockReset();
  getLoadedSave.mockReset();
  loadSave.mockReset();
  closeSave.mockReset();
  getSaveCharacters.mockReset();
  getCharacterProfile.mockReset();
  getCharacterStats.mockReset();
  getInventory.mockReset();
  getStorage.mockReset();
  getResources.mockReset();
  getResource.mockReset();
  getItemVariants.mockReset();
});

describe("wails application info adapter", () => {
  it("projects the generated backend result onto the application port", async () => {
    getApplicationInfo.mockResolvedValue(
      application.GetApplicationInfoResult.createFrom({
        applicationVersion: "2.0.0",
        supportedSchemas: [{ name: "game_catalog", minimumVersion: 1, currentVersion: 16 }],
        capabilities: ["catalog_read"],
      }),
    );

    await expect(wailsDesktopBridge.getApplicationInfo()).resolves.toEqual({
      version: "2.0.0",
      schemas: [{ name: "game_catalog", minimumVersion: 1, currentVersion: 16 }],
      capabilities: ["catalog_read"],
    });
    expect(getApplicationInfo).toHaveBeenCalledTimes(1);
  });

  it("passes every reported schema and capability through unchanged", async () => {
    getApplicationInfo.mockResolvedValue(
      application.GetApplicationInfoResult.createFrom({
        applicationVersion: "  2.0.0-rc.1+local  ",
        supportedSchemas: [
          { name: "game_catalog", minimumVersion: 1, currentVersion: 16 },
          { name: "future_schema", minimumVersion: 3, currentVersion: 4 },
        ],
        capabilities: ["catalog_read", "future_capability"],
      }),
    );

    const info = await wailsDesktopBridge.getApplicationInfo();

    // The adapter neither trims the version nor filters or reorders the lists:
    // the backend owns that contract.
    expect(info.version).toBe("  2.0.0-rc.1+local  ");
    expect(info.schemas.map((schema) => schema.name)).toEqual(["game_catalog", "future_schema"]);
    expect(info.capabilities).toEqual(["catalog_read", "future_capability"]);
  });

  it("replaces a transport failure with a stable code carrying no transport text", async () => {
    getApplicationInfo.mockRejectedValue(
      new Error(
        "goroutine 1 [running]: desktop.(*Bridge).GetApplicationInfo /Users/private/app.go:42",
      ),
    );

    await expect(wailsDesktopBridge.getApplicationInfo()).rejects.toThrow(
      new Error(bridgeFailureCode),
    );
  });
});

const session = saveengine.SessionInfo.createFrom({
  saveSessionID: "session-1",
  platform: "pc",
  format: "sl2_v2",
  unsavedChanges: true,
});

describe("wails save session adapter", () => {
  it("passes the source and the expected platform to the backend unchanged", async () => {
    loadSave.mockResolvedValue(session);

    await wailsDesktopBridge.loadSave("  /Volumes/A B/ER0000.sl2  ", "  PS4  ");

    // No trimming, no normalisation, no fallback: the backend owns path and
    // platform handling.
    expect(loadSave).toHaveBeenCalledWith("  /Volumes/A B/ER0000.sl2  ", "  PS4  ");
    expect(loadSave).toHaveBeenCalledTimes(1);
  });

  it("passes the session identifier to the reader and to the close call unchanged", async () => {
    getLoadedSave.mockResolvedValue(session);
    closeSave.mockResolvedValue(undefined);

    await wailsDesktopBridge.getLoadedSave("  Session ID  ");
    await wailsDesktopBridge.closeSave("  Session ID  ");

    expect(getLoadedSave).toHaveBeenCalledWith("  Session ID  ");
    expect(closeSave).toHaveBeenCalledWith("  Session ID  ");
  });

  it("maps every reported session field without normalising or defaulting it", async () => {
    getLoadedSave.mockResolvedValue(session);

    // Exactly the four fields the backend reports; nothing is added.
    await expect(wailsDesktopBridge.getLoadedSave("session-1")).resolves.toEqual({
      saveSessionID: "session-1",
      platform: "pc",
      format: "sl2_v2",
      unsavedChanges: true,
    });
  });

  it("carries an unknown platform and format through without rejecting them", async () => {
    loadSave.mockResolvedValue(
      saveengine.SessionInfo.createFrom({
        saveSessionID: "session-2",
        platform: "future_platform",
        format: "future_format",
        unsavedChanges: false,
      }),
    );

    await expect(wailsDesktopBridge.loadSave("source", "future_platform")).resolves.toEqual({
      saveSessionID: "session-2",
      platform: "future_platform",
      format: "future_format",
      unsavedChanges: false,
    });
  });

  it("replaces a failed session call with the stable code, on every session method", async () => {
    const transportError = new Error(
      "goroutine 7 [running]: saveengine.(*Engine).Load /Users/private/save.go:120",
    );
    loadSave.mockRejectedValue(transportError);
    getLoadedSave.mockRejectedValue(transportError);
    closeSave.mockRejectedValue(transportError);

    for (const call of [
      () => wailsDesktopBridge.loadSave("/Users/private/ER0000.sl2", "pc"),
      () => wailsDesktopBridge.getLoadedSave("session-1"),
      () => wailsDesktopBridge.closeSave("session-1"),
    ]) {
      const failure = await call().then(
        () => undefined,
        (error: unknown) => error as Error,
      );

      expect(failure?.message).toBe(bridgeFailureCode);
      // Neither the Go text, nor the stack, nor the path reaches the caller.
      expect(failure?.message).not.toContain("goroutine");
      expect(failure?.message).not.toContain("/Users/private");
      expect(failure?.stack ?? "").not.toContain("saveengine.(*Engine).Load");
    }
  });
});

const characters = saveengine.SaveCharacters.createFrom({
  saveSessionID: "session-1",
  characters: [
    { characterID: 0, active: true, name: "Tarnished", level: 150 },
    { characterID: 1, active: false, name: "", level: 0 },
  ],
});

const profile = saveengine.CharacterProfile.createFrom({
  saveSessionID: "session-1",
  characterID: 0,
  active: true,
  name: "Tarnished",
  level: 150,
  startingClassID: 3,
  gender: 1,
  secondsPlayed: 123456,
});

const stats = saveengine.CharacterStats.createFrom({
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
});

describe("wails character adapter", () => {
  it("passes the session identifier and the slot index to the backend unchanged", async () => {
    getSaveCharacters.mockResolvedValue(characters);
    getCharacterProfile.mockResolvedValue(profile);
    getCharacterStats.mockResolvedValue(stats);

    await wailsDesktopBridge.getSaveCharacters("  Session ID  ");
    await wailsDesktopBridge.getCharacterProfile("  Session ID  ", 0);
    await wailsDesktopBridge.getCharacterStats("  Session ID  ", 9);

    // No trimming and no slot-range check: the backend owns both.
    expect(getSaveCharacters).toHaveBeenCalledExactlyOnceWith("  Session ID  ");
    expect(getCharacterProfile).toHaveBeenCalledExactlyOnceWith("  Session ID  ", 0);
    expect(getCharacterStats).toHaveBeenCalledExactlyOnceWith("  Session ID  ", 9);
  });

  it("passes a slot index outside the backend range on instead of rejecting it", async () => {
    getCharacterProfile.mockResolvedValue(profile);
    getCharacterStats.mockResolvedValue(stats);

    await wailsDesktopBridge.getCharacterProfile("session-1", -1);
    await wailsDesktopBridge.getCharacterStats("session-1", 42);

    expect(getCharacterProfile).toHaveBeenCalledExactlyOnceWith("session-1", -1);
    expect(getCharacterStats).toHaveBeenCalledExactlyOnceWith("session-1", 42);
  });

  it("maps every reported slot summary field and nothing else", async () => {
    getSaveCharacters.mockResolvedValue(characters);

    // Exactly the fields the backend reports; no slot number, no status beyond
    // `active`, and an inactive slot is an ordinary result.
    await expect(wailsDesktopBridge.getSaveCharacters("session-1")).resolves.toEqual({
      saveSessionID: "session-1",
      characters: [
        { characterID: 0, active: true, name: "Tarnished", level: 150 },
        { characterID: 1, active: false, name: "", level: 0 },
      ],
    });
  });

  it("maps every reported profile field and nothing else", async () => {
    getCharacterProfile.mockResolvedValue(profile);

    await expect(wailsDesktopBridge.getCharacterProfile("session-1", 0)).resolves.toEqual({
      saveSessionID: "session-1",
      characterID: 0,
      active: true,
      name: "Tarnished",
      level: 150,
      startingClassID: 3,
      gender: 1,
      secondsPlayed: 123456,
    });
  });

  it("maps every reported statistics field and nothing else", async () => {
    getCharacterStats.mockResolvedValue(stats);

    await expect(wailsDesktopBridge.getCharacterStats("session-1", 0)).resolves.toEqual({
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
    });
  });

  it("carries unknown identifiers and out-of-range values through unnormalised", async () => {
    getCharacterProfile.mockResolvedValue(
      saveengine.CharacterProfile.createFrom({
        saveSessionID: "session-1",
        characterID: 4,
        active: false,
        name: "",
        level: 0,
        startingClassID: 250,
        gender: 200,
        secondsPlayed: 0,
      }),
    );
    getCharacterStats.mockResolvedValue(
      saveengine.CharacterStats.createFrom({ ...stats, vigor: 4294967295, maxHP: 0, hp: 999999 }),
    );

    const unknownProfile = await wailsDesktopBridge.getCharacterProfile("session-1", 4);
    const unknownStats = await wailsDesktopBridge.getCharacterStats("session-1", 4);

    // No class name, no gender label, no clamping and no default: an unknown
    // identifier stays exactly the number the backend reported.
    expect(unknownProfile.startingClassID).toBe(250);
    expect(unknownProfile.gender).toBe(200);
    expect(unknownProfile.active).toBe(false);
    expect(unknownStats.vigor).toBe(4294967295);
    expect(unknownStats.maxHP).toBe(0);
    expect(unknownStats.hp).toBe(999999);
  });

  it("replaces a failed character call with the stable code, on every method", async () => {
    const transportError = new Error(
      "goroutine 9 [running]: saveengine.(*Engine).GetCharacterStats /Users/private/stats.go:88",
    );
    getSaveCharacters.mockRejectedValue(transportError);
    getCharacterProfile.mockRejectedValue(transportError);
    getCharacterStats.mockRejectedValue(transportError);

    for (const call of [
      () => wailsDesktopBridge.getSaveCharacters("session-1"),
      () => wailsDesktopBridge.getCharacterProfile("session-1", 0),
      () => wailsDesktopBridge.getCharacterStats("session-1", 0),
    ]) {
      const failure = await call().then(
        () => undefined,
        (error: unknown) => error as Error,
      );

      expect(failure?.message).toBe(bridgeFailureCode);
      // An unknown session, an inactive slot and a transport failure are not
      // told apart here: that needs a structured backend error contract.
      expect(failure?.message).not.toContain("goroutine");
      expect(failure?.message).not.toContain("/Users/private");
    }
  });
});

const containerRecord = {
  ownedItemID: "owned-1",
  kind: "item",
  key: "weapon/uchigatana",
  gameID: 0x00bb8000,
  containerSection: "common",
  physicalIndex: 3,
  gaItemHandle: 0x8000000a,
  quantity: 1,
  acquisitionIndex: 42,
};

const inventoryPage = inventory.GetInventoryResult.createFrom({
  saveSessionID: "session-1",
  saveRevision: "  Revision 7  ",
  characterID: 0,
  active: true,
  records: [containerRecord],
  total: 1,
  page: 2,
  pageSize: 30,
});

const storagePage = inventory.GetStorageResult.createFrom({
  saveSessionID: "session-1",
  saveRevision: "  Revision 7  ",
  characterID: 0,
  active: true,
  records: [{ ...containerRecord, ownedItemID: "owned-2", physicalIndex: 7 }],
  total: 1,
  page: 2,
  pageSize: 30,
});

const expectedPage = {
  saveSessionID: "session-1",
  saveRevision: "  Revision 7  ",
  characterID: 0,
  active: true,
  records: [containerRecord],
  total: 1,
  page: 2,
  pageSize: 30,
};

describe("wails items adapter", () => {
  it("passes all five arguments to the Inventory binding in contract order", async () => {
    getInventory.mockResolvedValue(inventoryPage);

    await wailsDesktopBridge.getInventory({
      saveSessionID: "  Session ID  ",
      characterID: 9,
      containerSection: "  Common  ",
      page: 2,
      pageSize: 30,
    });

    // No trimming, no section default, no paging normalisation and no slot
    // range check: the backend owns all of them.
    expect(getInventory).toHaveBeenCalledExactlyOnceWith("  Session ID  ", 9, "  Common  ", 2, 30);
    expect(getStorage).not.toHaveBeenCalled();
  });

  it("passes all five arguments to the Storage binding in contract order", async () => {
    getStorage.mockResolvedValue(storagePage);

    await wailsDesktopBridge.getStorage({
      saveSessionID: "session-1",
      characterID: -1,
      containerSection: "",
      page: 0,
      pageSize: 0,
    });

    expect(getStorage).toHaveBeenCalledExactlyOnceWith("session-1", -1, "", 0, 0);
    expect(getInventory).not.toHaveBeenCalled();
  });

  it("maps every reported Inventory field and nothing else", async () => {
    getInventory.mockResolvedValue(inventoryPage);

    // Exactly the fields the backend reports: no name, no icon, no capacity, no
    // favourite state and no capability is invented here.
    await expect(
      wailsDesktopBridge.getInventory({
        saveSessionID: "session-1",
        characterID: 0,
        containerSection: "common",
        page: 2,
        pageSize: 30,
      }),
    ).resolves.toEqual(expectedPage);
  });

  it("maps every reported Storage field and nothing else", async () => {
    getStorage.mockResolvedValue(storagePage);

    await expect(
      wailsDesktopBridge.getStorage({
        saveSessionID: "session-1",
        characterID: 0,
        containerSection: "common",
        page: 2,
        pageSize: 30,
      }),
    ).resolves.toEqual({
      ...expectedPage,
      records: [{ ...containerRecord, ownedItemID: "owned-2", physicalIndex: 7 }],
    });
  });

  it("carries an inactive slot, an empty page and an opaque revision through", async () => {
    getInventory.mockResolvedValue(
      inventory.GetInventoryResult.createFrom({
        saveSessionID: "session-1",
        saveRevision: "",
        characterID: 4,
        active: false,
        records: [],
        total: 0,
        page: 99,
        pageSize: 30,
      }),
    );

    const page = await wailsDesktopBridge.getInventory({
      saveSessionID: "session-1",
      characterID: 4,
      containerSection: "common",
      page: 99,
      pageSize: 30,
    });

    // An inactive slot is an ordinary result, not an error, and the revision is
    // never generated or replaced by the adapter.
    expect(page).toEqual({
      saveSessionID: "session-1",
      saveRevision: "",
      characterID: 4,
      active: false,
      records: [],
      total: 0,
      page: 99,
      pageSize: 30,
    });
  });

  it("replaces a failed container call with the stable code, on both methods", async () => {
    const transportError = new Error(
      "goroutine 11 [running]: saveengine.(*Engine).GetInventory /Users/private/inventory.go:64",
    );
    getInventory.mockRejectedValue(transportError);
    getStorage.mockRejectedValue(transportError);

    const request = {
      saveSessionID: "session-1",
      characterID: 0,
      containerSection: "common",
      page: 1,
      pageSize: 30,
    };

    for (const call of [
      () => wailsDesktopBridge.getInventory(request),
      () => wailsDesktopBridge.getStorage(request),
    ]) {
      const failure = await call().then(
        () => undefined,
        (error: unknown) => error as Error,
      );

      // An unknown section, an unknown item ID and a transport failure are not
      // told apart here: that needs a structured backend error contract.
      expect(failure?.message).toBe(bridgeFailureCode);
      expect(failure?.message).not.toContain("goroutine");
      expect(failure?.message).not.toContain("/Users/private");
    }
  });
});

const catalogPage = catalog.GetResourcesResult.createFrom({
  resources: [
    { kind: "item", key: "weapon/uchigatana", family: "weapon", name: "Uchigatana" },
    { kind: "item", key: "goods/unnamed", family: "", name: "" },
    { kind: "future_kind", key: "future/key", family: "", name: "Future Resource" },
  ],
  total: 3,
  page: 1,
  pageSize: 50,
});

const catalogRequest = {
  resourceType: "item",
  family: "weapon",
  capability: "upgrade",
  endpointID: "",
  search: "uchi",
  page: 2,
  pageSize: 25,
};

describe("wails catalog adapter", () => {
  it("passes all seven arguments to the binding in contract order", async () => {
    getResources.mockResolvedValue(catalogPage);

    await wailsDesktopBridge.getResources({
      resourceType: "  Item  ",
      family: "  Weapon  ",
      capability: "  Upgrade  ",
      endpointID: "  get_resources  ",
      search: "  Uchi  ",
      page: 0,
      pageSize: 0,
    });

    // No trimming, no recasing, no filter default and no paging normalisation:
    // every one of them is the backend's contract.
    expect(getResources).toHaveBeenCalledExactlyOnceWith(
      "  Item  ",
      "  Weapon  ",
      "  Upgrade  ",
      "  get_resources  ",
      "  Uchi  ",
      0,
      0,
    );
  });

  it("maps every reported catalog field and every row, and nothing else", async () => {
    getResources.mockResolvedValue(catalogPage);

    // Exactly the four fields the backend reports per row: no icon, no
    // description, no capability, no provenance, no limit, no favourite state
    // and no safety level is invented here.
    await expect(wailsDesktopBridge.getResources(catalogRequest)).resolves.toEqual({
      resources: [
        { kind: "item", key: "weapon/uchigatana", family: "weapon", name: "Uchigatana" },
        { kind: "item", key: "goods/unnamed", family: "", name: "" },
        { kind: "future_kind", key: "future/key", family: "", name: "Future Resource" },
      ],
      total: 3,
      page: 1,
      pageSize: 50,
    });
  });

  it("keeps an empty name and an empty family instead of building a fallback", async () => {
    getResources.mockResolvedValue(catalogPage);

    const page = await wailsDesktopBridge.getResources(catalogRequest);

    // The key is never promoted to a name, and an unknown family stays empty.
    expect(page.resources[1].name).toBe("");
    expect(page.resources[1].family).toBe("");
    expect(page.resources[1].key).toBe("goods/unnamed");
  });

  it("reports the page and page size the backend served, not the requested ones", async () => {
    getResources.mockResolvedValue(catalogPage);

    const page = await wailsDesktopBridge.getResources({
      ...catalogRequest,
      page: 0,
      pageSize: 0,
    });

    // Zero paging resolves to the backend defaults; the adapter echoes neither
    // the request nor a default of its own.
    expect(page.page).toBe(1);
    expect(page.pageSize).toBe(50);
    expect(page.total).toBe(3);
  });

  it("carries an empty page through as an ordinary result", async () => {
    getResources.mockResolvedValue(
      catalog.GetResourcesResult.createFrom({
        resources: [],
        total: 0,
        page: 99,
        pageSize: 25,
      }),
    );

    await expect(wailsDesktopBridge.getResources(catalogRequest)).resolves.toEqual({
      resources: [],
      total: 0,
      page: 99,
      pageSize: 25,
    });
  });

  it("replaces a failed catalog call with the same stable code as every other port", async () => {
    getResources.mockRejectedValue(
      new Error("goroutine 13 [running]: catalog.GetResources /Users/private/get_resources.go:96"),
    );

    const failure = await wailsDesktopBridge.getResources(catalogRequest).then(
      () => undefined,
      (error: unknown) => error as Error,
    );

    // A rejected filter, an unknown resource type and a transport failure are
    // not told apart here: that needs a structured backend error contract.
    expect(failure?.message).toBe(bridgeFailureCode);
    expect(failure?.message).not.toContain("goroutine");
    expect(failure?.message).not.toContain("/Users/private");
  });
});

const fullProvenance = {
  source: "legacy_db_data",
  method: "regulation_row",
  table: "EquipParamWeapon",
  row: "1000000",
  field: "maxLevel",
};

/**
 * The backend omits an empty `table`, `row` or `field`, so this is what an
 * unresolved fact actually arrives as: a complete record with three absent
 * parts, never a missing provenance.
 */
const sparseProvenance = { source: "legacy_db_data", method: "unresolved" };

/** The same record as it must arrive above the port: absent parts become empty. */
const emptyPartsProvenance = {
  source: "legacy_db_data",
  method: "unresolved",
  table: "",
  row: "",
  field: "",
};

const knownFact = (value: unknown) => ({ known: true, value, provenance: fullProvenance });
const unknownFact = (value: unknown) => ({ known: false, value, provenance: sparseProvenance });

/**
 * One generated item resource covering every shape the projection has to
 * survive: resolved facts, unresolved ones keeping their raw value, empty
 * strings, zeros, absent optional facts, present optional ones, all five
 * capabilities and a capability the backend reports with `rules` null.
 */
const itemResource = catalog.GetResourceResult.createFrom({
  resource: {
    kind: "item",
    key: "000F4240",
    item: {
      gameID: knownFact(1000000),
      family: knownFact("weapon"),
      category: unknownFact(""),
      subcategory: knownFact(""),
      presentation: {
        name: knownFact("Dagger"),
        caption: unknownFact(""),
        description: knownFact("A small dagger."),
        location: unknownFact(""),
        iconPath: knownFact("MENU_Knowledge_00100.png"),
        textMetadata: {
          captionSource: knownFact("caption"),
          descriptionSource: knownFact("description"),
          locationSource: knownFact("location"),
          dlcSource: knownFact("base"),
          notes: knownFact("ignored"),
        },
      },
      storage: {
        recordMode: knownFact("separate_instances"),
        maxInventory: knownFact(600),
        safeModeMaxInventory: knownFact(99),
        maxStorage: unknownFact(0),
        "maxStorage-sfv": knownFact(0),
      },
      safety: {
        cutContent: knownFact(false),
        banRisk: knownFact(true),
        dlc: knownFact(false),
        noDatabase: unknownFact(false),
        scalesWithNG: knownFact(false),
        preOrder: knownFact(false),
      },
      capabilities: {
        upgrade: {
          known: true,
          enabled: true,
          rules: { model: "standard", maxLevel: 25, "maxLevel-sfv": knownFact(10) },
          provenance: fullProvenance,
          rulesEvidence: [fullProvenance],
        },
        infusion: {
          known: true,
          enabled: true,
          rules: { allowedAffinities: ["standard", "heavy"] },
          provenance: fullProvenance,
        },
        ashOfWarMount: {
          known: true,
          enabled: true,
          rules: { mode: "custom", weaponType: "dagger", compatibilityBit: 0 },
          provenance: sparseProvenance,
        },
        stack: { known: false, enabled: false, rules: null, provenance: sparseProvenance },
        equipment: {
          known: true,
          enabled: true,
          rules: { allowedSlots: [] },
          provenance: fullProvenance,
        },
      },
      // Everything below is deliberately outside the application contract of
      // this step and must not reach the port result.
      acquisition: {},
      modifiers: {},
      links: {},
      variants: [{ gameID: knownFact(1000100) }],
      aliases: [],
      unlocks: [],
      relatedTechnicalRecords: [],
      sourceRecords: [{ table: "EquipParamWeapon", rowID: 1000000 }],
      weapon: { physicalAttack: knownFact(73) },
    },
  },
});

describe("wails catalog resource adapter", () => {
  it("passes the kind and the key to the binding exactly as given", async () => {
    getResource.mockResolvedValue(itemResource);

    await wailsDesktopBridge.getResource({ kind: "  Item  ", key: " 000f4240 " });

    // No trimming, no recasing, no alias and no kind default: every one of them
    // is the backend's contract.
    expect(getResource).toHaveBeenCalledExactlyOnceWith("  Item  ", " 000f4240 ");
  });

  it("passes an empty kind and an empty key through instead of skipping the call", async () => {
    getResource.mockResolvedValue(itemResource);

    await wailsDesktopBridge.getResource({ kind: "", key: "" });

    // The empty pair is a real request the backend rejects; the adapter must
    // not decide that on its own.
    expect(getResource).toHaveBeenCalledExactlyOnceWith("", "");
  });

  it("maps the identity and the common item fields, and nothing else", async () => {
    getResource.mockResolvedValue(itemResource);

    await expect(
      wailsDesktopBridge.getResource({ kind: "item", key: "000F4240" }),
    ).resolves.toEqual({
      kind: "item",
      key: "000F4240",
      item: {
        gameID: { known: true, value: 1000000, provenance: fullProvenance },
        family: { known: true, value: "weapon", provenance: fullProvenance },
        category: { known: false, value: "", provenance: emptyPartsProvenance },
        subcategory: { known: true, value: "", provenance: fullProvenance },
        presentation: {
          name: { known: true, value: "Dagger", provenance: fullProvenance },
          caption: { known: false, value: "", provenance: emptyPartsProvenance },
          description: { known: true, value: "A small dagger.", provenance: fullProvenance },
          location: { known: false, value: "", provenance: emptyPartsProvenance },
          iconPath: {
            known: true,
            value: "MENU_Knowledge_00100.png",
            provenance: fullProvenance,
          },
        },
        storage: {
          recordMode: { known: true, value: "separate_instances", provenance: fullProvenance },
          maxInventory: { known: true, value: 600, provenance: fullProvenance },
          safeModeMaxInventory: { known: true, value: 99, provenance: fullProvenance },
          maxInventorySFV: null,
          maxStorage: { known: false, value: 0, provenance: emptyPartsProvenance },
          safeModeMaxStorage: null,
          maxStorageSFV: { known: true, value: 0, provenance: fullProvenance },
        },
        safety: {
          cutContent: { known: true, value: false, provenance: fullProvenance },
          banRisk: { known: true, value: true, provenance: fullProvenance },
          dlc: { known: true, value: false, provenance: fullProvenance },
          noDatabase: { known: false, value: false, provenance: emptyPartsProvenance },
          scalesWithNG: { known: true, value: false, provenance: fullProvenance },
          preOrder: { known: true, value: false, provenance: fullProvenance },
        },
        capabilities: {
          upgrade: {
            known: true,
            enabled: true,
            rules: {
              model: "standard",
              maxLevel: 25,
              maxLevelSFV: { known: true, value: 10, provenance: fullProvenance },
            },
            provenance: fullProvenance,
          },
          infusion: {
            known: true,
            enabled: true,
            rules: { allowedAffinities: ["standard", "heavy"] },
            provenance: fullProvenance,
          },
          ashOfWarMount: {
            known: true,
            enabled: true,
            rules: { mode: "custom", weaponType: "dagger", compatibilityBit: 0 },
            provenance: emptyPartsProvenance,
          },
          stack: {
            known: false,
            enabled: false,
            rules: null,
            provenance: emptyPartsProvenance,
          },
          equipment: {
            known: true,
            enabled: true,
            rules: { allowedSlots: [] },
            provenance: fullProvenance,
          },
        },
      },
    });
  });

  it("keeps an unknown fact, an empty string and a zero exactly as reported", async () => {
    getResource.mockResolvedValue(itemResource);

    const detail = await wailsDesktopBridge.getResource({ kind: "item", key: "000F4240" });

    // An unresolved fact keeps its raw value: no placeholder, no key promoted to
    // a name, no zero turned into an absent limit.
    expect(detail.item?.category).toEqual({
      known: false,
      value: "",
      provenance: emptyPartsProvenance,
    });
    // A known fact whose value happens to be empty or zero stays known.
    expect(detail.item?.subcategory.known).toBe(true);
    expect(detail.item?.subcategory.value).toBe("");
    expect(detail.item?.storage.maxStorage.value).toBe(0);
    expect(detail.item?.capabilities.ashOfWarMount.rules?.compatibilityBit).toBe(0);
    expect(detail.item?.capabilities.equipment.rules?.allowedSlots).toEqual([]);
  });

  it("maps an absent optional fact to null instead of a zero-valued fact", async () => {
    getResource.mockResolvedValue(itemResource);

    const detail = await wailsDesktopBridge.getResource({ kind: "item", key: "000F4240" });

    // Absent and zero are different answers and must stay distinguishable.
    expect(detail.item?.storage.maxInventorySFV).toBeNull();
    expect(detail.item?.storage.safeModeMaxStorage).toBeNull();
    expect(detail.item?.storage.maxStorageSFV).not.toBeNull();
  });

  it("maps a capability reported without rules to null rules", async () => {
    getResource.mockResolvedValue(itemResource);

    const detail = await wailsDesktopBridge.getResource({ kind: "item", key: "000F4240" });

    // No rule set is invented, and `enabled` is not derived from the rules.
    expect(detail.item?.capabilities.stack.rules).toBeNull();
    expect(detail.item?.capabilities.stack.enabled).toBe(false);
    expect(detail.item?.capabilities.stack.known).toBe(false);
  });

  it("reports safety as six independent facts without a derived level", async () => {
    getResource.mockResolvedValue(itemResource);

    const detail = await wailsDesktopBridge.getResource({ kind: "item", key: "000F4240" });

    expect(Object.keys(detail.item?.safety ?? {})).toEqual([
      "cutContent",
      "banRisk",
      "dlc",
      "noDatabase",
      "scalesWithNG",
      "preOrder",
    ]);
    // A ban-risk item gets no severity, no ordering and no verdict here.
    expect(JSON.stringify(detail)).not.toMatch(/warning|critical|riskLevel|safetyLevel/i);
  });

  it("keeps item null for a resource of another kind", async () => {
    getResource.mockResolvedValue(
      catalog.GetResourceResult.createFrom({
        resource: {
          kind: "class",
          key: "0",
          class: { startingClassID: knownFact(0), name: knownFact("Vagabond") },
        },
      }),
    );

    // The identity is exact and no other document is mapped onto the item shape.
    await expect(wailsDesktopBridge.getResource({ kind: "class", key: "0" })).resolves.toEqual({
      kind: "class",
      key: "0",
      item: null,
    });
  });

  it("replaces a failed detail call with the same stable code as every other port", async () => {
    getResource.mockRejectedValue(
      new Error(
        'unknown resource key " 000F4240" in kind "item" /Users/private/get_resource.go:96',
      ),
    );

    const failure = await wailsDesktopBridge.getResource({ kind: "item", key: " 000F4240" }).then(
      () => undefined,
      (error: unknown) => error as Error,
    );

    // An unknown kind, an unknown key and a transport failure are not told apart
    // here: that needs a structured backend error contract.
    expect(failure?.message).toBe(bridgeFailureCode);
    expect(failure?.message).not.toContain("unknown resource key");
    expect(failure?.message).not.toContain("/Users/private");
  });
});

/**
 * One generated variant list covering what the projection has to survive: two
 * variants in a fixed catalog order, resolved facts, an unresolved one keeping
 * its raw zero and empty string, and the `data` and `sourceRecords` blocks that
 * are deliberately outside the application contract of this step.
 */
const itemVariants = catalog.GetItemVariantsResult.createFrom({
  variants: [
    {
      gameID: knownFact(1000100),
      kind: knownFact("affinity"),
      affinity: knownFact("heavy"),
      upgradeLevel: knownFact(0),
      sourceRowID: knownFact(1000100),
      data: { family: knownFact("weapon"), weapon: { physicalAttack: knownFact(80) } },
      sourceRecords: [{ table: "EquipParamWeapon", rowID: 1000100 }],
    },
    {
      gameID: knownFact(1000001),
      kind: unknownFact(""),
      affinity: unknownFact(""),
      upgradeLevel: knownFact(1),
      sourceRowID: unknownFact(0),
      data: { family: knownFact("weapon") },
      sourceRecords: [],
    },
  ],
});

describe("wails catalog item variants adapter", () => {
  it("passes the kind and the key to the binding exactly as given", async () => {
    getItemVariants.mockResolvedValue(itemVariants);

    await wailsDesktopBridge.getItemVariants({ kind: "  Item  ", key: " 000f4240 " });

    // No trimming, no recasing, no alias and no kind default: every one of them
    // is the backend's contract.
    expect(getItemVariants).toHaveBeenCalledExactlyOnceWith("  Item  ", " 000f4240 ");
  });

  it("passes an empty kind and an empty key through instead of skipping the call", async () => {
    getItemVariants.mockResolvedValue(itemVariants);

    await wailsDesktopBridge.getItemVariants({ kind: "", key: "" });

    expect(getItemVariants).toHaveBeenCalledExactlyOnceWith("", "");
  });

  it("maps the five variant facts with their provenance, and nothing else", async () => {
    getItemVariants.mockResolvedValue(itemVariants);

    await expect(
      wailsDesktopBridge.getItemVariants({ kind: "item", key: "000F4240" }),
    ).resolves.toEqual({
      variants: [
        {
          gameID: { known: true, value: 1000100, provenance: fullProvenance },
          kind: { known: true, value: "affinity", provenance: fullProvenance },
          affinity: { known: true, value: "heavy", provenance: fullProvenance },
          upgradeLevel: { known: true, value: 0, provenance: fullProvenance },
          sourceRowID: { known: true, value: 1000100, provenance: fullProvenance },
        },
        {
          gameID: { known: true, value: 1000001, provenance: fullProvenance },
          kind: { known: false, value: "", provenance: emptyPartsProvenance },
          affinity: { known: false, value: "", provenance: emptyPartsProvenance },
          upgradeLevel: { known: true, value: 1, provenance: fullProvenance },
          sourceRowID: { known: false, value: 0, provenance: emptyPartsProvenance },
        },
      ],
    });
  });

  it("keeps the variant document data and the source records out of the port result", async () => {
    getItemVariants.mockResolvedValue(itemVariants);

    const result = await wailsDesktopBridge.getItemVariants({ kind: "item", key: "000F4240" });

    for (const variant of result.variants) {
      expect(Object.keys(variant)).toEqual([
        "gameID",
        "kind",
        "affinity",
        "upgradeLevel",
        "sourceRowID",
      ]);
    }
    // Neither the weapon statistics nor the parameter records reach the port.
    // `EquipParamWeapon` is not searched for: it is a legitimate part of the
    // provenance the five facts keep.
    expect(JSON.stringify(result)).not.toContain("physicalAttack");
    expect(JSON.stringify(result)).not.toContain("rowID");
  });

  it("keeps an unknown fact, an empty string and a zero exactly as reported", async () => {
    getItemVariants.mockResolvedValue(itemVariants);

    const result = await wailsDesktopBridge.getItemVariants({ kind: "item", key: "000F4240" });

    // No placeholder affinity, no upgrade level derived from the kind and no
    // zero row identifier turned into an absent one.
    expect(result.variants[1].affinity).toEqual({
      known: false,
      value: "",
      provenance: emptyPartsProvenance,
    });
    expect(result.variants[1].sourceRowID.value).toBe(0);
    // A known fact whose value happens to be zero stays known.
    expect(result.variants[0].upgradeLevel).toEqual({
      known: true,
      value: 0,
      provenance: fullProvenance,
    });
  });

  it("keeps the catalog order the backend reported", async () => {
    getItemVariants.mockResolvedValue(itemVariants);

    const result = await wailsDesktopBridge.getItemVariants({ kind: "item", key: "000F4240" });

    // No sorting by game identifier, affinity or upgrade level happens here.
    expect(result.variants.map((variant) => variant.gameID.value)).toEqual([1000100, 1000001]);
  });

  it("maps an item without variants to an empty list rather than a failure", async () => {
    getItemVariants.mockResolvedValue(catalog.GetItemVariantsResult.createFrom({ variants: [] }));

    // An item that carries no variant is a valid answer, and the base item is
    // never synthesised into one.
    await expect(
      wailsDesktopBridge.getItemVariants({ kind: "item", key: "10009C40" }),
    ).resolves.toEqual({ variants: [] });
  });

  it("replaces a failed variants call with the same stable code as every other port", async () => {
    getItemVariants.mockRejectedValue(
      new Error(
        'resource kind "class" has no item variants /Users/private/get_item_variants.go:96',
      ),
    );

    const failure = await wailsDesktopBridge.getItemVariants({ kind: "class", key: "0" }).then(
      () => undefined,
      (error: unknown) => error as Error,
    );

    expect(failure?.message).toBe(bridgeFailureCode);
    expect(failure?.message).not.toContain("has no item variants");
    expect(failure?.message).not.toContain("/Users/private");
  });
});
