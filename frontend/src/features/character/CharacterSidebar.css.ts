import { style } from "@vanilla-extract/css";
import { tokens } from "../../ui/tokens/contract.css";

/**
 * Feature layout only. Every control keeps its base styling from the shared UI
 * library; nothing here redefines a button, a card or a badge.
 */
export const sidebar = style({
  display: "flex",
  flexDirection: "column",
  gap: tokens.space.lg,
  // A narrow panel must shrink instead of pushing the layout sideways.
  minWidth: 0,
});

export const group = style({
  display: "flex",
  flexDirection: "column",
  gap: tokens.space.sm,
  minWidth: 0,
});

export const groupTitle = style({
  margin: 0,
  fontSize: tokens.fontSize.sm,
  color: tokens.color.textMuted,
});

export const list = style({
  listStyle: "none",
  margin: 0,
  padding: 0,
  display: "flex",
  flexDirection: "column",
  gap: tokens.space.xs,
  minWidth: 0,
});

/**
 * The selectable row is the shared `Button`. The doubled selector raises the
 * specificity above the button recipe so this layout wins over the single-line
 * control height without shipping a second button base style.
 */
export const row = style({
  selectors: {
    "&&": {
      width: "100%",
      height: "auto",
      display: "flex",
      flexDirection: "column",
      alignItems: "stretch",
      gap: tokens.space.xs,
      paddingBlock: tokens.space.sm,
      textAlign: "left",
      minWidth: 0,
    },
  },
});

export const rowHead = style({
  display: "flex",
  alignItems: "baseline",
  justifyContent: "space-between",
  gap: tokens.space.sm,
  minWidth: 0,
});

/** A save may carry an arbitrarily long name; it is shown, never rewritten. */
export const name = style({
  overflow: "hidden",
  textOverflow: "ellipsis",
  whiteSpace: "nowrap",
  minWidth: 0,
  fontWeight: 600,
});

export const level = style({
  flex: "none",
  fontSize: tokens.fontSize.sm,
  color: tokens.color.textMuted,
});

export const meta = style({
  fontSize: tokens.fontSize.sm,
  color: tokens.color.textMuted,
  overflow: "hidden",
  textOverflow: "ellipsis",
  whiteSpace: "nowrap",
});

export const inactiveRow = style({
  display: "flex",
  flexDirection: "column",
  gap: tokens.space.xs,
  paddingBlock: tokens.space.sm,
  paddingInline: tokens.space.md,
  border: `1px solid ${tokens.color.border}`,
  borderRadius: tokens.radius.sm,
  color: tokens.color.textMuted,
  minWidth: 0,
});

export const message = style({
  margin: 0,
  color: tokens.color.textMuted,
});

export const alert = style({
  margin: 0,
  padding: tokens.space.md,
  borderRadius: tokens.radius.sm,
  border: `1px solid ${tokens.color.danger}`,
  backgroundColor: tokens.color.dangerSurface,
  color: tokens.color.text,
});
