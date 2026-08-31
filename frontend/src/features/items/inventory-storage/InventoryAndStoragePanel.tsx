import { Trans, useLingui } from "@lingui/react/macro";
import { createColumnHelper, tableFeatures, useTable } from "@tanstack/react-table";
import { type RefObject, useMemo, useRef, useState } from "react";
import { catalogAssetURL } from "../../../application/catalog/catalogAssetURL";
import type {
  CatalogFact,
  CatalogResourcePresentationSummary,
} from "../../../application/catalog/catalogPort";
import { useCatalogResource } from "../../../application/catalog/useCatalogResource";
import type { ItemPage, ItemRecord } from "../../../application/items/itemsPort";
import { Badge } from "../../../ui/components/Badge/Badge";
import { Button } from "../../../ui/components/Button/Button";
import { Card } from "../../../ui/components/Card/Card";
import { Dialog } from "../../../ui/components/Dialog/Dialog";
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
  cardNavigation,
  container as containerColumn,
  containerHead,
  containerTitle,
  detailHeader,
  detailIcon,
  detailIconPlaceholder,
  emptyCell,
  grid,
  pagination,
  tile,
  tileIcon,
  tileIconPlaceholder,
  tileName,
  tileQuantity,
  workspace,
} from "./InventoryAndStoragePanel.css";
import { type ItemContainer, useInventoryAndStorage } from "./useInventoryAndStorage";

/**
 * The first read-only Inventory and Storage workspace. Both containers live in
 * one panel and one selection: the controller owns the two container pages, the
 * presentation batch, the revision comparison and the selected owned item, and
 * this component owns only the view mode and the two requested card numbers.
 *
 * Nothing is derived locally that a backend contract does not state. Names and
 * icons come from the catalog presentation batch, paging follows the values the
 * backend served, and a record without a resolved name keeps a neutral label
 * rather than a value synthesised from its identifiers.
 */
export type InventoryAndStoragePanelProps = {
  saveSessionID: string | undefined;
  saveRevision: string | undefined;
  characterID: number | undefined;
  containerSection: string;
};

type WorkspaceView = "grid" | "table";

/** The two requested card numbers, scoped to the workspace that requested them. */
type WorkspacePages = {
  saveSessionID: string | undefined;
  characterID: number | undefined;
  containerSection: string;
  inventory: number;
  storage: number;
};

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

type WorkspaceRow = {
  container: ItemContainer;
  record: ItemRecord;
};

/** One card is 5 × 6 physical fields, which is also the requested page size. */
const cardSize = 30;

const tableFeatureSet = tableFeatures({});
const columnHelper = createColumnHelper<typeof tableFeatureSet, WorkspaceRow>();

