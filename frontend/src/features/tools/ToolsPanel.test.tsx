import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { describe, expect, it, vi } from "vitest";
import type {
  RepairPlan,
  SaveValidationReport,
} from "../../application/diagnostics/diagnosticsPort";
import {
  makeDiagnosticsPort,
  makeSaveSessionPort,
  renderApp,
} from "../../test/renderWithProviders";
import type { BackupSettingsStatus } from "../save-session/useSaveSessionFlow";
import { ToolsPanel } from "./ToolsPanel";

const baseProps = {
  theme: "dark" as const,
  onThemeChange: () => {},
  locale: "en" as const,
  onLocaleChange: () => {},
  onBackupSettingsChange: () => {},
  onOpenStagedFile: () => {},
  onOpenLocalFile: () => {},
  // The shared save-mutation path is required; a test that does not mutate
  // supplies an explicit no-op rather than leaving the panel without one.
  applyMutationReceipt: () => Promise.resolve(),
};

const pcSession = {
  saveSessionID: "session-1",
  saveRevision: "7",
  platform: "pc",
  characterID: 0,
};

const reportWithIssue: SaveValidationReport = {
  saveSessionID: "session-1",
  saveRevision: "7",
  characterID: 0,
  active: true,
  coverage: [
    { scope: "inventory", checked: true, reason: "", recordsChecked: 12, unresolvedRecords: 0 },
  ],
  issues: [
    {
      id: "issue-1",
      code: "quantity_above_limit",
      severity: "error",
      scope: "inventory",
      message: "A stack exceeds the container limit.",
      ownedItemID: "owned-1",
    },
    {
      id: "issue-2",
      code: "unknown_record",
      severity: "error",
      scope: "inventory",
      message: "A stored record cannot be resolved.",
      ownedItemID: "owned-2",
    },
  ],
  errorCount: 2,
  warningCount: 0,
};

const plan: RepairPlan = {
  saveSessionID: "session-1",
  saveRevision: "7",
  characterID: 0,
  planToken: "token-abc",
  actions: [
    {
      issueIDs: ["issue-1"],
      scope: "inventory",
      operation: "set_owned_item_quantity",
      ownedItemID: "owned-1",
      targetValue: 99,
      description: "Clamp the stack to 99.",
    },
  ],
  rejected: [
    { issueID: "issue-2", code: "unknown_record", scope: "inventory", reason: "No safe repair." },
  ],
};

