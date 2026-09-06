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

export const presetBrowser = style({
  border: `1px solid ${tokens.color.border}`,
  borderRadius: tokens.radius.md,
  backgroundColor: tokens.color.surface,
  overflow: "hidden",
});

export const presetBrowserHead = style({
  display: "flex",
  alignItems: "center",
  gap: tokens.space.md,
  flexWrap: "wrap",
  padding: `${tokens.space.sm} ${tokens.space.md}`,
  borderBottom: `1px solid ${tokens.color.border}`,
  backgroundColor: tokens.color.surfaceSunken,
});

export const presetFilterSwitches = style({
  display: "flex",
  alignItems: "center",
  gap: tokens.space.md,
  flexWrap: "wrap",
});

export const presetSpacer = style({
  flex: 1,
});

export const presetBadges = style({
  display: "flex",
  alignItems: "center",
  gap: tokens.space.sm,
});

export const presetCarousel = style({
  position: "relative",
  minHeight: "430px",
  overflow: "hidden",
  background: `radial-gradient(ellipse at 50% 88%, color-mix(in srgb, ${tokens.color.accent} 14%, transparent), transparent 48%), ${tokens.color.surfaceRaised}`,
  perspective: "1100px",
  userSelect: "none",
  display: "flex",
  alignItems: "center",
  justifyContent: "center",
});

export const presetRing = style({
  position: "absolute",
  left: "50%",
  bottom: "8px",
  width: "72%",
  height: "92px",
  transform: "translateX(-50%)",
  border: `1px solid color-mix(in srgb, ${tokens.color.accent} 42%, ${tokens.color.border})`,
  borderRadius: "50%",
  boxShadow: `0 0 28px color-mix(in srgb, ${tokens.color.accent} 18%, transparent), inset 0 0 22px color-mix(in srgb, ${tokens.color.accent} 12%, transparent)`,
  pointerEvents: "none",
});

export const presetStage = style({
  position: "absolute",
  inset: "12px 64px 22px",
  transformStyle: "preserve-3d",
});

export const presetCard = style({
  position: "absolute",
  left: "50%",
  top: "14px",
  width: "246px",
  height: "356px",
  marginLeft: "-123px",
  display: "flex",
  flexDirection: "column",
  overflow: "hidden",
  border: `1px solid ${tokens.color.border}`,
  borderRadius: tokens.radius.md,
  backgroundColor: tokens.color.surface,
  opacity: "var(--card-opacity, 1)",
  zIndex: "var(--card-z, 1)",
  transform:
    "translateX(calc(var(--offset, 0) * 250px)) translateZ(calc(var(--distance, 0) * -90px)) rotateY(calc(var(--offset, 0) * -12deg)) scale(calc(1 - var(--distance, 0) * 0.06))",
  transition: "transform 0.28s ease, opacity 0.28s ease, border-color 0.2s",
  pointerEvents: "none",
  boxShadow: tokens.shadow.sm,
});

export const presetCardActive = style({
  borderColor: tokens.color.accent,
  boxShadow: `0 10px 35px rgba(0,0,0,0.45), 0 0 0 1px ${tokens.color.accent}`,
  pointerEvents: "auto",
});

export const presetCardPortrait = style({
  position: "relative",
  height: "222px",
  border: 0,
  borderBottom: `1px solid ${tokens.color.border}`,
  backgroundColor: tokens.color.surfaceSunken,
  cursor: "pointer",
  overflow: "hidden",
  padding: 0,
  display: "flex",
  alignItems: "center",
  justifyContent: "center",
  pointerEvents: "auto",
});

export const presetCardImage = style({
  width: "100%",
  height: "100%",
  objectFit: "cover",
});

export const presetCardPlaceholder = style({
  display: "flex",
  alignItems: "center",
  justifyContent: "center",
  width: "100%",
  height: "100%",
  color: tokens.color.textMuted,
  fontSize: tokens.fontSize.xs,
});

export const presetCardShade = style({
  position: "absolute",
  inset: "auto 0 0",
  height: "40%",
  background: "linear-gradient(transparent, rgba(0,0,0,0.35))",
  pointerEvents: "none",
});

export const presetCardFavorite = style({
  position: "absolute",
  zIndex: 3,
  top: "8px",
  right: "8px",
  width: "32px",
  height: "32px",
  borderRadius: "50%",
  border: `1px solid ${tokens.color.borderStrong}`,
  backgroundColor: tokens.color.surfaceRaised,
  color: tokens.color.accentText,
  cursor: "pointer",
  fontSize: "18px",
  display: "grid",
  placeItems: "center",
  padding: 0,
  pointerEvents: "auto",
  lineHeight: 1,
});

export const presetCardBody = style({
  display: "flex",
  flex: 1,
  flexDirection: "column",
  gap: "3px",
  padding: tokens.space.sm,
});

export const presetName = style({
  fontWeight: 600,
  fontSize: tokens.fontSize.sm,
  color: tokens.color.text,
  whiteSpace: "nowrap",
  overflow: "hidden",
  textOverflow: "ellipsis",
});

export const presetMeta = style({
  fontSize: tokens.fontSize.xs,
  color: tokens.color.textMuted,
  whiteSpace: "nowrap",
  overflow: "hidden",
  textOverflow: "ellipsis",
});

