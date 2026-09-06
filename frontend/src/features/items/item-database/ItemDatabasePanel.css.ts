import { style } from "@vanilla-extract/css";
import { tokens } from "../../../ui/tokens/contract.css";

/** Item Database layout only. Shared panel patterns live in ui/patterns. */

export const search = style({ flex: "1 1 240px", maxWidth: "480px" });
export const family = style({ flex: "0 1 180px" });

export const grid = style({
  display: "grid",
  gridTemplateColumns: "repeat(5, minmax(0, 1fr))",
  gap: tokens.space.md,
  minHeight: 0,
  "@media": {
    "screen and (max-width: 980px)": { gridTemplateColumns: "repeat(4, minmax(0, 1fr))" },
    "screen and (max-width: 720px)": { gridTemplateColumns: "repeat(2, minmax(0, 1fr))" },
  },
});

export const tile = style({
  minWidth: 0,
  width: "100%",
  height: "142px",
  flexDirection: "column",
  alignItems: "stretch",
  justifyContent: "space-between",
  padding: `${tokens.space.xl} ${tokens.space.sm} ${tokens.space.sm}`,
  textAlign: "left",
});

export const tileName = style({
  overflow: "hidden",
  color: tokens.color.text,
  fontWeight: 700,
  textOverflow: "ellipsis",
  whiteSpace: "nowrap",
});

export const tileMeta = style({
  overflow: "hidden",
  color: tokens.color.textMuted,
  fontSize: tokens.fontSize.sm,
  textOverflow: "ellipsis",
  whiteSpace: "nowrap",
});

export const pagination = style({
  display: "flex",
  alignItems: "center",
  justifyContent: "center",
  gap: tokens.space.sm,
});

export const variantList = style({
  display: "grid",
  gridTemplateColumns: "repeat(2, minmax(0, 1fr))",
  gap: tokens.space.sm,
  margin: 0,
  padding: 0,
  listStyle: "none",
  "@media": { "screen and (max-width: 560px)": { gridTemplateColumns: "1fr" } },
});

export const variant = style({
  padding: tokens.space.sm,
  border: `1px solid ${tokens.color.border}`,
  borderRadius: tokens.radius.sm,
  color: tokens.color.textMuted,
});

/** The cell of one grid result: the tile plus its selection and favourite controls. */
export const cell = style({ position: "relative", display: "flex", minWidth: 0 });

export const cellControls = style({
  position: "absolute",
  insetBlockStart: tokens.space.xs,
  insetInlineStart: tokens.space.xs,
  insetInlineEnd: tokens.space.xs,
  display: "flex",
  alignItems: "center",
  justifyContent: "space-between",
  gap: tokens.space.xs,
  pointerEvents: "none",
});

export const cellControl = style({ pointerEvents: "auto" });

export const tileIcon = style({
  width: "48px",
  height: "48px",
  objectFit: "contain",
  alignSelf: "center",
});

export const tileIconPlaceholder = style({
  width: "36px",
  height: "36px",
  borderRadius: tokens.radius.sm,
  background: tokens.color.surfaceHover,
});

/** The bar shown while at least one result is ticked. */
export const bulkBar = style({
  display: "flex",
  flexWrap: "wrap",
  alignItems: "center",
  gap: tokens.space.sm,
  padding: tokens.space.sm,
  border: `1px solid ${tokens.color.border}`,
  borderRadius: tokens.radius.md,
  background: tokens.color.surfaceRaised,
});

export const filters = style({
  display: "flex",
  flexWrap: "wrap",
  alignItems: "center",
  gap: tokens.space.sm,
});

export const flagRow = style({ display: "flex", flexWrap: "wrap", gap: tokens.space.xs });

export const quantityField = style({ width: "6rem" });

/** One row of the shared Add dialog: a label and its two quantity fields. */
export const addRow = style({
  display: "flex",
  flexWrap: "wrap",
  alignItems: "center",
  gap: tokens.space.sm,
  paddingBlock: tokens.space.xs,
});
