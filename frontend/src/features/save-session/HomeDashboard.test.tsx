import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { expect, it, vi } from "vitest";
import { App } from "../../App";
import {
  makeCharacterPort,
  makeSaveSessionPort,
  renderApp,
  stubCharacterSlot,
  stubSaveSession,
} from "../../test/renderWithProviders";
import { SaveSessionContent } from "./SaveSessionPanel";
import { useSaveSessionFlow } from "./useSaveSessionFlow";

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
  await userEvent.click(
    within(screen.getByRole("navigation", { name: "Modules" })).getByRole("button", {
      name: "Character",
    }),
  );
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
  expect(screen.getByText("copy-save-date")).toBeVisible();
  expect(screen.getByText("Backup reported by last save").nextElementSibling).toHaveTextContent(
    "Unavailable",
  );
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

it("renders selected-slot versions and only metadata belonging to the current session", async () => {
  function Harness() {
    const flow = useSaveSessionFlow();
    const [laterRevision, setLaterRevision] = useState(false);
    const [foreignBackup, setForeignBackup] = useState(false);
    return (
      <>
        <button onClick={() => setLaterRevision(true)}>Advance revision</button>
        <button onClick={() => setForeignBackup(true)}>Foreign backup result</button>
        <SaveSessionContent
          showCharacterSidebar
          flow={{
            ...flow,
            session:
              flow.session === undefined
                ? undefined
                : {
                    ...flow.session,
                    saveRevision: laterRevision ? "next" : flow.session.saveRevision,
                  },
            lastSaveResult: {
              saveSessionID: foreignBackup ? "other-session" : stubSaveSession.saveSessionID,
              saveRevision: "0",
              operationID: "saved",
              operationKind: "save",
              changedScopes: ["save.session"],
              target: stubSaveSession.sourcePath,
              backupPath: "/backups/reported-copy.sl2",
              warnings: [],
              retentionNoticeRequired: false,
            },
          }}
        />
      </>
    );
  }
  await renderApp(<Harness />, {
    characterPort: makeCharacterPort({
      getSaveCharacters: (saveSessionID) =>
        Promise.resolve({
          saveSessionID,
          saveRevision: "0",
          characters: [
            { characterID: 0, active: true, name: "First", level: 30 },
            { characterID: 4, active: true, name: "Fifth", level: 40 },
            { characterID: 9, active: true, name: "Tenth", level: 50 },
          ],
          slots: [
            stubCharacterSlot(0, "active", { slotVersion: 0x4c, slotVersionKnown: true }),
            stubCharacterSlot(4, "active", { slotVersion: 0xe6, slotVersionKnown: true }),
            stubCharacterSlot(9, "active"),
          ],
        }),
    }),
  });
  await userEvent.click(screen.getByRole("button", { name: "Open Save" }));
  const home = within(screen.getByRole("region", { name: "Save session" }));
  expect(await home.findByText("Slot 1 · 0x4C")).toBeVisible();
  expect(home.queryByText(/Slot 0\b/)).toBeNull();
  await userEvent.click(screen.getByRole("button", { name: /Fifth.*RL 40/ }));
  expect(await home.findByText("Slot 5 · 0xE6")).toBeVisible();
  expect(home.queryByText("Slot 1 · 0x4C")).toBeNull();
  await userEvent.click(screen.getByRole("button", { name: /Tenth.*RL 50/ }));
  expect(await home.findByText("Slot 10 · Version unavailable")).toBeVisible();
  expect(home.queryByText(/0x00/)).toBeNull();

  const backups = within(screen.getByRole("region", { name: "Local backups" }));
  expect(backups.getByText("/backups/reported-copy.sl2")).toBeVisible();
  await userEvent.click(screen.getByRole("button", { name: "Foreign backup result" }));
  expect(backups.queryByText("/backups/reported-copy.sl2")).toBeNull();
  expect(backups.getByText("Unavailable")).toBeVisible();

  await userEvent.click(screen.getByRole("button", { name: "Advance revision" }));
  expect(home.getByText("Slot version").nextElementSibling).toHaveTextContent("Unavailable");
  expect(home.queryByText(/Slot 10 ·/)).toBeNull();
  expect(
    within(screen.getByRole("region", { name: "Active characters" })).getByText("Unavailable"),
  ).toBeVisible();
});
