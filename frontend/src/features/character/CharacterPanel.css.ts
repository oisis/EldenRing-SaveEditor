import { style } from "@vanilla-extract/css";
import { tokens } from "../../ui/tokens/contract.css";

export const subnav = style({
  display: "flex",
  gap: tokens.space.xs,
  marginBottom: tokens.space.md,
});

export const sectionGrid = style({
  display: "grid",
  gridTemplateColumns: "repeat(2, minmax(0, 1fr))",
  gap: tokens.space.md,
  "@media": {
    "screen and (max-width: 800px)": {
      gridTemplateColumns: "1fr",
    },
  },
});

export const identityCard = style({
  marginBottom: tokens.space.md,
});

export const identityGrid = style({
  display: "grid",
  gridTemplateColumns: "repeat(auto-fit, minmax(200px, 1fr))",
  gap: tokens.space.md,
  alignItems: "end",
});

export const fieldGroup = style({
  display: "flex",
  flexDirection: "column",
  gap: tokens.space.xs,
});

export const fieldLabel = style({
  fontSize: tokens.fontSize.sm,
  fontWeight: 600,
  color: tokens.color.textMuted,
});

export const nameForm = style({
  display: "flex",
  gap: tokens.space.xs,
  alignItems: "center",
});

export const attributeRow = style({
  display: "flex",
  alignItems: "center",
  gap: tokens.space.sm,
  padding: `${tokens.space.xs} 0`,
});

export const attributeName = style({
  width: "100px",
  fontSize: tokens.fontSize.sm,
  fontWeight: 600,
  textTransform: "capitalize",
});

export const attributeSlider = style({
  flex: "1 1 auto",
  cursor: "pointer",
  accentColor: tokens.color.accent,
});

export const attributeInput = style({
  width: "60px",
});

export const statGrid = style({
  display: "grid",
  gridTemplateColumns: "repeat(auto-fit, minmax(140px, 1fr))",
  gap: tokens.space.sm,
});

export const statBox = style({
  display: "flex",
  flexDirection: "column",
  gap: tokens.space.xs,
  padding: tokens.space.sm,
  borderRadius: tokens.radius.sm,
  backgroundColor: tokens.color.surfaceRaised,
  border: `1px solid ${tokens.color.border}`,
});

export const statBoxLabel = style({
  fontSize: tokens.fontSize.sm,
  fontWeight: 600,
  color: tokens.color.textMuted,
  textTransform: "uppercase",
  letterSpacing: "0.05em",
});

export const statBoxValue = style({
  fontSize: tokens.fontSize.md,
  fontWeight: 700,
});

export const statBoxSub = style({
  fontSize: tokens.fontSize.sm,
  color: tokens.color.textMuted,
});

export const presetContainer = style({
  display: "flex",
  flexDirection: "column",
  gap: tokens.space.lg,
});

export const presetViewer = style({
  display: "flex",
  flexDirection: "column",
  alignItems: "center",
  gap: tokens.space.md,
  padding: tokens.space.md,
  borderRadius: tokens.radius.md,
  backgroundColor: tokens.color.surfaceRaised,
  border: `1px solid ${tokens.color.border}`,
});

export const presetImage = style({
  maxHeight: "360px",
  maxWidth: "100%",
  objectFit: "contain",
  borderRadius: tokens.radius.sm,
});

export const presetImagePlaceholder = style({
  width: "280px",
  height: "360px",
  display: "flex",
  alignItems: "center",
  justifyContent: "center",
  borderRadius: tokens.radius.sm,
  backgroundColor: tokens.color.surface,
  color: tokens.color.textMuted,
  fontSize: tokens.fontSize.sm,
});

export const presetControls = style({
  display: "flex",
  alignItems: "center",
  gap: tokens.space.md,
  width: "100%",
  maxWidth: "480px",
  justifyContent: "center",
});

export const presetTags = style({
  display: "flex",
  flexWrap: "wrap",
  gap: tokens.space.xs,
  justifyContent: "center",
});

export const favoritesGrid = style({
  display: "grid",
  gridTemplateColumns: "repeat(auto-fill, minmax(220px, 1fr))",
  gap: tokens.space.sm,
});

export const favoriteSlotCard = style({
  display: "flex",
  flexDirection: "column",
  gap: tokens.space.xs,
  padding: tokens.space.sm,
  borderRadius: tokens.radius.sm,
  border: `1px solid ${tokens.color.border}`,
  backgroundColor: tokens.color.surfaceRaised,
});

export const favoriteSlotHeader = style({
  display: "flex",
  justifyContent: "space-between",
  alignItems: "center",
});

export const favoriteSlotActions = style({
  display: "flex",
  gap: tokens.space.xs,
  marginTop: tokens.space.xs,
});
