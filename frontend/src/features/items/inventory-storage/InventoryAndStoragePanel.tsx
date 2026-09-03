import { Trans, useLingui } from "@lingui/react/macro";
import { createColumnHelper, tableFeatures, useTable } from "@tanstack/react-table";
import { type RefObject, useRef, useState } from "react";
import { catalogAssetURL } from "../../../application/catalog/catalogAssetURL";
import type { CatalogFact } from "../../../application/catalog/catalogPort";
import { useCatalogResource } from "../../../application/catalog/useCatalogResource";
import type { AppError } from "../../../application/errors/appError";
import type {
  ItemMutationReceipt,
  OwnedItemRow,
  OwnedItemsPage,
} from "../../../application/items/itemsPort";
import { useItemMutations } from "../../../application/items/useItemMutations";
import { useItemPreferences } from "../../../application/preferences/itemPreferences";
import { Badge } from "../../../ui/components/Badge/Badge";
import { Button } from "../../../ui/components/Button/Button";
import { Card } from "../../../ui/components/Card/Card";
import { Checkbox } from "../../../ui/components/Checkbox/Checkbox";
import { Dialog } from "../../../ui/components/Dialog/Dialog";
import { Input } from "../../../ui/components/Input/Input";
import { Select } from "../../../ui/components/Select/Select";
import { Table, TableFrame } from "../../../ui/components/Table/Table";
import {
  actionCell,
  alert,
  detailHeading,
  detailText,
  fact,
  factLabel,
  facts,
  factValue,
  message,
  panel,
  spacer,
  tableFrame,
  toolbar,
  viewSwitch,
  visuallyHidden,
} from "../../../ui/patterns/panel.css";
import {
  bulkBar,
  cardNavigation,
  cell,
  cellControl,
  cellControls,
  container as containerColumn,
  containerHead,
  containerTitle,
  detailHeader,
  detailIcon,
  detailIconPlaceholder,
  emptyCell,
  filterField,
  filters as filterRow,
  flagRow,
  grid,
  pagination,
  quantityField,
  searchField,
  tile,
  tileIcon,
  tileIconPlaceholder,
  tileName,
  tileQuantity,
  workspace,
} from "./InventoryAndStoragePanel.css";
import { type ItemContainerName, useItemsWorkspace } from "./useItemsWorkspace";

/**
 * The shared Inventory and Storage workspace.
 *
 * Both containers live in one panel, one filter set and one selection. The
 * backend owns everything that is not presentation: which records exist, their
 * names, icons, categories and limits, the order they are in, which mutations
 * each record accepts and how a batch is committed. This component owns the
 * view mode, the filters, the two card numbers and the selection intent, and it
 * renders no action the backend did not declare.
 *
 * Every mutation goes through the one shared receipt path supplied by the
 * session controller, so the revision, the history and the invalidated views
 * always come from the backend receipt and never from a second local mapping.
 */
export type InventoryAndStoragePanelProps = {
  saveSessionID: string | undefined;
  saveRevision: string | undefined;
  characterID: number | undefined;
  containerSection: string;
  /**
   * The session controller's post-mutation step. It is the single path from a
   * mutation receipt to the refreshed session and the invalidated views; this
   * panel never invalidates a query itself.
   */
  applyMutationReceipt: (receipt: ItemMutationReceipt) => Promise<unknown>;
  /** True while the session controller is running an operation of its own. */
  sessionBusy: boolean;
};

type WorkspaceView = "grid" | "table";

type WorkspaceRow = {
  container: ItemContainerName;
  record: OwnedItemRow;
};

/** One card is 5 × 6 physical fields, which is also the requested page size. */
const cardSize = 30;

/**
 * The sort orders the backend accepts. The empty value is the container's own
 * order and is the only one in which the manual Inventory order can be changed.
 */
const sortOrders = ["", "name", "category", "quantity"] as const;

const tableFeatureSet = tableFeatures({});
const columnHelper = createColumnHelper<typeof tableFeatureSet, WorkspaceRow>();

