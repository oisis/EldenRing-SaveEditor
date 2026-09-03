import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type {
  CatalogItemDatabasePage,
  CatalogPort,
} from "../../../application/catalog/catalogPort";
import type { ItemMutationReceipt, ItemsPort } from "../../../application/items/itemsPort";
import {
  makeCatalogPort,
  makeItemsPort,
  renderApp,
  stubCatalogItemVariants,
} from "../../../test/renderWithProviders";
import { ItemDatabasePanel } from "./ItemDatabasePanel";

const firstPage: CatalogItemDatabasePage = {
  safetyProfile: "safe",
  resources: [
    {
      kind: "item",
      key: "weapon/uchigatana",
      gameID: 0x00bb8000,
      gameIDKnown: true,
      family: "weapon",
      category: "melee_armaments",
      subcategory: "katana",
      name: "Uchigatana",
      iconPath: "assets/icons/items/uchigatana.png",
      banRisk: false,
      cutContent: false,
      dlc: false,
      preOrder: false,
    },
    {
      kind: "item",
      key: "goods/unnamed",
      gameID: 0,
      gameIDKnown: false,
      family: "goods",
      category: "",
      subcategory: "",
      name: "",
      iconPath: "",
      banRisk: false,
      cutContent: false,
      dlc: false,
      preOrder: false,
    },
    {
      kind: "item",
      key: "goods/risky",
      gameID: 0x4000abcd,
      gameIDKnown: true,
      family: "goods",
      category: "consumables",
      subcategory: "",
      name: "Rune Arc",
      iconPath: "",
      banRisk: true,
      cutContent: false,
      dlc: false,
      preOrder: false,
    },
  ],
  categories: [
    { category: "consumables", count: 1 },
    { category: "melee_armaments", count: 1 },
  ],
  total: 45,
  page: 1,
  pageSize: 20,
};

function ports(overrides: { catalog?: Partial<CatalogPort>; items?: Partial<ItemsPort> } = {}) {
  const base = makeCatalogPort();
  const getItemDatabase = vi.fn(
    overrides.catalog?.getItemDatabase ?? (() => Promise.resolve(firstPage)),
  );
  const getResource = vi.fn(overrides.catalog?.getResource ?? base.getResource);
  const getItemVariants = vi.fn(overrides.catalog?.getItemVariants ?? base.getItemVariants);
  const addItemsToContainers = vi.fn(
    overrides.items?.addItemsToContainers ?? makeItemsPort().addItemsToContainers,
  );
  return {
    getItemDatabase,
    getResource,
    getItemVariants,
    addItemsToContainers,
    catalogPort: makeCatalogPort({ getItemDatabase, getResource, getItemVariants }),
    itemsPort: makeItemsPort({ addItemsToContainers }),
  };
}

type RenderOptions = {
  locale?: "en" | "pl";
  withSave?: boolean;
  showItemID?: boolean;
  applyMutationReceipt?: (receipt: ItemMutationReceipt) => Promise<unknown>;
};

function renderPanel(port: ReturnType<typeof ports>, options: RenderOptions = {}) {
  const save = options.withSave === true;
  return renderApp(
    <ItemDatabasePanel
      saveSessionID={save ? "session-1" : undefined}
      saveRevision={save ? "0" : undefined}
      characterID={save ? 0 : undefined}
      applyMutationReceipt={
        save ? (options.applyMutationReceipt ?? (() => Promise.resolve())) : undefined
      }
      sessionBusy={false}
    />,
    {
      catalogPort: port.catalogPort,
      itemsPort: port.itemsPort,
      locale: options.locale,
      showItemID: options.showItemID,
    },
  );
}

/** The exact column set of the results table, in render order. */
function columnNames(table: HTMLElement): string[] {
  return within(table)
    .getAllByRole("columnheader")
    .map((header) => header.textContent ?? "");
}

