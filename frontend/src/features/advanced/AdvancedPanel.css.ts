import { style } from "@vanilla-extract/css";
import { tokens } from "../../ui/tokens/contract.css";

export const subnav = style({
  display: "flex",
  flexWrap: "wrap",
  gap: tokens.space.sm,
});

export const groupSection = style({
  display: "flex",
  flexDirection: "column",
  gap: tokens.space.md,
  border: `1px solid ${tokens.color.border}`,
  borderRadius: tokens.radius.md,
  background: tokens.color.surfaceRaised,
  padding: tokens.space.md,
});

export const groupHeader = style({
  display: "flex",
  flexWrap: "wrap",
  justifyContent: "space-between",
  alignItems: "center",
  gap: tokens.space.sm,
});

export const groupTitle = style({
  margin: 0,
  fontSize: tokens.fontSize.md,
  fontWeight: 700,
});

export const groupDescription = style({
  margin: 0,
  fontSize: tokens.fontSize.sm,
  color: tokens.color.textMuted,
});

export const presetButtons = style({
  display: "flex",
  flexWrap: "wrap",
  gap: tokens.space.xs,
});

export const presetRolesCard = style({
  display: "flex",
  flexDirection: "column",
  gap: tokens.space.md,
  border: `1px solid ${tokens.color.border}`,
  borderRadius: tokens.radius.md,
  background: tokens.color.surfaceRaised,
  padding: tokens.space.md,
});

export const presetRoleItem = style({
  display: "flex",
  flexWrap: "wrap",
  justifyContent: "space-between",
  alignItems: "center",
  gap: tokens.space.sm,
  paddingBlock: tokens.space.xs,
  borderBottom: `1px solid ${tokens.color.border}`,
  selectors: {
    "&:last-child": {
      borderBottom: "none",
    },
  },
});

export const presetRoleTitle = style({
  margin: 0,
  fontSize: tokens.fontSize.sm,
  fontWeight: 600,
});

export const presetMissingMessage = style({
  fontSize: tokens.fontSize.sm,
  color: tokens.color.danger,
  margin: 0,
});

export const controlsGrid = style({
  display: "grid",
  gridTemplateColumns: "repeat(auto-fill, minmax(320px, 1fr))",
  gap: tokens.space.md,
});

export const paramControl = style({
  display: "flex",
  flexDirection: "column",
  gap: tokens.space.xs,
  padding: tokens.space.sm,
  border: `1px solid ${tokens.color.border}`,
  borderRadius: tokens.radius.sm,
  backgroundColor: tokens.color.surface,
});

export const paramHeader = style({
  display: "flex",
  justifyContent: "space-between",
  alignItems: "baseline",
  gap: tokens.space.xs,
});

export const paramLabel = style({
  fontWeight: 600,
  fontSize: tokens.fontSize.sm,
});

export const paramKey = style({
  fontSize: tokens.fontSize.sm,
  color: tokens.color.textMuted,
  fontFamily: "monospace",
});

export const paramDescription = style({
  margin: 0,
  fontSize: tokens.fontSize.sm,
  color: tokens.color.textMuted,
  lineHeight: 1.4,
});

export const paramInputs = style({
  display: "flex",
  alignItems: "center",
  gap: tokens.space.sm,
  marginTop: tokens.space.xs,
});

export const paramSlider = style({
  flex: "1 1 auto",
  cursor: "pointer",
  accentColor: tokens.color.accent,
});

export const paramNumberInput = style({
  width: "80px",
  textAlign: "right",
});

export const paramUnit = style({
  fontSize: tokens.fontSize.sm,
  color: tokens.color.textMuted,
  minWidth: "16px",
});

export const actionsBar = style({
  display: "flex",
  flexWrap: "wrap",
  alignItems: "center",
  gap: tokens.space.sm,
  paddingTop: tokens.space.sm,
  borderTop: `1px solid ${tokens.color.border}`,
});

export const notice = style({
  display: "flex",
  flexDirection: "column",
  gap: tokens.space.xs,
  padding: tokens.space.md,
  border: `1px solid ${tokens.color.border}`,
  borderRadius: tokens.radius.sm,
  backgroundColor: tokens.color.surfaceRaised,
  fontSize: tokens.fontSize.sm,
  color: tokens.color.textMuted,
});
