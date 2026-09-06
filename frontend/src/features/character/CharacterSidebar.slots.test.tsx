import { fireEvent, screen, waitFor } from "@testing-library/react";
import { useState } from "react";
import { describe, expect, it, vi } from "vitest";
import type { CharacterSummary, SaveCharacters } from "../../application/character/characterPort";
import type { MutationReceipt } from "../../application/save-session/saveSessionPort";
import {
  makeCharacterPort,
  renderApp,
  stubCharacterMutationReceipt,
  stubCharacterProfile,
  stubCharacterStats,
  stubSlotsFor,
} from "../../test/renderWithProviders";
import { CharacterSidebar } from "./CharacterSidebar";
import { useCharacterSelection } from "./useCharacterSelection";

function summary(characterID: number, overrides: Partial<CharacterSummary> = {}): CharacterSummary {
  return { characterID, active: false, name: "", level: 0, ...overrides };
}

const zeroAndOne: readonly CharacterSummary[] = [
  summary(0, { active: true, name: "Zero", level: 150 }),
  summary(1),
];

/** A promise the test resolves itself, so an unresolved request stays unresolved. */
function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (reason: unknown) => void;
  const promise = new Promise<T>((settle, fail) => {
    resolve = settle;
    reject = fail;
  });
  return { promise, resolve, reject };
}

/**
 * The list the backend currently reports. The revision is part of it because the
 * character read is rejected when it answers for another revision.
 */
type Listing = { revision: string; characters: readonly CharacterSummary[] };

function listingPort(listing: Listing, overrides = {}) {
  return makeCharacterPort({
    getSaveCharacters: (saveSessionID): Promise<SaveCharacters> =>
      Promise.resolve({
        saveSessionID,
        saveRevision: listing.revision,
        characters: listing.characters,
        slots: stubSlotsFor(listing.characters),
      }),
    getCharacterProfile: (saveSessionID, characterID) =>
      Promise.resolve({ ...stubCharacterProfile, saveSessionID, characterID }),
    getCharacterStats: (saveSessionID, characterID) =>
      Promise.resolve({ ...stubCharacterStats, saveSessionID, characterID }),
    ...overrides,
  });
}

/**
 * The shell of one save session, reduced to what the slot management depends on:
 * the revision the list was read for and the revision the operations are sent
 * with. They move apart in the real shell while a changed session is re-read, so
 * the harness drives them separately.
 */
function Harness({
  applyMutationReceipt,
  listing,
}: {
  applyMutationReceipt: (receipt: MutationReceipt) => Promise<unknown>;
  listing: Listing;
}) {
  const [listRevision, setListRevision] = useState("0");
  const [manageRevision, setManageRevision] = useState("0");

  return (
    <>
      <button type="button" onClick={() => setManageRevision("9")}>
        drift revision
      </button>
      <button type="button" onClick={() => setListRevision(listing.revision)}>
        reload list
      </button>
      <CharacterSidebar
        model={useCharacterSelection("session-1", listRevision)}
        management={{
          saveSessionID: "session-1",
          saveRevision: manageRevision,
          applyMutationReceipt,
        }}
      />
    </>
  );
}

/**
 * A control of the surrounding shell. The dialog is modal, so the rest of the
 * document is hidden from the role queries while it is open.
 */
function clickShell(label: string) {
  fireEvent.click(screen.getByText(label));
}

async function openDeleteConfirmation() {
  fireEvent.click(await screen.findByRole("button", { name: "Manage slot — Zero" }));
  fireEvent.click(await screen.findByRole("button", { name: "Delete character" }));
  return screen.getByRole("button", { name: "Confirm" });
}

