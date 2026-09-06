import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { expect, it, vi } from "vitest";
import { App } from "../../App";
import { makeSaveSessionPort, renderApp, stubSaveSession } from "../../test/renderWithProviders";

it("routes Home read-only shortcuts directly to their subtabs without opening a save", async () => {
  const loadSave = vi.fn();
  await renderApp(<App />, { saveSessionPort: makeSaveSessionPort({ loadSave }) });
  expect(screen.getByRole("heading", { name: "Current save" })).toBeVisible();
  expect(screen.getByText("No recent files.")).toBeVisible();
  expect(screen.queryByRole("region", { name: "Local backups" })).toBeNull();

  for (const label of ["Appearance Presets", "Item Database", "Settings", "About & Updates"]) {
    await userEvent.click(
      within(screen.getByRole("region", { name: "Available without a save" })).getByRole("button", {
        name: label,
      }),
    );
    expect(screen.getByRole("button", { name: label, pressed: true })).toBeVisible();
    expect(screen.queryByRole("button", { name: "Open Save" })).toBeNull();
    await userEvent.click(
      within(screen.getByRole("navigation", { name: "Modules" })).getByRole("button", {
        name: "Home",
      }),
    );
  }
  expect(loadSave).not.toHaveBeenCalled();
  await userEvent.click(within(screen.getByRole("navigation", { name: "Modules" })).getByRole("button", { name: "Character" }));
  expect(screen.getByRole("button", { name: "Profile", pressed: true })).toBeVisible();
});

it("renders backend summaries and recent file rows without presenting stale history as current", async () => {
  const recent = {
    path: "C:\\Saves\\ER0000.sl2",
    platform: "pc",
    format: "sl2_v2",
    lastOpenedAt: "2026-09-06T12:00:00Z",
  };
  const removeRecentFile = vi.fn(() => Promise.resolve([]));
  const loadSave = vi.fn(() => Promise.resolve(stubSaveSession));
  await renderApp(<App />, {
    saveSessionPort: makeSaveSessionPort({
      loadSave,
      getRecentFiles: () => Promise.resolve([recent]),
      recordRecentFile: () => Promise.resolve([recent]),
      removeRecentFile,
      getOperationHistory: (saveSessionID) =>
        Promise.resolve({
          saveSessionID,
          saveRevision: "old-revision",
          operations: [],
          undoCount: 0,
          redoCount: 0,
        }),
      getSaveLifecycleSettings: () =>
        Promise.resolve({
          backupRetention: 7,
          retentionNoticeShown: false,
          backupNamePattern: "copy-{filename}-{timestamp}",
          backupNameExample: "copy-save-date",
        }),
    }),
  });
  const recentFiles = screen.getByRole("region", { name: "Recent files" });
  await within(recentFiles).findByText(recent.path);
  expect(recentFiles.querySelector("time")).toHaveAttribute("datetime", recent.lastOpenedAt);
  expect(recentFiles.querySelector("time")).not.toHaveTextContent(recent.lastOpenedAt);
  await userEvent.click(await within(recentFiles).findByRole("button", { name: /ER0000.sl2.*pc/ }));
  await waitFor(() => expect(loadSave).toHaveBeenCalledWith(recent.path, "", "local"));
  expect(await screen.findByRole("heading", { name: "Local backups" })).toBeVisible();
  expect(screen.getByText("copy-{filename}-{timestamp}")).toBeVisible();
  expect(
    within(screen.getByRole("region", { name: "Local backups" })).getByText("7"),
  ).toBeVisible();
  expect(
    within(screen.getByRole("region", { name: "Active characters" })).getByText("1 active"),
  ).toBeVisible();
  expect(
    within(screen.getByRole("region", { name: "Pending changes" })).getByText("Unavailable"),
  ).toBeVisible();
  expect(screen.getByText("Not validated for the current revision.")).toBeVisible();
  await userEvent.click(
    within(recentFiles).getByRole("button", { name: `Remove recent file: ${recent.path}` }),
  );
  expect(removeRecentFile).toHaveBeenCalledExactlyOnceWith(recent.path);
  expect(await screen.findByText("No recent files.")).toBeVisible();
});
