import { fireEvent, screen, waitFor } from "@testing-library/react";
import { useState } from "react";
import { describe, expect, it, vi } from "vitest";
import type { AppearancePort } from "../../application/appearance/appearancePort";
import type { CharacterPort } from "../../application/character/characterPort";
import type { FavoritesPort } from "../../application/favorites/favoritesPort";
import {
  makeAppearancePort,
  makeCharacterPort,
  makeFavoritesPort,
  renderApp,
  stubCharacterMutationReceipt,
  stubCharacterStats,
} from "../../test/renderWithProviders";
import { CharacterPanel, type CharacterPanelProps } from "./CharacterPanel";

/**
 * Keeps the panel mounted while the selected slot changes. That is exactly the
 * situation in which a draft could otherwise outlive the character it was typed
 * for, so the switch must happen without a remount.
 */
function PanelHarness({
  initialCharacterID,
  ...panelProps
}: CharacterPanelProps & { initialCharacterID: number | undefined }) {
  const [characterID, setCharacterID] = useState(initialCharacterID);
  return (
    <>
      <button type="button" onClick={() => setCharacterID(1)}>
        Switch character slot
      </button>
      <CharacterPanel {...panelProps} characterID={characterID} />
    </>
  );
}

function setup(
  overrides: {
    characterPort?: Partial<CharacterPort>;
    appearancePort?: Partial<AppearancePort>;
    favoritesPort?: Partial<FavoritesPort>;
    saveSessionID?: string | undefined;
    saveRevision?: string | undefined;
    characterID?: number | undefined;
    sessionBusy?: boolean;
  } = {},
) {
  const setCharacterName = vi.fn(
    overrides.characterPort?.setCharacterName ??
      (() => Promise.resolve(stubCharacterMutationReceipt)),
  );
  const setCharacterStats = vi.fn(
    overrides.characterPort?.setCharacterStats ??
      (() => Promise.resolve(stubCharacterMutationReceipt)),
  );
  const applyAppearancePreset = vi.fn(
    overrides.appearancePort?.applyAppearancePreset ??
      (() => Promise.resolve(stubCharacterMutationReceipt)),
  );
  const applyFavoritePreset = vi.fn(
    overrides.favoritesPort?.applyFavoritePreset ??
      (() => Promise.resolve(stubCharacterMutationReceipt)),
  );
  const applyMutationReceipt = vi.fn(() => Promise.resolve());

  const characterPort = makeCharacterPort({
    ...overrides.characterPort,
    setCharacterName,
    setCharacterStats,
  });
  const appearancePort = makeAppearancePort({
    ...overrides.appearancePort,
    applyAppearancePreset,
  });
  const favoritesPort = makeFavoritesPort({
    ...overrides.favoritesPort,
    applyFavoritePreset,
  });

  return {
    setCharacterName,
    setCharacterStats,
    applyAppearancePreset,
    applyFavoritePreset,
    applyMutationReceipt,
    render: () =>
      renderApp(
        <PanelHarness
          saveSessionID={"saveSessionID" in overrides ? overrides.saveSessionID : "session-1"}
          saveRevision={"saveRevision" in overrides ? overrides.saveRevision : "0"}
          initialCharacterID={"characterID" in overrides ? overrides.characterID : 0}
          applyMutationReceipt={applyMutationReceipt}
          sessionBusy={overrides.sessionBusy ?? false}
        />,
        { characterPort, appearancePort, favoritesPort },
      ),
  };
}

