import { Trans } from "@lingui/react/macro";
import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { App } from "../App";
import { makePort, renderApp, stubApplicationInfo } from "../test/renderWithProviders";
import { activateLocale } from "./i18n";
import { locales, sourceLocale } from "./locales";

/**
 * Test-only fixture. It carries the deliberately untranslated source string
 * that proves the English fallback; that message must never appear in the
 * production UI.
 */
function Sample() {
  return (
    <div>
      <p data-testid="translated">
        <Trans>Backend</Trans>
      </p>
      <p data-testid="untranslated">
        <Trans>This message is intentionally left untranslated in Polish.</Trans>
      </p>
    </div>
  );
}

describe("localization", () => {
  it("keeps one source of truth for the supported locales", () => {
    expect(sourceLocale).toBe("en");
    expect(locales).toEqual(["en", "pl"]);
  });

  it("renders English source strings", async () => {
    await renderApp(<Sample />, { locale: "en" });

    expect(screen.getByTestId("translated")).toHaveTextContent("Backend");
  });

  it("switches the language from the UI, including the application chrome", async () => {
    await renderApp(<App />, { locale: "en" });

    expect(screen.getByRole("navigation", { name: "Language" })).toBeInTheDocument();
    expect(screen.getByRole("navigation", { name: "Theme" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Backend" })).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Polish" }));

    expect(await screen.findByRole("heading", { name: "Backend aplikacji" })).toBeInTheDocument();
    expect(screen.getByRole("navigation", { name: "Język" })).toBeInTheDocument();
    expect(screen.getByRole("navigation", { name: "Motyw" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "polski" })).toBeInTheDocument();
  });

  it("keeps the semantic document language in sync with the active locale", async () => {
    await activateLocale("en");
    expect(document.documentElement.lang).toBe("en");

    await activateLocale("pl");
    expect(document.documentElement.lang).toBe("pl");

    await activateLocale("en");
    expect(document.documentElement.lang).toBe("en");
  });

  it("does not touch query keys, cache or data when the language changes", async () => {
    const getApplicationInfo = vi.fn(makePort().getApplicationInfo);

    await renderApp(<App />, { locale: "en", port: { getApplicationInfo } });
    expect(await screen.findByText(stubApplicationInfo.version)).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Polish" }));
    expect(await screen.findByRole("heading", { name: "Backend aplikacji" })).toBeInTheDocument();

    // Same query key, same cache entry, same data: no refetch was triggered.
    expect(getApplicationInfo).toHaveBeenCalledTimes(1);
    expect(screen.getByText(stubApplicationInfo.version)).toBeInTheDocument();
  });

  it("falls back to English for a message missing in Polish", async () => {
    await activateLocale("pl");
    await renderApp(<Sample />, { locale: "pl" });

    expect(screen.getByTestId("translated")).toHaveTextContent("Backend aplikacji");
    expect(screen.getByTestId("untranslated")).toHaveTextContent(
      "This message is intentionally left untranslated in Polish.",
    );
  });
});