export function InventoryAndStoragePanel({
  saveSessionID,
  saveRevision,
  characterID,
  containerSection,
  applyMutationReceipt,
  sessionBusy,
}: InventoryAndStoragePanelProps) {
  const { t } = useLingui();
  const returnFocusRef = useRef<HTMLButtonElement | null>(null);
  const preferences = useItemPreferences();
  const mutations = useItemMutations(applyMutationReceipt);

  const [view, setView] = useState<WorkspaceView>("grid");
  const [search, setSearch] = useState("");
  const [category, setCategory] = useState("");
  const [sort, setSort] = useState<string>("");
  const [favoritesOnly, setFavoritesOnly] = useState(false);
  const [positionDraft, setPositionDraft] = useState("");
  // The two card numbers belong to one workspace identity. A session, slot or
  // section change starts both containers at card 1 during render, so a new
  // identity is never asked for a card number the previous one reached.
  const [pages, setPages] = useState({
    saveSessionID,
    characterID,
    containerSection,
    inventory: 1,
    storage: 1,
  });
  const isCurrentWorkspace =
    pages.saveSessionID === saveSessionID &&
    pages.characterID === characterID &&
    pages.containerSection === containerSection;
  if (!isCurrentWorkspace) {
    setPages({ saveSessionID, characterID, containerSection, inventory: 1, storage: 1 });
  }
  const inventoryPage = isCurrentWorkspace ? pages.inventory : 1;
  const storagePage = isCurrentWorkspace ? pages.storage : 1;
  const setInventoryPage = (card: number) =>
    setPages({
      saveSessionID,
      characterID,
      containerSection,
      inventory: card,
      storage: storagePage,
    });
  const setStoragePage = (card: number) =>
    setPages({
      saveSessionID,
      characterID,
      containerSection,
      inventory: inventoryPage,
      storage: card,
    });

  const model = useItemsWorkspace({
    saveSessionID,
    saveRevision,
    characterID,
    containerSection,
    filters: { search, category, sort, favoritesOnly },
    favorites: preferences.favorites,
    inventoryPage,
    storagePage,
    pageSize: cardSize,
  });

  const hasSlot = saveSessionID !== undefined && characterID !== undefined;
  // Without a session, a slot or a revision there is nothing a mutation could
  // address, so no save mutation is offered at all.
  const canMutate = hasSlot && saveRevision !== undefined && !sessionBusy && !mutations.isBusy;
  const nameUnavailable = t`Name unavailable`;

  const detail = useCatalogResource(model.openedRow?.kind, model.openedRow?.key);

  const labels: Record<ItemContainerName, ContainerLabels> = {
    inventory: {
      title: t`Inventory`,
      grid: t`Inventory items`,
      pages: t`Inventory cards`,
      previous: t`Previous inventory card`,
      next: t`Next inventory card`,
      loading: t`Loading the Inventory…`,
      error: t`Unable to load the Inventory.`,
      empty: t`No Inventory item matches the current filters.`,
    },
    storage: {
      title: t`Storage`,
      grid: t`Storage items`,
      pages: t`Storage cards`,
      previous: t`Previous storage card`,
      next: t`Next storage card`,
      loading: t`Loading the Storage Box…`,
      error: t`Unable to load the Storage Box.`,
      empty: t`No Storage item matches the current filters.`,
    },
  };
  const sortLabels: Record<(typeof sortOrders)[number], string> = {
    "": t`Sort: default order`,
    name: t`Sort: name`,
    category: t`Sort: category`,
    quantity: t`Sort: quantity`,
  };

  const rows: WorkspaceRow[] = [
    ...(model.inventory.data?.records ?? []).map((record) => ({
      container: "inventory" as const,
      record,
    })),
    ...(model.storage.data?.records ?? []).map((record) => ({
      container: "storage" as const,
      record,
    })),
  ];

  const changeFilter = (apply: () => void) => {
    apply();
    setInventoryPage(1);
    setStoragePage(1);
    model.clearSelection();
  };

  const runQuantity = async (record: OwnedItemRow, value: string) => {
    // The field sets a quantity, it never removes a record: 0 is not a
    // shorthand for "delete", so it is refused instead of being sent. A value
    // above the maximum the backend reported is refused for the same reason —
    // it can only be rejected. The backend stays the deciding validator; this
    // is the interface declining to send a request it already knows is invalid.
    const quantity = Number(value);
    if (!Number.isInteger(quantity) || quantity < 1) return;
    if (record.maxQuantityKnown && quantity > record.maxQuantity) return;
    if (!canMutate || saveSessionID === undefined || characterID === undefined) return;
    if (saveRevision === undefined) return;
    await mutations.setQuantity({
      saveSessionID,
      characterID,
      expectedRevision: saveRevision,
      ownedItemID: record.ownedItemID,
      quantity,
    });
  };

  const selectedInventory = model.selectedIn("inventory");
  const selectedStorage = model.selectedIn("storage");
  const selectedRows = rows.filter((row) =>
    model.isSelected(row.container, row.record.ownedItemID),
  );
  const canMoveSelectionToStorage =
    canMutate &&
    selectedStorage.length === 0 &&
    selectedInventory.length > 0 &&
    selectedRows.every((row) => row.record.actions.moveToStorage);
  const canMoveSelectionToInventory =
    canMutate &&
    selectedInventory.length === 0 &&
    selectedStorage.length > 0 &&
    selectedRows.every((row) => row.record.actions.moveToInventory);
  const canRemoveSelection =
    canMutate && selectedRows.length > 0 && selectedRows.every((row) => row.record.actions.remove);

  const runBatch = async (call: () => Promise<boolean>) => {
    if (await call()) model.clearSelection();
  };

  /**
   * The anchored group move. The record whose control was used is the anchor
   * and lands on `targetPosition`; the other selected records keep their side
   * of it. The group is the current selection when the anchor is part of it,
   * and the anchor alone otherwise.
   */
  const reorder = async (anchor: OwnedItemRow, targetPosition: number) => {
    if (!canMutate || saveSessionID === undefined || characterID === undefined) return;
    if (saveRevision === undefined || targetPosition < 0) return;
    const group = model.isSelected("inventory", anchor.ownedItemID)
      ? (model.inventory.data?.records ?? [])
          .filter(
            (record) => record.actions.reorder && model.isSelected("inventory", record.ownedItemID),
          )
          .map((record) => record.ownedItemID)
      : [anchor.ownedItemID];
    await mutations.reorderItems({
      saveSessionID,
      characterID,
      expectedRevision: saveRevision,
      anchorOwnedItemID: anchor.ownedItemID,
      groupOwnedItemIDs: group.includes(anchor.ownedItemID) ? group : [anchor.ownedItemID],
      targetPosition,
    });
  };

  const columns = columnHelper.columns([
    columnHelper.display({
      id: "select",
      header: t`Select`,
      cell: ({ row }) => (
        <Checkbox
          checked={model.isSelected(row.original.container, row.original.record.ownedItemID)}
          aria-label={t`Select ${displayName(row.original.record, nameUnavailable)}`}
          onChange={() =>
            model.toggleSelection(row.original.container, row.original.record.ownedItemID)
          }
        />
      ),
    }),
    columnHelper.display({
      id: "container",
      header: t`Container`,
      cell: ({ row }) => labels[row.original.container].title,
    }),
    columnHelper.display({
      id: "name",
      header: t`Name`,
      cell: ({ row }) => displayName(row.original.record, nameUnavailable),
    }),
    ...(preferences.showItemID
      ? [
          columnHelper.display({
            id: "itemID",
            header: t`Item ID`,
            cell: ({ row }) => formatItemID(row.original.record.gameID),
          }),
        ]
      : []),
    columnHelper.display({
      id: "category",
      header: t`Category`,
      cell: ({ row }) => row.original.record.category || t`Unavailable`,
    }),
    columnHelper.display({
      id: "quantity",
      header: t`Quantity`,
      cell: ({ row }) => (
        <QuantityCell
          record={row.original.record}
          editable={canMutate && row.original.record.actions.setQuantity}
          label={t`Quantity of ${displayName(row.original.record, nameUnavailable)}`}
          onCommit={(value) => void runQuantity(row.original.record, value)}
        />
      ),
    }),
    columnHelper.display({
      id: "position",
      header: t`Position`,
      cell: ({ row }) => String(row.original.record.physicalIndex),
    }),
    columnHelper.display({
      id: "favorite",
      header: t`Favorite`,
      cell: ({ row }) => (
        <FavoriteButton record={row.original.record} unavailable={nameUnavailable} />
      ),
    }),
    columnHelper.display({
      id: "details",
      header: t`Details`,
      cell: ({ row }) => (
        <Button
          size="sm"
          onClick={(event) => {
            returnFocusRef.current = event.currentTarget;
            model.openItem(row.original.container, row.original.record.ownedItemID);
          }}
        >
          <Trans>View</Trans>
        </Button>
      ),
    }),
  ]);
  const tableModel = useTable({ features: tableFeatureSet, columns, data: rows }, (state) => state);

  return (
    <Card aria-label={t`Inventory and Storage`} className={panel}>
      <div className={toolbar}>
        <h2 className={visuallyHidden}>
          <Trans>Inventory and Storage</Trans>
        </h2>
        <div className={filterRow}>
          <label className={visuallyHidden} htmlFor="items-workspace-search">
            <Trans>Search items</Trans>
          </label>
          <Input
            id="items-workspace-search"
            className={searchField}
            type="search"
            value={search}
            placeholder={t`Search items`}
            onChange={(event) => {
              const value = event.currentTarget.value;
              changeFilter(() => setSearch(value));
            }}
          />
          <label className={visuallyHidden} htmlFor="items-workspace-category">
            <Trans>Category</Trans>
          </label>
          <Select
            id="items-workspace-category"
            className={filterField}
            value={category}
            onChange={(event) => {
              const value = event.currentTarget.value;
              changeFilter(() => setCategory(value));
            }}
          >
            <option value="">{t`All categories`}</option>
            {model.categories.map((facet) => (
              <option key={facet.category} value={facet.category}>
                {`${facet.category} (${facet.count})`}
              </option>
            ))}
          </Select>
          <label className={visuallyHidden} htmlFor="items-workspace-sort">
            <Trans>Sort order</Trans>
          </label>
          <Select
            id="items-workspace-sort"
            className={filterField}
            value={sort}
            onChange={(event) => {
              const value = event.currentTarget.value;
              changeFilter(() => setSort(value));
            }}
          >
            {sortOrders.map((value) => (
              <option key={value || "default"} value={value}>
                {sortLabels[value]}
              </option>
            ))}
          </Select>
          <Button
            size="sm"
            pressed={favoritesOnly}
            onClick={() => changeFilter(() => setFavoritesOnly(!favoritesOnly))}
          >
            <Trans>Favorites only</Trans>
          </Button>
        </div>
        <span className={spacer} />
        <fieldset className={viewSwitch}>
          <legend className={visuallyHidden}>
            <Trans>View</Trans>
          </legend>
          <Button size="sm" pressed={view === "grid"} onClick={() => setView("grid")}>
            <Trans>Grid</Trans>
          </Button>
          <Button size="sm" pressed={view === "table"} onClick={() => setView("table")}>
            <Trans>Table</Trans>
          </Button>
        </fieldset>
      </div>

      {hasSlot ? null : (
        <p className={message}>
          <Trans>
            Load a save and select a character to read the Inventory and the Storage Box.
          </Trans>
        </p>
      )}

      {mutations.error ? (
        <p role="alert" className={alert}>
          {mutationErrorText(mutations.error, t`The change was not applied.`)}
        </p>
      ) : null}

      {hasSlot && model.selected.length > 0 ? (
        <fieldset className={bulkBar}>
          <legend className={visuallyHidden}>
            <Trans>Selected items</Trans>
          </legend>
          <Badge>
            <Trans>Selected: {model.selected.length}</Trans>
          </Badge>
          {canMoveSelectionToStorage ? (
            <Button
              size="sm"
              onClick={() =>
                void runBatch(() =>
                  mutations.moveToStorage({
                    saveSessionID: saveSessionID ?? "",
                    characterID: characterID ?? 0,
                    expectedRevision: saveRevision ?? "",
                    ownedItemIDs: selectedInventory,
                  }),
                )
              }
            >
              <Trans>Move to Storage</Trans>
            </Button>
          ) : null}
          {canMoveSelectionToInventory ? (
            <Button
              size="sm"
              onClick={() =>
                void runBatch(() =>
                  mutations.moveToInventory({
                    saveSessionID: saveSessionID ?? "",
                    characterID: characterID ?? 0,
                    expectedRevision: saveRevision ?? "",
                    ownedItemIDs: selectedStorage,
                  }),
                )
              }
            >
              <Trans>Move to Inventory</Trans>
            </Button>
          ) : null}
          {canRemoveSelection ? (
            <Button
              size="sm"
              onClick={() =>
                void runBatch(() =>
                  mutations.removeItems({
                    saveSessionID: saveSessionID ?? "",
                    characterID: characterID ?? 0,
                    expectedRevision: saveRevision ?? "",
                    ownedItemIDs: selectedRows.map((row) => row.record.ownedItemID),
                  }),
                )
              }
            >
              <Trans>Remove</Trans>
            </Button>
          ) : null}
          <Button size="sm" onClick={() => model.clearSelection()}>
            <Trans>Clear selection</Trans>
          </Button>
        </fieldset>
      ) : null}

      {hasSlot && view === "grid" ? (
        <div className={workspace}>
          {(["inventory", "storage"] as const).map((name) => (
            <div className={containerColumn} key={name}>
              <ContainerHeader
                labels={labels[name]}
                page={name === "inventory" ? model.inventory.data : model.storage.data}
                requestedPage={name === "inventory" ? inventoryPage : storagePage}
                onPage={(card) =>
                  name === "inventory" ? setInventoryPage(card) : setStoragePage(card)
                }
              />
              <ContainerStatus
                labels={labels[name]}
                query={name === "inventory" ? model.inventory : model.storage}
              />
              <ContainerGrid
                labels={labels[name]}
                container={name}
                records={
                  (name === "inventory" ? model.inventory.data : model.storage.data)?.records ?? []
                }
                model={model}
                nameUnavailable={nameUnavailable}
                returnFocusRef={returnFocusRef}
              />
            </div>
          ))}
        </div>
      ) : null}

      {hasSlot && view === "table" ? (
        <>
          <div className={cardNavigation}>
            <ContainerHeader
              labels={labels.inventory}
              page={model.inventory.data}
              requestedPage={inventoryPage}
              onPage={setInventoryPage}
            />
            <ContainerHeader
              labels={labels.storage}
              page={model.storage.data}
              requestedPage={storagePage}
              onPage={setStoragePage}
            />
          </div>
          <ContainerStatus labels={labels.inventory} query={model.inventory} />
          <ContainerStatus labels={labels.storage} query={model.storage} />
          <TableFrame className={tableFrame}>
            <Table aria-label={t`Inventory and Storage records`}>
              <thead>
                {tableModel.getHeaderGroups().map((headerGroup) => (
                  <tr key={headerGroup.id}>
                    {headerGroup.headers.map((header) => (
                      <th
                        key={header.id}
                        className={header.column.id === "details" ? actionCell : undefined}
                      >
                        {header.isPlaceholder ? null : <tableModel.FlexRender header={header} />}
                      </th>
                    ))}
                  </tr>
                ))}
              </thead>
              <tbody>
                {tableModel.getRowModel().rows.map((row) => (
                  <tr key={row.id}>
                    {row.getAllCells().map((tableCell) => (
                      <td
                        key={tableCell.id}
                        className={tableCell.column.id === "details" ? actionCell : undefined}
                      >
                        <tableModel.FlexRender cell={tableCell} />
                      </td>
                    ))}
                  </tr>
                ))}
              </tbody>
            </Table>
          </TableFrame>
        </>
      ) : null}

      <Dialog
        open={model.openedRow !== null}
        onOpenChange={(open) => {
          if (!open) model.closeItem();
        }}
        title={<Trans>Item details</Trans>}
        description={<Trans>Information from the save and GameCatalog.</Trans>}
        closeLabel={<Trans>Close</Trans>}
        returnFocusRef={returnFocusRef}
      >
        {model.openedRow ? (
          <ItemDetailContent
            container={labels[model.openedContainer ?? "inventory"].title}
            record={model.openedRow}
            detail={detail}
            canMutate={canMutate}
            sortIsDefault={sort === ""}
            positionDraft={positionDraft}
            onPositionDraft={setPositionDraft}
            onQuantity={(value) => void runQuantity(model.openedRow as OwnedItemRow, value)}
            onReorder={(target) => void reorder(model.openedRow as OwnedItemRow, target)}
            onMove={() => {
              const record = model.openedRow as OwnedItemRow;
              const input = {
                saveSessionID: saveSessionID ?? "",
                characterID: characterID ?? 0,
                expectedRevision: saveRevision ?? "",
                ownedItemIDs: [record.ownedItemID],
              };
              void runBatch(() =>
                record.container === "inventory"
                  ? mutations.moveToStorage(input)
                  : mutations.moveToInventory(input),
              );
              model.closeItem();
            }}
            onRemove={() => {
              const record = model.openedRow as OwnedItemRow;
              void runBatch(() =>
                mutations.removeItems({
                  saveSessionID: saveSessionID ?? "",
                  characterID: characterID ?? 0,
                  expectedRevision: saveRevision ?? "",
                  ownedItemIDs: [record.ownedItemID],
                }),
              );
              model.closeItem();
            }}
            unavailable={nameUnavailable}
          />
        ) : null}
      </Dialog>
    </Card>
  );
}