export const presetTagsRow = style({
  display: "flex",
  flexWrap: "wrap",
  gap: "4px",
  margin: "2px 0",
});

export const presetCardActions = style({
  display: "flex",
  alignItems: "center",
  justifyContent: "space-between",
  marginTop: "auto",
  gap: tokens.space.xs,
});

export const presetArrow = style({
  position: "absolute",
  zIndex: 40,
  top: "46%",
  width: "42px",
  height: "58px",
  border: `1px solid ${tokens.color.borderStrong}`,
  borderRadius: tokens.radius.sm,
  backgroundColor: tokens.color.surfaceRaised,
  color: tokens.color.text,
  cursor: "pointer",
  display: "grid",
  placeItems: "center",
  fontSize: tokens.fontSize.lg,
  ":hover": {
    color: tokens.color.accentText,
    borderColor: tokens.color.accent,
  },
  ":disabled": {
    opacity: 0.4,
    cursor: "not-allowed",
  },
});

export const presetArrowPrev = style({
  left: "14px",
});

export const presetArrowNext = style({
  right: "14px",
});

export const presetRange = style({
  display: "flex",
  alignItems: "center",
  gap: tokens.space.md,
  width: "min(560px, 100%)",
  margin: "0 auto",
  padding: `${tokens.space.sm} ${tokens.space.md}`,
  borderTop: `1px solid ${tokens.color.border}`,
});

export const presetRangeInput = style({
  flex: 1,
  accentColor: tokens.color.accent,
  cursor: "pointer",
});

export const presetRangeLabel = style({
  fontFamily: tokens.font.mono,
  fontSize: tokens.fontSize.xs,
  color: tokens.color.textMuted,
  whiteSpace: "nowrap",
});

export const mirrorFavoritesSection = style({
  marginTop: tokens.space.lg,
});

export const mirrorFavoritesHead = style({
  display: "flex",
  alignItems: "center",
  justifyContent: "space-between",
  marginBottom: tokens.space.sm,
  flexWrap: "wrap",
  gap: tokens.space.xs,
});

export const mirrorFavoritesList = style({
  display: "grid",
  gridTemplateColumns: "repeat(auto-fill, minmax(280px, 1fr))",
  gap: tokens.space.sm,
});

export const mirrorFavoriteCard = style({
  display: "grid",
  gridTemplateColumns: "48px minmax(0, 1fr) auto",
  alignItems: "center",
  gap: tokens.space.sm,
  padding: tokens.space.sm,
  border: `1px solid ${tokens.color.border}`,
  borderRadius: tokens.radius.sm,
  backgroundColor: tokens.color.surface,
});

export const mirrorFavoritePortrait = style({
  width: "48px",
  height: "48px",
  display: "grid",
  placeItems: "center",
  overflow: "hidden",
  border: `1px solid ${tokens.color.border}`,
  borderRadius: "50%",
  backgroundColor: tokens.color.surfaceSunken,
  fontSize: tokens.fontSize.xs,
  color: tokens.color.textMuted,
});

export const mirrorFavoriteDetails = style({
  display: "flex",
  flexDirection: "column",
  gap: "2px",
  minWidth: 0,
});

export const mirrorFavoriteTitle = style({
  fontSize: tokens.fontSize.sm,
  fontWeight: 600,
  color: tokens.color.text,
  whiteSpace: "nowrap",
  overflow: "hidden",
  textOverflow: "ellipsis",
});

export const mirrorFavoriteSlot = style({
  fontSize: tokens.fontSize.xs,
  color: tokens.color.textFaint,
});

export const mirrorFavoriteActions = style({
  display: "flex",
  alignItems: "center",
  gap: tokens.space.xs,
});

export const presetPreviewModal = style({
  display: "grid",
  gridTemplateColumns: "minmax(240px, 0.85fr) minmax(280px, 1.15fr)",
  gap: tokens.space.lg,
  alignItems: "start",
  "@media": {
    "screen and (max-width: 700px)": {
      gridTemplateColumns: "1fr",
    },
  },
});

export const presetPreviewPortrait = style({
  minHeight: "340px",
  display: "grid",
  placeItems: "center",
  border: `1px solid ${tokens.color.border}`,
  borderRadius: tokens.radius.md,
  overflow: "hidden",
  backgroundColor: tokens.color.surfaceSunken,
});

export const presetPreviewImage = style({
  width: "100%",
  maxHeight: "340px",
  objectFit: "contain",
});

export const presetPreviewInfo = style({
  display: "flex",
  flexDirection: "column",
  gap: tokens.space.md,
});

// Backward-compatible exports for any references
export const presetViewer = presetCard;
export const presetNeighbor = presetCard;
export const presetImage = presetCardImage;
export const presetImagePlaceholder = presetCardPlaceholder;
export const presetControls = presetCardActions;
export const presetTags = presetTagsRow;
export const favoritesGrid = mirrorFavoritesList;
export const favoriteSlotCard = mirrorFavoriteCard;
export const favoriteSlotHeader = mirrorFavoriteDetails;
export const favoriteSlotActions = mirrorFavoriteActions;