describe("ItemDatabasePanel", () => {
  it("reads the catalog with no save loaded and invents no missing value", async () => {
    const port = ports();
    await renderPanel(port);

    expect(await screen.findByRole("button", { name: "Uchigatana" })).toBeInTheDocument();
    expect(port.getItemDatabase).toHaveBeenCalledExactlyOnceWith({
      family: "",
      category: "",
      search: "",
      favoritesOnly: false,
      favorites: [],
      sort: "",
      page: 1,
      pageSize: 20,
    });
    expect(screen.getByText("Results: 45")).toBeInTheDocument();
    expect(screen.getByText("Name unavailable")).toBeInTheDocument();
    expect(document.body).not.toHaveTextContent("weapon/uchigatana");
  });

  it("passes every filter verbatim and returns to the first page", async () => {
    const port = ports({
      catalog: {
        getItemDatabase: (request) => Promise.resolve({ ...firstPage, page: request.page }),
      },
    });
    await renderPanel(port);

    await screen.findByText("Uchigatana");
    fireEvent.click(screen.getByRole("button", { name: "Next" }));
    await waitFor(() =>
      expect(port.getItemDatabase).toHaveBeenLastCalledWith(expect.objectContaining({ page: 2 })),
    );

    fireEvent.change(screen.getByRole("searchbox", { name: "Search items" }), {
      target: { value: "  Uchi  " },
    });
    await waitFor(() =>
      expect(port.getItemDatabase).toHaveBeenLastCalledWith(
        expect.objectContaining({ search: "  Uchi  ", page: 1 }),
      ),
    );

    fireEvent.change(screen.getByRole("combobox", { name: "Item family" }), {
      target: { value: "spirit_ash" },
    });
    await waitFor(() =>
      expect(port.getItemDatabase).toHaveBeenLastCalledWith(
        expect.objectContaining({ family: "spirit_ash" }),
      ),
    );
    fireEvent.change(screen.getByRole("combobox", { name: "Category" }), {
      target: { value: "consumables" },
    });
    await waitFor(() =>
      expect(port.getItemDatabase).toHaveBeenLastCalledWith(
        expect.objectContaining({ category: "consumables" }),
      ),
    );
    fireEvent.change(screen.getByRole("combobox", { name: "Sort order" }), {
      target: { value: "game_id" },
    });
    await waitFor(() =>
      expect(port.getItemDatabase).toHaveBeenLastCalledWith(
        expect.objectContaining({
          family: "spirit_ash",
          category: "consumables",
          sort: "game_id",
          search: "  Uchi  ",
          page: 1,
        }),
      ),
    );
  });

  it("filters by favourites through the backend rather than through the served page", async () => {
    const port = ports();
    await renderPanel(port);
    await screen.findByText("Uchigatana");

    fireEvent.click(screen.getByRole("button", { name: "Add Uchigatana to favorites" }));
    expect(
      await screen.findByRole("button", { name: "Remove Uchigatana from favorites" }),
    ).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Favorites only" }));
    await waitFor(() =>
      expect(port.getItemDatabase).toHaveBeenLastCalledWith(
        expect.objectContaining({
          favoritesOnly: true,
          favorites: [{ kind: "item", key: "weapon/uchigatana" }],
          page: 1,
        }),
      ),
    );
  });

  it("lists the exact table columns for Show Item ID off and on", async () => {
    const port = ports({
      catalog: {
        getItemDatabase: (request) =>
          Promise.resolve({ ...firstPage, pageSize: request.pageSize, total: 75 }),
      },
    });
    const first = await renderPanel(port);
    await screen.findByText("Uchigatana");
    fireEvent.click(screen.getByRole("button", { name: "Table" }));

    const table = await screen.findByRole("table", { name: "Item results" });
    expect(columnNames(table)).toEqual([
      "Select",
      "Name",
      "Family",
      "Category",
      "Favorite",
      "Details",
    ]);
    expect(within(table).getAllByRole("row")).toHaveLength(firstPage.resources.length + 1);
    await waitFor(() =>
      expect(port.getItemDatabase).toHaveBeenLastCalledWith(
        expect.objectContaining({ page: 1, pageSize: 50 }),
      ),
    );

    // `Show Item ID` is a presentational preference: it adds exactly one column
    // and changes no backend request.
    first.unmount();

    const second = ports();
    const view = await renderPanel(second, { showItemID: true });
    await screen.findByText("Uchigatana");
    fireEvent.click(screen.getByRole("button", { name: "Table" }));
    const withIdentifier = await screen.findByRole("table", { name: "Item results" });
    expect(columnNames(withIdentifier)).toEqual([
      "Select",
      "Name",
      "Item ID",
      "Family",
      "Category",
      "Favorite",
      "Details",
    ]);
    expect(within(withIdentifier).getByRole("cell", { name: "0x00BB8000" })).toBeInTheDocument();
    expect(second.getItemDatabase.mock.calls.every(([request]) => !("showItemID" in request))).toBe(
      true,
    );
    view.unmount();
  });

  it("moves through backend pages with previous and next controls", async () => {
    const port = ports({
      catalog: {
        getItemDatabase: (request) =>
          Promise.resolve({ ...firstPage, page: request.page === 0 ? 1 : request.page }),
      },
    });
    await renderPanel(port);

    await screen.findByText("Page 1");
    expect(screen.getByRole("button", { name: "Previous" })).toBeDisabled();
    fireEvent.click(screen.getByRole("button", { name: "Next" }));

    await screen.findByText("Page 2");
    expect(port.getItemDatabase).toHaveBeenLastCalledWith(expect.objectContaining({ page: 2 }));
    fireEvent.click(screen.getByRole("button", { name: "Previous" }));
    await screen.findByText("Page 1");
  });

  it("opens the shared modal with common detail and variants, then restores focus on close", async () => {
    const port = ports();
    await renderPanel(port);

    const tile = await screen.findByRole("button", { name: "Uchigatana" });
    tile.focus();
    fireEvent.click(tile);

    const dialog = await screen.findByRole("dialog", { name: "Item details" });
    expect(await within(dialog).findByRole("heading", { name: "Dagger" })).toBeInTheDocument();
    expect(within(dialog).getByText("A small dagger.")).toBeInTheDocument();
    expect(within(dialog).getByText("heavy · +0")).toBeInTheDocument();
    expect(within(dialog).getByText("+1")).toBeInTheDocument();
    expect(port.getResource).toHaveBeenCalledExactlyOnceWith({
      kind: "item",
      key: "weapon/uchigatana",
    });
    expect(port.getItemVariants).toHaveBeenCalledExactlyOnceWith({
      kind: "item",
      key: "weapon/uchigatana",
    });

    fireEvent.click(within(dialog).getByRole("button", { name: "Close" }));
    await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
    expect(tile).toHaveFocus();
  });

  it("offers no save mutation without an active save session", async () => {
    const port = ports();
    await renderPanel(port);

    fireEvent.click(await screen.findByRole("checkbox", { name: "Select Uchigatana" }));
    const bar = await screen.findByRole("group", { name: "Selected items" });
    expect(within(bar).queryByRole("button", { name: "Add" })).toBeNull();
    expect(within(bar).getByRole("button", { name: "Clear selection" })).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Uchigatana" }));
    const dialog = await screen.findByRole("dialog", { name: "Item details" });
    expect(within(dialog).queryByRole("button", { name: "Add" })).toBeNull();
    expect(port.addItemsToContainers).not.toHaveBeenCalled();
  });

  it("adds a multi-selection as one batch with separate Inventory and Storage amounts", async () => {
    const applyMutationReceipt = vi.fn((_receipt: ItemMutationReceipt) => Promise.resolve());
    const port = ports();
    await renderPanel(port, { withSave: true, applyMutationReceipt });

    fireEvent.click(await screen.findByRole("checkbox", { name: "Select Uchigatana" }));
    fireEvent.click(screen.getByRole("checkbox", { name: "Select Name unavailable" }));
    fireEvent.click(await screen.findByRole("button", { name: "Add" }));

    const dialog = await screen.findByRole("dialog", { name: "Add items" });
    const storageFields = within(dialog).getAllByRole("spinbutton", { name: "Storage" });
    fireEvent.change(storageFields[0], { target: { value: "5" } });
    fireEvent.click(within(dialog).getByRole("button", { name: "Add to save" }));

    await waitFor(() =>
      expect(port.addItemsToContainers).toHaveBeenCalledExactlyOnceWith({
        saveSessionID: "session-1",
        characterID: 0,
        expectedRevision: "0",
        confirmBanRisk: false,
        items: [
          {
            kind: "item",
            key: "weapon/uchigatana",
            inventoryQuantity: 1,
            storageQuantity: 5,
          },
          { kind: "item", key: "goods/unnamed", inventoryQuantity: 1, storageQuantity: 0 },
        ],
      }),
    );
    await waitFor(() => expect(applyMutationReceipt).toHaveBeenCalledOnce());
    expect(applyMutationReceipt.mock.calls[0][0]).toEqual(
      expect.objectContaining({ operationKind: "add_items_to_containers", saveRevision: "1" }),
    );
  });

  it("requires an explicit confirmation before a ban-risk resource is added", async () => {
    const port = ports();
    await renderPanel(port, { withSave: true });

    fireEvent.click(await screen.findByRole("checkbox", { name: "Select Rune Arc" }));
    fireEvent.click(await screen.findByRole("button", { name: "Add" }));

    const dialog = await screen.findByRole("dialog", { name: "Add items" });
    const submit = within(dialog).getByRole("button", { name: "Add to save" });
    expect(submit).toBeDisabled();

    const confirmation = within(dialog).getByRole("checkbox", { name: /ban risk/i });
    fireEvent.click(confirmation);
    expect(submit).toBeEnabled();
    fireEvent.click(submit);

    await waitFor(() =>
      expect(port.addItemsToContainers).toHaveBeenCalledExactlyOnceWith(
        expect.objectContaining({ confirmBanRisk: true }),
      ),
    );
  });

  it("keeps detail and variant failures independent and hides transport errors", async () => {
    const port = ports({
      catalog: {
        getResource: () => Promise.reject(new Error("bridge_call_failed /Users/private")),
        getItemVariants: () => Promise.resolve(stubCatalogItemVariants),
      },
    });
    await renderPanel(port);

    fireEvent.click(await screen.findByRole("button", { name: "Uchigatana" }));
    const dialog = await screen.findByRole("dialog", { name: "Item details" });
    expect(await within(dialog).findByRole("alert")).toHaveTextContent(
      "Unable to load item details.",
    );
    expect(document.body).not.toHaveTextContent("bridge_call_failed");
    expect(document.body).not.toHaveTextContent("/Users/private");
  });

  it("renders safe loading, empty and failed-list states", async () => {
    const pending = ports({ catalog: { getItemDatabase: () => new Promise(() => undefined) } });
    const pendingView = await renderPanel(pending);
    expect(screen.getByRole("status")).toHaveTextContent("Loading items…");
    pendingView.unmount();

    const empty = ports({
      catalog: {
        getItemDatabase: () =>
          Promise.resolve({
            safetyProfile: "safe",
            resources: [],
            categories: [],
            total: 0,
            page: 1,
            pageSize: 20,
          }),
      },
    });
    const emptyView = await renderPanel(empty);
    expect(await screen.findByText("No items match the current search.")).toBeInTheDocument();
    emptyView.unmount();

    const failed = ports({
      catalog: { getItemDatabase: () => Promise.reject(new Error("bridge_call_failed secret")) },
    });
    await renderPanel(failed);
    expect(await screen.findByRole("alert")).toHaveTextContent("Unable to load the Item Database.");
    expect(document.body).not.toHaveTextContent("bridge_call_failed");
  });

  it("renders the panel controls and safe list failure in Polish", async () => {
    const port = ports();
    await renderPanel(port, { locale: "pl" });

    expect(await screen.findByRole("button", { name: "Uchigatana" })).toBeInTheDocument();
    expect(screen.getByRole("searchbox", { name: "Szukaj przedmiotów" })).toBeInTheDocument();
    expect(screen.getByRole("combobox", { name: "Rodzina przedmiotu" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Siatka" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Tabela" })).toBeInTheDocument();
    expect(screen.getByText("Wyniki: 45")).toBeInTheDocument();
  });
});