type ContainerLabels = {
  title: string;
  grid: string;
  pages: string;
  previous: string;
  next: string;
  loading: string;
  error: string;
  empty: string;
};

function ContainerHeader({
  labels,
  page,
  requestedPage,
  onPage,
}: {
  labels: ContainerLabels;
  page: OwnedItemsPage | undefined;
  requestedPage: number;
  onPage: (page: number) => void;
}) {
  // The backend decides which card it served; a request is never assumed to be
  // the answer, so every control below reads the served values.
  const servedPage = page?.page ?? requestedPage;
  const servedPageSize = page?.pageSize ?? cardSize;
  const total = page?.total ?? 0;
  const cardCount = servedPageSize > 0 ? Math.max(1, Math.ceil(total / servedPageSize)) : 1;

  return (
    <div className={containerHead}>
      <h3 className={containerTitle}>{labels.title}</h3>
      <Badge>
        <Trans>Items: {total}</Trans>
      </Badge>
      <nav className={pagination} aria-label={labels.pages}>
        <Button
          size="sm"
          aria-label={labels.previous}
          disabled={servedPage <= 1}
          onClick={() => onPage(servedPage - 1)}
        >
          <Trans>Previous</Trans>
        </Button>
        <Badge aria-current="page">
          <Trans>
            Card {servedPage} of {cardCount}
          </Trans>
        </Badge>
        <Button
          size="sm"
          aria-label={labels.next}
          disabled={servedPage * servedPageSize >= total}
          onClick={() => onPage(servedPage + 1)}
        >
          <Trans>Next</Trans>
        </Button>
      </nav>
    </div>
  );
}

