import { QueryClient, type UseQueryResult } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";
import {
  makeEquipmentPort,
  stubCharacterEquipment,
  stubCharacterEquippedSpells,
  stubCharacterLoadout,
  stubCharacterPhysickMixture,
  stubCharacterPouchItems,
  stubCharacterQuickItems,
  TestProviders,
} from "../../test/renderWithProviders";
import { type noCharacter, queryKeys } from "../queryKeys";
import type {
  CharacterEquipment,
  CharacterEquippedSpells,
  CharacterLoadout,
  CharacterPhysickMixture,
  CharacterPouchItems,
  CharacterQuickItems,
  EquipmentPort,
  EquipmentRequest,
} from "./equipmentPort";
import {
  type EquipmentQuery,
  useCharacterLoadout,
  useEquipment,
  useEquippedSpells,
  usePhysickMixture,
  usePouchItems,
  useQuickItems,
} from "./useEquipment";

/**
 * The hooks are exercised through an injected `EquipmentPort` stub. The
 * generated bindings are never mocked here: that belongs to the adapter test.
 *
 * The client keeps the library defaults on purpose, so a hook that dropped its
 * own `retry: false` would be caught instead of being covered by a test-only
 * default.
 */
function setup(port: EquipmentPort) {
  const queryClient = new QueryClient();
  const wrapper = ({ children }: { children: ReactNode }) => (
    <TestProviders queryClient={queryClient} equipmentPort={port}>
      {children}
    </TestProviders>
  );

  return { queryClient, wrapper };
}

const request = { saveSessionID: "session-1", saveRevision: "0", characterID: 0 };
const backendRequest = { saveSessionID: "session-1", characterID: 0 };

type Getter<T> = (request: EquipmentRequest) => Promise<T>;

/**
 * One getter under test, kept generic so that each hook is checked against its
 * own port method and its own result type rather than through a widened one.
 */
type GetterCase<T> = {
  hook: (query: EquipmentQuery) => UseQueryResult<T, Error>;
  /** Replaces exactly this getter on an otherwise complete stub port. */
  portWith: (call: Getter<T>) => EquipmentPort;
  stub: T;
  key: (
    saveSessionID: string,
    characterID: number | typeof noCharacter,
    saveRevision: string,
  ) => readonly unknown[];
};

const equipmentCase: GetterCase<CharacterEquipment> = {
  hook: useEquipment,
  portWith: (call) => makeEquipmentPort({ getEquipment: call }),
  stub: stubCharacterEquipment,
  key: queryKeys.equipment,
};

const characterLoadoutCase: GetterCase<CharacterLoadout> = {
  hook: useCharacterLoadout,
  portWith: (call) => makeEquipmentPort({ getCharacterLoadout: call }),
  stub: stubCharacterLoadout,
  key: queryKeys.characterLoadout,
};

const quickItemsCase: GetterCase<CharacterQuickItems> = {
  hook: useQuickItems,
  portWith: (call) => makeEquipmentPort({ getQuickItems: call }),
  stub: stubCharacterQuickItems,
  key: queryKeys.quickItems,
};

const pouchItemsCase: GetterCase<CharacterPouchItems> = {
  hook: usePouchItems,
  portWith: (call) => makeEquipmentPort({ getPouchItems: call }),
  stub: stubCharacterPouchItems,
  key: queryKeys.pouchItems,
};

const physickMixtureCase: GetterCase<CharacterPhysickMixture> = {
  hook: usePhysickMixture,
  portWith: (call) => makeEquipmentPort({ getPhysickMixture: call }),
  stub: stubCharacterPhysickMixture,
  key: queryKeys.physickMixture,
};

const equippedSpellsCase: GetterCase<CharacterEquippedSpells> = {
  hook: useEquippedSpells,
  portWith: (call) => makeEquipmentPort({ getEquippedSpells: call }),
  stub: stubCharacterEquippedSpells,
  key: queryKeys.equippedSpells,
};

