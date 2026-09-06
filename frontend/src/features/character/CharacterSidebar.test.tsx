import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type {
  CharacterPort,
  CharacterSummary,
  SaveCharacters,
} from "../../application/character/characterPort";
import {
  makeCatalogPort,
  makeCharacterPort,
  renderApp,
  stubCharacterProfile,
  stubCharacterStats,
  stubSlotsFor,
} from "../../test/renderWithProviders";
import { CharacterSidebar } from "./CharacterSidebar";
import { useCharacterSelection } from "./useCharacterSelection";

function summary(characterID: number, overrides: Partial<CharacterSummary> = {}): CharacterSummary {
  return { characterID, active: false, name: "", level: 0, ...overrides };
}

function saveCharacters(
  saveSessionID: string,
  characters: readonly CharacterSummary[],
): SaveCharacters {
  return { saveSessionID, saveRevision: "0", characters, slots: stubSlotsFor(characters) };
}

/**
 * The panel is presentational, so the screen that will own the session is
 * simulated by this harness. It is the same composition the future Character
 * screen uses: controller above, panel below.
 */
function Harness({ saveSessionID }: { saveSessionID?: string }) {
  return <CharacterSidebar model={useCharacterSelection(saveSessionID, "0")} />;
}

function listing(characters: readonly CharacterSummary[], overrides: Partial<CharacterPort> = {}) {
  return makeCharacterPort({
    getSaveCharacters: (id) => Promise.resolve(saveCharacters(id, characters)),
    getCharacterProfile: (saveSessionID, characterID) =>
      Promise.resolve({ ...stubCharacterProfile, saveSessionID, characterID }),
    getCharacterStats: (saveSessionID, characterID) =>
      Promise.resolve({ ...stubCharacterStats, saveSessionID, characterID }),
    ...overrides,
  });
}

/** Ten physical slots, three of them active, exactly as the backend reports. */
const tenSlots: readonly CharacterSummary[] = [
  summary(0, { active: true, name: "Zero", level: 150 }),
  summary(1),
  summary(2, { active: true, name: "Second", level: 90 }),
  summary(3),
  summary(4),
  summary(5, { active: true, name: "Fifth", level: 42 }),
  summary(6),
  summary(7),
  summary(8),
  summary(9),
];

function slotLabels() {
  return screen
    .getAllByRole("listitem")
    .map((item) => item.textContent?.match(/Slot \d+/)?.[0] ?? "");
}

