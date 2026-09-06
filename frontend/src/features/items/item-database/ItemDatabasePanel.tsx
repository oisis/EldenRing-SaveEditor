import { Trans, useLingui } from "@lingui/react/macro";
import { createColumnHelper, tableFeatures, useTable } from "@tanstack/react-table";
import { useRef, useState } from "react";
import { catalogAssetURL } from "../../../application/catalog/catalogAssetURL";
import type {
  CatalogFact,
  CatalogItemDatabaseEntry,
} from "../../../application/catalog/catalogPort";
import type {
  AddItemsRequestEntry,
  ItemMutationReceipt,
} from "../../../application/items/itemsPort";
import { useItemMutations } from "../../../application/items/useItemMutations";
import { useItemPreferences } from "../../../application/preferences/itemPreferences";
import { Badge } from "../../../ui/components/Badge/Badge";
import { Button } from "../../../ui/components/Button/Button";
import { workspaceStack } from "../../../ui/patterns/workspace.css";
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
  addRow,
  bulkBar,
  cell,
  cellControl,
  cellControls,
  family as familyControl,
  filters as filterRow,
  flagRow,
  grid,
  pagination,
  quantityField,
  search as searchControl,
  tile,
  tileIcon,
  tileIconPlaceholder,
  tileMeta,
  tileName,
  variant,
  variantList,
} from "./ItemDatabasePanel.css";
import { useItemCatalog } from "./useItemCatalog";

/**
 * The Item Database.
 *
 * It works as a read-only catalog with no save loaded, and gains exactly one
 * save mutation when a session and a character are active: the shared Add
 * dialog, which commits one atomic batch through the same receipt path every
 * other Items mutation uses. Which resources exist at all is the backend's
 * decision under the global Safety Profile; this screen never filters by risk.
 */
export type ItemDatabasePanelProps = {
  saveSessionID?: string | undefined;
  saveRevision?: string | undefined;
  characterID?: number | undefined;
  /**
   * The session controller's post-mutation step. It is absent while no session
   * controller is mounted, which is also when no save mutation is offered.
   */
  applyMutationReceipt?: ((receipt: ItemMutationReceipt) => Promise<unknown>) | undefined;
  sessionBusy?: boolean;
};

type ItemDatabaseView = "grid" | "table";

const gridPageSize = 20;
const tablePageSize = 50;
const tableFeatureSet = tableFeatures({});
const columnHelper = createColumnHelper<typeof tableFeatureSet, CatalogItemDatabaseEntry>();

const families = [
  "",
  "weapon",
  "armor",
  "talisman",
  "ash_of_war",
  "spell",
  "spirit_ash",
  "goods",
  "gesture",
] as const;

/** The sort orders the backend accepts; the empty value is the catalog order. */
const sortOrders = ["", "name", "category", "game_id"] as const;

/** The quantities of one requested resource, as typed into the Add dialog. */
type AddDraft = {
  inventory: string;
  storage: string;
};

