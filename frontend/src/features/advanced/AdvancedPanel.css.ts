import { style } from "@vanilla-extract/css";
import { tokens } from "../../ui/tokens/contract.css";

export { subnav } from "../../ui/patterns/workspace.css";

export const networkGrid = style({
  display: "grid",
  gridTemplateColumns: "repeat(2, minmax(0, 1fr))",
  gap: tokens.space.lg,
  "@media": { "screen and (max-width: 1000px)": { gridTemplateColumns: "minmax(0, 1fr)" } },
});

export const groupSection = style({
  display: "flex",
  flexDirection: "column",
  gap: tokens.space.md,
  border: `1px solid ${tokens.color.border}`,
  borderRadius: tokens.radius.md,
  background: tokens.color.surface,
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
  display: "grid",
  gridTemplateColumns: "repeat(3, minmax(0, 1fr))",
  gap: tokens.space.md,
  border: `1px solid ${tokens.color.border}`,
  borderRadius: tokens.radius.md,
  background: tokens.color.surfaceRaised,
  padding: tokens.space.md,
  "@media": {
    "screen and (max-width: 1100px)": { gridTemplateColumns: "repeat(2, minmax(0, 1fr))" },
  },
});

export const presetRoleItem = style({
  display: "flex",
  flexWrap: "wrap",
  justifyContent: "space-between",
  alignItems: "center",
  gap: tokens.space.sm,
  paddingBlock: tokens.space.xs,
  minWidth: 0,
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
  display: "flex",
  flexDirection: "column",
  gap: tokens.space.sm,
});

export const paramControl = style({
  display: "grid",
  gridTemplateColumns: "110px minmax(0, 1fr)",
  alignItems: "center",
  gap: tokens.space.xs,
  padding: tokens.space.sm,
  borderBottom: `1px solid ${tokens.color.border}`,
  minWidth: 0,
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
  display: "none",
});

export const paramDescription = style({
  gridColumn: "1 / -1",
  gridRow: "2",
  margin: 0,
  fontSize: tokens.fontSize.sm,
  color: tokens.color.textMuted,
  lineHeight: 1.4,
});

export const paramInputs = style({
  gridColumn: "2",
  gridRow: "1",
  minWidth: 0,
  display: "flex",
  alignItems: "center",
  gap: tokens.space.sm,
  marginTop: tokens.space.xs,
});

export const paramSlider = style({
  flex: "1 1 auto",
  minWidth: 0,
  cursor: "pointer",
  accentColor: tokens.color.accent,
});

export const paramNumberInput = style({
  selectors: { "&&": { width: "64px", flex: "0 0 64px", textAlign: "right" } },
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
