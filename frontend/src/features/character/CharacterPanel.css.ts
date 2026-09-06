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
  padding: 0,
  gap: 0,
  overflow: "hidden",
});

export const cardHeader = style({
  display: "flex",
  alignItems: "center",
  justifyContent: "space-between",
  flexWrap: "wrap",
  gap: tokens.space.sm,
  padding: `${tokens.space.md} ${tokens.space.lg}`,
  borderBottom: `1px solid ${tokens.color.border}`,
});

export const cardTitle = style({
  margin: 0,
  fontSize: tokens.fontSize.md,
  fontWeight: 600,
  letterSpacing: "0.01em",
});

export const cardBadges = style({
  display: "flex",
  alignItems: "center",
  gap: tokens.space.sm,
  flexWrap: "wrap",
});

export const cardBody = style({
  padding: tokens.space.lg,
  display: "flex",
  flexDirection: "column",
  gap: tokens.space.md,
});

export const profileSections = style({
  display: "grid",
  gridTemplateColumns: "repeat(2, minmax(0, 1fr))",
  gap: tokens.space.xl,
  "@media": {
    "screen and (max-width: 1000px)": {
      gridTemplateColumns: "minmax(0, 1fr)",
      gap: tokens.space.lg,
    },
  },
});

export const profileSection = style({
  display: "flex",
  flexDirection: "column",
  gap: tokens.space.md,
  minWidth: 0,
  selectors: {
    "& + &": {
      borderLeft: `1px solid ${tokens.color.border}`,
      paddingLeft: tokens.space.xl,
    },
  },
  "@media": {
    "screen and (max-width: 1000px)": {
      selectors: {
        "& + &": {
          borderLeft: "none",
          paddingLeft: 0,
          borderTop: `1px solid ${tokens.color.border}`,
          paddingTop: tokens.space.md,
        },
      },
    },
  },
});

export const profileSectionTitle = style({
  fontSize: tokens.fontSize.xs,
  fontWeight: 700,
  letterSpacing: "0.08em",
  textTransform: "uppercase",
  color: tokens.color.textMuted,
  margin: 0,
});

