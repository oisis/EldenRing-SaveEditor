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
  textTransform: "uppercase",
  letterSpacing: "0.06em",
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
      borderColor: "transparent",
      backgroundColor: "transparent",
    },
    '&&[aria-pressed="true"]': {
      borderColor: tokens.color.accent,
      backgroundColor: tokens.color.surfaceHover,
    },
    "&&:hover": { backgroundColor: tokens.color.surfaceHover },
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
  border: "1px solid transparent",
  borderRadius: tokens.radius.sm,
  color: tokens.color.textMuted,
  minWidth: 0,
  flex: 1,
});

/**
 * One list entry: the slot itself and its management control side by side. The
 * slot must shrink, the control must not.
 */
export const entry = style({
  display: "flex",
  alignItems: "stretch",
  gap: tokens.space.xs,
  minWidth: 0,
});

/** The vertical ellipsis. It keeps the shared button base and only squares it. */
export const manage = style({
  selectors: {
    "&&": {
      flex: "none",
      width: tokens.controlHeight.md,
      paddingInline: 0,
      height: "auto",
    },
  },
});

/**
 * The inactive-group disclosure. It is the shared button reduced to a quiet
 * full-width row, so the group keeps one visual weight below the active list.
 */
export const disclosure = style({
  selectors: {
    "&&": {
      width: "100%",
      justifyContent: "flex-start",
      gap: tokens.space.xs,
      borderColor: "transparent",
      backgroundColor: "transparent",
      color: tokens.color.textMuted,
      fontSize: tokens.fontSize.sm,
    },
    "&&:hover": { backgroundColor: tokens.color.surfaceHover },
  },
});

/** The state word of an inactive slot; `Residual data` is the notable one. */
export const stateLabel = style({
  fontWeight: 600,
  overflow: "hidden",
  textOverflow: "ellipsis",
  whiteSpace: "nowrap",
  minWidth: 0,
});

export const dialogSection = style({
  display: "flex",
  flexDirection: "column",
  gap: tokens.space.sm,
  minWidth: 0,
});

export const dialogActions = style({
  display: "flex",
  flexWrap: "wrap",
  gap: tokens.space.sm,
});