function ContainerStatus({
  labels,
  query,
}: {
  labels: ContainerLabels;
  query: { isPending: boolean; isError: boolean; isSuccess: boolean; data?: OwnedItemsPage };
}) {
  if (query.isPending) {
    return (
      <p role="status" className={message}>
        {labels.loading}
      </p>
    );
  }
  if (query.isError) {
    return (
      <p role="alert" className={alert}>
        {labels.error}
      </p>
    );
  }
  if (query.isSuccess && (query.data?.records.length ?? 0) === 0) {
    return <p className={message}>{labels.empty}</p>;
  }
  return null;
}

function ContainerGrid({
  labels,
  container,
  records,
  model,
  nameUnavailable,
  returnFocusRef,
}: {
  labels: ContainerLabels;
  container: ItemContainerName;
  records: readonly OwnedItemRow[];
  model: ReturnType<typeof useItemsWorkspace>;
  nameUnavailable: string;
  returnFocusRef: RefObject<HTMLButtonElement | null>;
}) {
  const { t } = useLingui();
  // A card keeps its physical fields even when the served page is partial: an
  // empty field is a neutral placeholder and never a save record.
  const cells = Array.from(
    { length: Math.max(cardSize, records.length) },
    (_, index) => records[index] ?? null,
  );

  return (
    <section className={grid} aria-label={labels.grid}>
      {cells.map((record, index) =>
        record ? (
          <div className={cell} key={record.ownedItemID}>
            <Button
              className={tile}
              // The tile opens the details, so its accessible name is the item
              // itself; the quantity beside it is presentation, not identity.
              aria-label={displayName(record, nameUnavailable)}
              pressed={model.openedRow?.ownedItemID === record.ownedItemID}
              onClick={(event) => {
                returnFocusRef.current = event.currentTarget;
                model.openItem(container, record.ownedItemID);
              }}
            >
              {catalogAssetURL(record.iconPath) ? (
                <img className={tileIcon} src={catalogAssetURL(record.iconPath)} alt="" />
              ) : (
                <span className={tileIconPlaceholder} aria-hidden="true" />
              )}
              <span className={tileName}>{displayName(record, nameUnavailable)}</span>
              <span className={tileQuantity}>{quantityText(record)}</span>
            </Button>
            <div className={cellControls}>
              <Checkbox
                className={cellControl}
                checked={model.isSelected(container, record.ownedItemID)}
                aria-label={t`Select ${displayName(record, nameUnavailable)}`}
                onChange={() => model.toggleSelection(container, record.ownedItemID)}
              />
              <FavoriteButton
                className={cellControl}
                record={record}
                unavailable={nameUnavailable}
              />
            </div>
          </div>
        ) : (
          // biome-ignore lint/suspicious/noArrayIndexKey: an empty field has no identity of its own.
          <div key={`empty-${index}`} className={emptyCell} aria-hidden="true" />
        ),
      )}
    </section>
  );
}

