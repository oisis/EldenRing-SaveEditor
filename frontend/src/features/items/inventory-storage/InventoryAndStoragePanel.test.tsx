import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { useState } from "react";
import { describe, expect, it, vi } from "vitest";
import type { CatalogPort } from "../../../application/catalog/catalogPort";
import type {
  ItemMutationReceipt,
  ItemsPort,
  OwnedItemRow,
  OwnedItemsPage,
  OwnedItemsRequest,
} from "../../../application/items/itemsPort";
import {
  makeCatalogPort,
  makeItemsPort,
  renderApp,
  stubCatalogResourceDetail,
  stubOwnedInventoryPage,
  stubOwnedInventoryRow,
  stubOwnedStoragePage,
  stubOwnedStorageRow,
} from "../../../test/renderWithProviders";
import {
  InventoryAndStoragePanel,
  type InventoryAndStoragePanelProps,
} from "./InventoryAndStoragePanel";

function row(base: OwnedItemRow, overrides: Partial<OwnedItemRow>): OwnedItemRow {
  return { ...base, ...overrides, actions: { ...base.actions, ...overrides.actions } };
}

const inventoryPage: OwnedItemsPage = {
  ...stubOwnedInventoryPage,
  records: [row(stubOwnedInventoryRow, { ownedItemID: "inv-1", physicalIndex: 3 })],
  total: 45,
};

const storagePage: OwnedItemsPage = {
  ...stubOwnedStoragePage,
  records: [
    row(stubOwnedStorageRow, {
      ownedItemID: "sto-1",
      kind: "goods",
      name: "Golden Rune",
      iconPath: "",
      physicalIndex: 9,
      quantity: 7,
      maxQuantity: 99,
      category: "consumables",
    }),
  ],
  total: 1,
};

function makePorts(overrides: { items?: Partial<ItemsPort>; catalog?: Partial<CatalogPort> } = {}) {
  const getOwnedItems = vi.fn(
    overrides.items?.getOwnedItems ??
      ((request: OwnedItemsRequest) =>
        Promise.resolve(
          request.container === "storage"
            ? { ...storagePage, page: request.page }
            : { ...inventoryPage, page: request.page },
        )),
  );
  const moveOwnedItemsToStorage = vi.fn(
    overrides.items?.moveOwnedItemsToStorage ?? makeItemsPort().moveOwnedItemsToStorage,
  );
  const moveOwnedItemsToInventory = vi.fn(
    overrides.items?.moveOwnedItemsToInventory ?? makeItemsPort().moveOwnedItemsToInventory,
  );
  const removeOwnedItems = vi.fn(
    overrides.items?.removeOwnedItems ?? makeItemsPort().removeOwnedItems,
  );
  const reorderInventoryItems = vi.fn(
    overrides.items?.reorderInventoryItems ?? makeItemsPort().reorderInventoryItems,
  );
  const setOwnedItemQuantity = vi.fn(
    overrides.items?.setOwnedItemQuantity ?? makeItemsPort().setOwnedItemQuantity,
  );
  const getResource = vi.fn(
    overrides.catalog?.getResource ?? (() => Promise.resolve(stubCatalogResourceDetail)),
  );

  return {
    getOwnedItems,
    moveOwnedItemsToStorage,
    moveOwnedItemsToInventory,
    removeOwnedItems,
    reorderInventoryItems,
    setOwnedItemQuantity,
    getResource,
    itemsPort: makeItemsPort({
      getOwnedItems,
      moveOwnedItemsToStorage,
      moveOwnedItemsToInventory,
      removeOwnedItems,
      reorderInventoryItems,
      setOwnedItemQuantity,
    }),
    catalogPort: makeCatalogPort({ getResource }),
  };
}

type RenderOptions = {
  locale?: "en" | "pl";
  saveSessionID?: string | undefined;
  saveRevision?: string | undefined;
  characterID?: number | undefined;
  sessionBusy?: boolean;
  showItemID?: boolean;
  applyMutationReceipt?: (receipt: ItemMutationReceipt) => Promise<unknown>;
};