describe("ToolsPanel", () => {
  it("edits the backup name pattern and sends the whole backup policy", async () => {
    const onBackupSettingsChange = vi.fn();
    await renderApp(
      <ToolsPanel
        {...baseProps}
        backupRetention={12}
        backupNamePattern="{filename}.{timestamp}"
        backupNameExample="ER0000.sl2.20260824202530_bak"
        onBackupSettingsChange={onBackupSettingsChange}
      />,
    );

    // The example comes from the backend; the frontend renders no name of its own.
    expect(screen.getByText("ER0000.sl2.20260824202530_bak")).toBeVisible();

    const pattern = screen.getByLabelText("Backup name pattern");
    expect(pattern).toHaveValue("{filename}.{timestamp}");
    // The value is pasted rather than typed: user-event reads "{" as the start
    // of a key descriptor, and the tokens are literal braces.
    await userEvent.clear(pattern);
    await userEvent.click(pattern);
    await userEvent.paste("saveforge-{timestamp}-{filename}");
    await userEvent.tab();

    // The retention limit travels with it: the backend owns one setting.
    await waitFor(() =>
      expect(onBackupSettingsChange).toHaveBeenCalledWith({
        backupRetention: 12,
        backupNamePattern: "saveforge-{timestamp}-{filename}",
      }),
    );
  });

  /**
   * Drives the panel exactly as the save session flow does: a write goes to
   * pending, and the backend's own answer decides whether it becomes success or
   * error. Nothing here compares a draft with a stored value.
   */
  function BackupSettingsHarness({
    answer,
  }: {
    answer: (pattern: string) => Promise<string>;
  }) {
    const [stored, setStored] = useState("{filename}.{timestamp}");
    const [status, setStatus] = useState<BackupSettingsStatus>();
    return (
      <>
      <span data-testid="stored-pattern">{stored}</span>
      <ToolsPanel
        {...baseProps}
        backupRetention={10}
        backupNamePattern={stored}
        backupNameExample="ER0000.sl2.20260824202530_bak"
        backupSettingsStatus={status}
        onBackupSettingsChange={({ backupNamePattern }) => {
          setStatus({ pattern: backupNamePattern, state: "pending" });
          void answer(backupNamePattern)
            .then((accepted) => {
              setStored(accepted);
              setStatus({ pattern: backupNamePattern, state: "success" });
            })
            .catch(() => setStatus({ pattern: backupNamePattern, state: "error" }));
        }}
      />
      </>
    );
  }

  async function submitPattern(pattern: string) {
    const field = screen.getByLabelText("Backup name pattern");
    await userEvent.clear(field);
    await userEvent.click(field);
    await userEvent.paste(pattern);
    await userEvent.tab();
  }

  it("does not call a delayed but accepted pattern a rejection", async () => {
    // The backend accepted the pattern and answered late. The old screen read
    // "the stored value still differs from my draft" as a refusal; only the
    // operation's own outcome may say that.
    let settle = (_: string) => {};
    await renderApp(
      <BackupSettingsHarness
        answer={(pattern) =>
          new Promise<string>((resolve) => {
            settle = () => resolve(pattern);
          })
        }
      />,
    );

    await submitPattern("saveforge-{timestamp}-{filename}");
    // While the answer is outstanding the screen states exactly that, and never
    // a rejection.
    expect(await screen.findByText("Storing the backup name pattern…")).toBeVisible();
    expect(screen.queryByRole("alert")).toBeNull();

    settle("");
    expect(await screen.findByText("The backup name pattern was stored.")).toBeVisible();
    expect(screen.queryByRole("alert")).toBeNull();
  });

  it("keeps an unsent edit when the answer to the previous one arrives", async () => {
    let settle = (_: string) => {};
    await renderApp(
      <BackupSettingsHarness
        answer={(pattern) =>
          new Promise<string>((resolve) => {
            settle = () => resolve(pattern);
          })
        }
      />,
    );

    await submitPattern("pattern-a-{timestamp}-{filename}");
    const field = screen.getByLabelText("Backup name pattern");
    // The answer to A is still outstanding while B is typed and not submitted.
    await userEvent.clear(field);
    await userEvent.click(field);
    await userEvent.paste("pattern-b-{timestamp}-{filename}");

    settle("");
    // The backend confirmed A and the stored example follows it, but the field
    // still belongs to the user.
    await waitFor(() => expect(screen.getByTestId("stored-pattern")).toHaveTextContent(
      "pattern-a-{timestamp}-{filename}",
    ));
    expect(field).toHaveValue("pattern-b-{timestamp}-{filename}");
  });

  it("states that a rejected pattern was not stored", async () => {
    await renderApp(
      <BackupSettingsHarness answer={() => Promise.reject(new Error("refused"))} />,
    );

    await submitPattern("../{filename}.{timestamp}");

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "This pattern was not accepted",
    );

    // The answer belongs to the pattern it was written with: once the user types
    // another one, that refusal no longer describes what is on screen.
    const field = screen.getByLabelText("Backup name pattern");
    await userEvent.click(field);
    await userEvent.paste("x");
    await waitFor(() => expect(screen.queryByRole("alert")).toBeNull());
  });

  it("reports the outcome of an empty pattern, which restores the default", async () => {
    // An empty pattern is a legitimate value the backend answers with the
    // default. It is a success, not a refusal, even though what comes back is
    // not what was typed.
    await renderApp(
      <BackupSettingsHarness answer={() => Promise.resolve("{filename}.{timestamp}")} />,
    );

    const field = screen.getByLabelText("Backup name pattern");
    await userEvent.clear(field);
    await userEvent.tab();

    expect(await screen.findByText("The backup name pattern was stored.")).toBeVisible();
    expect(screen.queryByRole("alert")).toBeNull();
  });

  it("navigates the subtabs and keeps the existing settings on Settings", async () => {
    await renderApp(<ToolsPanel {...baseProps} backupRetention={12} />);

    const navigation = screen.getByRole("navigation", { name: "Tools sections" });
    expect(
      within(navigation)
        .getAllByRole("button")
        .map((button) => button.textContent),
    ).toEqual(["Settings", "Templates", "Deployment", "Save Manager", "About & Updates"]);

    // The settings that already worked keep working, and the retention limit is
    // the backend's value rather than a frontend default.
    expect(screen.getByLabelText("Theme")).toHaveValue("dark");
    expect(screen.getByLabelText("Language")).toHaveValue("en");
    expect(await screen.findByRole("combobox", { name: "Safety profile" })).toHaveValue(
      "safe",
    );
    expect(screen.getByLabelText("Show Item ID")).toBeInstanceOf(HTMLInputElement);
    expect(screen.getByLabelText("Automatic backups kept")).toHaveValue(12);

    // Only settings backed by a real host contract are enabled. Debug Mode and
    // local logs remain explicit placeholders until their semantics exist.
    await waitFor(() =>
      expect(screen.getByLabelText("Skip Review Changes for normal operations")).toBeEnabled(),
    );
    expect(screen.getByLabelText("Debug Mode")).toBeDisabled();
    expect(screen.getByLabelText("Debug Mode")).not.toBeChecked();
    expect(screen.getByLabelText("Always create a remote backup")).toBeEnabled();
    // The stored policy is "ask", so the "always" switch is off rather than
    // defaulted on.
    expect(screen.getByLabelText("Always create a remote backup")).not.toBeChecked();
    expect(screen.getByRole("button", { name: "Open log directory" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Open configuration directory" })).toBeEnabled();

    await userEvent.click(within(navigation).getByRole("button", { name: "Templates" }));
    expect(screen.getByRole("region", { name: "Build Templates" })).toBeVisible();
    expect(screen.queryByLabelText("Theme")).toBeNull();

    await userEvent.click(within(navigation).getByRole("button", { name: "Deployment" }));
    expect(screen.getByRole("region", { name: "Deployment targets" })).toBeVisible();

    await userEvent.click(within(navigation).getByRole("button", { name: "Save Manager" }));
    expect(screen.getByRole("region", { name: "Save Manager" })).toBeVisible();

    await userEvent.click(within(navigation).getByRole("button", { name: "About & Updates" }));
    expect(await screen.findByText("2.0.0-test")).toBeVisible();
  });

  it("sends the Steam ID as a string of the current session and routes the receipt", async () => {
    const setSaveAccountID = vi.fn(() =>
      Promise.resolve({
        operationID: "operation-1",
        operationKind: "set_save_account_id",
        saveSessionID: "session-1",
        saveRevision: "8",
        changedScopes: ["save.session"] as const,
      }),
    );
    const applyMutationReceipt = vi.fn(() => Promise.resolve());

    await renderApp(
      <ToolsPanel
        {...baseProps}
        {...pcSession}
        applyMutationReceipt={applyMutationReceipt}
        sessionBusy={false}
      />,
      { saveSessionPort: makeSaveSessionPort({ setSaveAccountID }) },
    );

    const field = screen.getByLabelText("Steam ID");
    // A value that cannot survive a round trip through a JS number.
    await userEvent.type(field, "76561198000000001");
    await userEvent.click(screen.getByRole("button", { name: "Set Steam ID" }));

    await waitFor(() => expect(setSaveAccountID).toHaveBeenCalledTimes(1));
    expect(setSaveAccountID).toHaveBeenCalledWith("session-1", "76561198000000001", "7");
    expect(applyMutationReceipt).toHaveBeenCalledWith(
      expect.objectContaining({ saveSessionID: "session-1", saveRevision: "8" }),
    );
    // The backend reports no stored identifier, so nothing is kept on screen.
    await waitFor(() => expect(field).toHaveValue(""));
  });

  it("binds the report and the plan to the current context and applies only the plan", async () => {
    // The first answer describes another revision of the same slot; it is not
    // this context's plan, so nothing derived from it may be applied.
    const plans = [{ ...plan, saveRevision: "9" }, plan];
    const getRepairPlan = vi.fn(() => Promise.resolve(plans.shift() ?? plan));
    const applyRepairs = vi.fn(() =>
      Promise.resolve({
        operationID: "operation-2",
        operationKind: "apply_repairs",
        saveSessionID: "session-1",
        saveRevision: "8",
        changedScopes: ["inventory"] as const,
        characterID: 0,
        applied: true,
        actions: plan.actions,
        rejected: plan.rejected,
      }),
    );
    const applyMutationReceipt = vi.fn(() => Promise.resolve());

    await renderApp(
      <ToolsPanel
        {...baseProps}
        {...pcSession}
        applyMutationReceipt={applyMutationReceipt}
        sessionBusy={false}
      />,
      {
        diagnosticsPort: makeDiagnosticsPort({
          getSaveValidationReport: () => Promise.resolve(reportWithIssue),
          getRepairPlan,
          applyRepairs,
        }),
      },
    );

    // Nothing is scanned until the user asks for it.
    expect(screen.queryByText("A stack exceeds the container limit.")).toBeNull();
    await userEvent.click(screen.getByRole("button", { name: "Scan for problems" }));
    expect(await screen.findByText("A stack exceeds the container limit.")).toBeVisible();

    await userEvent.click(screen.getByLabelText("A stack exceeds the container limit."));
    await userEvent.click(screen.getByLabelText("A stored record cannot be resolved."));
    await userEvent.click(screen.getByRole("button", { name: "Preview repair plan" }));

    await waitFor(() => expect(getRepairPlan).toHaveBeenCalledTimes(1));
    // Both selected findings are sent in report order, and the backend accounts
    // for exactly those two: one action and one rejection.
    expect(getRepairPlan).toHaveBeenCalledWith({
      saveSessionID: "session-1",
      characterID: 0,
      saveRevision: "7",
      issueIDs: ["issue-1", "issue-2"],
    });
    expect(screen.queryByRole("button", { name: "Apply planned repairs" })).toBeNull();
    expect(applyRepairs).not.toHaveBeenCalled();

    await userEvent.click(screen.getByRole("button", { name: "Preview repair plan" }));
    expect(await screen.findByText("Clamp the stack to 99.")).toBeVisible();
    // A finding the backend refuses to plan for is shown with its reason rather
    // than dropped, so it can never look handled.
    expect(screen.getByText("No safe repair.")).toBeVisible();

    await userEvent.click(screen.getByRole("button", { name: "Apply planned repairs" }));

    await waitFor(() => expect(applyRepairs).toHaveBeenCalledTimes(1));
    expect(applyRepairs).toHaveBeenCalledWith({
      saveSessionID: "session-1",
      characterID: 0,
      issueIDs: ["issue-1", "issue-2"],
      planToken: "token-abc",
      expectedRevision: "7",
    });
    expect(applyMutationReceipt).toHaveBeenCalledWith(
      expect.objectContaining({ saveRevision: "8", applied: true }),
    );
  });
});
