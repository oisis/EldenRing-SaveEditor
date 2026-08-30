import { globalStyle, style } from "@vanilla-extract/css";
import { tokens } from "../../tokens/contract.css";

export const frame = style({
  minHeight: 0,
  overflow: "auto",
  border: `1px solid ${tokens.color.border}`,
  borderRadius: tokens.radius.md,
  backgroundColor: tokens.color.surface,
});

export const table = style({
  width: "100%",
  borderCollapse: "collapse",
  color: tokens.color.text,
  fontSize: tokens.fontSize.md,
});

globalStyle(`${table} th`, {
  position: "sticky",
  top: 0,
  zIndex: 1,
  padding: tokens.space.sm,
  borderBottom: `1px solid ${tokens.color.borderStrong}`,
  backgroundColor: tokens.color.surfaceRaised,
  color: tokens.color.textMuted,
  textAlign: "left",
  fontWeight: 700,
});

globalStyle(`${table} td`, {
  padding: tokens.space.sm,
  borderBottom: `1px solid ${tokens.color.border}`,
  verticalAlign: "middle",
});

globalStyle(`${table} tbody tr:last-child td`, { borderBottom: 0 });
globalStyle(`${table} tbody tr:hover`, { backgroundColor: tokens.color.surfaceHover });