export function ItemDatabasePanel({
  saveSessionID,
  saveRevision,
  characterID,
  applyMutationReceipt,
  sessionBusy = false,
}: ItemDatabasePanelProps) {
  const { t } = useLingui();
  const returnFocusRef = useRef<HTMLButtonElement | null>(null);
  const addReturnFocusRef = useRef<HTMLButtonElement | null>(null);
  const preferences = useItemPreferences();
  // Without a session controller there is no receipt path, so the mutation
  // runner is given one that can never be reached: no Add control is rendered.
  const mutations = useItemMutations(applyMutationReceipt ?? (() => Promise.resolve()));

  const [view, setView] = useState<ItemDatabaseView>("grid");
  const [search, setSearch] = useState("");
  const [family, setFamily] = useState("");
  const [category, setCategory] = useState("");
  const [sort, setSort] = useState<string>("");
  const [favoritesOnly, setFavoritesOnly] = useState(false);
  const [requestedPage, setRequestedPage] = useState(1);
  const [addOpen, setAddOpen] = useState(false);
  const [addDrafts, setAddDrafts] = useState<Record<string, AddDraft>>({});
  const [confirmBanRisk, setConfirmBanRisk] = useState(false);

  const pageSize = view === "grid" ? gridPageSize : tablePageSize;
  const model = useItemCatalog({
    family,
    category,
    search,
    favoritesOnly,
    favorites: preferences.favorites,
    sort,
    page: requestedPage,
    pageSize,
  });

  const canMutate =
    applyMutationReceipt !== undefined &&
    saveSessionID !== undefined &&
    saveRevision !== undefined &&
    characterID !== undefined &&
    !sessionBusy &&
    !mutations.isBusy;

  const nameUnavailable = t`Name unavailable`;
  const unavailable = t`Unavailable`;
  const familyLabels: Record<(typeof families)[number], string> = {
    "": t`All families`,
    weapon: t`Weapons`,
    armor: t`Armor`,
    talisman: t`Talismans`,
    ash_of_war: t`Ashes of War`,
    spell: t`Spells`,
    spirit_ash: t`Spirit Ashes`,
    goods: t`Goods`,
    gesture: t`Gestures`,
  };
  const sortLabels: Record<(typeof sortOrders)[number], string> = {
    "": t`Sort: catalog order`,
    name: t`Sort: name`,
    category: t`Sort: category`,
    game_id: t`Sort: item ID`,
  };

  const changeFilter = (apply: () => void) => {
    apply();
    setRequestedPage(1);
  };

  const openAdd = (rows: readonly CatalogItemDatabaseEntry[], element: HTMLButtonElement) => {
    addReturnFocusRef.current = element;
    setAddDrafts(
      Object.fromEntries(rows.map((row) => [rowToken(row), { inventory: "1", storage: "0" }])),
    );
    setConfirmBanRisk(false);
    setAddOpen(true);
  };

  const addRows = model.rows.filter((row) => rowToken(row) in addDrafts);
  const addNeedsBanRiskConfirmation = addRows.some((row) => row.banRisk);
  const addEntries: AddItemsRequestEntry[] = addRows.flatMap((row) => {
    const draft = addDrafts[rowToken(row)];
    const inventoryQuantity = toQuantity(draft?.inventory);
    const storageQuantity = toQuantity(draft?.storage);
    return inventoryQuantity + storageQuantity === 0
      ? []
      : [{ kind: row.kind, key: row.key, inventoryQuantity, storageQuantity }];
  });
  const canSubmitAdd =
    canMutate && addEntries.length > 0 && (!addNeedsBanRiskConfirmation || confirmBanRisk);

  const submitAdd = async () => {
    if (!canSubmitAdd || saveSessionID === undefined || saveRevision === undefined) return;
    if (characterID === undefined) return;
    const applied = await mutations.addItems({
      saveSessionID,
      characterID,
      expectedRevision: saveRevision,
      items: addEntries,
      confirmBanRisk,
    });
    if (applied) {
      setAddOpen(false);
      setAddDrafts({});
      setConfirmBanRisk(false);
      model.clearSelection();
    }
  };

  const columns = columnHelper.columns([
    columnHelper.display({
      id: "select",
      header: t`Select`,
      cell: ({ row }) => (
        <Checkbox
          checked={model.isSelected(row.original)}
          aria-label={t`Select ${displayName(row.original, nameUnavailable)}`}
          onChange={() => model.toggleSelection(row.original)}
        />
      ),
    }),
    columnHelper.accessor("name", {
      header: t`Name`,
      cell: (value) => displayName({ name: value.getValue() }, nameUnavailable),
    }),
    ...(preferences.showItemID
      ? [
          columnHelper.display({
            id: "itemID",
            header: t`Item ID`,
            cell: ({ row }) =>
              row.original.gameIDKnown ? formatItemID(row.original.gameID) : unavailable,
          }),
        ]
      : []),
    columnHelper.accessor("family", {
      header: t`Family`,
      cell: (value) => familyLabel(value.getValue(), familyLabels, unavailable),
    }),
    columnHelper.accessor("category", {
      header: t`Category`,
      cell: (value) => value.getValue() || unavailable,
    }),
    columnHelper.display({
      id: "favorite",
      header: t`Favorite`,
      cell: ({ row }) => <FavoriteButton row={row.original} unavailable={nameUnavailable} />,
    }),
    columnHelper.display({
      id: "details",
      header: t`Details`,
      cell: ({ row }) => (
        <Button
          size="sm"
          onClick={(event) => {
            returnFocusRef.current = event.currentTarget;
            model.openItem(row.original);
          }}
        >
          <Trans>View</Trans>
        </Button>
      ),
    }),
  ]);
  const tableModel = useTable(
    { features: tableFeatureSet, columns, data: model.rows },
    (state) => state,
  );

  const servedPage = model.resources.data?.page ?? requestedPage;
  const servedPageSize = model.resources.data?.pageSize ?? pageSize;
  const total = model.resources.data?.total ?? 0;
  const categories = model.resources.data?.categories ?? [];

  return (
    <section aria-label={t`Item Database`} className={`${panel} ${workspaceStack}`}>
      <div className={toolbar}>
        <div className={filterRow}>
          <label className={visuallyHidden} htmlFor="item-database-search">
            <Trans>Search items</Trans>
          </label>
          <Input
            id="item-database-search"
            className={searchControl}
            type="search"
            value={search}
            placeholder={t`Search items`}
            onChange={(event) => {
              const value = event.currentTarget.value;
              changeFilter(() => setSearch(value));
            }}
          />

          <label className={visuallyHidden} htmlFor="item-database-family">
            <Trans>Item family</Trans>
          </label>
          <Select
            id="item-database-family"
            className={familyControl}
            value={family}
            onChange={(event) => {
              const value = event.currentTarget.value;
              changeFilter(() => setFamily(value));
            }}
          >
            {families.map((value) => (
              <option key={value || "all"} value={value}>
                {familyLabels[value]}
              </option>
            ))}
          </Select>

          <label className={visuallyHidden} htmlFor="item-database-category">
            <Trans>Category</Trans>
          </label>
          <Select
            id="item-database-category"
            className={familyControl}
            value={category}
            onChange={(event) => {
              const value = event.currentTarget.value;
              changeFilter(() => setCategory(value));
            }}
          >
            <option value="">{t`All categories`}</option>
            {categories.map((facet) => (
              <option key={facet.category} value={facet.category}>
                {`${facet.category} (${facet.count})`}
              </option>
            ))}
          </Select>

          <label className={visuallyHidden} htmlFor="item-database-sort">
            <Trans>Sort order</Trans>
          </label>
          <Select
            id="item-database-sort"
            className={familyControl}
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
        <Badge>
          <Trans>Results: {total}</Trans>
        </Badge>
        <fieldset className={viewSwitch}>
          <legend className={visuallyHidden}>
            <Trans>View</Trans>
          </legend>
          <Button
            size="sm"
            pressed={view === "grid"}
            onClick={() => changeFilter(() => setView("grid"))}
          >
            <Trans>Grid</Trans>
          </Button>
          <Button
            size="sm"
            pressed={view === "table"}
            onClick={() => changeFilter(() => setView("table"))}
          >
            <Trans>Table</Trans>
          </Button>
        </fieldset>
      </div>

      {mutations.error ? (
        <p role="alert" className={alert}>
          {mutations.error.message === ""
            ? t`The change was not applied.`
            : mutations.error.message}
        </p>
      ) : null}

      {model.selected.length > 0 ? (
        <fieldset className={bulkBar}>
          <legend className={visuallyHidden}>
            <Trans>Selected items</Trans>
          </legend>
          <Badge>
            <Trans>Selected: {model.selected.length}</Trans>
          </Badge>
          {canMutate ? (
            <Button size="sm" onClick={(event) => openAdd(model.selectedRows, event.currentTarget)}>
              <Trans>Add</Trans>
            </Button>
          ) : null}
          <Button size="sm" onClick={() => model.clearSelection()}>
            <Trans>Clear selection</Trans>
          </Button>
        </fieldset>
      ) : null}

      {model.resources.isPending ? (
        <p role="status" className={message}>
          <Trans>Loading items…</Trans>
        </p>
      ) : null}
      {model.resources.isError ? (
        <p role="alert" className={alert}>
          <Trans>Unable to load the Item Database.</Trans>
        </p>
      ) : null}
      {model.resources.isSuccess && model.rows.length === 0 ? (
        <p className={message}>
          <Trans>No items match the current search.</Trans>
        </p>
      ) : null}

      {model.resources.isSuccess && model.rows.length > 0 && view === "grid" ? (
        <section className={grid} aria-label={t`Item results`}>
          {model.rows.map((row) => (
            <div className={cell} key={rowToken(row)}>
              <Button
                className={tile}
                // The tile opens the details, so its accessible name is the item
                // itself; the family beside it is presentation, not identity.
                aria-label={displayName(row, nameUnavailable)}
                pressed={model.opened?.kind === row.kind && model.opened.key === row.key}
                onClick={(event) => {
                  returnFocusRef.current = event.currentTarget;
                  model.openItem(row);
                }}
              >
                {catalogAssetURL(row.iconPath) ? (
                  <img className={tileIcon} src={catalogAssetURL(row.iconPath)} alt="" />
                ) : (
                  <span className={tileIconPlaceholder} aria-hidden="true" />
                )}
                <span className={tileName}>{displayName(row, nameUnavailable)}</span>
                <span className={tileMeta}>
                  {familyLabel(row.family, familyLabels, unavailable)}
                </span>
              </Button>
              <div className={cellControls}>
                <Checkbox
                  className={cellControl}
                  checked={model.isSelected(row)}
                  aria-label={t`Select ${displayName(row, nameUnavailable)}`}
                  onChange={() => model.toggleSelection(row)}
                />
                <FavoriteButton className={cellControl} row={row} unavailable={nameUnavailable} />
              </div>
            </div>
          ))}
        </section>
      ) : null}

      {model.resources.isSuccess && model.rows.length > 0 && view === "table" ? (
        <TableFrame className={tableFrame}>
          <Table aria-label={t`Item results`}>
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
      ) : null}

      {model.resources.isSuccess && total > servedPageSize ? (
        <nav className={pagination} aria-label={t`Item Database pages`}>
          <Button
            size="sm"
            disabled={servedPage <= 1}
            onClick={() => setRequestedPage(servedPage - 1)}
          >
            <Trans>Previous</Trans>
          </Button>
          <Badge>
            <Trans>Page {servedPage}</Trans>
          </Badge>
          <Button
            size="sm"
            disabled={servedPage * servedPageSize >= total}
            onClick={() => setRequestedPage(servedPage + 1)}
          >
            <Trans>Next</Trans>
          </Button>
        </nav>
      ) : null}

      <Dialog
        open={model.opened !== null}
        onOpenChange={(open) => {
          if (!open) model.closeItem();
        }}
        title={<Trans>Item details</Trans>}
        description={<Trans>Information from GameCatalog.</Trans>}
        closeLabel={<Trans>Close</Trans>}
        returnFocusRef={returnFocusRef}
      >
        <ItemDetailContent
          model={model}
          familyLabels={familyLabels}
          canMutate={canMutate}
          onAdd={(row, element) => {
            model.closeItem();
            openAdd([row], element);
          }}
        />
      </Dialog>

      <Dialog
        open={addOpen}
        onOpenChange={(open) => {
          if (!open) setAddOpen(false);
        }}
        title={<Trans>Add items</Trans>}
        description={<Trans>State how much of each item goes to Inventory and to Storage.</Trans>}
        closeLabel={<Trans>Cancel</Trans>}
        returnFocusRef={addReturnFocusRef}
      >
        {addRows.map((row) => (
          <div className={addRow} key={rowToken(row)}>
            <span>{displayName(row, nameUnavailable)}</span>
            <label htmlFor={`add-inventory-${rowToken(row)}`}>
              <Trans>Inventory</Trans>
            </label>
            <Input
              id={`add-inventory-${rowToken(row)}`}
              className={quantityField}
              type="number"
              min={0}
              step={1}
              value={addDrafts[rowToken(row)]?.inventory ?? "0"}
              onChange={(event) =>
                setAddDrafts((current) => ({
                  ...current,
                  [rowToken(row)]: {
                    inventory: event.target.value,
                    storage: current[rowToken(row)]?.storage ?? "0",
                  },
                }))
              }
            />
            <label htmlFor={`add-storage-${rowToken(row)}`}>
              <Trans>Storage</Trans>
            </label>
            <Input
              id={`add-storage-${rowToken(row)}`}
              className={quantityField}
              type="number"
              min={0}
              step={1}
              value={addDrafts[rowToken(row)]?.storage ?? "0"}
              onChange={(event) =>
                setAddDrafts((current) => ({
                  ...current,
                  [rowToken(row)]: {
                    inventory: current[rowToken(row)]?.inventory ?? "0",
                    storage: event.target.value,
                  },
                }))
              }
            />
            {row.banRisk ? (
              <Badge tone="accent">
                <Trans>Ban risk</Trans>
              </Badge>
            ) : null}
          </div>
        ))}

        {addNeedsBanRiskConfirmation ? (
          <p>
            <label htmlFor="add-confirm-ban-risk">
              <Checkbox
                id="add-confirm-ban-risk"
                checked={confirmBanRisk}
                onChange={(event) => setConfirmBanRisk(event.target.checked)}
              />{" "}
              <Trans>
                I understand that at least one of these items carries a ban risk and I confirm
                adding it.
              </Trans>
            </label>
          </p>
        ) : null}

        <Button disabled={!canSubmitAdd} onClick={() => void submitAdd()}>
          <Trans>Add to save</Trans>
        </Button>
      </Dialog>
    </section>
  );
}

