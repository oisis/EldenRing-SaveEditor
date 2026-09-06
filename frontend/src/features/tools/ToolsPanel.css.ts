import { globalStyle, style } from "@vanilla-extract/css";
import { tokens } from "../../ui/tokens/contract.css";

/** The horizontal subtab strip of Tools, shaped like the other panel subnavs. */
export { subnav } from "../../ui/patterns/workspace.css";

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
  alignItems: "flex-start",
  gap: tokens.space.md,
});

export const settingField = style({
  display: "flex",
  minWidth: 0,
  flex: "0 0 auto",
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

globalStyle(`.${settingsRow} > .${settingField}`, { flex: "1 1 220px" });
globalStyle(`.${fieldWithAction} > input`, { flex: "1 1 120px", width: 0, minWidth: 0 });

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

export const findingText = style({
  display: "flex",
  flexDirection: "column",
  gap: tokens.space.xs,
});

export const findingMeta = style({ color: tokens.color.textMuted, fontSize: tokens.fontSize.sm });

export const actionsBar = style({
  display: "flex",
  flexWrap: "wrap",
  gap: tokens.space.sm,
});

/**
 * The About & Updates grid: two cards per row, and every card in a row as tall
 * as the tallest one. `align-items: stretch` on a grid row is what section
 * 4.10.5 asks for, without any measurement in JavaScript.
 */
export const cardGrid = style({
  display: "grid",
  gridTemplateColumns: "repeat(2, minmax(0, 1fr))",
  alignItems: "stretch",
  gap: tokens.space.lg,
  "@media": { "screen and (max-width: 850px)": { gridTemplateColumns: "minmax(0, 1fr)" } },
});

/** One About card: its footer actions sit at the bottom of the stretched card. */
export const gridCard = style({
  display: "flex",
  flexDirection: "column",
  gap: tokens.space.sm,
  minWidth: 0,
  minHeight: "180px",
});

/** The action row a stretched card pins to its bottom edge. */
export const cardActions = style({
  display: "flex",
  flexWrap: "wrap",
  gap: tokens.space.sm,
  marginTop: "auto",
});

/** The two-column form of the target and template editors. */
export const formGrid = style({
  display: "grid",
  gridTemplateColumns: "repeat(auto-fit, minmax(220px, 1fr))",
  gap: tokens.space.md,
});

/** The stage list a finished long operation reports. */
export const stageList = style({
  display: "flex",
  flexDirection: "column",
  gap: tokens.space.xs,
  margin: 0,
  padding: 0,
  listStyle: "none",
  color: tokens.color.textMuted,
  fontSize: tokens.fontSize.sm,
});

/** A row of badges, used for template tags and backup tags. */
export const badgeRow = style({
  display: "flex",
  flexWrap: "wrap",
  gap: tokens.space.xs,
});
