import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { EquipmentPort } from "../../application/equipment/equipmentPort";
import {
  makeEquipmentPort,
  renderApp,
  stubEquipmentCandidatesPage,
  stubEquipmentMutationReceipt,
} from "../../test/renderWithProviders";
import { EquipmentPanel } from "./EquipmentPanel";

function ports(overrides: Partial<EquipmentPort> = {}) {
  const getEquipmentCandidates = vi.fn(
    overrides.getEquipmentCandidates ?? (() => Promise.resolve(stubEquipmentCandidatesPage)),
  );
  const setEquippedArmaments = vi.fn(
    overrides.setEquippedArmaments ?? (() => Promise.resolve(stubEquipmentMutationReceipt)),
  );
  return {
    getEquipmentCandidates,
    setEquippedArmaments,
    port: makeEquipmentPort({ ...overrides, getEquipmentCandidates, setEquippedArmaments }),
  };
}

function panel(
  equipmentPort: EquipmentPort,
  applyMutationReceipt = vi.fn(() => Promise.resolve()),
) {
  return {
    applyMutationReceipt,
    render: () =>
      renderApp(
        <EquipmentPanel
          saveSessionID="session-1"
          saveRevision="0"
          characterID={0}
          applyMutationReceipt={applyMutationReceipt}
        />,
        { equipmentPort },
      ),
  };
}