/**
 * The favourite toggle of one resource. A favourite is a presentational
 * preference identified by the canonical `(kind, key)` pair; toggling it reaches
 * no backend endpoint and changes no save.
 */
function FavoriteButton({
  row,
  unavailable,
  className,
}: {
  row: CatalogItemDatabaseEntry;
  unavailable: string;
  className?: string;
}) {
  const { t } = useLingui();
  const preferences = useItemPreferences();
  const isFavorite = preferences.isFavorite({ kind: row.kind, key: row.key });
  const name = displayName(row, unavailable);

  return (
    <Button
      size="sm"
      className={className}
      pressed={isFavorite}
      aria-label={isFavorite ? t`Remove ${name} from favorites` : t`Add ${name} to favorites`}
      onClick={() => preferences.toggleFavorite({ kind: row.kind, key: row.key })}
    >
      <span aria-hidden="true">{isFavorite ? "★" : "☆"}</span>
    </Button>
  );
}

function ItemDetailContent({
  model,
  familyLabels,
  canMutate,
  onAdd,
}: {
  model: ReturnType<typeof useItemCatalog>;
  familyLabels: Record<(typeof families)[number], string>;
  canMutate: boolean;
  onAdd: (row: CatalogItemDatabaseEntry, element: HTMLButtonElement) => void;
}) {
  const { t } = useLingui();
  const opened = model.opened;
  const row = opened
    ? (model.rows.find((entry) => entry.kind === opened.kind && entry.key === opened.key) ?? null)
    : null;

  if (model.detail.isPending) {
    return (
      <p role="status" className={message}>
        <Trans>Loading item details…</Trans>
      </p>
    );
  }
  if (model.detail.isError) {
    return (
      <p role="alert" className={alert}>
        <Trans>Unable to load item details.</Trans>
      </p>
    );
  }

  const item = model.detail.data?.item;
  if (!item) {
    return (
      <p className={message}>
        <Trans>The selected resource has no item details.</Trans>
      </p>
    );
  }

  const unavailable = t`Unavailable`;
  const name = factText(item.presentation.name, t`Name unavailable`);
  const caption = factText(item.presentation.caption, "");
  const description = factText(item.presentation.description, "");
  const location = factText(item.presentation.location, unavailable);
  const familyValue = item.family.known ? item.family.value : "";
  const iconPath = item.presentation.iconPath.known ? item.presentation.iconPath.value : "";

  return (
    <>
      <h3 className={detailHeading}>{name}</h3>
      {catalogAssetURL(iconPath) ? (
        <img className={tileIcon} src={catalogAssetURL(iconPath)} alt="" />
      ) : null}

      <div className={flagRow}>
        {row?.banRisk ? (
          <Badge tone="accent">
            <Trans>Ban risk</Trans>
          </Badge>
        ) : null}
        {row?.cutContent ? (
          <Badge>
            <Trans>Cut content</Trans>
          </Badge>
        ) : null}
        {row?.dlc ? (
          <Badge>
            <Trans>DLC</Trans>
          </Badge>
        ) : null}
        {row?.preOrder ? (
          <Badge>
            <Trans>Pre-order</Trans>
          </Badge>
        ) : null}
      </div>

      {caption ? <p className={detailText}>{caption}</p> : null}
      {description ? <p>{description}</p> : null}

      <dl className={facts}>
        <div className={fact}>
          <dt className={factLabel}>
            <Trans>Family</Trans>
          </dt>
          <dd className={factValue}>{familyLabel(familyValue, familyLabels, unavailable)}</dd>
        </div>
        <Fact label={t`Category`} value={item.category} unavailable={unavailable} />
        <Fact label={t`Subcategory`} value={item.subcategory} unavailable={unavailable} />
        <div className={fact}>
          <dt className={factLabel}>
            <Trans>Location</Trans>
          </dt>
          <dd className={factValue}>{location}</dd>
        </div>
        <div className={fact}>
          <dt className={factLabel}>
            <Trans>Item ID</Trans>
          </dt>
          <dd className={factValue}>
            {item.gameID.known ? formatItemID(item.gameID.value) : unavailable}
          </dd>
        </div>
        <div className={fact}>
          <dt className={factLabel}>
            <Trans>Maximum in Inventory</Trans>
          </dt>
          <dd className={factValue}>
            {item.storage.maxInventory.known
              ? String(item.storage.maxInventory.value)
              : unavailable}
          </dd>
        </div>
        <div className={fact}>
          <dt className={factLabel}>
            <Trans>Maximum in Storage</Trans>
          </dt>
          <dd className={factValue}>
            {item.storage.maxStorage.known ? String(item.storage.maxStorage.value) : unavailable}
          </dd>
        </div>
        <div className={fact}>
          <dt className={factLabel}>
            <Trans>Stacks</Trans>
          </dt>
          <dd className={factValue}>
            {item.capabilities.stack.known
              ? item.capabilities.stack.enabled
                ? t`Yes`
                : t`No`
              : unavailable}
          </dd>
        </div>
        <div className={fact}>
          <dt className={factLabel}>
            <Trans>Upgradeable</Trans>
          </dt>
          <dd className={factValue}>
            {item.capabilities.upgrade.known
              ? item.capabilities.upgrade.enabled
                ? t`Yes`
                : t`No`
              : unavailable}
          </dd>
        </div>
      </dl>

      {canMutate && row ? (
        <Button size="sm" onClick={(event) => onAdd(row, event.currentTarget)}>
          <Trans>Add</Trans>
        </Button>
      ) : null}

      <h3 className={detailHeading}>
        <Trans>Variants</Trans>
      </h3>
      {model.variants.isPending ? (
        <p role="status" className={message}>
          <Trans>Loading variants…</Trans>
        </p>
      ) : null}
      {model.variants.isError ? (
        <p role="alert" className={alert}>
          <Trans>Unable to load item variants.</Trans>
        </p>
      ) : null}
      {model.variants.isSuccess && model.variants.data.variants.length === 0 ? (
        <p className={message}>
          <Trans>No variants are available.</Trans>
        </p>
      ) : null}
      {model.variants.isSuccess && model.variants.data.variants.length > 0 ? (
        <ul className={variantList}>
          {model.variants.data.variants.map((entry) => (
            <li
              className={variant}
              key={`${entry.gameID.value}:${entry.kind.value}:${entry.affinity.value}:${entry.upgradeLevel.value}:${entry.sourceRowID.value}`}
            >
              {variantText(entry.affinity, entry.upgradeLevel, t`Variant details unavailable`)}
            </li>
          ))}
        </ul>
      ) : null}
    </>
  );
}

