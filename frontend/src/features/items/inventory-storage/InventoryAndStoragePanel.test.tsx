import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { useState } from "react";
import { describe, expect, it, vi } from "vitest";
import type {
  CatalogPort,
  CatalogResourcePresentationIdentity,
} from "../../../application/catalog/catalogPort";
import type {
  ItemPage,
  ItemPageRequest,
  ItemRecord,
  ItemsPort,
} from "../../../application/items/itemsPort";
import {
  makeCatalogPort,
  makeItemsPort,
  renderApp,
  stubCatalogResourceDetail,
  stubInventoryPage,
} from "../../../test/renderWithProviders";
import {
  InventoryAndStoragePanel,
  type InventoryAndStoragePanelProps,
} from "./InventoryAndStoragePanel";

const uchigatana: ItemRecord = stubInventoryPage.records[0];

function record(overrides: Partial<ItemRecord>): ItemRecord {
  return { ...uchigatana, ...overrides };
}

function page(overrides: Partial<ItemPage>): ItemPage {
  return { ...stubInventoryPage, ...overrides };
}

const inventoryPage = page({
  records: [record({ ownedItemID: "inv-1", key: "weapon/uchigatana", physicalIndex: 3 })],
  total: 45,
  page: 1,
  pageSize: 30,
});

const storagePage = page({
  records: [
    record({
      ownedItemID: "sto-1",
      kind: "goods",
      key: "weapon/uchigatana",
      physicalIndex: 9,
      quantity: 7,
    }),
  ],
  total: 1,
  page: 1,
  pageSize: 30,
});

/**
 * The batch answers with one name per exact identity so a test can prove that
 * the same `key` under two different `kind` values stays two resources.
 */
const presentationNames: Record<string, { name: string; iconPath: string }> = {
  "item/weapon/uchigatana": {
    name: "Uchigatana",
    iconPath: "assets/icons/items/uchigatana.png",
  },
  "goods/weapon/uchigatana": { name: "Golden Rune", iconPath: "" },
};

function makePorts(overrides: { items?: Partial<ItemsPort>; catalog?: Partial<CatalogPort> } = {}) {
  const getInventory = vi.fn(
    overrides.items?.getInventory ?? (() => Promise.resolve(inventoryPage)),
  );
  const getStorage = vi.fn(overrides.items?.getStorage ?? (() => Promise.resolve(storagePage)));
  const getResourcePresentationSummaries = vi.fn(
    overrides.catalog?.getResourcePresentationSummaries ??
      ((identities: readonly CatalogResourcePresentationIdentity[]) =>
        Promise.resolve({
          resources: identities.map(({ kind, key }) => ({
            kind,
            key,
            name: presentationNames[`${kind}/${key}`]?.name ?? "",
            iconPath: presentationNames[`${kind}/${key}`]?.iconPath ?? "",
          })),
        })),
  );
  const getResource = vi.fn(
    overrides.catalog?.getResource ?? (() => Promise.resolve(stubCatalogResourceDetail)),
  );

  return {
    getInventory,
    getStorage,
    getResourcePresentationSummaries,
    getResource,
    itemsPort: makeItemsPort({ getInventory, getStorage }),
    catalogPort: makeCatalogPort({ getResourcePresentationSummaries, getResource }),
  };
}