/**
 * The favourite toggle of one resource. A favourite is a presentational
 * preference identified by the canonical `(kind, key)` pair; toggling it reaches
 * no backend endpoint and changes no save.
 */
function FavoriteButton({
  record,
  unavailable,
  className,
}: {
  record: { kind: string; key: string; name: string };
  unavailable: string;
  className?: string;
}) {
  const { t } = useLingui();
  const preferences = useItemPreferences();
  const isFavorite = preferences.isFavorite({ kind: record.kind, key: record.key });
  const name = record.name === "" ? unavailable : record.name;

  return (
    <Button
      size="sm"
      className={className}
      pressed={isFavorite}
      aria-label={isFavorite ? t`Remove ${name} from favorites` : t`Add ${name} to favorites`}
      onClick={() => preferences.toggleFavorite({ kind: record.kind, key: record.key })}
    >
      <span aria-hidden="true">{isFavorite ? "★" : "☆"}</span>
    </Button>
  );
}

/** `Owned / Max`, editable exactly when the backend accepts a quantity change. */
function QuantityCell({
  record,
  editable,
  label,
  onCommit,
}: {
  record: OwnedItemRow;
  editable: boolean;
  label: string;
  onCommit: (value: string) => void;
}) {
  const [draft, setDraft] = useState<string | null>(null);
  if (!editable) return <span>{quantityText(record)}</span>;

  const commit = () => {
    if (draft !== null && draft !== String(record.quantity)) onCommit(draft);
    setDraft(null);
  };

  return (
    <span>
      <Input
        className={quantityField}
        type="number"
        min={1}
        max={record.maxQuantityKnown ? record.maxQuantity : undefined}
        step={1}
        aria-label={label}
        value={draft ?? String(record.quantity)}
        onChange={(event) => setDraft(event.currentTarget.value)}
        onBlur={commit}
        onKeyDown={(event) => {
          if (event.key === "Enter") commit();
        }}
      />
      {record.maxQuantityKnown ? <span>{` / ${record.maxQuantity}`}</span> : null}
    </span>
  );
}

