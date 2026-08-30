import { style } from "@vanilla-extract/css";
import { tokens } from "../tokens/contract.css";

/**
 * Layout shared by every screen panel: the card frame, its toolbar row, the
 * status lines, the scrolling table frame and the detail fact list. A screen
 * imports these instead of restating them, so two panels cannot drift apart.
 * Anything specific to one screen stays in that feature's own stylesheet.
 */

export const panel = style({ minHeight: 0 });

export const toolbar = style({
  display: "flex",
  flexWrap: "wrap",
  alignItems: "center",
  gap: tokens.space.sm,
});

export const spacer = style({ flex: "1 1 auto" });

export const viewSwitch = style({
  display: "flex",
  gap: tokens.space.xs,
  margin: 0,
  padding: 0,
  border: 0,
});

export const message = style({ margin: 0, color: tokens.color.textMuted });
export const alert = style({ margin: 0, color: tokens.color.danger });

/**
 * The boxed variant of the same failure: a standing danger panel rather than
 * one danger-coloured line. Both variants stay named, because a screen picks
 * between them by weight, not by accident.
 */
export const alertPanel = style({
  margin: 0,
  padding: tokens.space.md,
  borderRadius: tokens.radius.sm,
  border: `1px solid ${tokens.color.danger}`,
  backgroundColor: tokens.color.dangerSurface,
  color: tokens.color.text,
});

export const tableFrame = style({ height: "min(520px, calc(100vh - 260px))", minHeight: "280px" });
export const actionCell = style({ width: "1%", whiteSpace: "nowrap" });

export const detailHeading = style({ margin: 0, fontSize: tokens.fontSize.lg });
export const detailText = style({ margin: 0, color: tokens.color.textMuted });

export const facts = style({
  display: "grid",
  gridTemplateColumns: "repeat(3, minmax(0, 1fr))",
  gap: tokens.space.sm,
  margin: 0,
  "@media": { "screen and (max-width: 560px)": { gridTemplateColumns: "1fr" } },
});

export const fact = style({
  display: "flex",
  minWidth: 0,
  flexDirection: "column",
  gap: tokens.space.xs,
  padding: tokens.space.sm,
  border: `1px solid ${tokens.color.border}`,
  borderRadius: tokens.radius.sm,
  backgroundColor: tokens.color.surfaceRaised,
});

export const factLabel = style({ color: tokens.color.textMuted, fontSize: tokens.fontSize.sm });
export const factValue = style({ overflowWrap: "anywhere", fontWeight: 700 });

export const visuallyHidden = style({
  position: "absolute",
  width: "1px",
  height: "1px",
  padding: 0,
  margin: "-1px",
  overflow: "hidden",
  clip: "rect(0, 0, 0, 0)",
  whiteSpace: "nowrap",
  border: 0,
});
