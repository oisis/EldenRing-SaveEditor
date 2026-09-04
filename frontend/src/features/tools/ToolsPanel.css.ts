import { style } from "@vanilla-extract/css";
import { tokens } from "../../ui/tokens/contract.css";

/** The horizontal subtab strip of Tools, shaped like the other panel subnavs. */
export const subnav = style({
  display: "flex",
  flexWrap: "wrap",
  gap: tokens.space.sm,
});

/** The stack of setting cards inside one subtab. */
export const sections = style({
  display: "flex",
  flexDirection: "column",
  gap: tokens.space.md,
  minWidth: 0,
});

/**
 * One row of settings that wraps to a column when the window is too narrow for
 * the whole row, which is what section 4.10.1 asks of `Application` and of
 * `Save behavior`.
 */
export const settingsRow = style({
  display: "flex",
  flexWrap: "wrap",
  alignItems: "flex-end",
  gap: tokens.space.md,
});

export const settingField = style({
  display: "flex",
  minWidth: 0,
  flex: "1 1 220px",
  flexDirection: "column",
  gap: tokens.space.xs,
});

/** A field that carries its own submit button next to the control. */
export const fieldWithAction = style({
  display: "flex",
  flexWrap: "wrap",
  alignItems: "center",
  gap: tokens.space.sm,
});

export const settingList = style({
  display: "flex",
  flexDirection: "column",
  gap: tokens.space.sm,
  margin: 0,
  padding: 0,
  listStyle: "none",
});

export const settingItem = style({
  display: "flex",
  flexWrap: "wrap",
  alignItems: "center",
  gap: tokens.space.sm,
});

/** The finding and plan lists of Save integrity. */
export const findings = style({
  display: "flex",
  flexDirection: "column",
  gap: tokens.space.xs,
  margin: 0,
  padding: 0,
  listStyle: "none",
});

export const finding = style({
  display: "flex",
  alignItems: "flex-start",
  gap: tokens.space.sm,
  padding: tokens.space.sm,
  border: `1px solid ${tokens.color.border}`,
  borderRadius: tokens.radius.sm,
  backgroundColor: tokens.color.surfaceRaised,
});

export const findingText = style({ display: "flex", flexDirection: "column", gap: tokens.space.xs });

export const findingMeta = style({ color: tokens.color.textMuted, fontSize: tokens.fontSize.sm });

export const actionsBar = style({
  display: "flex",
  flexWrap: "wrap",
  gap: tokens.space.sm,
});
