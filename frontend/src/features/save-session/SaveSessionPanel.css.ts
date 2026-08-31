import { style } from "@vanilla-extract/css";
import { tokens } from "../../ui/tokens/contract.css";

export const layout = style({
  display: "grid",
  gridTemplateColumns: "minmax(0, 1fr) minmax(0, 260px)",
  gap: tokens.space.lg,
  alignItems: "start",
  "@media": { "screen and (max-width: 720px)": { gridTemplateColumns: "1fr" } },
});

export const stack = style({
  display: "flex",
  minWidth: 0,
  flexDirection: "column",
  gap: tokens.space.md,
});

/**
 * The standing warnings banner. It is deliberately not the danger panel: a save
 * with warnings is editable, and only a blocked one gets the danger treatment.
 */
export const warningBanner = style({
  display: "flex",
  flexWrap: "wrap",
  alignItems: "center",
  gap: tokens.space.sm,
  margin: 0,
  padding: tokens.space.md,
  borderRadius: tokens.radius.sm,
  border: `1px solid ${tokens.color.accent}`,
  backgroundColor: tokens.color.surfaceRaised,
  color: tokens.color.text,
});

export const reportList = style({
  display: "flex",
  flexDirection: "column",
  gap: tokens.space.xs,
  margin: 0,
  paddingLeft: tokens.space.lg,
});

export const reportItem = style({ overflowWrap: "anywhere" });

export const reportScope = style({
  color: tokens.color.textMuted,
  fontFamily: tokens.font.mono,
  fontSize: tokens.fontSize.sm,
});