export const profileGrid = style({
  display: "grid",
  gridTemplateColumns: "repeat(2, minmax(0, 1fr))",
  gap: `${tokens.space.sm} ${tokens.space.md}`,
  alignItems: "start",
  "@media": {
    "screen and (max-width: 600px)": {
      gridTemplateColumns: "minmax(0, 1fr)",
    },
  },
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

export const field = style({
  display: "flex",
  flexDirection: "column",
  gap: "3px",
  alignSelf: "start",
  minWidth: 0,
});

export const fieldLabelRow = style({
  display: "flex",
  alignItems: "baseline",
  justifyContent: "space-between",
  gap: tokens.space.xs,
});

export const fieldLabel = style({
  fontSize: tokens.fontSize.sm,
  fontWeight: 500,
  color: tokens.color.textMuted,
  lineHeight: 1.35,
});

export const fieldLabelSuffix = style({
  fontFamily: tokens.font.mono,
  fontSize: tokens.fontSize.xs,
  color: tokens.color.textFaint,
});

export const fieldRow = style({
  display: "flex",
  alignItems: "center",
  gap: tokens.space.xs,
});

globalStyle(`.${fieldRow} > input`, {
  flex: "1 1 auto",
  width: 0,
  minWidth: 0,
});

export const nameForm = style({
  display: "flex",
  flexWrap: "wrap",
  gap: tokens.space.xs,
  alignItems: "center",
});

globalStyle(`.${nameForm} > input`, { flex: "1 1 120px", width: 0, minWidth: 0 });

export const fieldHint = style({
  fontSize: tokens.fontSize.xs,
  color: tokens.color.textMuted,
  lineHeight: 1.4,
});

export const fieldWarning = style({
  border: `1px solid ${tokens.color.warning}`,
  borderRadius: tokens.radius.sm,
  padding: "3px 6px",
  backgroundColor: tokens.color.warningSurface,
  color: tokens.color.warning,
  fontSize: tokens.fontSize.xs,
  lineHeight: 1.35,
  display: "flex",
  alignItems: "center",
  gap: "4px",
});

export const fieldDanger = style({
  border: `1px solid ${tokens.color.danger}`,
  borderRadius: tokens.radius.sm,
  padding: "3px 6px",
  backgroundColor: tokens.color.dangerSurface,
  color: tokens.color.danger,
  fontSize: tokens.fontSize.xs,
  lineHeight: 1.35,
  display: "flex",
  alignItems: "center",
  gap: "4px",
});

export const profileAdvanced = style({
  marginTop: tokens.space.md,
});

export const addSettingsAccordion = style({
  border: `1px solid ${tokens.color.accent}`,
  borderRadius: tokens.radius.md,
  backgroundColor: tokens.color.surface,
  overflow: "hidden",
  minWidth: 0,
  boxShadow: `inset 3px 0 0 ${tokens.color.accent}`,
});

export const addSettingsHeading = style({
  display: "flex",
  alignItems: "center",
  justifyContent: "space-between",
  padding: `${tokens.space.sm} ${tokens.space.md}`,
  cursor: "pointer",
  backgroundColor: tokens.color.surfaceRaised,
  minHeight: "40px",
  userSelect: "none",
});

export const addSettingsTitleGroup = style({
  display: "flex",
  alignItems: "center",
  gap: tokens.space.sm,
});

export const addSettingsTitle = style({
  margin: 0,
  fontSize: tokens.fontSize.sm,
  fontWeight: 600,
});

export const addSettingsBody = style({
  display: "flex",
  flexDirection: "column",
  gap: tokens.space.md,
  padding: tokens.space.md,
  borderTop: `1px solid ${tokens.color.border}`,
});

export const addSettingsGrid = style({
  display: "grid",
  gridTemplateColumns: "repeat(2, minmax(0, 1fr))",
  gap: tokens.space.md,
  "@media": {
    "screen and (max-width: 760px)": {
      gridTemplateColumns: "minmax(0, 1fr)",
    },
  },
});

export const addSettingsSwitches = style({
  display: "grid",
  gridTemplateColumns: "repeat(3, minmax(0, 1fr))",
  gap: tokens.space.md,
  paddingTop: tokens.space.sm,
  borderTop: `1px solid ${tokens.color.border}`,
  alignItems: "start",
  "@media": {
    "screen and (max-width: 900px)": {
      gridTemplateColumns: "minmax(0, 1fr)",
    },
  },
});

export const switchLabel = style({
  display: "flex",
  alignItems: "flex-start",
  gap: tokens.space.xs,
  cursor: "pointer",
  fontSize: tokens.fontSize.sm,
  color: tokens.color.text,
  userSelect: "none",
});

export const switchText = style({
  display: "flex",
  flexDirection: "column",
  gap: "2px",
});

export const profileLower = style({
  display: "grid",
  gridTemplateColumns: "repeat(2, minmax(0, 1fr))",
  gap: tokens.space.md,
  marginTop: tokens.space.md,
  alignItems: "start",
  "@media": {
    "screen and (max-width: 1000px)": {
      gridTemplateColumns: "minmax(0, 1fr)",
    },
  },
});

export const profilePanel = style({
  minWidth: 0,
});

export const summaryContent = style({
  display: "flex",
  alignItems: "center",
  justifyContent: "space-between",
  width: "100%",
  gap: tokens.space.sm,
});

export const summaryTitle = style({
  margin: 0,
  fontSize: tokens.fontSize.md,
  fontWeight: 600,
});

export const summaryMeta = style({
  fontFamily: tokens.font.mono,
  fontSize: tokens.fontSize.xs,
  color: tokens.color.textFaint,
  overflow: "hidden",
  textOverflow: "ellipsis",
  whiteSpace: "nowrap",
});

export const rangeControl = style({
  display: "grid",
  gridTemplateColumns: "104px minmax(0, 1fr) 64px",
  alignItems: "center",
  gap: tokens.space.sm,
  minHeight: "30px",
  padding: "2px 0",
});

export const rangeLabel = style({
  fontSize: tokens.fontSize.sm,
  fontWeight: 500,
  color: tokens.color.textMuted,
  overflow: "hidden",
  textOverflow: "ellipsis",
  whiteSpace: "nowrap",
});

export const rangeSlider = style({
  width: "100%",
  minWidth: 0,
  cursor: "pointer",
  accentColor: tokens.color.accent,
});

export const rangeNumberInput = style({
  selectors: {
    "&&": {
      width: "64px",
      flex: "0 0 64px",
      textAlign: "right",
      fontFamily: tokens.font.mono,
    },
  },
});

export const derivedGroup = style({
  display: "flex",
  flexDirection: "column",
  gap: tokens.space.xs,
  selectors: {
    "& + &": {
      marginTop: tokens.space.sm,
    },
  },
});

export const derivedGroupTitle = style({
  margin: "0 0 4px",
  fontSize: tokens.fontSize.xs,
  fontWeight: 700,
  letterSpacing: "0.08em",
  textTransform: "uppercase",
  color: tokens.color.textMuted,
});

export const derivedGrid = style({
  display: "grid",
  gridTemplateColumns: "repeat(auto-fit, minmax(130px, 1fr))",
  gap: tokens.space.xs,
});

export const derivedStat = style({
  display: "flex",
  alignItems: "baseline",
  justifyContent: "space-between",
  gap: tokens.space.xs,
  minHeight: "30px",
  padding: "4px 8px",
  border: `1px solid ${tokens.color.border}`,
  borderRadius: tokens.radius.sm,
  backgroundColor: tokens.color.surfaceRaised,
});

export const derivedStatLabel = style({
  color: tokens.color.textMuted,
  fontSize: tokens.fontSize.xs,
  overflow: "hidden",
  textOverflow: "ellipsis",
  whiteSpace: "nowrap",
});

export const derivedStatValue = style({
  fontFamily: tokens.font.mono,
  fontSize: tokens.fontSize.sm,
  fontWeight: 500,
  color: tokens.color.accentText,
  whiteSpace: "nowrap",
});

export const derivedStatSub = style({
  fontSize: tokens.fontSize.xs,
  color: tokens.color.textFaint,
});

export const profileNote = style({
  marginTop: tokens.space.md,
  fontSize: tokens.fontSize.xs,
  color: tokens.color.textMuted,
});

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
