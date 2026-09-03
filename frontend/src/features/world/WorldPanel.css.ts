import { style } from "@vanilla-extract/css";
import { tokens } from "../../ui/tokens/contract.css";

/** World layout only. Shared panel patterns live in ui/patterns. */

export const subnav = style({
  display: "flex",
  flexWrap: "wrap",
  gap: tokens.space.sm,
});

export const datasets = style({
  display: "flex",
  flexDirection: "column",
  gap: tokens.space.md,
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
  listStyle: "none",
  selectors: { "&::-webkit-details-marker": { display: "none" } },
});

export const datasetTitle = style({ margin: 0, fontSize: tokens.fontSize.md, fontWeight: 700 });

export const datasetBody = style({
  display: "flex",
  flexDirection: "column",
  gap: tokens.space.md,
  padding: `0 ${tokens.space.md} ${tokens.space.md}`,
});

export const areaTitle = style({
  margin: 0,
  fontSize: tokens.fontSize.sm,
  color: tokens.color.textMuted,
  textTransform: "uppercase",
  letterSpacing: "0.06em",
});

export const entryList = style({
  display: "grid",
  gridTemplateColumns: "repeat(auto-fill, minmax(240px, 1fr))",
  gap: tokens.space.sm,
  margin: `${tokens.space.xs} 0 0`,
  padding: 0,
  listStyle: "none",
});

export const entry = style({
  display: "flex",
  flexDirection: "column",
  gap: tokens.space.xs,
  padding: tokens.space.sm,
  border: `1px solid ${tokens.color.border}`,
  borderRadius: tokens.radius.sm,
  minWidth: 0,
});

export const entryName = style({ fontWeight: 700, overflowWrap: "anywhere" });

export const entryMeta = style({
  color: tokens.color.textMuted,
  fontSize: tokens.fontSize.sm,
  overflowWrap: "anywhere",
});

export const stepList = style({
  display: "flex",
  flexDirection: "column",
  gap: tokens.space.xs,
  margin: 0,
  padding: 0,
  listStyle: "none",
});
