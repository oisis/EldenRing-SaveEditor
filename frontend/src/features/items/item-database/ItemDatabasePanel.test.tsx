import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { CatalogPort, CatalogResourcesPage } from "../../../application/catalog/catalogPort";
import {
  makeCatalogPort,
  renderApp,
  stubCatalogItemVariants,
} from "../../../test/renderWithProviders";
import { ItemDatabasePanel } from "./ItemDatabasePanel";

const firstPage: CatalogResourcesPage = {
  resources: [
    { kind: "item", key: "weapon/uchigatana", family: "weapon", name: "Uchigatana" },
    { kind: "item", key: "goods/unnamed", family: "goods", name: "" },
    { kind: "item", key: "armor/blaidd", family: "armor", name: "Blaidd's Armor" },
  ],
  total: 45,
  page: 1,
  pageSize: 20,
};

function catalogPort(overrides: Partial<CatalogPort> = {}) {
  const base = makeCatalogPort();
  const getResources = vi.fn(overrides.getResources ?? (() => Promise.resolve(firstPage)));
  const getResource = vi.fn(overrides.getResource ?? base.getResource);
  const getItemVariants = vi.fn(overrides.getItemVariants ?? base.getItemVariants);
  return {
    getResources,
    getResource,
    getItemVariants,
    port: makeCatalogPort({ getResources, getResource, getItemVariants }),
  };
}

