import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { useState } from "react";
import { describe, expect, it } from "vitest";
import type { WorldPort } from "../../application/world/worldPort";
import { makeWorldPort, renderApp } from "../../test/renderWithProviders";
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
});
