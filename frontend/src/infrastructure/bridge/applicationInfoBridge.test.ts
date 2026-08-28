import { beforeEach, describe, expect, it, vi } from "vitest";
import { GetApplicationInfo } from "../../../wailsjs/go/desktop/Bridge";
import { application } from "../../../wailsjs/go/models";
import { bridgeFailureCode, wailsApplicationInfoBridge } from "./applicationInfoBridge";

vi.mock("../../../wailsjs/go/desktop/Bridge", () => ({
  GetApplicationInfo: vi.fn(),
}));

const getApplicationInfo = vi.mocked(GetApplicationInfo);

beforeEach(() => {
  getApplicationInfo.mockReset();
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

    await expect(wailsApplicationInfoBridge.getApplicationInfo()).resolves.toEqual({
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

    const info = await wailsApplicationInfoBridge.getApplicationInfo();

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

    await expect(wailsApplicationInfoBridge.getApplicationInfo()).rejects.toThrow(
      new Error(bridgeFailureCode),
    );
  });
});
