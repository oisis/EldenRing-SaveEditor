import { cleanup, fireEvent, screen, waitFor, within } from "@testing-library/react";
import { useState } from "react";
import { describe, expect, it, vi } from "vitest";
import type {
  WorldMutationCapability,
  WorldOperationKind,
  WorldPort,
} from "../../application/world/worldPort";
import {
  makeWorldPort,
  renderApp,
  stubWorldMutationReceipt,
} from "../../test/renderWithProviders";
import { WorldPanel, type WorldPanelProps } from "./WorldPanel";

const identity = { saveSessionID: "session-1", saveRevision: "3", characterID: 0, active: true };

/**
 * One answer per slot, the way a real backend answers: a stub that always named
 * slot 0 would hide a panel that kept showing the previous character's data.
 */
function worldPortForSlots(): Partial<WorldPort> {
  const grace = (name: string, regionLabel: string, visited: boolean) => ({
    kind: "grace",
    key: name,
    name,
    regionLabel,
    bossArena: false,
    dungeonType: "",
    visited,
  });

  return {
    getGraces: ({ characterID }) =>
      Promise.resolve({
        ...identity,
        characterID,
        graces:
          characterID === 0
            ? [
                grace("First Step", "Limgrave", true),
                grace("Church of Elleh", "Limgrave", false),
                grace("Roundtable Hold", "", false),
              ]
            : [grace("Stormveil Cliffside", "Stormveil Castle", true)],
      }),
  };
}

/** Keeps the panel mounted while the selected slot changes. */
function PanelHarness(props: WorldPanelProps) {
  const [characterID, setCharacterID] = useState(props.characterID);
  return (
    <>
      <button type="button" onClick={() => setCharacterID(1)}>
        Switch character slot
      </button>
      <WorldPanel {...props} characterID={characterID} />
    </>
  );
}

/**
 * The backend's own capability entry. The risk and its reason are values the
 * screen receives, never values it words itself, so a test states them here
 * exactly as a backend would.
 */
function capability(
  operationKind: WorldOperationKind,
  supportsBulk = false,
): WorldMutationCapability {
  return {
    operationKind,
    risk: "warning",
    riskReason: "This operation changes world progression state.",
    supportsBulk,
  };
}

function graceSection() {
  return screen.getByRole("group", { name: "Sites of Grace" });
}

