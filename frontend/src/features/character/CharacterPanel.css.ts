import { globalStyle, style } from "@vanilla-extract/css";
import { tokens } from "../../ui/tokens/contract.css";

export { subnav } from "../../ui/patterns/workspace.css";

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
  marginBottom: 0,
});

export const profileSections = style({
  display: "grid",
  gridTemplateColumns: "repeat(2, minmax(0, 1fr))",
  gap: tokens.space.xl,
  "@media": { "screen and (max-width: 1000px)": { gridTemplateColumns: "minmax(0, 1fr)" } },
});
export const profileSection = style({
  display: "flex",
  flexDirection: "column",
  gap: tokens.space.md,
  minWidth: 0,
});

export const identityGrid = style({
  display: "grid",
  gridTemplateColumns: "repeat(2, minmax(0, 1fr))",
  gap: tokens.space.md,
  alignItems: "start",
  "@media": { "screen and (max-width: 760px)": { gridTemplateColumns: "minmax(0, 1fr)" } },
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
  flexWrap: "wrap",
  gap: tokens.space.xs,
  alignItems: "center",
});

globalStyle(`.${nameForm} > input`, { flex: "1 1 120px", width: 0, minWidth: 0 });

export const attributeRow = style({
  display: "flex",
  alignItems: "center",
  gap: tokens.space.sm,
  padding: `${tokens.space.xs} 0`,
});

export const attributeName = style({
  flex: "0 0 90px",
  fontSize: tokens.fontSize.sm,
  fontWeight: 600,
  textTransform: "capitalize",
});

export const attributeSlider = style({
  flex: "1 1 auto",
  minWidth: 0,
  cursor: "pointer",
  accentColor: tokens.color.accent,
});

export const attributeInput = style({
  width: "60px",
  flexShrink: 0,
});

export const statGrid = style({
  display: "grid",
  gridTemplateColumns: "repeat(2, minmax(0, 1fr))",
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
  width: "100%",
  maxWidth: "280px",
  justifySelf: "center",
});

export const presetStage = style({
  display: "grid",
  gridTemplateColumns: "minmax(0, 1fr) minmax(0, 280px) minmax(0, 1fr)",
  alignItems: "center",
  gap: tokens.space.xl,
  padding: `${tokens.space.xl} 0`,
  overflow: "hidden",
  "@media": { "screen and (max-width: 760px)": { gridTemplateColumns: "minmax(0, 1fr)" } },
});

export const presetNeighbor = style({
  justifySelf: "center",
  width: "100%",
  maxWidth: "210px",
  height: "300px",
  objectFit: "cover",
  borderRadius: tokens.radius.md,
  opacity: 0.45,
  "@media": { "screen and (max-width: 760px)": { display: "none" } },
});

export const presetImage = style({
  height: "300px",
  width: "100%",
  objectFit: "contain",
  borderRadius: tokens.radius.sm,
});

export const presetImagePlaceholder = style({
  width: "100%",
  height: "300px",
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
  gridTemplateColumns: "repeat(5, minmax(0, 1fr))",
  gap: tokens.space.sm,
  "@media": {
    "screen and (max-width: 1100px)": { gridTemplateColumns: "repeat(3, minmax(0, 1fr))" },
    "screen and (max-width: 720px)": { gridTemplateColumns: "repeat(2, minmax(0, 1fr))" },
  },
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
  flexWrap: "wrap",
  gap: tokens.space.xs,
  marginTop: tokens.space.xs,
});