describe("ItemDatabasePanel", () => {
  it("renders the first backend page as a five-column grid without inventing missing data", async () => {
    const { port, getResources } = catalogPort();
    await renderApp(<ItemDatabasePanel />, { catalogPort: port });

    expect(await screen.findByRole("button", { name: /Uchigatana/ })).toBeInTheDocument();
    expect(getResources).toHaveBeenCalledExactlyOnceWith({
      resourceType: "item",
      endpointID: "",
      family: "",
      capability: "",
      search: "",
      page: 1,
      pageSize: 20,
    });
    expect(screen.getByText("Results: 45")).toBeInTheDocument();
    expect(screen.getByText("Name unavailable")).toBeInTheDocument();
    expect(document.body).not.toHaveTextContent("weapon/uchigatana");
    expect(document.body).not.toHaveTextContent("MENU_Knowledge_00100.png");
    expect(screen.queryByRole("button", { name: /favorite/i })).toBeNull();
    expect(screen.queryByRole("button", { name: /add/i })).toBeNull();
  });

  it("passes search and family filters verbatim and returns to the first page", async () => {
    const secondPage = { ...firstPage, page: 2 };
    const { port, getResources } = catalogPort({
      getResources: (request) => Promise.resolve(request.page === 2 ? secondPage : firstPage),
    });
    await renderApp(<ItemDatabasePanel />, { catalogPort: port });

    await screen.findByText("Uchigatana");
    fireEvent.click(screen.getByRole("button", { name: "Next" }));
    await waitFor(() =>
      expect(getResources).toHaveBeenLastCalledWith(expect.objectContaining({ page: 2 })),
    );

    fireEvent.change(screen.getByRole("searchbox", { name: "Search items" }), {
      target: { value: "  Uchi  " },
    });
    await waitFor(() =>
      expect(getResources).toHaveBeenLastCalledWith({
        resourceType: "item",
        endpointID: "",
        family: "",
        capability: "",
        search: "  Uchi  ",
        page: 1,
        pageSize: 20,
      }),
    );

    fireEvent.change(screen.getByRole("combobox", { name: "Item family" }), {
      target: { value: "spirit_ash" },
    });
    await waitFor(() =>
      expect(getResources).toHaveBeenLastCalledWith(
        expect.objectContaining({ family: "spirit_ash", search: "  Uchi  ", page: 1 }),
      ),
    );
  });

  it("uses TanStack Table for the semantic table view and requests its fixed row window", async () => {
    const tablePage = { ...firstPage, total: 75, pageSize: 50 };
    const { port, getResources } = catalogPort({
      getResources: (request) => Promise.resolve(request.pageSize === 50 ? tablePage : firstPage),
    });
    await renderApp(<ItemDatabasePanel />, { catalogPort: port });

    await screen.findByText("Uchigatana");
    fireEvent.click(screen.getByRole("button", { name: "Table" }));

    const table = await screen.findByRole("table", { name: "Item results" });
    expect(within(table).getByRole("columnheader", { name: "Name" })).toBeInTheDocument();
    expect(within(table).getByRole("columnheader", { name: "Family" })).toBeInTheDocument();
    expect(within(table).getAllByRole("row")).toHaveLength(firstPage.resources.length + 1);
    expect(screen.queryByRole("navigation", { name: "Item Database pages" })).toBeNull();
    await waitFor(() =>
      expect(getResources).toHaveBeenLastCalledWith(
        expect.objectContaining({ page: 1, pageSize: 50 }),
      ),
    );
  });

  it("moves through backend pages with previous and next controls instead of numbered pagination", async () => {
    const { port, getResources } = catalogPort({
      getResources: (request) =>
        Promise.resolve({ ...firstPage, page: request.page === 0 ? 1 : request.page }),
    });
    await renderApp(<ItemDatabasePanel />, { catalogPort: port });

    await screen.findByText("Page 1");
    expect(screen.getByRole("button", { name: "Previous" })).toBeDisabled();
    fireEvent.click(screen.getByRole("button", { name: "Next" }));

    await screen.findByText("Page 2");
    expect(getResources).toHaveBeenLastCalledWith(expect.objectContaining({ page: 2 }));
    fireEvent.click(screen.getByRole("button", { name: "Previous" }));
    await screen.findByText("Page 1");
  });

  it("opens the shared modal with common detail and variants, then restores focus on close", async () => {
    const { port, getResource, getItemVariants } = catalogPort();
    await renderApp(<ItemDatabasePanel />, { catalogPort: port });

    const tile = await screen.findByRole("button", { name: /Uchigatana/ });
    tile.focus();
    fireEvent.click(tile);

    const dialog = await screen.findByRole("dialog", { name: "Item details" });
    expect(await within(dialog).findByRole("heading", { name: "Dagger" })).toBeInTheDocument();
    expect(within(dialog).getByText("A small dagger.")).toBeInTheDocument();
    expect(within(dialog).getByText("heavy · +0")).toBeInTheDocument();
    expect(within(dialog).getByText("+1")).toBeInTheDocument();
    expect(getResource).toHaveBeenCalledExactlyOnceWith({
      kind: "item",
      key: "weapon/uchigatana",
    });
    expect(getItemVariants).toHaveBeenCalledExactlyOnceWith({
      kind: "item",
      key: "weapon/uchigatana",
    });
    expect(dialog).not.toHaveTextContent("MENU_Knowledge_00100.png");
    expect(dialog).not.toHaveTextContent("1000000");

    fireEvent.click(within(dialog).getByRole("button", { name: "Close" }));
    await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
    expect(tile).toHaveFocus();
  });

  it("keeps detail and variant failures independent and hides transport errors", async () => {
    const { port } = catalogPort({
      getResource: () => Promise.reject(new Error("bridge_call_failed /Users/private")),
      getItemVariants: () => Promise.resolve(stubCatalogItemVariants),
    });
    await renderApp(<ItemDatabasePanel />, { catalogPort: port });

    fireEvent.click(await screen.findByRole("button", { name: /Uchigatana/ }));
    const dialog = await screen.findByRole("dialog", { name: "Item details" });
    expect(await within(dialog).findByRole("alert")).toHaveTextContent(
      "Unable to load item details.",
    );
    expect(document.body).not.toHaveTextContent("bridge_call_failed");
    expect(document.body).not.toHaveTextContent("/Users/private");
  });

  it("renders safe loading, empty and failed-list states", async () => {
    const pending = catalogPort({ getResources: () => new Promise(() => undefined) });
    const pendingView = await renderApp(<ItemDatabasePanel />, { catalogPort: pending.port });
    expect(screen.getByRole("status")).toHaveTextContent("Loading items…");
    pendingView.unmount();

    const empty = catalogPort({
      getResources: () => Promise.resolve({ resources: [], total: 0, page: 1, pageSize: 20 }),
    });
    const emptyView = await renderApp(<ItemDatabasePanel />, { catalogPort: empty.port });
    expect(await screen.findByText("No items match the current search.")).toBeInTheDocument();
    emptyView.unmount();

    const failed = catalogPort({
      getResources: () => Promise.reject(new Error("bridge_call_failed secret")),
    });
    await renderApp(<ItemDatabasePanel />, { catalogPort: failed.port });
    expect(await screen.findByRole("alert")).toHaveTextContent("Unable to load the Item Database.");
    expect(document.body).not.toHaveTextContent("bridge_call_failed");
  });

  it("renders the panel controls and safe list failure in Polish", async () => {
    const { port } = catalogPort();
    await renderApp(<ItemDatabasePanel />, { catalogPort: port, locale: "pl" });

    expect(await screen.findByRole("button", { name: /Uchigatana/ })).toBeInTheDocument();
    expect(screen.getByRole("searchbox", { name: "Szukaj przedmiotów" })).toBeInTheDocument();
    expect(screen.getByRole("combobox", { name: "Rodzina przedmiotu" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Siatka" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Tabela" })).toBeInTheDocument();
    expect(screen.getByText("Wyniki: 45")).toBeInTheDocument();
  });
});