function renderPanel(
  ports: ReturnType<typeof makePorts>,
  options: { locale?: "en" | "pl"; saveSessionID?: string; characterID?: number } = {},
) {
  return renderApp(
    <InventoryAndStoragePanel
      saveSessionID={"saveSessionID" in options ? options.saveSessionID : "session-1"}
      characterID={"characterID" in options ? options.characterID : 0}
      containerSection="common"
    />,
    { itemsPort: ports.itemsPort, catalogPort: ports.catalogPort, locale: options.locale },
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

const servedPagePorts = {
  items: {
    getInventory: (request: ItemPageRequest) =>
      Promise.resolve({ ...inventoryPage, page: request.page }),
    getStorage: (request: ItemPageRequest) =>
      Promise.resolve({ ...storagePage, total: 45, page: request.page }),
  },
};

function cardsRequestedBy(
  getter: ReturnType<typeof makePorts>["getInventory"],
  identity: Partial<ItemPageRequest>,
): number[] {
  return getter.mock.calls
    .map(([request]) => request as ItemPageRequest)
    .filter((request) =>
      Object.entries(identity).every(
        ([field, value]) => request[field as keyof ItemPageRequest] === value,
      ),
    )
    .map((request) => request.page);
}

async function switchWorkspaceAfterPagingBoth(
  ports: ReturnType<typeof makePorts>,
  to: InventoryAndStoragePanelProps,
) {
  await renderApp(
    <WorkspaceSwitch
      from={{ saveSessionID: "session-1", characterID: 0, containerSection: "common" }}
      to={to}
    />,
    { itemsPort: ports.itemsPort, catalogPort: ports.catalogPort },
  );

  fireEvent.click(await screen.findByRole("button", { name: "Next inventory card" }));
  fireEvent.click(await screen.findByRole("button", { name: "Next storage card" }));
  await waitFor(() =>
    expect(ports.getInventory).toHaveBeenLastCalledWith(expect.objectContaining({ page: 2 })),
  );
  await waitFor(() =>
    expect(ports.getStorage).toHaveBeenLastCalledWith(expect.objectContaining({ page: 2 })),
  );

  fireEvent.click(screen.getByRole("button", { name: "switch workspace" }));
}

describe("InventoryAndStoragePanel", () => {
  it("renders one workspace holding both containers as full 5 × 6 cards", async () => {
    const ports = makePorts();
    await renderPanel(ports);

    const inventoryGrid = await screen.findByRole("region", { name: "Inventory items" });
    const storageGrid = screen.getByRole("region", { name: "Storage items" });
    expect(screen.getAllByRole("region", { name: "Inventory and Storage" })).toHaveLength(1);
    expect(inventoryGrid.children).toHaveLength(30);
    expect(storageGrid.children).toHaveLength(30);
    expect(within(inventoryGrid).getAllByRole("button")).toHaveLength(1);
    expect(within(storageGrid).getAllByRole("button")).toHaveLength(1);

    expect(ports.getInventory).toHaveBeenCalledExactlyOnceWith({
      saveSessionID: "session-1",
      characterID: 0,
      containerSection: "common",
      page: 1,
      pageSize: 30,
    });
    expect(ports.getStorage).toHaveBeenCalledExactlyOnceWith({
      saveSessionID: "session-1",
      characterID: 0,
      containerSection: "common",
      page: 1,
      pageSize: 30,
    });
  });

  it("names and illustrates tiles from the presentation batch and separates identical keys", async () => {
    const ports = makePorts();
    await renderPanel(ports);

    const inventoryGrid = await screen.findByRole("region", { name: "Inventory items" });
    const storageGrid = screen.getByRole("region", { name: "Storage items" });
    await within(inventoryGrid).findByText("Uchigatana");

    // The same key under two kinds resolves to two distinct catalog resources.
    expect(within(storageGrid).getByText("Golden Rune")).toBeInTheDocument();
    expect(ports.getResourcePresentationSummaries).toHaveBeenCalledExactlyOnceWith([
      { kind: "item", key: "weapon/uchigatana" },
      { kind: "goods", key: "weapon/uchigatana" },
    ]);

    const icon = within(inventoryGrid).getByRole("presentation", { hidden: true });
    expect(icon).toHaveAttribute("src", "/catalog-assets/assets/icons/items/uchigatana.png");
    expect(within(storageGrid).queryByRole("presentation", { hidden: true })).toBeNull();

    expect(document.body).not.toHaveTextContent("weapon/uchigatana");
    expect(document.body).not.toHaveTextContent("inv-1");
  });

  it("navigates the two container cards independently without numbered pagination", async () => {
    const ports = makePorts({
      items: {
        getInventory: (request) => Promise.resolve({ ...inventoryPage, page: request.page }),
      },
    });
    await renderPanel(ports);

    await screen.findByText("Card 1 of 2");
    expect(screen.getByRole("button", { name: "Previous inventory card" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Next storage card" })).toBeDisabled();
    expect(screen.queryByRole("button", { name: "2" })).toBeNull();

    fireEvent.click(screen.getByRole("button", { name: "Next inventory card" }));
    await waitFor(() =>
      expect(ports.getInventory).toHaveBeenLastCalledWith(expect.objectContaining({ page: 2 })),
    );
    await screen.findByText("Card 2 of 2");
    expect(ports.getStorage).toHaveBeenCalledOnce();
  });

  it("restarts both containers at card 1 after a session and character change", async () => {
    const ports = makePorts(servedPagePorts);
    const next = { saveSessionID: "session-2", characterID: 1, containerSection: "common" };
    await switchWorkspaceAfterPagingBoth(ports, next);

    await waitFor(() =>
      expect(ports.getInventory).toHaveBeenLastCalledWith(expect.objectContaining(next)),
    );
    await waitFor(() =>
      expect(ports.getStorage).toHaveBeenLastCalledWith(expect.objectContaining(next)),
    );

    const inventoryCards = cardsRequestedBy(ports.getInventory, { saveSessionID: "session-2" });
    const storageCards = cardsRequestedBy(ports.getStorage, { saveSessionID: "session-2" });
    expect(inventoryCards[0]).toBe(1);
    expect(storageCards[0]).toBe(1);
    expect(inventoryCards).not.toContain(2);
    expect(storageCards).not.toContain(2);
  });

  it("restarts both containers at card 1 after a container section change", async () => {
    const ports = makePorts(servedPagePorts);
    const next = { saveSessionID: "session-1", characterID: 0, containerSection: "key" };
    await switchWorkspaceAfterPagingBoth(ports, next);

    await waitFor(() =>
      expect(ports.getInventory).toHaveBeenLastCalledWith(expect.objectContaining(next)),
    );
    await waitFor(() =>
      expect(ports.getStorage).toHaveBeenLastCalledWith(expect.objectContaining(next)),
    );

    const inventoryCards = cardsRequestedBy(ports.getInventory, { containerSection: "key" });
    const storageCards = cardsRequestedBy(ports.getStorage, { containerSection: "key" });
    expect(inventoryCards[0]).toBe(1);
    expect(storageCards[0]).toBe(1);
    expect(inventoryCards).not.toContain(2);
    expect(storageCards).not.toContain(2);
  });

  it("switches to one shared table that keeps the container of every record", async () => {
    const ports = makePorts();
    await renderPanel(ports);

    await screen.findByText("Uchigatana");
    fireEvent.click(screen.getByRole("button", { name: "Table" }));

    const table = await screen.findByRole("table", { name: "Inventory and Storage records" });
    for (const column of ["Container", "Name", "Quantity", "Position", "Details"]) {
      expect(within(table).getByRole("columnheader", { name: column })).toBeInTheDocument();
    }
    const rows = within(table).getAllByRole("row");
    expect(rows).toHaveLength(3);
    expect(within(rows[1]).getByRole("cell", { name: "Inventory" })).toBeInTheDocument();
    expect(within(rows[2]).getByRole("cell", { name: "Storage" })).toBeInTheDocument();
    expect(within(rows[2]).getByRole("cell", { name: "7" })).toBeInTheDocument();
    expect(within(rows[2]).getByRole("cell", { name: "9" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Next inventory card" })).toBeInTheDocument();
    expect(table).not.toHaveTextContent("2147483658");
  });

  it("opens the detail modal for the exact selected identity and restores focus on close", async () => {
    const ports = makePorts();
    await renderPanel(ports);

    const tile = await screen.findByRole("button", { name: /Golden Rune/ });
    tile.focus();
    fireEvent.click(tile);

    const dialog = await screen.findByRole("dialog", { name: "Item details" });
    expect(ports.getResource).toHaveBeenCalledExactlyOnceWith({
      kind: "goods",
      key: "weapon/uchigatana",
    });
    expect(await within(dialog).findByRole("heading", { name: "Dagger" })).toBeInTheDocument();
    expect(within(dialog).getByText("Storage")).toBeInTheDocument();
    expect(within(dialog).getByText("7")).toBeInTheDocument();
    expect(within(dialog).getByText("9")).toBeInTheDocument();
    expect(within(dialog).getAllByText("600")).toHaveLength(2);
    expect(dialog).not.toHaveTextContent("weapon/uchigatana");

    fireEvent.click(within(dialog).getByRole("button", { name: "Close" }));
    await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
    expect(tile).toHaveFocus();
  });

  it("reports loading, empty and failed containers independently", async () => {
    const pending = makePorts({
      items: {
        getInventory: () => new Promise<ItemPage>(() => undefined),
        getStorage: () => Promise.resolve(page({ records: [], total: 0 })),
      },
    });
    const pendingView = await renderPanel(pending);
    expect(await screen.findByText("Loading the Inventory…")).toHaveAttribute("role", "status");
    expect(await screen.findByText("This Storage card is empty.")).toBeInTheDocument();
    pendingView.unmount();

    const failed = makePorts({
      items: { getStorage: () => Promise.reject(new Error("bridge_call_failed /Users/private")) },
    });
    await renderPanel(failed);
    expect(await screen.findByRole("alert")).toHaveTextContent("Unable to load the Storage Box.");
    expect(await screen.findByText("Uchigatana")).toBeInTheDocument();
    expect(document.body).not.toHaveTextContent("bridge_call_failed");
    expect(document.body).not.toHaveTextContent("/Users/private");
  });

  it("keeps save records when the presentation batch fails", async () => {
    const ports = makePorts({
      catalog: {
        getResourcePresentationSummaries: () =>
          Promise.reject(new Error("bridge_call_failed secret")),
      },
    });
    await renderPanel(ports);

    const inventoryGrid = await screen.findByRole("region", { name: "Inventory items" });
    expect(within(inventoryGrid).getAllByRole("button")).toHaveLength(1);
    expect(await screen.findByText("Item names and icons are unavailable.")).toBeInTheDocument();
    expect(within(inventoryGrid).getByText("Name unavailable")).toBeInTheDocument();
    expect(document.body).not.toHaveTextContent("bridge_call_failed");
  });

  it("keeps a failing detail query technical-free and outside both lists", async () => {
    const ports = makePorts({
      catalog: {
        getResource: () => Promise.reject(new Error("bridge_call_failed /Users/private")),
      },
    });
    await renderPanel(ports);

    fireEvent.click(await screen.findByRole("button", { name: /Uchigatana/ }));
    const dialog = await screen.findByRole("dialog", { name: "Item details" });
    expect(await within(dialog).findByRole("alert")).toHaveTextContent(
      "Unable to load item details.",
    );
    expect(within(dialog).getByText("Uchigatana")).toBeInTheDocument();
    expect(document.body).not.toHaveTextContent("bridge_call_failed");
    expect(document.body).not.toHaveTextContent("/Users/private");
    expect(
      screen.getByRole("region", { name: "Inventory items", hidden: true }).children,
    ).toHaveLength(30);
  });

  it("warns when the two containers were read at different save revisions", async () => {
    const ports = makePorts({
      items: { getStorage: () => Promise.resolve({ ...storagePage, saveRevision: "revision-2" }) },
    });
    await renderPanel(ports);

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "The Inventory and the Storage Box were read at different save revisions.",
    );
  });

  it("asks for nothing without a session or a character", async () => {
    const withoutSession = makePorts();
    const view = await renderPanel(withoutSession, { saveSessionID: undefined });
    expect(
      screen.getByText(
        "Load a save and select a character to read the Inventory and the Storage Box.",
      ),
    ).toBeInTheDocument();
    expect(screen.queryByRole("region", { name: "Inventory items" })).toBeNull();
    view.unmount();

    const withoutCharacter = makePorts();
    await renderPanel(withoutCharacter, { characterID: undefined });
    expect(withoutSession.getInventory).not.toHaveBeenCalled();
    expect(withoutSession.getStorage).not.toHaveBeenCalled();
    expect(withoutCharacter.getInventory).not.toHaveBeenCalled();
    expect(withoutCharacter.getStorage).not.toHaveBeenCalled();
    expect(withoutCharacter.getResourcePresentationSummaries).not.toHaveBeenCalled();
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

  it("offers no mutating control anywhere in the read-only workspace", async () => {
    const ports = makePorts();
    await renderPanel(ports);

    fireEvent.click(await screen.findByRole("button", { name: /Uchigatana/ }));
    await screen.findByRole("dialog", { name: "Item details" });

    for (const pattern of [/move/i, /delete/i, /remove/i, /favorite/i, /add/i, /save/i, /apply/i]) {
      expect(screen.queryByRole("button", { name: pattern })).toBeNull();
    }
    expect(screen.queryByRole("textbox")).toBeNull();
    expect(screen.queryByRole("spinbutton")).toBeNull();
    expect(screen.queryByRole("checkbox")).toBeNull();
  });
});