export function InventoryAndStoragePanel({
  saveSessionID,
  saveRevision,
  characterID,
  containerSection,
}: InventoryAndStoragePanelProps) {
  const { t } = useLingui();
  const returnFocusRef = useRef<HTMLButtonElement | null>(null);
  const [view, setView] = useState<WorkspaceView>("grid");
  const [pages, setPages] = useState<WorkspacePages>({
    saveSessionID,
    characterID,
    containerSection,
    inventory: 1,
    storage: 1,
  });

  // The requested cards belong to one workspace identity. A session, character
  // or section change starts both containers at card 1 during render, so the
  // new identity is never asked for a card number the previous one reached.
  const isCurrentWorkspace =
    pages.saveSessionID === saveSessionID &&
    pages.characterID === characterID &&
    pages.containerSection === containerSection;
  if (!isCurrentWorkspace) {
    setPages({ saveSessionID, characterID, containerSection, inventory: 1, storage: 1 });
  }
  const inventoryPage = isCurrentWorkspace ? pages.inventory : 1;
  const storagePage = isCurrentWorkspace ? pages.storage : 1;
  const setCardNumber = (container: ItemContainer, card: number) =>
    setPages({
      saveSessionID,
      characterID,
      containerSection,
      inventory: container === "inventory" ? card : inventoryPage,
      storage: container === "storage" ? card : storagePage,
    });

  const model = useInventoryAndStorage({
    saveSessionID,
    saveRevision,
    characterID,
    containerSection,
    inventory: { page: inventoryPage, pageSize: cardSize },
    storage: { page: storagePage, pageSize: cardSize },
  });

  // The detail query is the selection's own lifecycle: it is skipped entirely
  // while nothing is selected and it fails independently of both lists.
  const detail = useCatalogResource(model.selected?.record.kind, model.selected?.record.key);

  const hasSlot = saveSessionID !== undefined && characterID !== undefined;
  const nameUnavailable = t`Name unavailable`;

  const labels: Record<ItemContainer, ContainerLabels> = {
    inventory: {
      title: t`Inventory`,
      grid: t`Inventory items`,
      pages: t`Inventory cards`,
      previous: t`Previous inventory card`,
      next: t`Next inventory card`,
      loading: t`Loading the Inventory…`,
      error: t`Unable to load the Inventory.`,
      empty: t`This Inventory card is empty.`,
    },
    storage: {
      title: t`Storage`,
      grid: t`Storage items`,
      pages: t`Storage cards`,
      previous: t`Previous storage card`,
      next: t`Next storage card`,
      loading: t`Loading the Storage Box…`,
      error: t`Unable to load the Storage Box.`,
      empty: t`This Storage card is empty.`,
    },
  };

  const inventoryRecords = model.inventory.data?.records ?? [];
  const storageRecords = model.storage.data?.records ?? [];
  const rows = useMemo<WorkspaceRow[]>(
    () => [
      ...(model.inventory.data?.records ?? []).map((record) => ({
        container: "inventory" as const,
        record,
      })),
      ...(model.storage.data?.records ?? []).map((record) => ({
        container: "storage" as const,
        record,
      })),
    ],
    [model.inventory.data, model.storage.data],
  );

  const columns = columnHelper.columns([
    columnHelper.display({
      id: "container",
      header: t`Container`,
      cell: ({ row }) => labels[row.original.container].title,
    }),
    columnHelper.display({
      id: "name",
      header: t`Name`,
      cell: ({ row }) => displayName(model.presentationFor(row.original.record), nameUnavailable),
    }),
    columnHelper.display({
      id: "quantity",
      header: t`Quantity`,
      cell: ({ row }) => String(row.original.record.quantity),
    }),
    columnHelper.display({
      id: "position",
      header: t`Position`,
      cell: ({ row }) => String(row.original.record.physicalIndex),
    }),
    columnHelper.display({
      id: "details",
      header: t`Details`,
      cell: ({ row }) => (
        <Button
          size="sm"
          onClick={(event) => {
            returnFocusRef.current = event.currentTarget;
            model.selectItem(row.original.container, row.original.record.ownedItemID);
          }}
        >
          <Trans>View</Trans>
        </Button>
      ),
    }),
  ]);
  const tableModel = useTable({ features: tableFeatureSet, columns, data: rows }, (state) => state);

  const headers = (
    <>
      <ContainerHeader
        labels={labels.inventory}
        page={model.inventory.data}
        requestedPage={inventoryPage}
        onPage={(card) => setCardNumber("inventory", card)}
      />
      <ContainerHeader
        labels={labels.storage}
        page={model.storage.data}
        requestedPage={storagePage}
        onPage={(card) => setCardNumber("storage", card)}
      />
    </>
  );

  return (
    <Card aria-label={t`Inventory and Storage`} className={panel}>
      <div className={toolbar}>
        <h2 className={visuallyHidden}>
          <Trans>Inventory and Storage</Trans>
        </h2>
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

      {hasSlot && model.revisionState === "mismatch" ? (
        <p role="alert" className={alert}>
          <Trans>
            The Inventory and the Storage Box were read at different save revisions. Reload the
            character before relying on this view.
          </Trans>
        </p>
      ) : null}

      {hasSlot && model.presentations.isError ? (
        <p role="status" className={message}>
          <Trans>Item names and icons are unavailable.</Trans>
        </p>
      ) : null}

      {hasSlot && view === "grid" ? (
        <div className={workspace}>
          <div className={containerColumn}>
            <ContainerHeader
              labels={labels.inventory}
              page={model.inventory.data}
              requestedPage={inventoryPage}
              onPage={(card) => setCardNumber("inventory", card)}
            />
            <ContainerStatus labels={labels.inventory} query={model.inventory} />
            <ContainerGrid
              labels={labels.inventory}
              container="inventory"
              records={inventoryRecords}
              model={model}
              nameUnavailable={nameUnavailable}
              returnFocusRef={returnFocusRef}
            />
          </div>
          <div className={containerColumn}>
            <ContainerHeader
              labels={labels.storage}
              page={model.storage.data}
              requestedPage={storagePage}
              onPage={(card) => setCardNumber("storage", card)}
            />
            <ContainerStatus labels={labels.storage} query={model.storage} />
            <ContainerGrid
              labels={labels.storage}
              container="storage"
              records={storageRecords}
              model={model}
              nameUnavailable={nameUnavailable}
              returnFocusRef={returnFocusRef}
            />
          </div>
        </div>
      ) : null}

      {hasSlot && view === "table" ? (
        <>
          <div className={cardNavigation}>{headers}</div>
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
                    {row.getAllCells().map((cell) => (
                      <td
                        key={cell.id}
                        className={cell.column.id === "details" ? actionCell : undefined}
                      >
                        <tableModel.FlexRender cell={cell} />
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
        open={model.selected !== null}
        onOpenChange={(open) => {
          if (!open) model.clearSelection();
        }}
        title={<Trans>Item details</Trans>}
        description={<Trans>Read-only information from the save and GameCatalog.</Trans>}
        closeLabel={<Trans>Close</Trans>}
        returnFocusRef={returnFocusRef}
      >
        {model.selected ? (
          <ItemDetailContent
            container={labels[model.selected.container].title}
            record={model.selected.record}
            presentation={model.selectedPresentation}
            detail={detail}
          />
        ) : null}
      </Dialog>
    </Card>
  );
}

function ContainerHeader({
  labels,
  page,
  requestedPage,
  onPage,
}: {
  labels: ContainerLabels;
  page: ItemPage | undefined;
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
  query: { isPending: boolean; isError: boolean; isSuccess: boolean; data?: ItemPage };
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
  container: ItemContainer;
  records: readonly ItemRecord[];
  model: ReturnType<typeof useInventoryAndStorage>;
  nameUnavailable: string;
  returnFocusRef: RefObject<HTMLButtonElement | null>;
}) {
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
          <ItemTile
            key={record.ownedItemID}
            record={record}
            presentation={model.presentationFor(record)}
            nameUnavailable={nameUnavailable}
            selected={
              model.selected?.container === container &&
              model.selected.record.ownedItemID === record.ownedItemID
            }
            onSelect={(element) => {
              returnFocusRef.current = element;
              model.selectItem(container, record.ownedItemID);
            }}
          />
        ) : (
          // biome-ignore lint/suspicious/noArrayIndexKey: an empty field has no identity of its own.
          <div key={`empty-${index}`} className={emptyCell} aria-hidden="true" />
        ),
      )}
    </section>
  );
}

function ItemTile({
  record,
  presentation,
  nameUnavailable,
  selected,
  onSelect,
}: {
  record: ItemRecord;
  presentation: CatalogResourcePresentationSummary | null;
  nameUnavailable: string;
  selected: boolean;
  onSelect: (element: HTMLButtonElement) => void;
}) {
  const icon = presentation ? catalogAssetURL(presentation.iconPath) : undefined;

  return (
    <Button className={tile} pressed={selected} onClick={(event) => onSelect(event.currentTarget)}>
      {icon ? (
        <img className={tileIcon} src={icon} alt="" />
      ) : (
        <span className={tileIconPlaceholder} aria-hidden="true" />
      )}
      <span className={tileName}>{displayName(presentation, nameUnavailable)}</span>
      <span className={tileQuantity}>×{record.quantity}</span>
    </Button>
  );
}

function ItemDetailContent({
  container,
  record,
  presentation,
  detail,
}: {
  container: string;
  record: ItemRecord;
  presentation: CatalogResourcePresentationSummary | null;
  detail: ReturnType<typeof useCatalogResource>;
}) {
  const { t } = useLingui();
  const item = detail.data?.item ?? null;
  const catalogName = item?.presentation.name;
  const name =
    catalogName?.known && catalogName.value !== ""
      ? catalogName.value
      : displayName(presentation, t`Name unavailable`);
  const catalogIconPath = item?.presentation.iconPath;
  const iconPath =
    catalogIconPath?.known && catalogIconPath.value !== ""
      ? catalogIconPath.value
      : (presentation?.iconPath ?? "");
  const icon = catalogAssetURL(iconPath);
  const description = knownText(item?.presentation.description);

  return (
    <>
      <div className={detailHeader}>
        {icon ? (
          <img className={detailIcon} src={icon} alt="" />
        ) : (
          <span className={detailIconPlaceholder} aria-hidden="true" />
        )}
        <h3 className={detailHeading}>{name}</h3>
      </div>

      <dl className={facts}>
        <Fact label={t`Container`} value={container} />
        <Fact label={t`Quantity`} value={String(record.quantity)} />
        <Fact label={t`Position`} value={String(record.physicalIndex)} />
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
      {detail.isSuccess && !item ? (
        <p className={message}>
          <Trans>The selected resource has no item details.</Trans>
        </p>
      ) : null}

      {description ? <p className={detailText}>{description}</p> : null}

      {item ? (
        <dl className={facts}>
          <OptionalFact label={t`Category`} value={knownText(item.category)} />
          <OptionalFact label={t`Subcategory`} value={knownText(item.subcategory)} />
          <OptionalFact
            label={t`Maximum in Inventory`}
            value={knownNumber(item.storage.maxInventory)}
          />
          <OptionalFact
            label={t`Maximum in Storage`}
            value={knownNumber(item.storage.maxStorage)}
          />
        </dl>
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

function displayName(
  presentation: CatalogResourcePresentationSummary | null,
  unavailable: string,
): string {
  return presentation && presentation.name !== "" ? presentation.name : unavailable;
}

/** A fact the backend did not resolve is absent, never a synthesised value. */
function knownText(value: CatalogFact<string> | undefined): string | null {
  return value?.known && value.value !== "" ? value.value : null;
}

function knownNumber(value: CatalogFact<number> | undefined): string | null {
  return value?.known ? String(value.value) : null;
}
