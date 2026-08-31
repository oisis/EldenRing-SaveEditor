import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";
import { App } from "../../App";
import { renderApp, stubSaveSession } from "../../test/renderWithProviders";
import { fileNameFromPath } from "./AppShell";

describe("AppShell", () => {
  it("renders the accepted module order and keeps file actions on Home", async () => {
    await renderApp(<App />);

    const navigation = screen.getByRole("navigation", { name: "Modules" });
    expect(
      within(navigation)
        .getAllByRole("button")
        .map((button) => button.textContent),
    ).toEqual(["Home", "Character", "Items", "Equipment", "World", "Advanced", "Tools"]);
    expect(screen.getByRole("button", { name: "Open Save" })).toBeEnabled();
    expect(screen.getByRole("button", { name: "Save" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Save As" })).toBeDisabled();

    await userEvent.click(within(navigation).getByRole("button", { name: "Character" }));

    expect(screen.getByText(/Character workspace will be implemented/i)).toBeVisible();
    expect(screen.queryByRole("button", { name: "Open Save" })).toBeNull();
  });

  it("places the opened file and characters in the global sidebar", async () => {
    await renderApp(<App />);

    await userEvent.click(screen.getByRole("button", { name: "Open Save" }));

    expect(await screen.findByText("ER0000.sl2")).toBeVisible();
    expect(screen.getByText("Tarnished")).toBeVisible();
    expect(screen.getByText(stubSaveSession.sourcePath)).toBeVisible();
    expect(screen.getByText("Saved")).toBeVisible();
    expect(screen.getAllByRole("complementary", { name: "Characters" })).toHaveLength(1);
  });

  it("opens and closes the centred diagnostic console from the bottom bar", async () => {
    await renderApp(<App />);

    const toggle = screen.getByRole("button", { name: /Console.*No live messages/i });
    expect(toggle).toHaveAttribute("aria-expanded", "false");

    await userEvent.click(toggle);

    expect(screen.getByRole("region", { name: "Diagnostic console" })).toBeVisible();
    expect(toggle).toHaveAttribute("aria-expanded", "true");

    await userEvent.click(screen.getByRole("button", { name: "Close" }));
    await waitFor(() =>
      expect(screen.queryByRole("region", { name: "Diagnostic console" })).toBeNull(),
    );
  });

  it("keeps Item Database reachable without an open save", async () => {
    await renderApp(<App />);

    await userEvent.click(screen.getByRole("button", { name: "Items" }));
    await userEvent.click(screen.getByRole("button", { name: "Item Database" }));

    expect(await screen.findByRole("region", { name: "Item Database" })).toBeVisible();
    expect(screen.getByRole("searchbox", { name: "Search items" })).toBeVisible();
  });
});

describe("fileNameFromPath", () => {
  it.each([
    ["/Users/Tarnished/Elden Ring/ER0000.sl2", "ER0000.sl2"],
    [String.raw`C:\\Users\\Tarnished\\ER0000.sl2`, "ER0000.sl2"],
    ["ER0000.sl2", "ER0000.sl2"],
    ["", ""],
  ])("presents %s without changing the source path", (source, expected) => {
    expect(fileNameFromPath(source)).toBe(expected);
  });
});
