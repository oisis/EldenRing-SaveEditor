import { style } from "@vanilla-extract/css";
import { tokens } from "../../../ui/tokens/contract.css";

/**
 * Inventory and Storage layout only. Every control keeps the appearance of its
 * canonical component; nothing here redefines a button, a badge, a table or a
 * dialog, and the shared panel patterns live in ui/patterns.
 */

const iconSize = "32px";

export const workspace = style({
  display: "grid",
  gridTemplateColumns: "repeat(2, minmax(0, 1fr))",
  gap: tokens.space.lg,
  minHeight: 0,
  "@media": { "screen and (max-width: 820px)": { gridTemplateColumns: "1fr" } },
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
  gap: "6px",
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
  width: "100%",
  height: "auto",
  aspectRatio: "1",
  position: "relative",
  flexDirection: "column",
  alignItems: "stretch",
  justifyContent: "space-between",
  padding: tokens.space.xs,
  textAlign: "center",
});

export const emptyCell = style({
  aspectRatio: "1",
  border: `1px dashed ${tokens.color.border}`,
  borderRadius: tokens.radius.sm,
  backgroundColor: tokens.color.surfaceRaised,
});

export const tileIcon = style({
  width: iconSize,
  height: iconSize,
  alignSelf: "center",
  objectFit: "contain",
  margin: "auto",
});

export const tileIconPlaceholder = style({
  width: iconSize,
  height: iconSize,
  alignSelf: "center",
  border: `1px dashed ${tokens.color.border}`,
  borderRadius: tokens.radius.sm,
  margin: "auto",
});

export const tileName = style({
  position: "absolute",
  width: "1px",
  height: "1px",
  clipPath: "inset(50%)",
  overflow: "hidden",
  color: tokens.color.text,
  fontSize: tokens.fontSize.sm,
  textOverflow: "ellipsis",
  whiteSpace: "nowrap",
});

export const tileQuantity = style({
  position: "absolute",
  bottom: "3px",
  right: "4px",
  maxWidth: "calc(100% - 8px)",
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

/** The cell of one physical field: the tile plus its selection and favourite controls. */
export const cell = style({
  position: "relative",
  display: "flex",
  minWidth: 0,
});

/** The two overlay controls of a tile, kept out of the tile's own click target. */
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

export const cellControl = style({
  pointerEvents: "auto",
});

/** The bulk action bar shown while at least one record is selected. */
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

export const searchField = style({ flex: "1 1 220px", maxWidth: "420px" });
export const filterField = style({ flex: "0 1 180px" });
export const quantityField = style({ width: "6rem" });

/** The badge row of a tile or a detail dialog: safety and presentation flags. */
export const flagRow = style({
  display: "flex",
  flexWrap: "wrap",
  gap: tokens.space.xs,
});
