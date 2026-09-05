import { act, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { App } from "../../App";
import type { DiagnosticEventPage } from "../../application/settings/settingsPort";
import {
  makeSaveSessionPort,
  makeSettingsPort,
  renderApp,
  stubSaveSession,
} from "../../test/renderWithProviders";
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

  it("opens Review Changes from the global operation toolbar", async () => {
    await renderApp(<App />);
    await userEvent.click(screen.getByRole("button", { name: "Open Save" }));
    await screen.findByText(stubSaveSession.sourcePath);

    const changes = screen.getByRole("button", { name: "Changes" });
    expect(changes).toBeEnabled();
    await userEvent.click(changes);

    expect(await screen.findByRole("dialog", { name: "Review Changes" })).toBeVisible();
    expect(screen.getByText("Validation passed.")).toBeVisible();
    expect(screen.getByRole("button", { name: "Save As" })).toBeEnabled();
  });

  it("requires Save As for a temporary session", async () => {
    const temporary = { ...stubSaveSession, sourceKind: "temporary" as const };
    await renderApp(<App />, {
      saveSessionPort: makeSaveSessionPort({
        loadSave: () => Promise.resolve(temporary),
        getLoadedSave: () => Promise.resolve(temporary),
      }),
    });
    await userEvent.click(screen.getByRole("button", { name: "Open Save" }));
    await screen.findByText(stubSaveSession.sourcePath);

    expect(screen.getByRole("button", { name: "Save" })).toBeDisabled();
    await userEvent.click(screen.getByRole("button", { name: "Changes" }));

    const dialog = await screen.findByRole("dialog", { name: "Review Changes" });
    expect(within(dialog).getByRole("button", { name: "Save" })).toBeDisabled();
    expect(within(dialog).getByRole("button", { name: "Save As" })).toBeEnabled();
  });

  it("requires an explicit decision before closing a dirty session", async () => {
    const dirty = { ...stubSaveSession, unsavedChanges: true };
    const closeSave = vi.fn(() => Promise.resolve());
    await renderApp(<App />, {
      saveSessionPort: makeSaveSessionPort({
        closeSave,
        loadSave: () => Promise.resolve(dirty),
        getLoadedSave: () => Promise.resolve(dirty),
      }),
    });
    await userEvent.click(screen.getByRole("button", { name: "Open Save" }));
    await screen.findByText(stubSaveSession.sourcePath);

    await userEvent.click(screen.getByRole("button", { name: "Close Save" }));

    const dialog = await screen.findByRole("dialog", { name: "Unsaved changes" });
    expect(within(dialog).getByRole("button", { name: "Save…" })).toBeVisible();
    expect(within(dialog).getByRole("button", { name: "Discard" })).toBeVisible();
    expect(closeSave).not.toHaveBeenCalled();

    await userEvent.click(within(dialog).getByRole("button", { name: "Cancel" }));

    await waitFor(() =>
      expect(screen.queryByRole("dialog", { name: "Unsaved changes" })).toBeNull(),
    );
    expect(closeSave).not.toHaveBeenCalled();
    expect(screen.getByText(stubSaveSession.sourcePath)).toBeVisible();
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

describe("AppShell diagnostic console", () => {
  const event = (seq: number, severity: string, message: string) => ({
    seq,
    timestamp: `2026-09-05T12:00:0${seq}Z`,
    severity,
    event: "operation_finished",
    message,
  });

  it("renders the instance stream incrementally and only while expanded", async () => {
    const getDiagnosticEvents = vi
      .fn()
      .mockResolvedValueOnce({
        records: [event(1, "info", "application started")],
        nextCursor: "1",
        hasMore: false,
        totalBuffered: 1,
        cursorExpired: false,
      })
      .mockResolvedValue({
        records: [event(2, "error", "operation finished")],
        nextCursor: "2",
        hasMore: false,
        totalBuffered: 2,
        cursorExpired: false,
      });

    await renderApp(<App />, { settingsPort: makeSettingsPort({ getDiagnosticEvents }) });

    // Nothing polls while the console is collapsed.
    expect(getDiagnosticEvents).not.toHaveBeenCalled();

    await userEvent.click(screen.getByRole("button", { name: /Console/ }));

    const console_ = await screen.findByRole("region", { name: "Diagnostic console" });
    expect(await within(console_).findByText("application started")).toBeVisible();
    // The second poll continues from the cursor the first one returned, so the
    // first record is kept rather than requested again.
    await waitFor(() => expect(getDiagnosticEvents).toHaveBeenCalledTimes(2), { timeout: 2000 });
    expect(getDiagnosticEvents).toHaveBeenLastCalledWith({
      cursor: "1",
      limit: 200,
      severity: "",
    });
    expect(await within(console_).findByText("operation finished")).toBeVisible();
    expect(within(console_).getAllByText("application started")).toHaveLength(1);

    // The level filter narrows what is rendered without re-reading the stream.
    await userEvent.selectOptions(within(console_).getByLabelText("Level"), "error");
    expect(within(console_).queryByText("application started")).toBeNull();
    expect(within(console_).getByText("operation finished")).toBeVisible();
  });

  it("ignores a pending diagnostic response after closing and reopening the console", async () => {
    let resolveOld!: (page: DiagnosticEventPage) => void;
    const pending = new Promise<DiagnosticEventPage>((resolve) => {
      resolveOld = resolve;
    });
    const getDiagnosticEvents = vi
      .fn()
      .mockReturnValueOnce(pending)
      .mockResolvedValue({
        records: [event(2, "info", "current record")],
        nextCursor: "2",
        hasMore: false,
        totalBuffered: 1,
        cursorExpired: false,
      });
    await renderApp(<App />, { settingsPort: makeSettingsPort({ getDiagnosticEvents }) });
    await userEvent.click(screen.getByRole("button", { name: /Console/ }));
    expect(getDiagnosticEvents).toHaveBeenCalledTimes(1);
    await userEvent.click(screen.getByRole("button", { name: "Close" }));
    await userEvent.click(screen.getByRole("button", { name: /Console/ }));
    const console_ = within(screen.getByRole("region", { name: "Diagnostic console" }));
    expect(await console_.findByText("current record")).toBeVisible();
    await act(async () =>
      resolveOld({
        records: [event(1, "error", "obsolete record")],
        nextCursor: "1",
        hasMore: false,
        totalBuffered: 1,
        cursorExpired: false,
      }),
    );
    expect(screen.queryByText("obsolete record")).toBeNull();
    expect(console_.getAllByText("current record")).toHaveLength(1);
  });

  it("states an explicit failure instead of rendering the transport message", async () => {
    const getDiagnosticEvents = vi.fn(() => Promise.reject(new Error("bridge internals")));

    await renderApp(<App />, { settingsPort: makeSettingsPort({ getDiagnosticEvents }) });
    await userEvent.click(screen.getByRole("button", { name: /Console/ }));

    expect(await screen.findByText("The diagnostic messages could not be read.")).toBeVisible();
    expect(screen.queryByText(/bridge internals/)).toBeNull();
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
