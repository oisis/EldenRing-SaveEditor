import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  CloseSave,
  GetApplicationInfo,
  GetCharacterProfile,
  GetCharacterStats,
  GetLoadedSave,
  GetSaveCharacters,
  LoadSave,
} from "../../../wailsjs/go/desktop/Bridge";
import { application, saveengine } from "../../../wailsjs/go/models";
import { bridgeFailureCode, wailsDesktopBridge } from "./desktopBridge";

vi.mock("../../../wailsjs/go/desktop/Bridge", () => ({
  CloseSave: vi.fn(),
  GetApplicationInfo: vi.fn(),
  GetCharacterProfile: vi.fn(),
  GetCharacterStats: vi.fn(),
  GetLoadedSave: vi.fn(),
  GetSaveCharacters: vi.fn(),
  LoadSave: vi.fn(),
}));

const getApplicationInfo = vi.mocked(GetApplicationInfo);
const getLoadedSave = vi.mocked(GetLoadedSave);
const loadSave = vi.mocked(LoadSave);
const closeSave = vi.mocked(CloseSave);
const getSaveCharacters = vi.mocked(GetSaveCharacters);
const getCharacterProfile = vi.mocked(GetCharacterProfile);
const getCharacterStats = vi.mocked(GetCharacterStats);

beforeEach(() => {
  getApplicationInfo.mockReset();
  getLoadedSave.mockReset();
  loadSave.mockReset();
  closeSave.mockReset();
  getSaveCharacters.mockReset();
  getCharacterProfile.mockReset();
  getCharacterStats.mockReset();
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
