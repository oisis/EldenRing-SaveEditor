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
  height: "112px",
  flexDirection: "column",
  alignItems: "stretch",
  justifyContent: "space-between",
  padding: tokens.space.md,
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
