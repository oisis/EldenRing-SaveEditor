import { Trans, useLingui } from "@lingui/react/macro";
import { createColumnHelper, tableFeatures, useTable } from "@tanstack/react-table";
import { useRef, useState } from "react";
import type { CatalogFact, CatalogResourceSummary } from "../../../application/catalog/catalogPort";
import { Badge } from "../../../ui/components/Badge/Badge";
import { Button } from "../../../ui/components/Button/Button";
import { Card } from "../../../ui/components/Card/Card";
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
  family as familyControl,
  grid,
  message,
  pagination,
  panel,
  search as searchControl,
  spacer,
  tableFrame,
  tile,
  tileMeta,
  tileName,
  toolbar,
  variant,
  variantList,
  viewSwitch,
  visuallyHidden,
} from "./ItemDatabasePanel.css";
import { useItemDatabase } from "./useItemDatabase";

type ItemDatabaseView = "grid" | "table";

const gridPageSize = 20;
const tablePageSize = 50;
const tableFeatureSet = tableFeatures({});
const columnHelper = createColumnHelper<typeof tableFeatureSet, CatalogResourceSummary>();

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

/**
 * The first read-only Item Database view. It renders only fields and filters
 * the existing catalog contract can state; unavailable domain features are not
 * simulated locally.
 */
export function ItemDatabasePanel() {
  const { t } = useLingui();
  const returnFocusRef = useRef<HTMLButtonElement | null>(null);
  const [view, setView] = useState<ItemDatabaseView>("grid");
  const [search, setSearch] = useState("");
  const [family, setFamily] = useState("");
  const [requestedPage, setRequestedPage] = useState(1);
  const pageSize = view === "grid" ? gridPageSize : tablePageSize;
  const model = useItemDatabase({
    family,
    capability: "",
    search,
    page: requestedPage,
    pageSize,
  });
  const rows = model.resources.data?.resources ?? [];
  const selected = model.selected;

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

  const columns = columnHelper.columns([
    columnHelper.accessor("name", {
      header: t`Name`,
      cell: (cell) => displayName(cell.getValue(), t`Name unavailable`),
    }),
    columnHelper.accessor("family", {
      header: t`Family`,
      cell: (cell) => familyLabel(cell.getValue(), familyLabels, t`Unavailable`),
    }),
    columnHelper.display({
      id: "details",
      header: t`Details`,
      cell: ({ row }) => (
        <Button
          size="sm"
          onClick={(event) => {
            returnFocusRef.current = event.currentTarget;
            model.selectItem(row.original.kind, row.original.key);
          }}
        >
          <Trans>View</Trans>
        </Button>
      ),
    }),
  ]);
  const tableModel = useTable({ features: tableFeatureSet, columns, data: rows }, (state) => state);

  const servedPage = model.resources.data?.page ?? requestedPage;
  const servedPageSize = model.resources.data?.pageSize ?? pageSize;
  const total = model.resources.data?.total ?? 0;
  const hasPrevious = servedPage > 1;
  const hasNext = servedPage * servedPageSize < total;

  const changeSearch = (value: string) => {
    setSearch(value);
    setRequestedPage(1);
  };
  const changeFamily = (value: string) => {
    setFamily(value);
    setRequestedPage(1);
  };
  const changeView = (next: ItemDatabaseView) => {
    setView(next);
    setRequestedPage(1);
  };

  return (
    <Card aria-label={t`Item Database`} className={panel}>
      <div className={toolbar}>
        <label className={visuallyHidden} htmlFor="item-database-search">
          <Trans>Search items</Trans>
        </label>
        <Input
          id="item-database-search"
          className={searchControl}
          type="search"
          value={search}
          placeholder={t`Search items`}
          onChange={(event) => changeSearch(event.currentTarget.value)}
        />

        <label className={visuallyHidden} htmlFor="item-database-family">
          <Trans>Item family</Trans>
        </label>
        <Select
          id="item-database-family"
          className={familyControl}
          value={family}
          onChange={(event) => changeFamily(event.currentTarget.value)}
        >
          {families.map((value) => (
            <option key={value || "all"} value={value}>
              {familyLabels[value]}
            </option>
          ))}
        </Select>

        <span className={spacer} />
        <Badge>
          <Trans>Results: {total}</Trans>
        </Badge>
        <fieldset className={viewSwitch}>
          <legend className={visuallyHidden}>
            <Trans>View</Trans>
          </legend>
          <Button size="sm" pressed={view === "grid"} onClick={() => changeView("grid")}>
            <Trans>Grid</Trans>
          </Button>
          <Button size="sm" pressed={view === "table"} onClick={() => changeView("table")}>
            <Trans>Table</Trans>
          </Button>
        </fieldset>
      </div>

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
      {model.resources.isSuccess && rows.length === 0 ? (
        <p className={message}>
          <Trans>No items match the current search.</Trans>
        </p>
      ) : null}

      {model.resources.isSuccess && rows.length > 0 && view === "grid" ? (
        <section className={grid} aria-label={t`Item results`}>
          {rows.map((row) => (
            <Button
              key={`${row.kind}:${row.key}`}
              className={tile}
              pressed={selected?.kind === row.kind && selected.key === row.key}
              onClick={(event) => {
                returnFocusRef.current = event.currentTarget;
                model.selectItem(row.kind, row.key);
              }}
            >
              <span className={tileName}>{displayName(row.name, t`Name unavailable`)}</span>
              <span className={tileMeta}>
                {familyLabel(row.family, familyLabels, t`Unavailable`)}
              </span>
            </Button>
          ))}
        </section>
      ) : null}

      {model.resources.isSuccess && rows.length > 0 && view === "table" ? (
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
      ) : null}

      {model.resources.isSuccess && view === "grid" && total > servedPageSize ? (
        <nav className={pagination} aria-label={t`Item Database pages`}>
          <Button
            size="sm"
            disabled={!hasPrevious}
            onClick={() => setRequestedPage(servedPage - 1)}
          >
            <Trans>Previous</Trans>
          </Button>
          <Badge>
            <Trans>Page {servedPage}</Trans>
          </Badge>
          <Button size="sm" disabled={!hasNext} onClick={() => setRequestedPage(servedPage + 1)}>
            <Trans>Next</Trans>
          </Button>
        </nav>
      ) : null}

      <Dialog
        open={selected !== null}
        onOpenChange={(open) => {
          if (!open) model.clearSelection();
        }}
        title={<Trans>Item details</Trans>}
        description={<Trans>Read-only information from GameCatalog.</Trans>}
        closeLabel={<Trans>Close</Trans>}
        returnFocusRef={returnFocusRef}
      >
        <ItemDetailContent model={model} familyLabels={familyLabels} />
      </Dialog>
    </Card>
  );
}