function describeGetter<T>(name: string, getter: GetterCase<T>) {
  const { hook, portWith, stub } = getter;

  describe(name, () => {
    it("passes the session and the slot to the port exactly as given", async () => {
      const call = vi.fn<Getter<T>>(() => Promise.resolve(stub));
      const { wrapper } = setup(portWith(call));

      const { result } = renderHook(() => hook(request), { wrapper });

      await waitFor(() => expect(result.current.data).toEqual(stub));
      // No trimming and no slot normalisation: the backend owns both.
      expect(call).toHaveBeenCalledExactlyOnceWith(backendRequest);
    });

    it("asks the backend for nothing without a session identifier", async () => {
      const call = vi.fn<Getter<T>>(() => Promise.resolve(stub));
      const { wrapper } = setup(portWith(call));

      const view = renderHook(
        ({ id }: { id?: string }) => hook({ ...request, saveSessionID: id }),
        {
          wrapper,
          initialProps: {},
        },
      );

      expect(view.result.current.fetchStatus).toBe("idle");
      // An empty identifier is just as absent as a missing one.
      view.rerender({ id: "" });
      expect(call).not.toHaveBeenCalled();

      view.rerender({ id: "session-1" });
      await waitFor(() => expect(view.result.current.data).toEqual(stub));
      expect(call).toHaveBeenCalledExactlyOnceWith(backendRequest);
    });

    it("asks the backend for nothing without a character slot", async () => {
      const call = vi.fn<Getter<T>>(() => Promise.resolve(stub));
      const { wrapper } = setup(portWith(call));

      const view = renderHook(
        ({ slot }: { slot?: number }) => hook({ ...request, characterID: slot }),
        { wrapper, initialProps: {} },
      );

      expect(view.result.current.fetchStatus).toBe("idle");
      expect(call).not.toHaveBeenCalled();

      // Slot 0 is an ordinary slot, not an absent one.
      view.rerender({ slot: 0 });
      await waitFor(() => expect(view.result.current.data).toEqual(stub));
      expect(call).toHaveBeenCalledExactlyOnceWith(backendRequest);
    });

    it("cannot reach the port through a manual refetch while an identifier is missing", async () => {
      const call = vi.fn<Getter<T>>(() => Promise.resolve(stub));
      const { wrapper } = setup(portWith(call));

      const withoutSession = renderHook(() => hook({ ...request, saveSessionID: undefined }), {
        wrapper,
      });
      const withoutSlot = renderHook(() => hook({ ...request, characterID: undefined }), {
        wrapper,
      });

      // `enabled` would still run the query function here; `skipToken` cannot.
      await withoutSession.result.current.refetch();
      await withoutSlot.result.current.refetch();

      expect(call).not.toHaveBeenCalled();
    });

    it("reports a rejected call without retrying it and without a fallback", async () => {
      const call = vi.fn<Getter<T>>(() => Promise.reject(new Error("bridge_call_failed")));
      const { wrapper } = setup(portWith(call));

      const { result } = renderHook(() => hook(request), { wrapper });

      await waitFor(() => expect(result.current.isError).toBe(true));
      expect(call).toHaveBeenCalledTimes(1);
      // The failure stays the query's own state: no local fallback value.
      expect(result.current.data).toBeUndefined();
    });

    it("fails as a wiring error when no EquipmentPortProvider is above the hook", () => {
      // The port is read before any query is set up, so a tree without the
      // provider fails immediately instead of silently rendering an empty view.
      expect(() => renderHook(() => hook(request))).toThrow(
        "EquipmentPortProvider is missing above this component",
      );
    });

    it("keeps two sessions and two slots in separate cache entries", async () => {
      const call = vi.fn<Getter<T>>((requested) =>
        Promise.resolve({ ...stub, saveSessionID: requested.saveSessionID }),
      );
      const { queryClient, wrapper } = setup(portWith(call));

      renderHook(() => hook({ saveSessionID: "session-1", saveRevision: "0", characterID: 0 }), {
        wrapper,
      });
      renderHook(() => hook({ saveSessionID: "session-2", saveRevision: "0", characterID: 0 }), {
        wrapper,
      });
      renderHook(() => hook({ saveSessionID: "session-1", saveRevision: "0", characterID: 1 }), {
        wrapper,
      });

      // Three distinct scopes, three calls and three cached entries: no pair of
      // them shared a key.
      await waitFor(() => expect(call).toHaveBeenCalledTimes(3));
      expect(queryClient.getQueryData(getter.key("session-1", 0, "0"))).toEqual(stub);
      expect(queryClient.getQueryData(getter.key("session-2", 0, "0"))).toEqual({
        ...stub,
        saveSessionID: "session-2",
      });
      expect(queryClient.getQueryData(getter.key("session-1", 1, "0"))).toEqual(stub);
    });
  });
}

describeGetter("useEquipment", equipmentCase);
describeGetter("useCharacterLoadout", characterLoadoutCase);
describeGetter("useQuickItems", quickItemsCase);
describeGetter("usePouchItems", pouchItemsCase);
describeGetter("usePhysickMixture", physickMixtureCase);
describeGetter("useEquippedSpells", equippedSpellsCase);

