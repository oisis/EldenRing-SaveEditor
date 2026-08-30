const catalogAssetURLPrefix = "/catalog-assets/";
const itemIconPathPrefix = "assets/icons/items/";

/**
 * Turns validated item-icon metadata into the Wails AssetServer URL. Unknown or
 * unsafe metadata returns undefined and can never become a host filesystem URL.
 */
export function catalogAssetURL(iconPath: string): string | undefined {
  if (
    !iconPath.startsWith(itemIconPathPrefix) ||
    !iconPath.endsWith(".png") ||
    iconPath.includes("\\")
  ) {
    return undefined;
  }
  const segments = iconPath.split("/");
  if (segments.some((segment) => segment === "" || segment === "." || segment === "..")) {
    return undefined;
  }
  return `${catalogAssetURLPrefix}${segments.map(encodeURIComponent).join("/")}`;
}
