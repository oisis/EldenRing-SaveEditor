import { globalStyle, style } from "@vanilla-extract/css";
import { tokens } from "../../ui/tokens/contract.css";

/** World layout only. Shared panel patterns live in ui/patterns. */

export { subnav } from "../../ui/patterns/workspace.css";

export const datasets = style({
  display: "flex",
  flexDirection: "column",
  gap: tokens.space.sm,
  minHeight: 0,
});

export const dataset = style({
  border: `1px solid ${tokens.color.border}`,
  borderRadius: tokens.radius.md,
  background: tokens.color.surfaceRaised,
});

export const datasetSummary = style({
  display: "flex",
  flexWrap: "wrap",
  alignItems: "center",
  gap: tokens.space.sm,
  padding: tokens.space.md,
  cursor: "pointer",
  selectors: {
    "&::before": { content: '"▸"', color: tokens.color.textMuted },
    "[open] > &::before": { content: '"▾"' },
  },
});

export const datasetTitle = style({ margin: 0, fontSize: tokens.fontSize.md, fontWeight: 700 });

export const datasetBody = style({
  display: "flex",
  flexDirection: "column",
  gap: tokens.space.md,
  padding: tokens.space.md,
  borderTop: `1px solid ${tokens.color.border}`,
});

export const areaTitle = style({
  margin: 0,
  fontSize: tokens.fontSize.sm,
  color: tokens.color.textMuted,
  textTransform: "uppercase",
  letterSpacing: "0.06em",
  display: "inline",
});

export const areaGrid = style({
  display: "grid",
  gridTemplateColumns: "repeat(2, minmax(0, 1fr))",
  gap: tokens.space.md,
  "@media": { "screen and (max-width: 1000px)": { gridTemplateColumns: "minmax(0, 1fr)" } },
});
export const area = style({
  border: `1px solid ${tokens.color.border}`,
  borderRadius: tokens.radius.sm,
  minWidth: 0,
  overflow: "hidden",
});
export const areaSummary = style({
  padding: tokens.space.sm,
  background: tokens.color.surfaceRaised,
});
export const search = style({ flex: "1 1 240px", width: "auto", maxWidth: "360px" });

export const entryList = style({
  display: "grid",
  gridTemplateColumns: "minmax(0, 1fr)",
  gap: 0,
  margin: `${tokens.space.xs} 0 0`,
  padding: 0,
  listStyle: "none",
});

export const entry = style({
  display: "grid",
  gridTemplateColumns: "minmax(0, 1fr) auto",
  gap: tokens.space.xs,
  padding: tokens.space.sm,
  borderBottom: `1px solid ${tokens.color.border}`,
  minWidth: 0,
});

export const entryName = style({ gridColumn: "1", fontWeight: 500, overflowWrap: "anywhere" });

export const entryMeta = style({
  gridColumn: "1",
  color: tokens.color.textMuted,
  fontSize: tokens.fontSize.sm,
  overflowWrap: "anywhere",
});
globalStyle(`.${entry} > button`, { gridColumn: "2", gridRow: "1 / span 2", alignSelf: "center" });

export const stepList = style({
  display: "flex",
  flexDirection: "column",
  gap: tokens.space.xs,
  margin: 0,
  padding: 0,
  listStyle: "none",
});

/** The row that carries a dataset's backend risk line and its bulk action. */
export const datasetControls = style({
  display: "flex",
  flexWrap: "wrap",
  alignItems: "center",
  gap: tokens.space.sm,
});

/** The explicit step picker and its Apply button inside one quest entry. */
export const stepPicker = style({
  display: "flex",
  flexWrap: "wrap",
  alignItems: "center",
  gap: tokens.space.xs,
});
