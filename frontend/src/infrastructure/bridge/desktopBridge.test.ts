import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  CloseSave,
  GetApplicationInfo,
  GetLoadedSave,
  LoadSave,
} from "../../../wailsjs/go/desktop/Bridge";
import { application, saveengine } from "../../../wailsjs/go/models";
import { bridgeFailureCode, wailsDesktopBridge } from "./desktopBridge";

vi.mock("../../../wailsjs/go/desktop/Bridge", () => ({
  CloseSave: vi.fn(),
  GetApplicationInfo: vi.fn(),
  GetLoadedSave: vi.fn(),
  LoadSave: vi.fn(),
}));

const getApplicationInfo = vi.mocked(GetApplicationInfo);
const getLoadedSave = vi.mocked(GetLoadedSave);
const loadSave = vi.mocked(LoadSave);
const closeSave = vi.mocked(CloseSave);

beforeEach(() => {
  getApplicationInfo.mockReset();
  getLoadedSave.mockReset();
  loadSave.mockReset();
  closeSave.mockReset();
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
