import { globalStyle, style } from "@vanilla-extract/css";
import { tokens } from "../tokens/contract.css";

/** Shared segmented module subnavigation from the approved desktop mockup. */
export const subnav = style({
  display: "inline-flex",
  alignSelf: "flex-start",
  width: "fit-content",
  maxWidth: "100%",
  overflowX: "auto",
  gap: "2px",
  padding: "2px",
  border: `1px solid ${tokens.color.border}`,
  borderRadius: tokens.radius.sm,
  backgroundColor: tokens.color.surfaceRaised,
  flexShrink: 0,
});
globalStyle(`.${subnav} > button`, {
  flex: "none",
  height: "28px",
  paddingInline: tokens.space.md,
  border: 0,
  borderRadius: "4px",
  background: "transparent",
  color: tokens.color.textMuted,
  whiteSpace: "nowrap",
});
globalStyle(`.${subnav} > button[aria-pressed="true"]`, {
  backgroundColor: tokens.color.surface,
  color: tokens.color.text,
  boxShadow: "0 1px 2px rgb(0 0 0 / 12%)",
  fontWeight: 600,
});

export const workspaceStack = style({
  display: "flex",
  flexDirection: "column",
  gap: tokens.space.lg,
  minWidth: 0,
});

export const disclosure = style({
  border: `1px solid ${tokens.color.border}`,
  borderRadius: tokens.radius.md,
  backgroundColor: tokens.color.surface,
  overflow: "hidden",
  minWidth: 0,
});
export const disclosureHeading = style({
  padding: `${tokens.space.md} ${tokens.space.lg}`,
  cursor: "pointer",
  backgroundColor: tokens.color.surfaceRaised,
});
globalStyle(`.${disclosureHeading} > h2`, {
  display: "inline",
  margin: 0,
  fontSize: tokens.fontSize.md,
});
export const disclosureBody = style({
  display: "flex",
  flexDirection: "column",
  gap: tokens.space.sm,
  padding: tokens.space.lg,
  borderTop: `1px solid ${tokens.color.border}`,
});
