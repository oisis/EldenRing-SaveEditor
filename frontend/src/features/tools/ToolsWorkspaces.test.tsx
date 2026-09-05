import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import {
  makeAboutPort,
  makeDeploymentPort,
  makeSettingsPort,
  makeTemplatePort,
  renderApp,
  stubHostSettings,
} from "../../test/renderWithProviders";
import { ToolsPanel } from "./ToolsPanel";

/**
 * One flow test per new Tools workspace and per new contract, rather than one
 * test per button: the thing worth protecting is that each panel reaches the
 * backend with the right arguments and answers a blocked outcome with an
 * explicit decision.
 */

const baseProps = {
  theme: "dark" as const,
  onThemeChange: () => {},
  locale: "en" as const,
  onLocaleChange: () => {},
  onBackupSettingsChange: () => {},
  onOpenStagedFile: () => {},
  onOpenLocalFile: () => {},
  applyMutationReceipt: () => Promise.resolve(),
};

const pcSession = {
  saveSessionID: "session-1",
  saveRevision: "7",
  platform: "pc",
  characterID: 0,
};

const template = {
  templateID: "template-1",
  name: "Strength build",
  description: "A heavy build.",
  tags: ["strength"],
  createdAt: "2026-01-01T00:00:00Z",
  updatedAt: "2026-01-02T00:00:00Z",
  schemaVersion: 2,
  selectedSections: ["stats"],
  inventoryItems: 3,
  storageItems: 1,
  warnings: 0,
  templateRevision: "1",
};

async function openSubtab(name: string) {
  const navigation = screen.getByRole("navigation", { name: "Tools sections" });
  await userEvent.click(within(navigation).getByRole("button", { name }));
}

