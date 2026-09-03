import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { useState } from "react";
import { describe, expect, it, vi } from "vitest";
import {
  makeNetworkPort,
  renderApp,
  stubNetworkParamValues,
} from "../../test/renderWithProviders";
import { AdvancedPanel, type AdvancedPanelProps } from "./AdvancedPanel";

function Harness(props: AdvancedPanelProps) {
  const [sessionID, setSessionID] = useState(props.saveSessionID);
  const [revision, setRevision] = useState(props.saveRevision);

  return (
    <>
      <button type="button" onClick={() => setRevision("4")}>
        Bump revision
      </button>
      <button type="button" onClick={() => { setSessionID("session-2"); setRevision("1"); }}>
        Switch session
      </button>
      <AdvancedPanel
        {...props}
        saveSessionID={sessionID}
        saveRevision={revision}
      />
    </>
  );
}

describe("AdvancedPanel", () => {
  it("renders notice when no save session is loaded", async () => {
    await renderApp(<AdvancedPanel />);
    expect(
      screen.getByText("Open a save to view and edit network tuning parameters."),
    ).toBeInTheDocument();
  });

  it("renders Super marchant placeholder when subtab is switched", async () => {
    await renderApp(<AdvancedPanel saveSessionID="session-1" saveRevision="3" />);
    fireEvent.click(screen.getByRole("button", { name: "Super marchant" }));
    expect(screen.getByRole("heading", { name: "Super marchant" })).toBeInTheDocument();
    expect(
      screen.getByText("This feature will be added in a future update."),
    ).toBeInTheDocument();
  });

  it("renders all 5 groups and all 22 parameter controls when session is loaded", async () => {
    await renderApp(<AdvancedPanel saveSessionID="session-1" saveRevision="3" />);

    // Existing test renders all 5 groups
    expect(await screen.findByRole("region", { name: "Invader" })).toBeInTheDocument();
    expect(screen.getByRole("region", { name: "Summon host" })).toBeInTheDocument();
    expect(screen.getByRole("region", { name: "Summon guest" })).toBeInTheDocument();
    expect(screen.getByRole("region", { name: "Hunter / Blue matchmaking" })).toBeInTheDocument();
    expect(screen.getByRole("region", { name: "Visitor / additional parameters" })).toBeInTheDocument();

    // And confirms all 6 preset roles
    const presetRolesSection = screen.getByRole("region", { name: "Preset roles" });
    expect(within(presetRolesSection).getByText("Reds / Invader")).toBeInTheDocument();
    expect(within(presetRolesSection).getByText("Summon signs")).toBeInTheDocument();
    expect(within(presetRolesSection).getByText("Blue matchmaking")).toBeInTheDocument();
    expect(within(presetRolesSection).getByText("Summon host")).toBeInTheDocument();
    expect(within(presetRolesSection).getByText("Summon guest")).toBeInTheDocument();
    expect(within(presetRolesSection).getByText("Hunter")).toBeInTheDocument();

    // Verify all 22 sliders are rendered by their unique aria-labels
    const expectedSliders = [
      "Max Targets",
      "Search Areas",
      "Request Interval",
      "Request Timeout",
      "Summon Timeout",
      "Sign Refresh Interval",
      "Signs Total Count",
      "Signs Per Cell",
      "Sign Get Max",
      "Sign Download Span",
      "Sign Upload Interval",
      "Sign Update Span",
      "Search Cooldown",
      "Max Blue Summons",
      "Visit List Size",
      "Reload Search Min",
      "Reload Search Max",
      "All Area Search Rate (Co-op)",
      "All Area Search Rate (Vs Blue)",
      "Visitor List Max",
      "Visitor Timeout",
      "Visitor Download Span",
    ];

    for (const label of expectedSliders) {
      expect(screen.getByRole("slider", { name: label })).toBeInTheDocument();
    }
  });

  it("applies group preset only to fields of that group", async () => {
    await renderApp(<AdvancedPanel saveSessionID="session-1" saveRevision="3" applyMutationReceipt={vi.fn()} />);

    // Manually modify a field outside reds role (e.g. summonTimeoutTime = 99)
    const summonRegion = await screen.findByRole("region", { name: "Summon host" });
    const summonTimeout = within(summonRegion).getByRole("slider", { name: "Summon Timeout" });
    fireEvent.change(summonTimeout, { target: { value: "99" } });
    expect(summonTimeout).toHaveValue("99");

    const presetRolesSection = screen.getByRole("region", { name: "Preset roles" });

    // Test Reds preset role
    const redsRow = within(presetRolesSection).getByText("Reds / Invader").closest("div")!.parentElement!;
    const maxTargetsSlider = within(screen.getByRole("region", { name: "Invader" })).getByRole("slider", { name: "Max Targets" });
    expect(maxTargetsSlider).toHaveValue("5");

    fireEvent.click(within(redsRow).getByRole("button", { name: "Faster" }));
    // In faster-reds, maxBreakInTargetListCount is 8
    expect(maxTargetsSlider).toHaveValue("8");
    // The manually modified field outside reds union must remain untouched (99)
    expect(summonTimeout).toHaveValue("99");

    // Test separate Hunter vs Blue presets
    const hunterRow = within(presetRolesSection).getByText("Hunter").closest("div")!.parentElement!;
    const blueRow = within(presetRolesSection).getByText("Blue matchmaking").closest("div")!.parentElement!;

    const searchCooldownSlider = within(screen.getByRole("region", { name: "Hunter / Blue matchmaking" })).getByRole("slider", { name: "Search Cooldown" });
    const allAreaRateSlider = within(screen.getByRole("region", { name: "Hunter / Blue matchmaking" })).getByRole("slider", { name: "All Area Search Rate (Co-op)" });

    // Apply Blue Faster preset
    fireEvent.click(within(blueRow).getByRole("button", { name: "Faster" }));
    // In faster-blue: reloadVisitListCoolTime is 8, allAreaSearchRateCoopBlue is 60
    expect(searchCooldownSlider).toHaveValue("8");
    expect(allAreaRateSlider).toHaveValue("60");

    // Restore Blue Vanilla so allAreaRateSlider goes back to 30
    fireEvent.click(within(blueRow).getByRole("button", { name: "Vanilla" }));
    expect(searchCooldownSlider).toHaveValue("20");
    expect(allAreaRateSlider).toHaveValue("30");

    // Apply Hunter Faster preset (distinct from Blue: reloadVisitListCoolTime is 10, maxVisitListCount is 8)
    const maxVisitSlider = within(screen.getByRole("region", { name: "Hunter / Blue matchmaking" })).getByRole("slider", { name: "Visit List Size" });
    fireEvent.click(within(hunterRow).getByRole("button", { name: "Faster" }));
    expect(searchCooldownSlider).toHaveValue("10");
    expect(maxVisitSlider).toHaveValue("8");

    // Restore Hunter Vanilla
    fireEvent.click(within(hunterRow).getByRole("button", { name: "Vanilla" }));
    expect(searchCooldownSlider).toHaveValue("20");
    expect(maxVisitSlider).toHaveValue("5");

    // Apply button becomes active (reds Faster is still applied)
    const applyBtn = screen.getByRole("button", { name: "Apply changes" });
    expect(applyBtn).not.toBeDisabled();
  });

  it("submits complete 22-field payload with current session and expectedRevision, and calls applyMutationReceipt", async () => {
    const applyReceiptSpy = vi.fn().mockResolvedValue(undefined);
    const setNetworkSettingsSpy = vi.fn().mockResolvedValue({
      operationID: "op-1",
      operationKind: "set_network_settings",
      saveSessionID: "session-1",
      saveRevision: "4",
      changedScopes: ["network"],
      networkSettings: { ...stubNetworkParamValues, maxBreakInTargetListCount: 8, breakInRequestIntervalTimeSec: 12, breakInRequestTimeOutSec: 8, breakInRequestAreaCount: 8 },
    });

    const networkPort = makeNetworkPort({
      setNetworkSettings: setNetworkSettingsSpy,
    });

    function ReceiptHarness() {
      const [hasReceipt, setHasReceipt] = useState(false);
      return (
        <>
          <button type="button" onClick={() => setHasReceipt(true)}>
            Enable receipt
          </button>
          <AdvancedPanel
            saveSessionID="session-1"
            saveRevision="3"
            applyMutationReceipt={hasReceipt ? applyReceiptSpy : undefined}
          />
        </>
      );
    }

    // When applyMutationReceipt is missing, Apply button must be disabled
    await renderApp(<ReceiptHarness />, { networkPort });

    const presetRolesSection = await screen.findByRole("region", { name: "Preset roles" });
    const redsRow = within(presetRolesSection).getByText("Reds / Invader").closest("div")!.parentElement!;
    fireEvent.click(within(redsRow).getByRole("button", { name: "Faster" }));

    const applyBtn = screen.getByRole("button", { name: "Apply changes" });
    expect(applyBtn).toBeDisabled();

    // Enable receipt -> button enables
    fireEvent.click(screen.getByRole("button", { name: "Enable receipt" }));
    expect(applyBtn).not.toBeDisabled();

    // When an invalid numeric edit exists, Apply button must be blocked
    const summonRegion = screen.getByRole("region", { name: "Summon host" });
    const summonTimeoutInput = within(summonRegion).getByRole("spinbutton", {
      name: "Summon Timeout numeric input",
    });
    expect(summonTimeoutInput).toHaveValue(45);
    expect(summonTimeoutInput).toHaveAttribute("aria-invalid", "false");

    // In a field belonging to another role, enter an invalid value whose draft value won't change
    fireEvent.change(summonTimeoutInput, { target: { value: "invalid" } });
    expect(summonTimeoutInput).toHaveValue(null);
    expect(summonTimeoutInput).toHaveAttribute("aria-invalid", "true");
    expect(applyBtn).toBeDisabled();

    // Re-apply Reds Faster preset: does not change summonTimeoutTime (stays 45 in draft),
    // but bumps resetToken which must clear the invalid text, restore valid draft display,
    // clear aria-invalid, and re-enable Apply button.
    fireEvent.click(within(redsRow).getByRole("button", { name: "Faster" }));
    expect(summonTimeoutInput).toHaveValue(45);
    expect(summonTimeoutInput).toHaveAttribute("aria-invalid", "false");
    expect(applyBtn).not.toBeDisabled();

    fireEvent.click(applyBtn);

    await waitFor(() => expect(setNetworkSettingsSpy).toHaveBeenCalledTimes(1));
    expect(setNetworkSettingsSpy).toHaveBeenCalledWith(
      "session-1",
      expect.objectContaining({
        ...stubNetworkParamValues,
        maxBreakInTargetListCount: 8,
        breakInRequestAreaCount: 8,
        breakInRequestIntervalTimeSec: 12,
        breakInRequestTimeOutSec: 8,
      }),
      "3",
    );
    expect(applyReceiptSpy).toHaveBeenCalledWith(
      expect.objectContaining({
        operationID: "op-1",
        operationKind: "set_network_settings",
        saveSessionID: "session-1",
        saveRevision: "4",
        changedScopes: ["network"],
      }),
    );
  });

  it("discards uncommitted draft when session or revision changes", async () => {
    let currentRevision = "3";
    const port = makeNetworkPort({
      getNetworkSettings: (sessionID) =>
        Promise.resolve({
          saveSessionID: sessionID,
          saveRevision: currentRevision,
          parameters: stubNetworkParamValues,
        }),
    });

    await renderApp(
      <Harness saveSessionID="session-1" saveRevision="3" applyMutationReceipt={vi.fn()} />,
      { networkPort: port },
    );

    const presetRolesSection = await screen.findByRole("region", { name: "Preset roles" });
    const redsRow = within(presetRolesSection).getByText("Reds / Invader").closest("div")!.parentElement!;
    fireEvent.click(within(redsRow).getByRole("button", { name: "Faster" }));

    const slider = within(screen.getByRole("region", { name: "Invader" })).getByRole("slider", { name: "Max Targets" });
    const numInput = within(screen.getByRole("region", { name: "Invader" })).getByRole("spinbutton", { name: "Max Targets numeric input" });
    expect(slider).toHaveValue("8");
    expect(numInput).toHaveValue(8);

    // Type in number input to simulate active text editing
    fireEvent.change(numInput, { target: { value: "14" } });
    expect(numInput).toHaveValue(14);

    // Bump revision
    currentRevision = "4";
    fireEvent.click(screen.getByRole("button", { name: "Bump revision" }));

    // Draft and text input from revision 3 must be discarded and replaced by fresh backend parameters
    await waitFor(() => {
      const reacquiredSlider = within(screen.getByRole("region", { name: "Invader" })).getByRole(
        "slider",
        { name: "Max Targets" },
      );
      const reacquiredNumInput = within(screen.getByRole("region", { name: "Invader" })).getByRole(
        "spinbutton",
        { name: "Max Targets numeric input" },
      );
      expect(reacquiredSlider).toHaveValue("5");
      expect(reacquiredNumInput).toHaveValue(5);
    });
  });
});