function Fact({
  label,
  value,
  unavailable,
}: {
  label: string;
  value: CatalogFact<string>;
  unavailable: string;
}) {
  return (
    <div className={fact}>
      <dt className={factLabel}>{label}</dt>
      <dd className={factValue}>{factText(value, unavailable)}</dd>
    </div>
  );
}

/** The cache and draft identity of one row: its exact backend pair. */
function rowToken(row: { kind: string; key: string }): string {
  return `${row.kind}/${row.key}`;
}

function displayName(row: { name: string }, unavailable: string) {
  return row.name === "" ? unavailable : row.name;
}

/** A quantity draft that is not a whole, non-negative number requests nothing. */
function toQuantity(value: string | undefined): number {
  const parsed = Number(value ?? "0");
  return Number.isInteger(parsed) && parsed > 0 ? parsed : 0;
}

/** The save-side identifier, rendered the way the backend documents it. */
function formatItemID(gameID: number): string {
  return `0x${gameID.toString(16).toUpperCase().padStart(8, "0")}`;
}

function factText(value: CatalogFact<string>, unavailable: string) {
  return value.known && value.value !== "" ? value.value : unavailable;
}

function familyLabel(
  value: string,
  labels: Record<(typeof families)[number], string>,
  unavailable: string,
) {
  if (value === "") return unavailable;
  return value in labels ? labels[value as (typeof families)[number]] : value;
}

function variantText(
  affinity: CatalogFact<string>,
  upgradeLevel: CatalogFact<number>,
  unavailable: string,
) {
  const parts: string[] = [];
  if (affinity.known && affinity.value !== "") parts.push(affinity.value);
  if (upgradeLevel.known) parts.push(`+${upgradeLevel.value}`);
  return parts.length > 0 ? parts.join(" · ") : unavailable;
}