describe("Tools workspaces", () => {
  it("previews a template against the open character and applies it through the shared receipt", async () => {
    const getBuildTemplatePreview = vi.fn(() =>
      Promise.resolve({
        templateID: "template-1",
        templateRevision: "1",
        characterID: 0,
        saveSessionID: "session-1",
        // The preview describes exactly the revision the session is on; the
        // Apply button stays disabled for any other one.
        saveRevision: "7",
        executable: true,
        plan: {
          stats: {
            fields: [
              { field: "vigor", change: { current: 10, target: 40, changed: true } },
              { field: "mind", change: { current: 12, target: 12, changed: false } },
            ],
            resultLevel: 60,
            resultSoulMemory: 0,
          },
        },
        blockingIssues: [],
      }),
    );
    const applyBuildTemplate = vi.fn(() =>
      Promise.resolve({
        operationID: "operation-1",
        operationKind: "apply_build_template",
        saveSessionID: "session-1",
        saveRevision: "8",
        changedScopes: ["character.stats"] as const,
      }),
    );
    const applyMutationReceipt = vi.fn(() => Promise.resolve());

    await renderApp(
      <ToolsPanel {...baseProps} {...pcSession} applyMutationReceipt={applyMutationReceipt} />,
      {
        templatePort: makeTemplatePort({
          getBuildTemplates: () =>
            Promise.resolve({ templates: [template], total: 1, page: 0, pageSize: 50 }),
          getBuildTemplatePreview,
          applyBuildTemplate,
        }),
      },
    );

    await openSubtab("Templates");
    expect(await screen.findByRole("rowheader", { name: "Strength build" })).toBeVisible();

    // Nothing is previewed until the user asks for it.
    expect(getBuildTemplatePreview).not.toHaveBeenCalled();
    await userEvent.click(screen.getByRole("button", { name: "Preview" }));
    expect(await screen.findByText("vigor 10 → 40")).toBeVisible();
    // Only the changed statistics are shown.
    expect(screen.queryByText("mind 12 → 12")).toBeNull();

    await userEvent.click(screen.getByRole("button", { name: "Apply template" }));
    await waitFor(() => expect(applyBuildTemplate).toHaveBeenCalledTimes(1));
    expect(applyBuildTemplate).toHaveBeenCalledWith(
      expect.objectContaining({
        saveSessionID: "session-1",
        characterID: 0,
        templateID: "template-1",
        expectedRevision: "7",
      }),
    );
    // The apply is routed through the shared save-mutation path, never a local
    // refresh of its own.
    await waitFor(() =>
      expect(applyMutationReceipt).toHaveBeenCalledWith(
        expect.objectContaining({ saveSessionID: "session-1", saveRevision: "8" }),
      ),
    );
  });

  it("approves only the host key fingerprint the backend observed", async () => {
    const sshTarget = {
      id: "target-1",
      name: "Steam Deck",
      kind: "ssh",
      savePath: "/home/deck/ER0000.sl2",
      host: "192.0.2.1",
      port: 22,
      user: "deck",
      keyPath: "/home/user/.ssh/id_ed25519",
      hostKeyTrusted: false,
      transferSupported: true,
    };
    const trustDeploymentHostKey = vi.fn(() =>
      Promise.resolve({
        targets: [{ ...sshTarget, hostKeyTrusted: true, hostKeyFingerprint: "SHA256:observed" }],
        availableKinds: ["local", "ssh"],
      }),
    );

    await renderApp(<ToolsPanel {...baseProps} />, {
      deploymentPort: makeDeploymentPort({
        getDeploymentTargets: () =>
          Promise.resolve({ targets: [sshTarget], availableKinds: ["local", "ssh"] }),
        testDeploymentTarget: (targetID: string) =>
          Promise.resolve({
            targetID,
            reachable: false,
            hostKeyTrusted: false,
            gameStatus: "unknown",
            saveExists: false,
            hostKeyPending: true,
            hostKeyChanged: false,
            observedFingerprint: "SHA256:observed",
          }),
        trustDeploymentHostKey,
      }),
    });

    await openSubtab("Deployment");
    await userEvent.click(await screen.findByRole("button", { name: "Select" }));
    await userEvent.click(screen.getByRole("button", { name: "Test the target" }));

    const dialog = await screen.findByRole("dialog", { name: "Approve the SSH host key?" });
    // The fingerprint is shown, never typed: there is no input for it.
    expect(within(dialog).getByText("SHA256:observed")).toBeVisible();
    expect(within(dialog).queryByRole("textbox")).toBeNull();

    await userEvent.click(within(dialog).getByRole("button", { name: "Approve this host key" }));
    await waitFor(() =>
      expect(trustDeploymentHostKey).toHaveBeenCalledWith({
        targetID: "target-1",
        fingerprint: "SHA256:observed",
      }),
    );
  });

  it("offers no approval for a changed host key", async () => {
    const sshTarget = {
      id: "target-1",
      name: "Steam Deck",
      kind: "ssh",
      savePath: "/home/deck/ER0000.sl2",
      host: "192.0.2.1",
      port: 22,
      user: "deck",
      keyPath: "/home/user/.ssh/id_ed25519",
      hostKeyTrusted: true,
      hostKeyFingerprint: "SHA256:approved",
      transferSupported: true,
    };
    const trustDeploymentHostKey = vi.fn();

    await renderApp(<ToolsPanel {...baseProps} />, {
      deploymentPort: makeDeploymentPort({
        getDeploymentTargets: () =>
          Promise.resolve({ targets: [sshTarget], availableKinds: ["local", "ssh"] }),
        testDeploymentTarget: (targetID: string) =>
          Promise.resolve({
            targetID,
            reachable: false,
            hostKeyTrusted: true,
            gameStatus: "unknown",
            saveExists: false,
            hostKeyPending: false,
            hostKeyChanged: true,
            observedFingerprint: "SHA256:different",
          }),
        trustDeploymentHostKey,
      }),
    });

    await openSubtab("Deployment");
    await userEvent.click(await screen.findByRole("button", { name: "Select" }));
    await userEvent.click(screen.getByRole("button", { name: "Test the target" }));

    const dialog = await screen.findByRole("dialog", { name: "The SSH host key changed" });
    expect(within(dialog).queryByRole("button", { name: "Approve this host key" })).toBeNull();
    expect(trustDeploymentHostKey).not.toHaveBeenCalled();
  });

  it("sends the configured status command with the rest of the target", async () => {
    const createDeploymentTarget = vi.fn(() =>
      Promise.resolve({ targets: [], availableKinds: ["local", "ssh"] }),
    );

    await renderApp(<ToolsPanel {...baseProps} />, {
      deploymentPort: makeDeploymentPort({
        getDeploymentTargets: () =>
          Promise.resolve({ targets: [], availableKinds: ["local", "ssh"] }),
        createDeploymentTarget,
      }),
    });

    await openSubtab("Deployment");
    await userEvent.click(await screen.findByRole("button", { name: "Add a target" }));
    await userEvent.type(screen.getByLabelText("Name"), "Steam Deck");
    await userEvent.type(screen.getByLabelText("Save path on the target"), "/home/deck/ER0000.sl2");
    await userEvent.type(screen.getByLabelText("Status command"), "pgrep eldenring");
    await userEvent.click(screen.getByRole("button", { name: "Save the target" }));

    await waitFor(() =>
      expect(createDeploymentTarget).toHaveBeenCalledWith(
        expect.objectContaining({
          name: "Steam Deck",
          kind: "local",
          savePath: "/home/deck/ER0000.sl2",
          statusCommand: "pgrep eldenring",
        }),
      ),
    );
  });

  it("answers a blocked deployment with one explicit decision and retries with it", async () => {
    const target = {
      id: "target-1",
      name: "Steam Deck",
      kind: "local",
      savePath: "/home/deck/ER0000.sl2",
      hostKeyTrusted: false,
      transferSupported: true,
    };
    const deployToTarget = vi
      .fn()
      .mockResolvedValueOnce({
        operationID: "operation-1",
        targetID: "target-1",
        completed: false,
        blocked: "game_status_unknown",
        targetState: "unchanged",
        gameStatus: "unknown",
        stages: [],
      })
      .mockResolvedValueOnce({
        operationID: "operation-1",
        targetID: "target-1",
        completed: true,
        targetState: "replaced_verified",
        gameStatus: "unknown",
        stages: [{ stage: "replace_target", completed: true }],
      });

    await renderApp(<ToolsPanel {...baseProps} {...pcSession} />, {
      deploymentPort: makeDeploymentPort({
        getDeploymentTargets: () =>
          Promise.resolve({ targets: [target], availableKinds: ["local", "ssh"] }),
        deployToTarget,
      }),
    });

    await openSubtab("Deployment");
    await userEvent.click(await screen.findByRole("button", { name: "Select" }));
    await userEvent.click(screen.getByRole("button", { name: "Upload" }));

    // Validation always runs first, and with the default settings the review
    // modal is shown before anything reaches the target.
    expect(
      await screen.findByRole("dialog", { name: "Review the changes before deploying" }),
    ).toBeVisible();
    expect(deployToTarget).not.toHaveBeenCalled();
    await userEvent.click(screen.getByRole("button", { name: "Deploy these changes" }));

    // The first answer is a block, and the target is stated as unchanged.
    expect(
      await screen.findByRole("dialog", { name: "The game state cannot be confirmed" }),
    ).toBeVisible();
    await waitFor(() => expect(deployToTarget).toHaveBeenCalledTimes(1));
    // No confirmation the user did not give is sent with the first attempt.
    expect(deployToTarget.mock.calls[0]?.[0]).not.toHaveProperty(
      "continueWithUnknownGameStatus",
    );

    await userEvent.click(screen.getByRole("button", { name: "Continue anyway" }));
    await waitFor(() => expect(deployToTarget).toHaveBeenCalledTimes(2));
    expect(deployToTarget.mock.calls[1]?.[0]).toEqual(
      expect.objectContaining({ continueWithUnknownGameStatus: true, launchAfter: false }),
    );
  });

  it("activates a backup only through the backend, which backs the target up first", async () => {
    const backup = {
      id: "backup-1",
      targetID: "target-1",
      fileName: "ER0000.sl2.20260101000000_bak",
      createdAt: "2026-01-01T00:00:00Z",
      manual: true,
      active: false,
      tags: ["before-dlc"],
      description: "Before the DLC",
    };
    const activateTargetBackup = vi.fn(() =>
      Promise.resolve({
        operation: {
          operationID: "activate-backup-1",
          targetID: "target-1",
          completed: true,
          targetState: "replaced_verified" as const,
          gameStatus: "unknown",
          stages: [{ stage: "backup_target", completed: true }],
        },
        backups: {
          targetID: "target-1",
          backups: [{ ...backup, active: true }],
          transferSupported: true,
        },
      }),
    );

    await renderApp(<ToolsPanel {...baseProps} />, {
      deploymentPort: makeDeploymentPort({
        getDeploymentTargets: () =>
          Promise.resolve({
            targets: [
              {
                id: "target-1",
                name: "Steam Deck",
                kind: "local",
                savePath: "/home/deck/ER0000.sl2",
                hostKeyTrusted: false,
                transferSupported: true,
              },
            ],
            availableKinds: ["local", "ssh"],
          }),
        getTargetBackups: () =>
          Promise.resolve({ targetID: "target-1", backups: [backup], transferSupported: true }),
        activateTargetBackup,
      }),
    });

    await openSubtab("Save Manager");
    await userEvent.selectOptions(await screen.findByLabelText("Target"), "target-1");
    expect(
      await screen.findByRole("rowheader", { name: "ER0000.sl2.20260101000000_bak" }),
    ).toBeVisible();
    // The table deliberately carries no Size column.
    expect(screen.queryByRole("columnheader", { name: "Size" })).toBeNull();

    await userEvent.click(screen.getByRole("button", { name: "Set active" }));
    await userEvent.click(
      screen.getByRole("button", { name: "Back up the target and activate" }),
    );

    await waitFor(() => expect(activateTargetBackup).toHaveBeenCalledTimes(1));
    expect(activateTargetBackup).toHaveBeenCalledWith(
      expect.objectContaining({
        targetID: "target-1",
        backupID: "backup-1",
        confirmRemoteBackup: true,
      }),
    );
  });

  it("checks for updates only when asked and reports the newer release", async () => {
    const checkForUpdates = vi.fn(() =>
      Promise.resolve({
        status: "available",
        currentVersion: "2.0.0",
        latestVersion: "2.1.0",
        comparisonPossible: true,
      }),
    );
    const openProjectLink = vi.fn(() => Promise.resolve());

    await renderApp(<ToolsPanel {...baseProps} />, {
      aboutPort: makeAboutPort({ checkForUpdates, openProjectLink }),
    });

    await openSubtab("About & Updates");
    // Nothing checks on mount: the specification allows one manual check only.
    expect(checkForUpdates).not.toHaveBeenCalled();

    await userEvent.click(screen.getByRole("button", { name: "Check for updates" }));
    await waitFor(() => expect(checkForUpdates).toHaveBeenCalledTimes(1));
    expect(await screen.findByText("2.1.0")).toBeVisible();

    // The address is never stated by the frontend: only an approved identifier.
    await userEvent.click(screen.getByRole("button", { name: "Open the repository" }));
    await waitFor(() => expect(openProjectLink).toHaveBeenCalledWith("repository"));
  });

  it("stores a host setting as one complete value", async () => {
    const setHostSettings = vi.fn(() =>
      Promise.resolve({ ...stubHostSettings, skipReviewForNormalRisk: true }),
    );

    await renderApp(<ToolsPanel {...baseProps} />, {
      settingsPort: makeSettingsPort({ setHostSettings }),
    });

    await waitFor(() =>
      expect(screen.getByLabelText("Skip Review Changes for normal operations")).toBeEnabled(),
    );
    await userEvent.click(screen.getByLabelText("Skip Review Changes for normal operations"));

    await waitFor(() => expect(setHostSettings).toHaveBeenCalledTimes(1));
    // The setting the user did not touch is sent with the value the
    // backend already held, never omitted or defaulted.
    expect(setHostSettings).toHaveBeenCalledWith({
      skipReviewForNormalRisk: true,
      remoteBackupPolicy: "ask",
    });
  });

  it("turns Debug Mode on through the backend and opens the log directory", async () => {
    const setDiagnosticMode = vi.fn((enabled: boolean) =>
      Promise.resolve({
        enabled,
        logDirectoryExists: true,
        localLoggingAvailable: true,
        droppedRecords: 0,
      }),
    );
    const openHostLocation = vi.fn(() => Promise.resolve());

    await renderApp(<ToolsPanel {...baseProps} />, {
      settingsPort: makeSettingsPort({ setDiagnosticMode, openHostLocation }),
    });

    const toggle = await screen.findByLabelText("Debug Mode");
    await waitFor(() => expect(toggle).toBeEnabled());
    expect(toggle).not.toBeChecked();

    await userEvent.click(toggle);
    await waitFor(() => expect(setDiagnosticMode).toHaveBeenCalledWith(true));
    // The rendered state is the one the backend confirmed, never a local guess.
    await waitFor(() => expect(screen.getByLabelText("Debug Mode")).toBeChecked());

    // The location is stated as an identifier; no path is sent from here.
    await userEvent.click(screen.getByRole("button", { name: "Open log directory" }));
    await waitFor(() => expect(openHostLocation).toHaveBeenCalledWith("logs"));
  });

  it("reports a Debug Mode read failure without exposing the transport error", async () => {
    await renderApp(<ToolsPanel {...baseProps} />, {
      settingsPort: makeSettingsPort({
        getDiagnosticMode: () => Promise.reject(new Error("private diagnostic failure")),
      }),
    });
    expect(await screen.findByText("The diagnostic settings could not be read.")).toBeVisible();
    expect(screen.getByLabelText("Debug Mode")).toBeDisabled();
    expect(screen.queryByText("private diagnostic failure")).toBeNull();
  });

  it("keeps the previous Debug Mode state when the backend refuses the change", async () => {
    const setDiagnosticMode = vi.fn(() => Promise.reject(new Error("refused")));

    await renderApp(<ToolsPanel {...baseProps} />, {
      settingsPort: makeSettingsPort({ setDiagnosticMode }),
    });

    const toggle = await screen.findByLabelText("Debug Mode");
    await waitFor(() => expect(toggle).toBeEnabled());
    await userEvent.click(toggle);

    expect(
      await screen.findByText("Debug Mode was not changed, so the previous state is still in effect."),
    ).toBeVisible();
    expect(screen.getByLabelText("Debug Mode")).not.toBeChecked();
  });
});
