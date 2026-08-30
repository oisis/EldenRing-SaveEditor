import { style } from "@vanilla-extract/css";
import { tokens } from "../../../ui/tokens/contract.css";

/**
 * Inventory and Storage layout only. Every control keeps the appearance of its
 * canonical component; nothing here redefines a button, a badge, a table or a
 * dialog, and the shared panel patterns live in ui/patterns.
 */

const cellSize = "96px";
const iconSize = "36px";

export const workspace = style({
  display: "grid",
  gridTemplateColumns: "repeat(2, minmax(0, 1fr))",
  gap: tokens.space.lg,
  minHeight: 0,
  "@media": { "screen and (max-width: 1080px)": { gridTemplateColumns: "1fr" } },
});

export const container = style({
  display: "flex",
  minWidth: 0,
  flexDirection: "column",
  gap: tokens.space.sm,
});

export const containerHead = style({
  display: "flex",
  flexWrap: "wrap",
  alignItems: "center",
  gap: tokens.space.sm,
});

export const containerTitle = style({
  margin: 0,
  fontSize: tokens.fontSize.md,
  textTransform: "uppercase",
  letterSpacing: "0.06em",
  color: tokens.color.textMuted,
});

export const pagination = style({
  display: "flex",
  alignItems: "center",
  gap: tokens.space.xs,
  marginInlineStart: "auto",
});

export const cardNavigation = style({
  display: "flex",
  flexWrap: "wrap",
  alignItems: "center",
  gap: tokens.space.lg,
});

export const grid = style({
  display: "grid",
  gridTemplateColumns: "repeat(5, minmax(0, 1fr))",
  gap: tokens.space.xs,
  padding: tokens.space.sm,
  border: `1px solid ${tokens.color.border}`,
  borderRadius: tokens.radius.md,
  backgroundColor: tokens.color.surface,
  "@media": {
    "screen and (max-width: 560px)": { gridTemplateColumns: "repeat(3, minmax(0, 1fr))" },
  },
});

export const tile = style({
  minWidth: 0,
  height: cellSize,
  flexDirection: "column",
  alignItems: "stretch",
  justifyContent: "space-between",
  padding: tokens.space.xs,
  textAlign: "center",
});

export const emptyCell = style({
  height: cellSize,
  border: `1px dashed ${tokens.color.border}`,
  borderRadius: tokens.radius.sm,
  backgroundColor: tokens.color.surfaceRaised,
});

export const tileIcon = style({
  width: iconSize,
  height: iconSize,
  alignSelf: "center",
  objectFit: "contain",
});

export const tileIconPlaceholder = style({
  width: iconSize,
  height: iconSize,
  alignSelf: "center",
  border: `1px dashed ${tokens.color.border}`,
  borderRadius: tokens.radius.sm,
});

export const tileName = style({
  overflow: "hidden",
  color: tokens.color.text,
  fontSize: tokens.fontSize.sm,
  textOverflow: "ellipsis",
  whiteSpace: "nowrap",
});

export const tileQuantity = style({
  overflow: "hidden",
  color: tokens.color.textMuted,
  fontSize: tokens.fontSize.sm,
  textOverflow: "ellipsis",
  whiteSpace: "nowrap",
});

export const detailHeader = style({
  display: "flex",
  alignItems: "center",
  gap: tokens.space.md,
});

export const detailIcon = style({ width: "48px", height: "48px", objectFit: "contain" });

export const detailIconPlaceholder = style({
  width: "48px",
  height: "48px",
  border: `1px dashed ${tokens.color.border}`,
  borderRadius: tokens.radius.sm,
});
