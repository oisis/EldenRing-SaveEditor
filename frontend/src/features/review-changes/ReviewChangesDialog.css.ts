import { style } from "@vanilla-extract/css";
import { tokens } from "../../ui/tokens/contract.css";

export const toolbar = style({
  display: "flex",
  flexWrap: "wrap",
  alignItems: "center",
  gap: tokens.space.sm,
});

export const summary = style({
  display: "flex",
  flexWrap: "wrap",
  gap: tokens.space.sm,
  margin: 0,
});

export const list = style({
  display: "flex",
  flexDirection: "column",
  gap: tokens.space.sm,
  margin: 0,
  padding: 0,
  listStyle: "none",
});

export const group = style({
  display: "flex",
  flexDirection: "column",
  gap: tokens.space.sm,
});

export const groupHeading = style({ margin: 0, fontSize: tokens.fontSize.md });

export const item = style({
  display: "grid",
  gridTemplateColumns: "minmax(0, 1fr) auto",
  gap: tokens.space.sm,
  padding: tokens.space.md,
  border: `1px solid ${tokens.color.border}`,
  borderRadius: tokens.radius.sm,
  backgroundColor: tokens.color.surfaceRaised,
});

export const itemBody = style({
  display: "flex",
  minWidth: 0,
  flexDirection: "column",
  gap: tokens.space.xs,
});

export const metadata = style({
  color: tokens.color.textMuted,
  fontFamily: tokens.font.mono,
  fontSize: tokens.fontSize.sm,
  overflowWrap: "anywhere",
});

export const state = style({
  margin: 0,
  color: tokens.color.textMuted,
  overflowWrap: "anywhere",
});

export const validation = style({
  display: "flex",
  flexDirection: "column",
  gap: tokens.space.sm,
  padding: tokens.space.md,
  border: `1px solid ${tokens.color.borderStrong}`,
  borderRadius: tokens.radius.sm,
});

export const validationList = style({ margin: 0, paddingLeft: tokens.space.lg });

export const actions = style({
  display: "flex",
  flexWrap: "wrap",
  justifyContent: "flex-end",
  gap: tokens.space.sm,
});