function ItemDetailContent({
  container,
  record,
  detail,
  canMutate,
  sortIsDefault,
  positionDraft,
  onPositionDraft,
  onQuantity,
  onReorder,
  onMove,
  onRemove,
  unavailable,
}: {
  container: string;
  record: OwnedItemRow;
  detail: ReturnType<typeof useCatalogResource>;
  canMutate: boolean;
  sortIsDefault: boolean;
  positionDraft: string;
  onPositionDraft: (value: string) => void;
  onQuantity: (value: string) => void;
  onReorder: (targetPosition: number) => void;
  onMove: () => void;
  onRemove: () => void;
  unavailable: string;
}) {
  const { t } = useLingui();
  const item = detail.data?.item ?? null;
  const description = knownText(item?.presentation.description);
  const icon = catalogAssetURL(record.iconPath);
  // The manual order is Inventory-only and exists only in the container's own
  // order: any other sort would place a record at a position the user cannot
  // see, so the controls are not offered at all.
  const canReorder =
    canMutate && sortIsDefault && record.actions.reorder && record.orderPositionKnown;

  return (
    <>
      <div className={detailHeader}>
        {icon ? (
          <img className={detailIcon} src={icon} alt="" />
        ) : (
          <span className={detailIconPlaceholder} aria-hidden="true" />
        )}
        <h3 className={detailHeading}>{displayName(record, unavailable)}</h3>
      </div>

      <div className={flagRow}>
        {record.banRisk ? (
          <Badge tone="accent">
            <Trans>Ban risk</Trans>
          </Badge>
        ) : null}
        {record.cutContent ? (
          <Badge>
            <Trans>Cut content</Trans>
          </Badge>
        ) : null}
        {record.dlc ? (
          <Badge>
            <Trans>DLC</Trans>
          </Badge>
        ) : null}
        {record.preOrder ? (
          <Badge>
            <Trans>Pre-order</Trans>
          </Badge>
        ) : null}
      </div>

      <dl className={facts}>
        <Fact label={t`Container`} value={container} />
        <Fact label={t`Quantity`} value={quantityText(record)} />
        <Fact label={t`Position`} value={String(record.physicalIndex)} />
        <OptionalFact label={t`Category`} value={record.category || null} />
        <OptionalFact label={t`Subcategory`} value={record.subcategory || null} />
        <OptionalFact label={t`Item ID`} value={formatItemID(record.gameID)} />
        <OptionalFact
          label={t`Order position`}
          value={record.orderPositionKnown ? String(record.orderPosition + 1) : null}
        />
      </dl>

      {detail.isPending ? (
        <p role="status" className={message}>
          <Trans>Loading item details…</Trans>
        </p>
      ) : null}
      {detail.isError ? (
        <p role="alert" className={alert}>
          <Trans>Unable to load item details.</Trans>
        </p>
      ) : null}

      {description ? <p className={detailText}>{description}</p> : null}

      {item ? (
        <dl className={facts}>
          <OptionalFact
            label={t`Maximum in Inventory`}
            value={knownNumber(item.storage.maxInventory)}
          />
          <OptionalFact
            label={t`Maximum in Storage`}
            value={knownNumber(item.storage.maxStorage)}
          />
          <OptionalFact
            label={t`Stacks`}
            value={
              item.capabilities.stack.known
                ? item.capabilities.stack.enabled
                  ? t`Yes`
                  : t`No`
                : null
            }
          />
          <OptionalFact
            label={t`Upgradeable`}
            value={
              item.capabilities.upgrade.known
                ? item.capabilities.upgrade.enabled
                  ? t`Yes`
                  : t`No`
                : null
            }
          />
        </dl>
      ) : null}

      {canMutate && record.actions.setQuantity ? (
        <p>
          <label htmlFor="item-detail-quantity">
            <Trans>Owned quantity</Trans>{" "}
          </label>
          <QuantityCell record={record} editable label={t`Owned quantity`} onCommit={onQuantity} />
        </p>
      ) : null}

      {canMutate && (record.actions.moveToStorage || record.actions.moveToInventory) ? (
        <Button size="sm" onClick={onMove}>
          {record.actions.moveToStorage ? (
            <Trans>Move to Storage</Trans>
          ) : (
            <Trans>Move to Inventory</Trans>
          )}
        </Button>
      ) : null}

      {canMutate && record.actions.remove ? (
        <Button size="sm" onClick={onRemove}>
          <Trans>Remove</Trans>
        </Button>
      ) : null}

      {canReorder ? (
        <fieldset className={flagRow}>
          <legend className={visuallyHidden}>
            <Trans>Manual sort order</Trans>
          </legend>
          <Button
            size="sm"
            disabled={record.orderPosition === 0}
            onClick={() => onReorder(record.orderPosition - 1)}
          >
            <Trans>Move up</Trans>
          </Button>
          <Button size="sm" onClick={() => onReorder(record.orderPosition + 1)}>
            <Trans>Move down</Trans>
          </Button>
          <label className={visuallyHidden} htmlFor="item-detail-position">
            <Trans>Target position</Trans>
          </label>
          <Input
            id="item-detail-position"
            className={quantityField}
            type="number"
            min={1}
            step={1}
            value={positionDraft}
            aria-label={t`Target position`}
            onChange={(event) => onPositionDraft(event.currentTarget.value)}
          />
          <Button
            size="sm"
            onClick={() => {
              const position = Number(positionDraft);
              if (Number.isInteger(position) && position >= 1) onReorder(position - 1);
            }}
          >
            <Trans>Move to position</Trans>
          </Button>
        </fieldset>
      ) : null}
    </>
  );
}

