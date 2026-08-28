import { describe, expect, it } from "vitest";
import { tokens } from "./contract.css";
import { darkTheme, eldenRingTheme, lightTheme, themeClassNames, themeNames } from "./themes.css";

/** Every token the contract declares, as flat `color.accent`-style paths. */
function tokenPaths(node: unknown, prefix = ""): string[] {
  if (typeof node === "string") {
    return [prefix];
  }
  return Object.entries(node as Record<string, unknown>).flatMap(([key, value]) =>
    tokenPaths(value, prefix ? `${prefix}.${key}` : key),
  );
}

describe("themes", () => {
  it("ships exactly the three themes of the first release", () => {
    expect(themeNames).toEqual(["light", "dark", "elden-ring"]);
    expect(Object.values(themeClassNames)).toEqual([lightTheme, darkTheme, eldenRingTheme]);
  });

  it("gives every theme its own class so themes cannot share values by accident", () => {
    expect(new Set(Object.values(themeClassNames)).size).toBe(themeNames.length);
    for (const className of Object.values(themeClassNames)) {
      expect(className.trim()).not.toBe("");
    }
  });

  it("substitutes values for the same token contract in every theme", () => {
    // A theme that dropped or invented a token would not compile, but this
    // pins the contract so a silently emptied token set is caught too.
    expect(tokenPaths(tokens)).toContain("color.accent");
    expect(tokenPaths(tokens)).toContain("motion.fast");
    expect(tokenPaths(tokens).length).toBeGreaterThan(20);
  });
});