describe("CharacterSidebar", () => {
  it("names the panel and groups active slots before inactive ones", async () => {
    await renderApp(<Harness saveSessionID="session-1" />, { characterPort: listing(tenSlots) });

    expect(await screen.findByRole("button", { name: /Zero/ })).toBeInTheDocument();
    expect(screen.getByRole("complementary", { name: "Characters" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Active characters" })).toBeInTheDocument();

    // The inactive slots start collapsed behind a counted disclosure, so only
    // the active ones are listed until it is opened.
    const inactive = screen.getByRole("button", { name: "7 inactive slots" });
    expect(inactive).toHaveAttribute("aria-expanded", "false");
    expect(slotLabels()).toEqual(["Slot 1", "Slot 3", "Slot 6"]);

    fireEvent.click(inactive);
    expect(inactive).toHaveAttribute("aria-expanded", "true");

    // Active slots first, and the reported order preserved inside both groups.
    expect(slotLabels()).toEqual([
      "Slot 1",
      "Slot 3",
      "Slot 6",
      "Slot 2",
      "Slot 4",
      "Slot 5",
      "Slot 7",
      "Slot 8",
      "Slot 9",
      "Slot 10",
    ]);
  });

  it("keeps the reported order inside each group when the backend scrambles it", async () => {
    const scrambled = [
      summary(3, { active: true, name: "Third", level: 60 }),
      summary(1),
      summary(0, { active: true, name: "Zero", level: 150 }),
      summary(2),
    ];
    await renderApp(<Harness saveSessionID="session-1" />, { characterPort: listing(scrambled) });

    expect(await screen.findByRole("button", { name: /Zero/ })).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "2 inactive slots" }));
    expect(slotLabels()).toEqual(["Slot 4", "Slot 1", "Slot 2", "Slot 3"]);
  });

  it("presents slot numbers from one and never a technical index", async () => {
    await renderApp(<Harness saveSessionID="session-1" />, { characterPort: listing(tenSlots) });

    expect(await screen.findByRole("button", { name: /Zero/ })).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "7 inactive slots" }));

    for (let slotNumber = 1; slotNumber <= 10; slotNumber += 1) {
      expect(screen.getByText(`Slot ${slotNumber} · Unknown`)).toBeInTheDocument();
    }
    expect(screen.queryByText(/\bSlot 0\b/)).toBeNull();
    expect(screen.queryByText(/\bSlot 11\b/)).toBeNull();
  });

  it("shows the name, the rune level and the reported starting class of an active character and nothing else", async () => {
    // The only class the catalog can name is the one the slot projection of the
    // fixture reports for an active slot.
    const catalogPort = makeCatalogPort({
      getResources: () =>
        Promise.resolve({
          resources: [{ kind: "class", key: "0", family: "", name: "Vagabond" }],
          total: 1,
          page: 1,
          pageSize: 50,
        }),
    });
    await renderApp(<Harness saveSessionID="session-1" />, {
      characterPort: listing(tenSlots),
      catalogPort,
    });

    const row = await screen.findByRole("button", { name: /Zero/ });
    expect(row).toHaveTextContent("Zero");
    expect(row).toHaveTextContent("RL 150");
    // The starting class comes from the slot projection, never from the profile.
    await waitFor(() => expect(row).toHaveTextContent("Slot 1 · Vagabond"));

    fireEvent.click(screen.getByRole("button", { name: "7 inactive slots" }));

    // Exactly what this fixture states: seven empty slots, no residual one, and
    // no class invented for a slot the backend reports none for.
    expect(screen.getAllByText("Empty")).toHaveLength(7);
    expect(screen.queryByText("Residual data")).toBeNull();
    expect(screen.getByText("Slot 2 · Unknown")).toBeInTheDocument();
    expect(screen.queryByText("Slot 2 · Vagabond")).toBeNull();

    // Nothing the current backend contract cannot state is rendered.
    const panel = screen.getByRole("complementary", { name: "Characters" });
    for (const forbidden of [
      String(stubCharacterProfile.secondsPlayed),
      "Play Time",
      "Starting Class",
      "Gender",
      "Body Type",
    ]) {
      expect(panel.textContent).not.toContain(forbidden);
    }
    // The raw starting class and gender identifiers never leak either.
    expect(screen.queryByText(String(stubCharacterProfile.startingClassID))).toBeNull();
    expect(screen.queryByText(String(stubCharacterProfile.gender))).toBeNull();
  });

  it("renders an inactive slot as a neutral, non-interactive row", async () => {
    await renderApp(<Harness saveSessionID="session-1" />, { characterPort: listing(tenSlots) });

    expect(await screen.findByRole("button", { name: /Zero/ })).toBeInTheDocument();

    // The three active slots and the disclosure are the only controls.
    expect(screen.getAllByRole("button")).toHaveLength(4);

    fireEvent.click(screen.getByRole("button", { name: "7 inactive slots" }));

    // The seven inactive slots of this fixture, classified as the backend
    // reports them, and none of them a control.
    expect(screen.getAllByRole("listitem")).toHaveLength(10);
    expect(screen.getAllByText("Empty")).toHaveLength(7);
    expect(screen.queryByText("Residual data")).toBeNull();
    expect(screen.getAllByRole("button")).toHaveLength(4);
    expect(screen.queryByRole("button", { name: /Empty/ })).toBeNull();
  });

  it("exposes the selection to assistive technology and moves it on a click", async () => {
    const getCharacterProfile = vi.fn((saveSessionID: string, characterID: number) =>
      Promise.resolve({ ...stubCharacterProfile, saveSessionID, characterID }),
    );
    const getCharacterStats = vi.fn((saveSessionID: string, characterID: number) =>
      Promise.resolve({ ...stubCharacterStats, saveSessionID, characterID }),
    );
    await renderApp(<Harness saveSessionID="session-1" />, {
      characterPort: listing(tenSlots, { getCharacterProfile, getCharacterStats }),
    });

    const zero = await screen.findByRole("button", { name: /Zero/ });
    const fifth = screen.getByRole("button", { name: /Fifth/ });
    expect(zero).toHaveAttribute("aria-pressed", "true");
    expect(fifth).toHaveAttribute("aria-pressed", "false");

    fireEvent.click(fifth);

    expect(fifth).toHaveAttribute("aria-pressed", "true");
    expect(zero).toHaveAttribute("aria-pressed", "false");
    expect(getCharacterProfile.mock.calls).toEqual([
      ["session-1", 0],
      ["session-1", 5],
    ]);
    expect(getCharacterStats.mock.calls.at(-1)).toEqual(["session-1", 5]);
  });

  it("renders a long, unusual name exactly as the backend reports it", async () => {
    const name = `Ἀχιλλεύς 🜃 ${"ⲛ".repeat(120)}`;
    await renderApp(<Harness saveSessionID="session-1" />, {
      characterPort: listing([summary(0, { active: true, name, level: 713 })]),
    });

    expect(await screen.findByText(name)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /RL 713/ })).toHaveTextContent(name);
  });

  it("says that no save is loaded and asks the backend for nothing", async () => {
    const getSaveCharacters = vi.fn(makeCharacterPort().getSaveCharacters);
    await renderApp(<Harness />, { characterPort: listing(tenSlots, { getSaveCharacters }) });

    expect(screen.getByText("No save loaded.")).toBeInTheDocument();
    expect(screen.queryByRole("status")).toBeNull();
    expect(getSaveCharacters).not.toHaveBeenCalled();
  });

  it("reports that the list is loading", async () => {
    await renderApp(<Harness saveSessionID="session-1" />, {
      characterPort: listing(tenSlots, { getSaveCharacters: () => new Promise(() => undefined) }),
    });

    expect(await screen.findByRole("status")).toHaveTextContent("Loading characters…");
  });

  it("reports a session without any active character", async () => {
    await renderApp(<Harness saveSessionID="session-1" />, {
      characterPort: listing([summary(0), summary(1)]),
    });

    expect(await screen.findByText("No active character is available.")).toBeInTheDocument();

    // The disclosure is the only control: no character can be selected.
    const inactive = screen.getByRole("button", { name: "2 inactive slots" });
    expect(screen.getAllByRole("button")).toHaveLength(1);

    fireEvent.click(inactive);
    expect(screen.getAllByText("Empty")).toHaveLength(2);
    expect(screen.getByText("Slot 1 · Unknown")).toBeInTheDocument();
    expect(screen.getByText("Slot 2 · Unknown")).toBeInTheDocument();
    expect(screen.getAllByRole("button")).toHaveLength(1);
  });

  it("shows a safe message instead of the transport error", async () => {
    await renderApp(<Harness saveSessionID="session-1" />, {
      characterPort: listing(tenSlots, {
        getSaveCharacters: () => Promise.reject(new Error("bridge_call_failed")),
      }),
    });

    expect(await screen.findByRole("alert")).toHaveTextContent("Unable to load characters.");
    expect(document.body.textContent).not.toContain("bridge_call_failed");
  });

  it("renders every label of the panel in Polish", async () => {
    await renderApp(<Harness saveSessionID="session-1" />, {
      characterPort: listing(tenSlots),
      locale: "pl",
    });

    expect(await screen.findByRole("button", { name: /Zero/ })).toBeInTheDocument();
    expect(screen.getByRole("complementary", { name: "Postacie" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Aktywne postacie" })).toBeInTheDocument();
    expect(screen.getByText("Slot 1 · Nieznana")).toBeInTheDocument();
    expect(screen.getByText("RL 150")).toBeInTheDocument();

    const inactive = screen.getByRole("button", { name: "7 nieaktywnych slotów" });
    fireEvent.click(inactive);
    expect(screen.getAllByText("Puste")).toHaveLength(7);
    expect(screen.getByText("Slot 2 · Nieznana")).toBeInTheDocument();
  });

  it("renders the Polish safe message for a failed list", async () => {
    await renderApp(<Harness saveSessionID="session-1" />, {
      characterPort: listing(tenSlots, {
        getSaveCharacters: () => Promise.reject(new Error("bridge_call_failed")),
      }),
      locale: "pl",
    });

    expect(await screen.findByRole("alert")).toHaveTextContent("Nie udało się wczytać postaci.");
    expect(document.body.textContent).not.toContain("bridge_call_failed");
  });
});