describe("CharacterPanel", () => {
  it("renders profile data and commits name and stats mutations with receipts", async () => {
    const { render, setCharacterName, setCharacterStats, applyMutationReceipt } = setup({
      characterPort: {
        // Both slots keep the stub name, so only the attribute value tells the
        // two cached characters apart.
        getCharacterStats: (saveSessionID, characterID) =>
          Promise.resolve({
            ...stubCharacterStats,
            saveSessionID,
            characterID,
            vigor: characterID === 0 ? 40 : 55,
          }),
      },
    });
    await render();

    // Renders character profile details
    expect(await screen.findByDisplayValue("Tarnished")).toBeInTheDocument();
    expect(screen.getByText("Type A")).toBeInTheDocument();
    expect(screen.getByText("150")).toBeInTheDocument();
    expect(screen.getByText("HP")).toBeInTheDocument();

    // Name mutation
    const nameInput = screen.getByLabelText("Name");
    fireEvent.change(nameInput, { target: { value: "Champion" } });
    fireEvent.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => expect(setCharacterName).toHaveBeenCalled());
    expect(setCharacterName).toHaveBeenCalledWith({
      saveSessionID: "session-1",
      characterID: 0,
      name: "Champion",
      expectedRevision: "0",
    });
    expect(applyMutationReceipt).toHaveBeenCalledWith(stubCharacterMutationReceipt);

    // Attribute mutation with recalculate policy
    fireEvent.click(screen.getByRole("button", { name: "Save Attributes" }));
    await waitFor(() => expect(setCharacterStats).toHaveBeenCalled());
    expect(setCharacterStats).toHaveBeenCalledWith({
      saveSessionID: "session-1",
      characterID: 0,
      attributes: {
        vigor: 40,
        mind: 20,
        endurance: 25,
        strength: 50,
        dexterity: 18,
        intelligence: 9,
        faith: 12,
        arcane: 7,
      },
      levelPolicy: "recalculate",
      expectedRevision: "0",
    });

    // A failed mutation shows the interface's own wording plus the stable code,
    // and never the backend message.
    setCharacterName.mockRejectedValueOnce(new Error("bridge exploded at /Users/host/save.sl2"));
    fireEvent.change(nameInput, { target: { value: "Nameless" } });
    fireEvent.click(screen.getByRole("button", { name: "Save" }));

    expect(await screen.findByText("The change was not applied.")).toBeInTheDocument();
    expect(screen.getByText("Error code: bridge_call_failed")).toBeInTheDocument();
    expect(screen.queryByText(/bridge exploded/)).not.toBeInTheDocument();

    // A draft belongs to one exact session, character and revision. Switching to
    // another cached slot that carries the same name must not show the previous
    // draft, and must not let it reach the mutation under the new slot's ID.
    fireEvent.change(nameInput, { target: { value: "Leaked" } });
    fireEvent.change(screen.getByLabelText("Vigor"), { target: { value: "99" } });
    expect(screen.getByLabelText("Name")).toHaveValue("Leaked");

    fireEvent.click(screen.getByRole("button", { name: "Switch character slot" }));

    expect(await screen.findAllByDisplayValue("55")).toHaveLength(2);
    expect(screen.getByLabelText("Name")).toHaveValue("Tarnished");
    expect(screen.queryByDisplayValue("99")).not.toBeInTheDocument();
    expect(screen.queryByDisplayValue("Leaked")).not.toBeInTheDocument();

    setCharacterName.mockClear();
    setCharacterStats.mockClear();

    fireEvent.change(screen.getByLabelText("Name"), { target: { value: "Second" } });
    fireEvent.click(screen.getByRole("button", { name: "Save" }));
    await waitFor(() => expect(setCharacterName).toHaveBeenCalled());
    expect(setCharacterName).toHaveBeenCalledWith({
      saveSessionID: "session-1",
      characterID: 1,
      name: "Second",
      expectedRevision: "0",
    });

    fireEvent.click(screen.getByRole("button", { name: "Save Attributes" }));
    await waitFor(() => expect(setCharacterStats).toHaveBeenCalled());
    expect(setCharacterStats).toHaveBeenCalledWith({
      saveSessionID: "session-1",
      characterID: 1,
      attributes: {
        vigor: 55,
        mind: 20,
        endurance: 25,
        strength: 50,
        dexterity: 18,
        intelligence: 9,
        faith: 12,
        arcane: 7,
      },
      levelPolicy: "recalculate",
      expectedRevision: "0",
    });
  });

  it("allows browsing presets read-only without save and disables apply", async () => {
    const noSave = setup({
      saveSessionID: undefined,
      saveRevision: undefined,
      characterID: undefined,
    });
    await noSave.render();

    fireEvent.click(screen.getByRole("button", { name: "Appearance Presets" }));
    expect(await screen.findByText("Geralt of Rivia, the Witcher")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Apply to Character" })).toBeDisabled();
    expect(
      screen.getByText("Mirror Favorites require an active save session."),
    ).toBeInTheDocument();

    // Moving to the last entry and then narrowing the list must not leave the
    // counter past its end.
    fireEvent.click(screen.getByRole("button", { name: "Next" }));
    expect(screen.getByText("2 / 2")).toBeInTheDocument();
    fireEvent.change(screen.getByLabelText("Body Type Filter"), { target: { value: "Type A" } });
    expect(screen.getByText("1 / 1")).toBeInTheDocument();
    expect(screen.getByText("Geralt of Rivia, the Witcher")).toBeInTheDocument();
  });

  it("applies appearance and favorite presets with active character and forwards receipts", async () => {
    const withSave = setup();
    await withSave.render();

    fireEvent.click(screen.getByRole("button", { name: "Appearance Presets" }));
    expect(await screen.findByText("Geralt of Rivia, the Witcher")).toBeInTheDocument();

    // Apply appearance preset
    fireEvent.click(screen.getByRole("button", { name: "Apply to Character" }));
    await waitFor(() => expect(withSave.applyAppearancePreset).toHaveBeenCalled());
    expect(withSave.applyAppearancePreset).toHaveBeenCalledWith({
      saveSessionID: "session-1",
      characterID: 0,
      presetID: "geralt-of-rivia-the-witcher",
      expectedRevision: "0",
    });
    expect(withSave.applyMutationReceipt).toHaveBeenCalledWith(stubCharacterMutationReceipt);

    // Apply mirror favorite preset
    const applyButtons = screen.getAllByRole("button", { name: "Apply" });
    fireEvent.click(applyButtons[0]);
    await waitFor(() => expect(withSave.applyFavoritePreset).toHaveBeenCalled());
    expect(withSave.applyFavoritePreset).toHaveBeenCalledWith({
      saveSessionID: "session-1",
      characterID: 0,
      favoriteSlotID: 0,
      expectedRevision: "0",
    });
  });

  it.each([
    {
      label: "pending",
      getCharacterStats: () => new Promise<never>(() => {}),
      status: "Loading attributes…",
    },
    {
      label: "failed",
      getCharacterStats: () => Promise.reject(new Error("unavailable")),
      status: "Unable to load the attributes of this character slot.",
    },
  ])(
    "never offers default attributes while the stats query is $label",
    async ({ getCharacterStats, status }) => {
      const { render, setCharacterStats } = setup({ characterPort: { getCharacterStats } });
      await render();

      expect(await screen.findByText(status)).toBeInTheDocument();
      expect(screen.queryByRole("button", { name: "Save Attributes" })).not.toBeInTheDocument();
      expect(screen.queryByLabelText("Vigor")).not.toBeInTheDocument();
      expect(setCharacterStats).not.toHaveBeenCalled();
    },
  );

  it("disables mutation controls when session is busy", async () => {
    const { render } = setup({ sessionBusy: true });
    await render();

    expect(await screen.findByDisplayValue("Tarnished")).toBeDisabled();
    expect(screen.getByRole("button", { name: "Save" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Save Attributes" })).toBeDisabled();
  });
});