function Fact({ label, value }: { label: string; value: string }) {
  return (
    <div className={fact}>
      <dt className={factLabel}>{label}</dt>
      <dd className={factValue}>{value}</dd>
    </div>
  );
}

function OptionalFact({ label, value }: { label: string; value: string | null }) {
  return value === null ? null : <Fact label={label} value={value} />;
}

function displayName(record: { name: string }, unavailable: string): string {
  return record.name === "" ? unavailable : record.name;
}

/** `Owned / Max`, or the owned amount alone when the catalog states no limit. */
function quantityText(record: OwnedItemRow): string {
  return record.maxQuantityKnown
    ? `${record.quantity} / ${record.maxQuantity}`
    : String(record.quantity);
}

/** The save-side identifier, rendered the way the backend documents it. */
function formatItemID(gameID: number): string {
  return `0x${gameID.toString(16).toUpperCase().padStart(8, "0")}`;
}

function mutationErrorText(error: AppError, fallback: string): string {
  return error.message === "" ? fallback : error.message;
}

/** A fact the backend did not resolve is absent, never a synthesised value. */
function knownText(value: CatalogFact<string> | undefined): string | null {
  return value?.known && value.value !== "" ? value.value : null;
}

function knownNumber(value: CatalogFact<number> | undefined): string | null {
  return value?.known ? String(value.value) : null;
}