function renderPanel(ports: ReturnType<typeof makePorts>, options: RenderOptions = {}) {
  return renderApp(
    <InventoryAndStoragePanel
      saveSessionID={"saveSessionID" in options ? options.saveSessionID : "session-1"}
      saveRevision={"saveRevision" in options ? options.saveRevision : "0"}
      characterID={"characterID" in options ? options.characterID : 0}
      containerSection="common"
      applyMutationReceipt={options.applyMutationReceipt ?? (() => Promise.resolve())}
      sessionBusy={options.sessionBusy ?? false}
    />,
    {
      itemsPort: ports.itemsPort,
      catalogPort: ports.catalogPort,
      locale: options.locale,
      showItemID: options.showItemID,
    },
  );
}

/**
 * Switches the whole workspace identity in one commit, the way the surrounding
 * screen will: the panel keeps its instance and only its props change.
 */
function WorkspaceSwitch({
  from,
  to,
}: {
  from: InventoryAndStoragePanelProps;
  to: InventoryAndStoragePanelProps;
}) {
  const [props, setProps] = useState(from);
  return (
    <>
      <button type="button" onClick={() => setProps(to)}>
        switch workspace
      </button>
      <InventoryAndStoragePanel {...props} />
    </>
  );
}

function requestsFor(
  getter: ReturnType<typeof makePorts>["getOwnedItems"],
  identity: Partial<OwnedItemsRequest>,
): OwnedItemsRequest[] {
  return getter.mock.calls
    .map(([request]) => request as OwnedItemsRequest)
    .filter((request) =>
      Object.entries(identity).every(
        ([field, value]) => request[field as keyof OwnedItemsRequest] === value,
      ),
    );
}

/** The exact column set of the shared table, in render order. */
function columnNames(table: HTMLElement): string[] {
  return within(table)
    .getAllByRole("columnheader")
    .map((header) => header.textContent ?? "");
}

async function openTable() {
  fireEvent.click(screen.getByRole("button", { name: "Table" }));
  return screen.findByRole("table", { name: "Inventory and Storage records" });
}