describe("WorldPanel", () => {
  it("shows the three sections, the backend groups, the counters and the search", async () => {
    await renderApp(
      <WorldPanel saveSessionID="session-1" saveRevision="3" characterID={0} />,
      { worldPort: makeWorldPort(worldPortForSlots()) },
    );

    // Exploration is the first section and carries the four exploration groups.
    for (const title of ["Regions", "Map Regions", "Sites of Grace", "Summoning Pools"]) {
      expect(await screen.findByRole("group", { name: title })).toBeInTheDocument();
    }
    expect(screen.queryByRole("group", { name: "Bosses" })).not.toBeInTheDocument();
    expect(screen.queryByRole("group", { name: "Gestures" })).not.toBeInTheDocument();

    const graces = graceSection();
    await waitFor(() => expect(within(graces).getByText("First Step")).toBeInTheDocument());

    // The counter is the backend's own boolean, counted over the entries only.
    expect(within(graces).getByText("1 / 3")).toBeInTheDocument();
    // The backend's region label groups the entries; a missing label falls back
    // to the neutral bucket rather than to an invented region.
    expect(within(graces).getByRole("region", { name: "Limgrave" })).toBeInTheDocument();
    expect(within(graces).getByRole("region", { name: "Other" })).toBeInTheDocument();

    fireEvent.change(screen.getByRole("searchbox", { name: "Search World entries" }), {
      target: { value: "elleh" },
    });
    expect(within(graceSection()).getByText("Church of Elleh")).toBeInTheDocument();
    expect(within(graceSection()).queryByText("First Step")).not.toBeInTheDocument();
    // The search narrows the rendered list only: the counter keeps describing
    // the whole section the backend answered with.
    expect(within(graceSection()).getByText("1 / 3")).toBeInTheDocument();

    // The other two sections are reachable and carry their own groups.
    fireEvent.click(screen.getByRole("button", { name: "Progress" }));
    expect(await screen.findByRole("group", { name: "Bosses" })).toBeInTheDocument();
    expect(screen.getByRole("group", { name: "Quests" })).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Unlocks" }));
    for (const title of [
      "Gestures",
      "Cookbooks",
      "Bell Bearings",
      "Whetblades",
      "Tutorials",
      "Colosseums",
      "Spectral Steed Attires",
    ]) {
      expect(await screen.findByRole("group", { name: title })).toBeInTheDocument();
    }

    // Read-only: the screen offers no writer at all, not even a disabled one.
    // The only buttons it renders are the three sections and the two expanders.
    expect(screen.queryAllByRole("checkbox")).toHaveLength(0);
    expect(screen.getAllByRole("button").map((element) => element.textContent)).toEqual([
      "Exploration",
      "Progress",
      "Unlocks",
      "Expand all",
      "Collapse all",
    ]);
  });

  it("never shows the previous slot's entries after the selection changes", async () => {
    await renderApp(
      <PanelHarness saveSessionID="session-1" saveRevision="3" characterID={0} />,
      { worldPort: makeWorldPort(worldPortForSlots()) },
    );

    await waitFor(() => expect(screen.getByText("First Step")).toBeInTheDocument());

    fireEvent.click(screen.getByRole("button", { name: "Switch character slot" }));

    // The panel is not remounted, so the guard has to be the query key: the
    // second slot's answer replaces the first one and nothing of slot 0 is left
    // on screen in the meantime.
    await waitFor(() => expect(screen.getByText("Stormveil Cliffside")).toBeInTheDocument());
    expect(screen.queryByText("First Step")).not.toBeInTheDocument();
    expect(screen.queryByText("Church of Elleh")).not.toBeInTheDocument();
  });

  it("fails closed while pending, on a failure and for an inactive slot", async () => {
    const failure = new Error("bridge call failed at /Users/someone/save.sl2");
    await renderApp(
      <WorldPanel saveSessionID="session-1" saveRevision="3" characterID={0} />,
      {
        worldPort: makeWorldPort({
          // Never settles, so the group stays pending.
          getRegions: () => new Promise<never>(() => {}),
          getMapRegions: () => Promise.reject(failure),
          getGraces: () =>
            Promise.resolve({ ...identity, active: false, graces: [] }),
        }),
      },
    );

    const regions = screen.getByRole("group", { name: "Regions" });
    expect(within(regions).getByRole("status")).toHaveTextContent("Loading");

    const mapRegions = screen.getByRole("group", { name: "Map Regions" });
    await waitFor(() =>
      expect(within(mapRegions).getByRole("alert")).toHaveTextContent(
        "Unable to load this World section.",
      ),
    );
    // The host path of the failure never reaches the screen.
    expect(screen.queryByText(/save\.sl2/)).not.toBeInTheDocument();

    const graces = graceSection();
    await waitFor(() =>
      expect(within(graces).getByText(/This character slot is empty/)).toBeInTheDocument(),
    );
    // No counter is shown for a slot that carries no state.
    expect(within(graces).queryByText("0 / 0")).not.toBeInTheDocument();
  });

  it("renders one backend capability, sends the exact opposite state and publishes the receipt", async () => {
    const setGraceVisited = vi.fn(() => Promise.resolve(stubWorldMutationReceipt));
    const applyMutationReceipt = vi.fn(() => Promise.resolve());

    await renderApp(
      <WorldPanel
        saveSessionID="session-1"
        saveRevision="3"
        characterID={0}
        applyMutationReceipt={applyMutationReceipt}
        sessionBusy={false}
      />,
      {
        worldPort: makeWorldPort({
          ...worldPortForSlots(),
          // Only one capability, so only that one writer may appear.
          getWorldMutationCapabilities: () => Promise.resolve([capability("set_grace_visited")]),
          setGraceVisited,
        }),
      },
    );

    const graces = graceSection();
    await waitFor(() => expect(within(graces).getByText("First Step")).toBeInTheDocument());

    // The backend risk level and the backend reason are both shown before the
    // operation runs, and neither is worded by the screen.
    expect(
      within(graceSection()).getByText(
        "Risk: warning - This operation changes world progression state.",
      ),
    ).toBeInTheDocument();
    // A dataset the backend published no capability for stays read-only.
    expect(
      within(screen.getByRole("group", { name: "Regions" })).queryByRole("button"),
    ).not.toBeInTheDocument();

    // "First Step" is visited, so the action writes the exact opposite value
    // under the revision it was rendered for.
    fireEvent.click(screen.getByRole("button", { name: "First Step: Set not visited" }));
    await waitFor(() => expect(setGraceVisited).toHaveBeenCalledOnce());
    expect(setGraceVisited).toHaveBeenCalledWith({
      saveSessionID: "session-1",
      characterID: 0,
      resourceKind: "grace",
      resourceKey: "First Step",
      value: false,
      expectedRevision: "3",
    });
    await waitFor(() =>
      expect(applyMutationReceipt).toHaveBeenCalledWith(stubWorldMutationReceipt),
    );

    // A busy session blocks the action: the button stays visible and disabled
    // rather than disappearing.
    cleanup();
    await renderApp(
      <WorldPanel
        saveSessionID="session-1"
        saveRevision="3"
        characterID={0}
        applyMutationReceipt={applyMutationReceipt}
        sessionBusy={true}
      />,
      {
        worldPort: makeWorldPort({
          ...worldPortForSlots(),
          getWorldMutationCapabilities: () => Promise.resolve([capability("set_grace_visited")]),
          setGraceVisited,
        }),
      },
    );
    await waitFor(() =>
      expect(screen.getByRole("button", { name: "First Step: Set not visited" })).toBeDisabled(),
    );
    expect(setGraceVisited).toHaveBeenCalledOnce();
  });

  it("applies a chosen quest step, removes Fog of War one-way and locks every attire in one call", async () => {
    const setQuestStep = vi.fn(() => Promise.resolve(stubWorldMutationReceipt));
    const setFogOfWarRemoved = vi.fn(() => Promise.resolve(stubWorldMutationReceipt));
    const setSpectralSteedAttire = vi.fn(() => Promise.resolve(stubWorldMutationReceipt));
    const lockAllSpectralSteedAttires = vi.fn(() => Promise.resolve(stubWorldMutationReceipt));

    await renderApp(
      <PanelHarness
        saveSessionID="session-1"
        saveRevision="3"
        characterID={0}
        applyMutationReceipt={() => Promise.resolve()}
        sessionBusy={false}
      />,
      {
        worldPort: makeWorldPort({
          getWorldMutationCapabilities: () =>
            Promise.resolve([
              capability("set_quest_step"),
              capability("set_fog_of_war_removed"),
              capability("set_spectral_steed_attire"),
              capability("lock_all_spectral_steed_attires", true),
            ]),
          getQuests: ({ characterID }) =>
            Promise.resolve({
              ...identity,
              characterID,
              quests: [
                {
                  kind: "quest",
                  key: "ranni",
                  name: "Ranni",
                  steps: [
                    {
                      stepKind: "quest_step",
                      stepKey: "met_ranni",
                      description: "Met Ranni",
                      location: "Church of Elleh",
                      matched: true,
                    },
                    {
                      stepKind: "quest_step",
                      stepKey: "joined_ranni",
                      description: "Joined Ranni",
                      location: "Ranni's Rise",
                      matched: false,
                    },
                  ],
                },
              ],
            }),
          getSpectralSteedAttires: ({ characterID }) =>
            Promise.resolve({
              ...identity,
              characterID,
              status: "resolved" as const,
              activeAttireKey: "default",
              attires: [
                {
                  attireKey: "default",
                  name: "Default Appearance",
                  owned: true,
                  requiredResourceKind: "",
                  requiredResourceKey: "",
                  iconPath: "",
                },
                {
                  attireKey: "tree_sentinel",
                  name: "Tree Sentinel Attire",
                  owned: true,
                  requiredResourceKind: "item",
                  requiredResourceKey: "goods/401EAA00",
                  iconPath: "",
                },
                {
                  attireKey: "funereal_night",
                  name: "Funereal Night Attire",
                  owned: false,
                  requiredResourceKind: "item",
                  requiredResourceKey: "goods/401EAA14",
                  iconPath: "",
                },
              ],
            }),
          setQuestStep,
          setFogOfWarRemoved,
          setSpectralSteedAttire,
          lockAllSpectralSteedAttires,
        }),
      },
    );

    // Fog of War is one-way: one action, no checkbox, no current state.
    const fog = await screen.findByRole("region", { name: "Fog of War" });
    expect(within(fog).queryAllByRole("checkbox")).toHaveLength(0);
    fireEvent.click(within(fog).getByRole("button", { name: "Remove Fog of War" }));
    await waitFor(() => expect(setFogOfWarRemoved).toHaveBeenCalledOnce());
    expect(setFogOfWarRemoved).toHaveBeenCalledWith({
      saveSessionID: "session-1",
      characterID: 0,
      removed: true,
      expectedRevision: "3",
    });

    // A quest step is chosen explicitly: Apply stays disabled until it is, and
    // `matched` never becomes a current step that could be inverted.
    fireEvent.click(screen.getByRole("button", { name: "Progress" }));
    const apply = await screen.findByRole("button", { name: "Ranni: apply step" });
    const questSelect = screen.getByRole("combobox", { name: "Ranni: quest step" });
    expect(apply).toBeDisabled();
    fireEvent.change(questSelect, {
      target: { value: "quest_step/joined_ranni" },
    });
    expect(questSelect).toHaveValue("quest_step/joined_ranni");
    expect(apply).toBeEnabled();

    // Switching character slot keeps the panel mounted, but the draft is scoped
    // to the context: the picker resets to empty and Apply is disabled.
    fireEvent.click(screen.getByRole("button", { name: "Switch character slot" }));
    await waitFor(() =>
      expect(screen.getByRole("combobox", { name: "Ranni: quest step" })).toHaveValue(""),
    );
    expect(screen.getByRole("button", { name: "Ranni: apply step" })).toBeDisabled();
    expect(setQuestStep).not.toHaveBeenCalled();

    // Reselecting a step on the new character sends the new characterID and revision.
    const newQuestSelect = screen.getByRole("combobox", { name: "Ranni: quest step" });
    fireEvent.change(newQuestSelect, {
      target: { value: "quest_step/joined_ranni" },
    });
    expect(screen.getByRole("button", { name: "Ranni: apply step" })).toBeEnabled();
    fireEvent.click(screen.getByRole("button", { name: "Ranni: apply step" }));
    await waitFor(() => expect(setQuestStep).toHaveBeenCalledOnce());
    expect(setQuestStep).toHaveBeenCalledWith({
      saveSessionID: "session-1",
      characterID: 1,
      questKind: "quest",
      questKey: "ranni",
      stepKind: "quest_step",
      stepKey: "joined_ranni",
      expectedRevision: "3",
    });

    // Only the default appearance and the owned one can be selected.
    fireEvent.click(screen.getByRole("button", { name: "Unlocks" }));
    const attireSection = await screen.findByRole("group", { name: "Spectral Steed Attires" });
    expect(
      within(attireSection).getByRole("button", { name: "Default Appearance: select appearance" }),
    ).toBeInTheDocument();
    expect(
      within(attireSection).queryByRole("button", {
        name: "Funereal Night Attire: select appearance",
      }),
    ).not.toBeInTheDocument();
    fireEvent.click(
      within(attireSection).getByRole("button", { name: "Tree Sentinel Attire: select appearance" }),
    );
    await waitFor(() => expect(setSpectralSteedAttire).toHaveBeenCalledOnce());
    expect(setSpectralSteedAttire).toHaveBeenCalledWith({
      saveSessionID: "session-1",
      characterID: 1,
      attireKey: "tree_sentinel",
      expectedRevision: "3",
    });

    // The bulk lock is exactly one atomic backend call, never a loop of the
    // single setters.
    fireEvent.click(
      within(attireSection).getByRole("button", { name: "Lock all Spectral Steed Attires" }),
    );
    await waitFor(() => expect(lockAllSpectralSteedAttires).toHaveBeenCalledOnce());
    expect(lockAllSpectralSteedAttires).toHaveBeenCalledWith({
      saveSessionID: "session-1",
      characterID: 1,
      expectedRevision: "3",
    });
    expect(setSpectralSteedAttire).toHaveBeenCalledOnce();
  });
});