describe("EquipmentPanel", () => {
  it("renders the backend loadout and never offers a picker for ammunition", async () => {
    const { port } = ports();
    await panel(port).render();

    // Names and states are the backend's own answer, including the technical
    // empty positions it already recognised for us.
    expect(await screen.findByRole("button", { name: "Right hand 1: Dagger" })).toBeEnabled();
    expect(screen.getByRole("button", { name: "Left hand 1: Empty" })).toBeEnabled();
    expect(screen.getByRole("button", { name: "Head: Empty" })).toBeEnabled();

    // A locked talisman position comes from the backend and opens nothing.
    expect(screen.getByRole("button", { name: "Talisman 2: Not unlocked yet" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Talisman 1: Moon of Nokstella" })).toBeEnabled();

    // Ammunition is presented and never editable: no confirmed writer exists.
    expect(screen.getByRole("button", { name: "Arrows 1: Empty" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Bolts 1: Empty" })).toBeDisabled();

    // The two capacity counts are carried as reported, not recomputed.
    expect(screen.getByText("Memory Slots: 1 / 7")).toBeInTheDocument();
    expect(screen.getByText("Talisman slots: 1")).toBeInTheDocument();
  });

  it("asks the backend for the candidates of the exact slot type it opened", async () => {
    const { port, getEquipmentCandidates } = ports();
    await panel(port).render();

    fireEvent.click(await screen.findByRole("button", { name: "Right hand 1: Dagger" }));

    await waitFor(() => expect(getEquipmentCandidates).toHaveBeenCalled());
    expect(getEquipmentCandidates).toHaveBeenCalledWith({
      saveSessionID: "session-1",
      characterID: 0,
      slotType: "right_hand",
      search: "",
      page: 1,
      pageSize: 24,
    });
  });

  it("commits the complete hand group and forwards the receipt", async () => {
    const { port, setEquippedArmaments } = ports();
    const view = panel(port);
    await view.render();

    fireEvent.click(await screen.findByRole("button", { name: "Left hand 2: Empty" }));
    const dialog = await screen.findByRole("dialog");
    // The second candidate is the one that is not already equipped elsewhere;
    // the backend states no name for it, and none is invented here.
    fireEvent.click(await within(dialog).findByRole("button", { name: /Name unavailable/ }));

    await waitFor(() => expect(setEquippedArmaments).toHaveBeenCalled());
    // The whole group travels in the backend's order — left 1, right 1, left 2,
    // right 2, left 3, right 3 — with the current revision, and only the opened
    // position changed. The occupied right hand keeps its own owned identity.
    expect(setEquippedArmaments).toHaveBeenCalledWith({
      saveSessionID: "session-1",
      characterID: 0,
      expectedRevision: "0",
      slotAssignments: [null, "owned-weapon-1", "owned-weapon-2", null, null, null],
    });
    await waitFor(() =>
      expect(view.applyMutationReceipt).toHaveBeenCalledWith(stubEquipmentMutationReceipt),
    );
  });

  it("refuses a candidate the group already carries at another position", async () => {
    const { port } = ports({
      // The Physick picker is served the very Crystal Tear the mixture already
      // carries at its first position.
      getEquipmentCandidates: (request) =>
        Promise.resolve(
          request.slotType === "physick"
            ? {
                ...stubEquipmentCandidatesPage,
                slotType: "physick",
                candidates: [
                  {
                    resource: { kind: "item", key: "40002AF9" },
                    name: "Crimson Crystal Tear",
                    iconPath: "",
                    banRisk: false,
                    cutContent: false,
                  },
                ],
                total: 1,
              }
            : stubEquipmentCandidatesPage,
        ),
    });
    await panel(port).render();

    fireEvent.click(await screen.findByRole("button", { name: "Left hand 1: Empty" }));
    const dialog = await screen.findByRole("dialog");

    // `owned-weapon-1` is the record the occupied right hand references, and the
    // setter rejects the same record twice, so the picker refuses it first.
    expect(await within(dialog).findByRole("button", { name: /Uchigatana/ })).toBeDisabled();
    expect(within(dialog).getByRole("button", { name: /Name unavailable/ })).toBeEnabled();

    // The same rule holds for the resource-addressed Physick group:
    // SetPhysickMixture rejects the same Crystal Tear in both positions, so the
    // picker of the free position refuses the one already mixed.
    fireEvent.click(within(dialog).getByRole("button", { name: "Cancel" }));
    fireEvent.click(await screen.findByRole("button", { name: "Crystal Tear 2: Empty" }));
    const physickDialog = await screen.findByRole("dialog");
    expect(
      await within(physickDialog).findByRole("button", { name: /Crimson Crystal Tear/ }),
    ).toBeDisabled();
  });

  it("reports a failed mutation safely and applies no receipt", async () => {
    const { port } = ports({
      setEquippedArmaments: () =>
        Promise.reject(new Error("bridge: /Users/tarnished/ER0000.sl2 revision conflict")),
    });
    const view = panel(port);
    await view.render();

    fireEvent.click(await screen.findByRole("button", { name: "Left hand 2: Empty" }));
    const dialog = await screen.findByRole("dialog");
    fireEvent.click(await within(dialog).findByRole("button", { name: /Name unavailable/ }));

    // The raw bridge message, including the host path in it, never reaches the
    // interface: the screen states its own localized wording instead.
    const failure = await screen.findByRole("alert");
    expect(failure).toHaveTextContent("The change was not applied.");
    expect(failure.textContent ?? "").not.toContain("/Users/");
    expect(view.applyMutationReceipt).not.toHaveBeenCalled();
  });

  it("offers no mutation while the session is busy", async () => {
    const { port, getEquipmentCandidates } = ports();
    await renderApp(
      <EquipmentPanel
        saveSessionID="session-1"
        saveRevision="0"
        characterID={0}
        applyMutationReceipt={vi.fn(() => Promise.resolve())}
        sessionBusy
      />,
      { equipmentPort: port },
    );

    expect(await screen.findByRole("button", { name: "Right hand 1: Dagger" })).toBeDisabled();
    expect(getEquipmentCandidates).not.toHaveBeenCalled();
  });

  it("states that an inactive character slot has no equipment", async () => {
    const { port, getEquipmentCandidates } = ports({
      getCharacterLoadout: () =>
        Promise.resolve({
          saveSessionID: "session-1",
          saveRevision: "0",
          characterID: 1,
          active: false,
          rightHand: [],
          leftHand: [],
          arrows: [],
          bolts: [],
          armor: [],
          talismans: [],
          quickItems: [],
          pouch: [],
          activeQuickItem: 0,
          physick: [],
          spells: [],
          activeSpellIndex: -1,
          usedMemorySlots: 0,
          availableMemorySlots: 0,
          unlockedTalismanSlots: 0,
        }),
    });
    await renderApp(
      <EquipmentPanel
        saveSessionID="session-1"
        saveRevision="0"
        characterID={1}
        applyMutationReceipt={vi.fn(() => Promise.resolve())}
      />,
      { equipmentPort: port },
    );

    expect(
      await screen.findByText("This character slot is empty, so it has no equipment to show."),
    ).toBeInTheDocument();
    expect(getEquipmentCandidates).not.toHaveBeenCalled();
  });
});