describe("InventoryAndStoragePanel", () => {
  it("renders one workspace holding both containers as full 5 × 6 cards", async () => {
    const ports = makePorts();
    await renderPanel(ports);

    await screen.findByRole("button", { name: "Uchigatana" });
    const inventoryGrid = await screen.findByRole("region", { name: "Inventory items" });
    const storageGrid = screen.getByRole("region", { name: "Storage items" });
    expect(screen.getAllByRole("region", { name: "Inventory and Storage" })).toHaveLength(1);
    expect(inventoryGrid.children).toHaveLength(30);
    expect(storageGrid.children).toHaveLength(30);

    expect(requestsFor(ports.getOwnedItems, { container: "inventory" })[0]).toEqual({
      saveSessionID: "session-1",
      characterID: 0,
      container: "inventory",
      containerSection: "common",
      search: "",
      category: "",
      favoritesOnly: false,
      favorites: [],
      sort: "",
      page: 1,
      pageSize: 30,
    });
    expect(requestsFor(ports.getOwnedItems, { container: "storage" })[0]).toEqual(
      expect.objectContaining({ container: "storage", page: 1, pageSize: 30 }),
    );
  });

  it("gives one tile exactly the controls its record allows, with accessible names", async () => {
    const ports = makePorts();
    await renderPanel(ports);

    const tile = await screen.findByRole("button", { name: "Uchigatana" });
    const inventoryGrid = screen.getByRole("region", { name: "Inventory items" });
    // Three controls and no more: open the details, select the record, toggle
    // the presentational favourite.
    expect(within(inventoryGrid).getAllByRole("button")).toHaveLength(2);
    expect(tile).toBeInTheDocument();
    expect(
      within(inventoryGrid).getByRole("button", { name: "Add Uchigatana to favorites" }),
    ).toBeInTheDocument();
    expect(
      within(inventoryGrid).getByRole("checkbox", { name: "Select Uchigatana" }),
    ).toBeInTheDocument();
    // The icon comes from the backend row and is decorative.
    const icons = inventoryGrid.querySelectorAll("img");
    expect(icons).toHaveLength(1);
    expect(icons[0]).toHaveAttribute("src", "/catalog-assets/assets/icons/items/uchigatana.png");
    expect(icons[0]).toHaveAttribute("alt", "");

    const storageGrid = screen.getByRole("region", { name: "Storage items" });
    expect(within(storageGrid).getByText("Golden Rune")).toBeInTheDocument();
    expect(storageGrid.querySelectorAll("img")).toHaveLength(0);
    expect(document.body).not.toHaveTextContent("weapon/uchigatana");
    expect(document.body).not.toHaveTextContent("inv-1");
  });

  it("sends every filter and the favourites to the backend, never filtering a served page", async () => {
    const ports = makePorts();
    await renderPanel(ports);
    await screen.findByRole("button", { name: "Uchigatana" });

    fireEvent.click(screen.getByRole("button", { name: "Add Uchigatana to favorites" }));
    fireEvent.change(screen.getByRole("searchbox", { name: "Search items" }), {
      target: { value: "uchi" },
    });
    fireEvent.change(screen.getByRole("combobox", { name: "Sort order" }), {
      target: { value: "name" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Favorites only" }));

    await waitFor(() =>
      expect(ports.getOwnedItems).toHaveBeenLastCalledWith(
        expect.objectContaining({
          search: "uchi",
          sort: "name",
          favoritesOnly: true,
          favorites: [{ kind: "item", key: "weapon/uchigatana" }],
          page: 1,
        }),
      ),
    );
  });

  it("navigates the two container cards independently without numbered pagination", async () => {
    const ports = makePorts();
    await renderPanel(ports);

    await screen.findByText("Card 1 of 2");
    expect(screen.getByRole("button", { name: "Previous inventory card" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Next storage card" })).toBeDisabled();
    expect(screen.queryByRole("button", { name: "2" })).toBeNull();

    fireEvent.click(screen.getByRole("button", { name: "Next inventory card" }));
    await waitFor(() =>
      expect(ports.getOwnedItems).toHaveBeenLastCalledWith(
        expect.objectContaining({ container: "inventory", page: 2 }),
      ),
    );
    await screen.findByText("Card 2 of 2");
    expect(requestsFor(ports.getOwnedItems, { container: "storage" }).map((r) => r.page)).toEqual([
      1,
    ]);
  });

  it("restarts both containers at card 1 after a workspace identity change", async () => {
    const ports = makePorts();
    const base: InventoryAndStoragePanelProps = {
      saveSessionID: "session-1",
      saveRevision: "0",
      characterID: 0,
      containerSection: "common",
      applyMutationReceipt: () => Promise.resolve(),
      sessionBusy: false,
    };
    const next: InventoryAndStoragePanelProps = {
      ...base,
      saveSessionID: "session-2",
      characterID: 1,
      containerSection: "key",
    };
    await renderApp(<WorkspaceSwitch from={base} to={next} />, {
      itemsPort: ports.itemsPort,
      catalogPort: ports.catalogPort,
    });

    const nextInventory = await screen.findByRole("button", { name: "Next inventory card" });
    await waitFor(() => expect(nextInventory).toBeEnabled());
    fireEvent.click(nextInventory);
    await waitFor(() =>
      expect(ports.getOwnedItems).toHaveBeenLastCalledWith(
        expect.objectContaining({ container: "inventory", page: 2 }),
      ),
    );

    fireEvent.click(screen.getByRole("button", { name: "switch workspace" }));
    await waitFor(() =>
      expect(
        requestsFor(ports.getOwnedItems, { saveSessionID: "session-2" }).length,
      ).toBeGreaterThan(0),
    );
    const pages = requestsFor(ports.getOwnedItems, { saveSessionID: "session-2" }).map(
      (request) => request.page,
    );
    expect(pages[0]).toBe(1);
    expect(pages).not.toContain(2);
    expect(
      requestsFor(ports.getOwnedItems, { saveSessionID: "session-2" }).every(
        (request) => request.characterID === 1 && request.containerSection === "key",
      ),
    ).toBe(true);
  });

  it("lists the exact table columns for Show Item ID off and on", async () => {
    const ports = makePorts();
    const first = await renderPanel(ports);
    await screen.findByText("Uchigatana");

    const table = await openTable();
    expect(columnNames(table)).toEqual([
      "Select",
      "Container",
      "Name",
      "Category",
      "Quantity",
      "Position",
      "Favorite",
      "Details",
    ]);

    // `Show Item ID` is a presentational preference of the host, so switching it
    // adds exactly one column and changes no backend request.
    first.unmount();

    const second = makePorts();
    const view = await renderPanel(second, { showItemID: true });
    await screen.findAllByText("Uchigatana");
    const withIdentifier = await openTable();
    expect(columnNames(withIdentifier)).toEqual([
      "Select",
      "Container",
      "Name",
      "Item ID",
      "Category",
      "Quantity",
      "Position",
      "Favorite",
      "Details",
    ]);
    expect(within(withIdentifier).getAllByRole("cell", { name: "0x00BB8000" })).toHaveLength(2);
    // The preference changes no backend request: the same eleven arguments are
    // sent with it on and off.
    expect(second.getOwnedItems.mock.calls.map(([request]) => request)).toEqual(
      ports.getOwnedItems.mock.calls
        .map(([request]) => request)
        .slice(0, second.getOwnedItems.mock.calls.length),
    );
    view.unmount();
  });

  it("shows the quantity as Owned / Max and commits an edited value once", async () => {
    const ports = makePorts();
    await renderPanel(ports);
    await screen.findByText("Uchigatana");
    const table = await openTable();

    const field = within(table).getByRole("spinbutton", { name: "Quantity of Golden Rune" });
    expect(field).toHaveValue(7);
    expect(within(table).getAllByText("/ 99").length).toBeGreaterThan(0);

    fireEvent.change(field, { target: { value: "12" } });
    fireEvent.blur(field);

    await waitFor(() =>
      expect(ports.setOwnedItemQuantity).toHaveBeenCalledExactlyOnceWith({
        saveSessionID: "session-1",
        characterID: 0,
        expectedRevision: "0",
        ownedItemID: "sto-1",
        quantity: 12,
      }),
    );
  });

  it("commits one atomic batch for a multi-selection and applies its receipt once", async () => {
    const applyMutationReceipt = vi.fn((_receipt: ItemMutationReceipt) => Promise.resolve());
    const ports = makePorts({
      items: {
        getOwnedItems: (request: OwnedItemsRequest) =>
          Promise.resolve(
            request.container === "storage"
              ? { ...storagePage, records: [] }
              : {
                  ...inventoryPage,
                  records: [
                    row(stubOwnedInventoryRow, { ownedItemID: "inv-1" }),
                    row(stubOwnedInventoryRow, {
                      ownedItemID: "inv-2",
                      name: "Rusted Key",
                      physicalIndex: 5,
                    }),
                  ],
                },
          ),
      },
    });
    await renderPanel(ports, { applyMutationReceipt });

    fireEvent.click(await screen.findByRole("checkbox", { name: "Select Uchigatana" }));
    fireEvent.click(screen.getByRole("checkbox", { name: "Select Rusted Key" }));
    fireEvent.click(await screen.findByRole("button", { name: "Move to Storage" }));

    await waitFor(() =>
      expect(ports.moveOwnedItemsToStorage).toHaveBeenCalledExactlyOnceWith({
        saveSessionID: "session-1",
        characterID: 0,
        expectedRevision: "0",
        ownedItemIDs: ["inv-1", "inv-2"],
      }),
    );
    // One receipt for the whole batch, applied exactly once through the shared
    // path; the panel invalidates nothing itself.
    await waitFor(() => expect(applyMutationReceipt).toHaveBeenCalledOnce());
    expect(applyMutationReceipt.mock.calls[0][0]).toEqual(
      expect.objectContaining({
        operationKind: "move_owned_items_to_storage",
        saveRevision: "1",
        changedScopes: ["save.session", "inventory", "storage", "diagnostics.report"],
      }),
    );
    expect(ports.moveOwnedItemsToInventory).not.toHaveBeenCalled();
    expect(ports.removeOwnedItems).not.toHaveBeenCalled();
  });

  it("hides every batch action the backend capabilities do not allow", async () => {
    const ports = makePorts({
      items: {
        getOwnedItems: (request: OwnedItemsRequest) =>
          Promise.resolve(
            request.container === "storage"
              ? { ...storagePage, records: [] }
              : {
                  ...inventoryPage,
                  records: [
                    row(stubOwnedInventoryRow, {
                      ownedItemID: "inv-1",
                      actions: {
                        moveToStorage: false,
                        moveToInventory: false,
                        remove: false,
                        setQuantity: true,
                        reorder: false,
                      },
                    }),
                  ],
                },
          ),
      },
    });
    await renderPanel(ports);

    fireEvent.click(await screen.findByRole("checkbox", { name: "Select Uchigatana" }));
    const bar = await screen.findByRole("group", { name: "Selected items" });
    expect(within(bar).queryByRole("button", { name: "Move to Storage" })).toBeNull();
    expect(within(bar).queryByRole("button", { name: "Move to Inventory" })).toBeNull();
    expect(within(bar).queryByRole("button", { name: "Remove" })).toBeNull();
    expect(within(bar).getByRole("button", { name: "Clear selection" })).toBeInTheDocument();
  });

  it("offers no save mutation without an active session and revision", async () => {
    const ports = makePorts();
    const view = await renderPanel(ports, { saveSessionID: undefined });
    expect(
      screen.getByText(
        "Load a save and select a character to read the Inventory and the Storage Box.",
      ),
    ).toBeInTheDocument();
    expect(screen.queryByRole("region", { name: "Inventory items" })).toBeNull();
    expect(ports.getOwnedItems).not.toHaveBeenCalled();
    view.unmount();

    // A session with no revision cannot address a snapshot, so the record is
    // shown and every save mutation stays absent.
    const withoutRevision = makePorts();
    await renderPanel(withoutRevision, { saveRevision: undefined });
    expect(withoutRevision.getOwnedItems).not.toHaveBeenCalled();
    for (const pattern of [/^Move to Storage$/, /^Move to Inventory$/, /^Remove$/]) {
      expect(screen.queryByRole("button", { name: pattern })).toBeNull();
    }
  });

  it("keeps mutations out of reach while the session controller is busy", async () => {
    const ports = makePorts();
    await renderPanel(ports, { sessionBusy: true });

    fireEvent.click(await screen.findByRole("checkbox", { name: "Select Uchigatana" }));
    const bar = await screen.findByRole("group", { name: "Selected items" });
    expect(within(bar).queryByRole("button", { name: "Move to Storage" })).toBeNull();
    expect(within(bar).queryByRole("button", { name: "Remove" })).toBeNull();
  });

  it("moves an anchored group to an explicit position from the detail dialog", async () => {
    const ports = makePorts({
      items: {
        getOwnedItems: (request: OwnedItemsRequest) =>
          Promise.resolve(
            request.container === "storage"
              ? { ...storagePage, records: [] }
              : {
                  ...inventoryPage,
                  records: [
                    row(stubOwnedInventoryRow, { ownedItemID: "inv-1", orderPosition: 0 }),
                    row(stubOwnedInventoryRow, {
                      ownedItemID: "inv-2",
                      name: "Rusted Key",
                      orderPosition: 1,
                    }),
                  ],
                },
          ),
      },
    });
    await renderPanel(ports);

    fireEvent.click(await screen.findByRole("checkbox", { name: "Select Uchigatana" }));
    fireEvent.click(screen.getByRole("checkbox", { name: "Select Rusted Key" }));
    fireEvent.click(screen.getByRole("button", { name: "Rusted Key" }));

    const dialog = await screen.findByRole("dialog", { name: "Item details" });
    fireEvent.change(within(dialog).getByRole("spinbutton", { name: "Target position" }), {
      target: { value: "4" },
    });
    fireEvent.click(within(dialog).getByRole("button", { name: "Move to position" }));

    await waitFor(() =>
      expect(ports.reorderInventoryItems).toHaveBeenCalledExactlyOnceWith({
        saveSessionID: "session-1",
        characterID: 0,
        expectedRevision: "0",
        anchorOwnedItemID: "inv-2",
        groupOwnedItemIDs: ["inv-1", "inv-2"],
        targetPosition: 3,
      }),
    );
  });

  it("offers no manual order outside the container's own sort order", async () => {
    const ports = makePorts();
    await renderPanel(ports);
    await screen.findByRole("button", { name: "Uchigatana" });

    fireEvent.change(screen.getByRole("combobox", { name: "Sort order" }), {
      target: { value: "name" },
    });
    fireEvent.click(await screen.findByRole("button", { name: "Uchigatana" }));

    const dialog = await screen.findByRole("dialog", { name: "Item details" });
    expect(within(dialog).queryByRole("button", { name: "Move up" })).toBeNull();
    expect(within(dialog).queryByRole("button", { name: "Move down" })).toBeNull();
    expect(within(dialog).queryByRole("button", { name: "Move to position" })).toBeNull();
  });

  it("opens the detail modal for the exact record and restores focus on close", async () => {
    const ports = makePorts();
    await renderPanel(ports);

    const tile = await screen.findByRole("button", { name: "Golden Rune" });
    tile.focus();
    fireEvent.click(tile);

    const dialog = await screen.findByRole("dialog", { name: "Item details" });
    expect(ports.getResource).toHaveBeenCalledExactlyOnceWith({
      kind: "goods",
      key: "weapon/uchigatana",
    });
    expect(within(dialog).getByRole("heading", { name: "Golden Rune" })).toBeInTheDocument();
    expect(within(dialog).getByText("Storage")).toBeInTheDocument();
    expect(within(dialog).getByText("9")).toBeInTheDocument();
    expect(dialog).not.toHaveTextContent("weapon/uchigatana");

    fireEvent.click(within(dialog).getByRole("button", { name: "Close" }));
    await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
    expect(tile).toHaveFocus();
  });

  it("reports loading, empty and failed containers independently", async () => {
    const pending = makePorts({
      items: {
        getOwnedItems: (request: OwnedItemsRequest) =>
          request.container === "inventory"
            ? new Promise<OwnedItemsPage>(() => undefined)
            : Promise.resolve({ ...storagePage, records: [], total: 0 }),
      },
    });
    const pendingView = await renderPanel(pending);
    expect(await screen.findByText("Loading the Inventory…")).toHaveAttribute("role", "status");
    expect(
      await screen.findByText("No Storage item matches the current filters."),
    ).toBeInTheDocument();
    pendingView.unmount();

    const failed = makePorts({
      items: {
        getOwnedItems: (request: OwnedItemsRequest) =>
          request.container === "storage"
            ? Promise.reject(new Error("bridge_call_failed /Users/private"))
            : Promise.resolve(inventoryPage),
      },
    });
    await renderPanel(failed);
    expect(await screen.findByRole("alert")).toHaveTextContent("Unable to load the Storage Box.");
    expect(await screen.findByText("Uchigatana")).toBeInTheDocument();
    expect(document.body).not.toHaveTextContent("bridge_call_failed");
    expect(document.body).not.toHaveTextContent("/Users/private");
  });

  it("rejects a container response from a different save revision", async () => {
    const ports = makePorts({
      items: {
        getOwnedItems: (request: OwnedItemsRequest) =>
          Promise.resolve(
            request.container === "storage"
              ? { ...storagePage, saveRevision: "revision-2" }
              : inventoryPage,
          ),
      },
    });
    await renderPanel(ports);

    expect(await screen.findByRole("alert")).toHaveTextContent("Unable to load the Storage Box.");
    expect(await screen.findByText("Uchigatana")).toBeInTheDocument();
  });

  it("keeps a failing detail query technical-free and outside both lists", async () => {
    const ports = makePorts({
      catalog: {
        getResource: () => Promise.reject(new Error("bridge_call_failed /Users/private")),
      },
    });
    await renderPanel(ports);

    fireEvent.click(await screen.findByRole("button", { name: "Uchigatana" }));
    const dialog = await screen.findByRole("dialog", { name: "Item details" });
    expect(await within(dialog).findByRole("alert")).toHaveTextContent(
      "Unable to load item details.",
    );
    expect(within(dialog).getByRole("heading", { name: "Uchigatana" })).toBeInTheDocument();
    expect(document.body).not.toHaveTextContent("bridge_call_failed");
    expect(document.body).not.toHaveTextContent("/Users/private");
  });

  it("renders the workspace labels in Polish", async () => {
    const ports = makePorts();
    await renderPanel(ports, { locale: "pl" });

    expect(await screen.findByRole("region", { name: "Ekwipunek i skrzynia" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Ekwipunek" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Skrzynia" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Siatka" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Tabela" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Poprzednia karta ekwipunku" })).toBeInTheDocument();
  });
});
