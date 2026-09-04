import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
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
import { ToolsPanel } from "./ToolsPanel";

const baseProps = {
  theme: "dark" as const,
  onThemeChange: () => {},
  locale: "en" as const,
  onLocaleChange: () => {},
  onBackupRetentionChange: () => {},
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
