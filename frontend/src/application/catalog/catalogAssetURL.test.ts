import { describe, expect, it } from "vitest";
import { catalogAssetURL } from "./catalogAssetURL";

describe("catalogAssetURL", () => {
  it("maps a validated embedded item icon onto the Wails asset route", () => {
    expect(catalogAssetURL("assets/icons/items/melee_armaments/dagger.png")).toBe(
      "/catalog-assets/assets/icons/items/melee_armaments/dagger.png",
    );
  });

  it("encodes each path segment without encoding its separators", () => {
    expect(catalogAssetURL("assets/icons/items/key items/item #1.png")).toBe(
      "/catalog-assets/assets/icons/items/key%20items/item%20%231.png",
    );
  });

  it("rejects unknown, non-icon and traversal metadata", () => {
    for (const value of [
      "",
      "/assets/icons/items/dagger.png",
      "assets/icons/items/../catalog.json.png",
      "assets/icons/items/melee_armaments\\dagger.png",
      "assets/icons/items/dagger.webp",
      "catalog.json",
    ]) {
      expect(catalogAssetURL(value)).toBeUndefined();
    }
  });
});