describe("the six equipment hooks together", () => {
  it("calls only its own port method", async () => {
    const port = makeEquipmentPort();
    const spies = {
      getCharacterLoadout: vi.fn(port.getCharacterLoadout),
      getEquipment: vi.fn(port.getEquipment),
      getQuickItems: vi.fn(port.getQuickItems),
      getPouchItems: vi.fn(port.getPouchItems),
      getPhysickMixture: vi.fn(port.getPhysickMixture),
      getEquippedSpells: vi.fn(port.getEquippedSpells),
    };
    const { wrapper } = setup(makeEquipmentPort(spies));

    const { result } = renderHook(() => useEquipment(request), { wrapper });

    await waitFor(() => expect(result.current.data).toEqual(stubCharacterEquipment));
    expect(spies.getEquipment).toHaveBeenCalledTimes(1);
    expect(spies.getCharacterLoadout).not.toHaveBeenCalled();
    expect(spies.getQuickItems).not.toHaveBeenCalled();
    expect(spies.getPouchItems).not.toHaveBeenCalled();
    expect(spies.getPhysickMixture).not.toHaveBeenCalled();
    expect(spies.getEquippedSpells).not.toHaveBeenCalled();
  });

  it("carries every backend value, length and order through untouched", async () => {
    const { wrapper } = setup(makeEquipmentPort());

    const equipment = renderHook(() => useEquipment(request), { wrapper });
    const loadout = renderHook(() => useCharacterLoadout(request), { wrapper });
    const quick = renderHook(() => useQuickItems(request), { wrapper });
    const pouch = renderHook(() => usePouchItems(request), { wrapper });
    const physick = renderHook(() => usePhysickMixture(request), { wrapper });
    const spells = renderHook(() => useEquippedSpells(request), { wrapper });

    await waitFor(() => expect(equipment.result.current.data).toBeDefined());
    await waitFor(() => expect(loadout.result.current.data).toBeDefined());
    await waitFor(() => expect(quick.result.current.data).toBeDefined());
    await waitFor(() => expect(pouch.result.current.data).toBeDefined());
    await waitFor(() => expect(physick.result.current.data).toBeDefined());
    await waitFor(() => expect(spells.result.current.data).toBeDefined());

    // Lengths and order are the backend's, and no hook trims, pads or sorts.
    expect(equipment.result.current.data?.slots).toEqual(stubCharacterEquipment.slots);
    expect(loadout.result.current.data).toEqual(stubCharacterLoadout);
    expect(equipment.result.current.data?.slots).toHaveLength(22);
    expect(quick.result.current.data?.items).toEqual(stubCharacterQuickItems.items);
    expect(pouch.result.current.data?.items).toEqual(stubCharacterPouchItems.items);
    expect(physick.result.current.data?.tears).toEqual(stubCharacterPhysickMixture.tears);
    expect(spells.result.current.data?.spells).toEqual(stubCharacterEquippedSpells.spells);

    // The maximum uint32 and the negative active slot survive the hook layer.
    expect(equipment.result.current.data?.slots[1]).toBe(0xffffffff);
    expect(quick.result.current.data?.activeQuick).toBe(-3);
    expect(physick.result.current.data?.tears[0]).toBe(0xffffffff);
    // An empty spell keeps the backend's own empty values.
    expect(spells.result.current.data?.spells[0]).toEqual({
      rawMagicParamID: 0xffffffff,
      resourceKey: "",
      name: "",
      memorySlots: 0,
    });
    // The two counts are the backend's own and are not recomputed from the
    // records, so they stay asymmetric.
    expect(spells.result.current.data?.usedMemorySlots).toBe(7);
    expect(spells.result.current.data?.availableMemorySlots).toBe(10);
  });
});

describe("equipment query keys", () => {
  const keys = (session: string, slot: number | typeof noCharacter, revision = "0") => ({
    equipment: queryKeys.equipment(session, slot, revision),
    characterLoadout: queryKeys.characterLoadout(session, slot, revision),
    quickItems: queryKeys.quickItems(session, slot, revision),
    pouchItems: queryKeys.pouchItems(session, slot, revision),
    physickMixture: queryKeys.physickMixture(session, slot, revision),
    equippedSpells: queryKeys.equippedSpells(session, slot, revision),
  });

  it("gives each of the six getters its own cache entry", () => {
    const built = Object.values(keys("session-1", 0)).map((key) => JSON.stringify(key));

    expect(new Set(built).size).toBe(6);
  });

  it("nests all six under the session prefix that CloseSave removes", () => {
    const prefix = queryKeys.saveSession("session-1");

    for (const built of Object.values(keys("session-1", 0))) {
      expect(built.slice(0, prefix.length)).toEqual([...prefix]);
    }
  });

  it("caches the six getters of one slot separately", async () => {
    const { queryClient, wrapper } = setup(makeEquipmentPort());

    renderHook(() => useEquipment(request), { wrapper });
    renderHook(() => useCharacterLoadout(request), { wrapper });
    renderHook(() => useQuickItems(request), { wrapper });
    renderHook(() => usePouchItems(request), { wrapper });
    renderHook(() => usePhysickMixture(request), { wrapper });
    renderHook(() => useEquippedSpells(request), { wrapper });

    await waitFor(() => expect(queryClient.getQueryCache().getAll()).toHaveLength(6));
    const scope = keys(request.saveSessionID, request.characterID);
    expect(queryClient.getQueryData(scope.equipment)).toEqual(stubCharacterEquipment);
    expect(queryClient.getQueryData(scope.characterLoadout)).toEqual(stubCharacterLoadout);
    expect(queryClient.getQueryData(scope.quickItems)).toEqual(stubCharacterQuickItems);
    expect(queryClient.getQueryData(scope.pouchItems)).toEqual(stubCharacterPouchItems);
    expect(queryClient.getQueryData(scope.physickMixture)).toEqual(stubCharacterPhysickMixture);
    expect(queryClient.getQueryData(scope.equippedSpells)).toEqual(stubCharacterEquippedSpells);
  });
});