describe("CharacterSidebar slot management", () => {
  it("drops the standing confirmation when the operation revision moves", async () => {
    const deleteCharacter = vi.fn(() => Promise.resolve(stubCharacterMutationReceipt));
    const listing: Listing = { revision: "0", characters: zeroAndOne };
    await renderApp(
      <Harness applyMutationReceipt={vi.fn(() => Promise.resolve())} listing={listing} />,
      {
        characterPort: listingPort(listing, { deleteCharacter }),
      },
    );

    expect(await openDeleteConfirmation()).toBeInTheDocument();

    // The user confirmed for revision 0; the session moved on before the click.
    clickShell("drift revision");

    expect(screen.queryByRole("button", { name: "Confirm" })).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Delete character" }));
    expect(deleteCharacter).not.toHaveBeenCalled();
  });

  it("closes the dialog when the slot leaves the reloaded list", async () => {
    const listing: Listing = { revision: "0", characters: zeroAndOne };
    await renderApp(
      <Harness applyMutationReceipt={vi.fn(() => Promise.resolve())} listing={listing} />,
      {
        characterPort: listingPort(listing),
      },
    );

    expect(await openDeleteConfirmation()).toBeInTheDocument();

    listing.revision = "1";
    listing.characters = [summary(1)];
    clickShell("reload list");

    await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
  });

  it("keeps the operation locked while the dialog is closed and opened again", async () => {
    const pending = deferred<MutationReceipt>();
    const deleteCharacter = vi.fn(() => pending.promise);
    const listing: Listing = { revision: "0", characters: zeroAndOne };
    await renderApp(
      <Harness applyMutationReceipt={vi.fn(() => Promise.resolve())} listing={listing} />,
      {
        characterPort: listingPort(listing, { deleteCharacter }),
      },
    );

    fireEvent.click(await openDeleteConfirmation());
    expect(deleteCharacter).toHaveBeenCalledTimes(1);

    // The dialog is a view of the operation, not its owner: a new one cannot
    // start a second mutation while the first is still unresolved.
    fireEvent.click(screen.getByRole("button", { name: "Close" }));
    await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());

    fireEvent.click(screen.getByRole("button", { name: "Manage slot — Zero" }));
    const deleteAgain = await screen.findByRole("button", { name: "Delete character" });
    expect(deleteAgain).toBeDisabled();
    fireEvent.click(deleteAgain);
    expect(deleteCharacter).toHaveBeenCalledTimes(1);

    pending.resolve(stubCharacterMutationReceipt);
    await waitFor(() => expect(deleteCharacter).toHaveBeenCalledTimes(1));
  });

  it("reports a committed mutation whose refresh failed as applied, not as rejected", async () => {
    const deleteCharacter = vi.fn(() => Promise.resolve(stubCharacterMutationReceipt));
    // The refresh fails once and works from then on, so the session can catch up
    // and the slot can be operated again.
    let refreshFails = true;
    const applyMutationReceipt = vi.fn(() =>
      refreshFails ? Promise.reject(new Error("bridge_call_failed")) : Promise.resolve(),
    );
    const listing: Listing = { revision: "0", characters: zeroAndOne };
    await renderApp(<Harness applyMutationReceipt={applyMutationReceipt} listing={listing} />, {
      characterPort: listingPort(listing, { deleteCharacter }),
    });

    fireEvent.click(await openDeleteConfirmation());

    const alert = await screen.findByText(/The slot operation was applied/);
    expect(alert).toHaveTextContent("could not be refreshed");
    expect(alert).toHaveAttribute("role", "alert");
    expect(document.body.textContent).not.toContain("was rejected");
    expect(document.body.textContent).not.toContain("bridge_call_failed");

    // Nothing invites the committed mutation to be repeated.
    expect(screen.queryByRole("button", { name: "Confirm" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Delete character" })).toBeNull();
    expect(deleteCharacter).toHaveBeenCalledTimes(1);

    // Reopening the dialog is not a synchronisation: the list is still the one
    // read before the commit, so the block and its message survive the reopen.
    fireEvent.click(screen.getByRole("button", { name: "Close" }));
    await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
    fireEvent.click(screen.getByRole("button", { name: "Manage slot — Zero" }));

    const reopened = await screen.findByText(/The slot operation was applied/);
    expect(reopened).toHaveTextContent("could not be refreshed");
    expect(screen.queryByRole("button", { name: "Delete character" })).toBeNull();
    expect(deleteCharacter).toHaveBeenCalledTimes(1);

    // The session caught up: the list is read for the revision the mutation
    // committed, which is the only thing that settles the incident.
    listing.revision = stubCharacterMutationReceipt.saveRevision;
    clickShell("reload list");
    await waitFor(() => expect(screen.queryByText(/The slot operation was applied/)).toBeNull());

    // A settled incident stays settled: a later revision is not a new
    // desynchronisation of the operation that was already reflected.
    refreshFails = false;
    clickShell("drift revision");
    listing.revision = "9";
    clickShell("reload list");

    const deleteAgain = await screen.findByRole("button", { name: "Delete character" });
    expect(deleteAgain).toBeEnabled();
    expect(screen.queryByText(/The slot operation was applied/)).toBeNull();

    fireEvent.click(deleteAgain);
    fireEvent.click(screen.getByRole("button", { name: "Confirm" }));
    await waitFor(() => expect(deleteCharacter).toHaveBeenCalledTimes(2));
    await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
    expect(document.body.textContent).not.toContain("could not be refreshed");
  });
});
