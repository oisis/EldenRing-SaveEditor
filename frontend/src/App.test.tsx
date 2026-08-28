import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { App } from "./App";
import { renderApp } from "./test/renderWithProviders";
import { themeClassNames } from "./ui/tokens/themes.css";

/** Every theme class currently present on the document element. */
function activeThemeClasses(): string[] {
  return Object.values(themeClassNames).filter((className) =>
    document.documentElement.classList.contains(className),
  );
}

describe("App", () => {
  it("scopes the theme to the document element so body and portals inherit it", async () => {
    await renderApp(<App />, { locale: "en" });

    expect(activeThemeClasses()).toEqual([themeClassNames.light]);

    fireEvent.click(screen.getByRole("button", { name: "Dark" }));

    // Exactly one theme class stays active and the previous one is removed.
    expect(activeThemeClasses()).toEqual([themeClassNames.dark]);

    fireEvent.click(screen.getByRole("button", { name: "Elden Ring" }));

    expect(activeThemeClasses()).toEqual([themeClassNames["elden-ring"]]);

    // The screen itself carries no theme class any more.
    expect(activeThemeClasses().some((c) => screen.getByRole("main").classList.contains(c))).toBe(
      false,
    );
  });

  it("renders no test-only content", async () => {
    const { container } = await renderApp(<App />, { locale: "en" });

    expect(container.textContent).not.toContain(
      "This message is intentionally left untranslated in Polish.",
    );
  });
});
