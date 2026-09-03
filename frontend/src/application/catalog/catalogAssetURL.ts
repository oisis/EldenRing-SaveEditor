const catalogAssetURLPrefix = "/catalog-assets/";
const itemIconPathPrefix = "assets/icons/items/";
const appearanceAssetPathPrefix = "assets/appearance/";

/**
 * Turns validated catalog asset metadata into the Wails AssetServer URL. Unknown or
 * unsafe metadata returns undefined and can never become a host filesystem URL.
 */
export function catalogAssetURL(assetPath: string): string | undefined {
  const isItemIcon = assetPath.startsWith(itemIconPathPrefix) && assetPath.endsWith(".png");
  const isAppearance =
    assetPath.startsWith(appearanceAssetPathPrefix) && assetPath.endsWith(".jpg");
  if ((!isItemIcon && !isAppearance) || assetPath.includes("\\")) {
    return undefined;
  }
  const segments = assetPath.split("/");
  if (segments.some((segment) => segment === "" || segment === "." || segment === "..")) {
    return undefined;
  }
  return `${catalogAssetURLPrefix}${segments.map(encodeURIComponent).join("/")}`;
}

export function appearancePresetAssetURL(imageFileName: string): string | undefined {
  if (!imageFileName || imageFileName.includes("/") || imageFileName.includes("\\")) {
    return undefined;
  }
  return catalogAssetURL(`${appearanceAssetPathPrefix}${imageFileName}`);
}
