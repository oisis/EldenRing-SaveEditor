import { describe, expect, it } from "vitest";
import { appearancePresetAssetURL, catalogAssetURL } from "./catalogAssetURL";

describe("catalogAssetURL", () => {
  it("maps a validated embedded item icon onto the Wails asset route", () => {
    expect(catalogAssetURL("assets/icons/items/melee_armaments/dagger.png")).toBe(
      "/catalog-assets/assets/icons/items/melee_armaments/dagger.png",
    );
    expect(catalogAssetURL("assets/appearance/geralt-of-rivia-the-witcher.jpg")).toBe(
      "/catalog-assets/assets/appearance/geralt-of-rivia-the-witcher.jpg",
    );
    expect(appearancePresetAssetURL("geralt-of-rivia-the-witcher.jpg")).toBe(
      "/catalog-assets/assets/appearance/geralt-of-rivia-the-witcher.jpg",
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
      "assets/appearance/../secret.jpg",
      "assets/appearance/geralt\\witcher.jpg",
      "catalog.json",
    ]) {
      expect(catalogAssetURL(value)).toBeUndefined();
    }

    for (const value of [
      "",
      "geralt/witcher.jpg",
      "geralt\\witcher.jpg",
      "../secret.jpg",
      "nested/preset.jpg",
      "invalid.png",
    ]) {
      expect(appearancePresetAssetURL(value)).toBeUndefined();
    }
  });
});