function ItemDetailContent({
  model,
  familyLabels,
}: {
  model: ReturnType<typeof useItemDatabase>;
  familyLabels: Record<(typeof families)[number], string>;
}) {
  const { t } = useLingui();

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

  const name = factText(item.presentation.name, t`Name unavailable`);
  const caption = factText(item.presentation.caption, "");
  const description = factText(item.presentation.description, "");
  const location = factText(item.presentation.location, t`Unavailable`);
  const familyValue = item.family.known ? item.family.value : "";
  const familyName = familyLabel(familyValue, familyLabels, t`Unavailable`);

  return (
    <>
      <h3 className={detailHeading}>{name}</h3>
      {caption ? <p className={detailText}>{caption}</p> : null}
      {description ? <p>{description}</p> : null}
      <dl className={facts}>
        <div className={fact}>
          <dt className={factLabel}>
            <Trans>Family</Trans>
          </dt>
          <dd className={factValue}>{familyName}</dd>
        </div>
        <Fact label={t`Category`} value={item.category} unavailable={t`Unavailable`} />
        <Fact label={t`Subcategory`} value={item.subcategory} unavailable={t`Unavailable`} />
        <div className={fact}>
          <dt className={factLabel}>
            <Trans>Location</Trans>
          </dt>
          <dd className={factValue}>{location}</dd>
        </div>
      </dl>
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

function displayName(value: string, unavailable: string) {
  return value === "" ? unavailable : value;
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
