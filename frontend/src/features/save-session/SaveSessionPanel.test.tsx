import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { AppErrorException } from "../../application/errors/appError";
import {
  makeCharacterPort,
  makeDiagnosticsPort,
  makeSaveSessionPort,
  renderApp,
  stubCleanValidationReport,
  stubSaveSession,
} from "../../test/renderWithProviders";
import { SaveSessionPanel } from "./SaveSessionPanel";

const inactiveOnly = makeCharacterPort({
  getSaveCharacters: () =>
    Promise.resolve({
      saveSessionID: stubSaveSession.saveSessionID,
      saveRevision: stubSaveSession.saveRevision,
      characters: [{ characterID: 0, active: false, name: "", level: 0 }],
    }),
});

async function openSave() {
  await userEvent.click(screen.getByRole("button", { name: "Open Save" }));
}

describe("SaveSessionPanel", () => {
  it("shows the opened file, its platform and its revision exactly as reported", async () => {
    await renderApp(<SaveSessionPanel />);
    await openSave();

    await waitFor(() => expect(screen.getByText(stubSaveSession.sourcePath)).toBeVisible());
    expect(screen.getByText("pc")).toBeVisible();
    expect(screen.getByText("sl2_v2")).toBeVisible();
    expect(screen.getByText("local")).toBeVisible();
    expect(screen.getByText("0")).toBeVisible();
    expect(screen.getByText("No unsaved changes")).toBeVisible();
  });

  it("offers no editing surface for a save with no active character", async () => {
    const closeSave = vi.fn(() => Promise.resolve());
    await renderApp(<SaveSessionPanel />, {
      characterPort: inactiveOnly,
      saveSessionPort: makeSaveSessionPort({ closeSave }),
    });
    await openSave();

    await waitFor(() =>
      expect(screen.getByRole("alert")).toHaveTextContent(/no active character/i),
    );
    // No character panel, no close action, and no file metadata: there is no
    // session to act on, and the one that had to be created was closed.
    expect(screen.queryByRole("complementary", { name: "Characters" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Close Save" })).toBeNull();
    expect(screen.queryByText(stubSaveSession.sourcePath)).toBeNull();
    expect(closeSave).toHaveBeenCalledExactlyOnceWith(stubSaveSession.saveSessionID);
  });

  it("never claims the rejected save was closed when its close failed", async () => {
    const closeSave = vi.fn(() => Promise.reject(new Error("bridge_call_failed")));
    await renderApp(<SaveSessionPanel />, {
      characterPort: inactiveOnly,
      saveSessionPort: makeSaveSessionPort({ closeSave }),
    });
    await openSave();

    await waitFor(() => expect(screen.getAllByRole("alert")).toHaveLength(2));
    const [refusal, cleanup] = screen.getAllByRole("alert");

    // The refusal states only what is known: the save was not opened for
    // editing. Whether the session created to find that out is gone is a
    // separate, unconfirmed outcome and may not be asserted here.
    expect(refusal).toHaveTextContent(/no active character/i);
    expect(refusal).not.toHaveTextContent(/no session was kept open/i);
    expect(refusal).not.toHaveTextContent(/closed/i);

    // The failed cleanup is shown on its own, and the session stays nameable.
    expect(cleanup).toHaveTextContent(/could not be closed and is still open/i);
    expect(screen.getByRole("button", { name: "Retry closing" })).toBeVisible();
    expect(closeSave).toHaveBeenCalledExactlyOnceWith(stubSaveSession.saveSessionID);
    // Nothing else may start beside an unconfirmed close.
    expect(screen.getByRole("button", { name: "Open Save" })).toBeDisabled();
  });

  it("says only that a refused load failed, never why the file was refused", async () => {
    await renderApp(<SaveSessionPanel />, {
      saveSessionPort: makeSaveSessionPort({
        loadSave: () => Promise.reject(new Error("bridge_call_failed")),
      }),
    });
    await openSave();

    const alert = await screen.findByRole("alert");
    expect(alert).toHaveTextContent(/The save was not opened for editing\./i);
    // The interface may not claim the container was damaged, unrecognised or
    // unsupported: a failed call carries no such reason, and inventing one
    // needs the structured backend error contract that does not exist yet.
    expect(alert).not.toHaveTextContent(/damaged|unrecognised|unsupported/i);
    // It may not claim anything about the backend's own state either. LoadSave
    // can have created a session before failing to report it, so no assurance
    // that none exists, or that none was left open, may be given here.
    expect(alert).not.toHaveTextContent(/no session/i);
    expect(screen.queryByRole("complementary", { name: "Characters" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Close Save" })).toBeNull();
  });

  it("shows the safe fallback and diagnostic ID for an unknown backend code", async () => {
    await renderApp(<SaveSessionPanel />, {
      saveSessionPort: makeSaveSessionPort({
        loadSave: () =>
          Promise.reject(
            new AppErrorException({
              code: "future_failure",
              message: "The backend could not complete this future operation.",
              params: {},
              severity: "error",
              stage: "mutation",
              retryable: false,
              fieldErrors: [],
              currentRevision: null,
              diagnosticID: "diag-future-1",
            }),
          ),
      }),
    });

    await openSave();

    const details = await screen.findByTestId("app-error-details");
    expect(details).toHaveTextContent("The backend could not complete this future operation.");
    expect(details).toHaveTextContent("diag-future-1");
    expect(details).not.toHaveTextContent("future_failure");
  });

  it("keeps the open session usable when the file dialog fails", async () => {
    let call = 0;
    await renderApp(<SaveSessionPanel />, {
      saveSessionPort: makeSaveSessionPort({
        selectSaveFile: () =>
          call++ === 0
            ? Promise.resolve(stubSaveSession.sourcePath)
            : Promise.reject(new Error("bridge_call_failed")),
      }),
    });
    await openSave();
    await waitFor(() => expect(screen.getByText(stubSaveSession.sourcePath)).toBeVisible());

    await openSave();

    await waitFor(() =>
      expect(screen.getByRole("alert")).toHaveTextContent(/file chooser could not be opened/i),
    );
    // The failure is about the operation, not the session: the metadata and the
    // Close action stay reachable, so no session becomes unreachable.
    expect(screen.getByText(stubSaveSession.sourcePath)).toBeVisible();
    expect(screen.getByRole("button", { name: "Close Save" })).toBeVisible();
  });

  it("keeps the session and offers a retry when closing it fails", async () => {
    let closes = 0;
    await renderApp(<SaveSessionPanel />, {
      saveSessionPort: makeSaveSessionPort({
        closeSave: () =>
          closes++ === 0 ? Promise.reject(new Error("bridge_call_failed")) : Promise.resolve(),
      }),
    });
    await openSave();
    await waitFor(() => expect(screen.getByText(stubSaveSession.sourcePath)).toBeVisible());

    await userEvent.click(screen.getByRole("button", { name: "Close Save" }));

    await waitFor(() =>
      expect(screen.getByRole("alert")).toHaveTextContent(/could not be closed and is still open/i),
    );
    // Nothing claims the session was closed while it is still open.
    expect(screen.getByText(stubSaveSession.sourcePath)).toBeVisible();

    await userEvent.click(screen.getByRole("button", { name: "Retry closing" }));

    await waitFor(() => expect(screen.queryByText(stubSaveSession.sourcePath)).toBeNull());
    expect(screen.queryByRole("alert")).toBeNull();
  });

  it("refuses to close a session with unsaved changes and discards nothing", async () => {
    const closeSave = vi.fn(() => Promise.resolve());
    await renderApp(<SaveSessionPanel />, {
      saveSessionPort: makeSaveSessionPort({
        closeSave,
        loadSave: () => Promise.resolve({ ...stubSaveSession, unsavedChanges: true }),
        getLoadedSave: () => Promise.resolve({ ...stubSaveSession, unsavedChanges: true }),
      }),
    });
    await openSave();
    await waitFor(() => expect(screen.getByText("Unsaved changes")).toBeVisible());

    await userEvent.click(screen.getByRole("button", { name: "Close Save" }));

    // The shell owns the Save / Discard / Cancel decision. The panel must not
    // bypass it by closing or replacing the dirty session.
    expect(closeSave).not.toHaveBeenCalled();
    expect(screen.getByText(stubSaveSession.sourcePath)).toBeVisible();
  });

  it("keeps a standing banner and a reachable report for a save with warnings", async () => {
    await renderApp(<SaveSessionPanel />, {
      diagnosticsPort: makeDiagnosticsPort({
        getSaveValidationReport: () =>
          Promise.resolve({
            ...stubCleanValidationReport,
            warningCount: 1,
            issues: [
              {
                id: "inventory-1",
                code: "unknown_item",
                severity: "warning",
                scope: "inventory",
                message: "An inventory record could not be resolved.",
                ownedItemID: "owned-9",
              },
            ],
          }),
      }),
    });
    await openSave();

    const banner = await screen.findByRole("alert");
    expect(banner).toHaveTextContent(/1 warning/);
    // The session stays editable behind the banner.
    expect(screen.getByRole("complementary", { name: "Characters" })).toBeVisible();

    await userEvent.click(screen.getByRole("button", { name: "View report" }));
    // The backend's own message, rendered verbatim.
    expect(
      await screen.findByText("An inventory record could not be resolved.", { exact: false }),
    ).toBeVisible();
  });

  it("reports a cancelled dialog without loading anything", async () => {
    const loadSave = vi.fn(() => Promise.resolve(stubSaveSession));
    await renderApp(<SaveSessionPanel />, {
      saveSessionPort: makeSaveSessionPort({
        selectSaveFile: () => Promise.resolve(""),
        loadSave,
      }),
    });
    await openSave();

    expect(await screen.findByRole("status")).toHaveTextContent("No file was chosen.");
    expect(loadSave).not.toHaveBeenCalled();
    expect(screen.queryByRole("complementary", { name: "Characters" })).toBeNull();
  });
});
