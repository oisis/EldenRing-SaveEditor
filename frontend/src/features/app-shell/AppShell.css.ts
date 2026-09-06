import { style } from "@vanilla-extract/css";
import { tokens } from "../../ui/tokens/contract.css";

const sidebarWidth = "258px";
const topbarHeight = "48px";
const consoleHeight = "30px";

export const shell = style({
  display: "grid",
  gridTemplateColumns: `${sidebarWidth} minmax(0, 1fr)`,
  gridTemplateRows: `${topbarHeight} minmax(0, 1fr) ${consoleHeight}`,
  gridTemplateAreas: '"brand topbar" "sidebar workspace" "console console"',
  width: "100%",
  height: "100vh",
  minWidth: 0,
  minHeight: 0,
  overflow: "hidden",
  // The shell stays transparent so the themed application backdrop, which the
  // Elden Ring theme paints as layered gradients, is what the user sees.
  backgroundColor: "transparent",
  "@media": {
    "screen and (max-width: 900px)": {
      gridTemplateColumns: "220px minmax(0, 1fr)",
    },
    "screen and (max-width: 720px)": {
      gridTemplateColumns: "minmax(0, 1fr)",
      gridTemplateRows: `44px auto 180px minmax(0, 1fr) ${consoleHeight}`,
      gridTemplateAreas: '"brand" "topbar" "sidebar" "workspace" "console"',
    },
  },
});

export const brand = style({
  gridArea: "brand",
  display: "flex",
  alignItems: "center",
  gap: tokens.space.sm,
  minWidth: 0,
  paddingInline: tokens.space.lg,
  borderRight: `1px solid ${tokens.color.border}`,
  borderBottom: `1px solid ${tokens.color.border}`,
  backgroundColor: tokens.color.surfaceRaised,
});

export const brandLogo = style({
  width: "32px",
  height: "32px",
  flex: "none",
  borderRadius: tokens.radius.sm,
});

export const brandName = style({
  minWidth: 0,
  overflow: "hidden",
  textOverflow: "ellipsis",
  whiteSpace: "nowrap",
  fontSize: tokens.fontSize.lg,
  fontWeight: 700,
  letterSpacing: "0.02em",
});

export const topbar = style({
  gridArea: "topbar",
  display: "flex",
  alignItems: "center",
  gap: tokens.space.sm,
  minWidth: 0,
  paddingInline: tokens.space.lg,
  borderBottom: `1px solid ${tokens.color.border}`,
  backgroundColor: tokens.color.surfaceRaised,
  "@media": {
    "screen and (max-width: 720px)": {
      minHeight: topbarHeight,
      overflowX: "auto",
    },
    "screen and (max-width: 1100px)": {
      paddingInline: tokens.space.sm,
      gap: tokens.space.xs,
    },
  },
});

/** The mockup's hairline between the session toolbar and the theme selector. */
export const topbarSeparator = style({
  flex: "none",
  width: "1px",
  height: "20px",
  backgroundColor: tokens.color.border,
  "@media": { "screen and (max-width: 1100px)": { display: "none" } },
});

export const moduleNav = style({
  display: "flex",
  alignItems: "center",
  gap: tokens.space.xs,
  alignSelf: "stretch",
  minWidth: 0,
  overflowX: "auto",
  scrollbarWidth: "thin",
});

export const moduleTab = style({
  flex: "none",
  minWidth: 0,
  paddingInline: tokens.space.md,
  border: 0,
  borderBottom: "2px solid transparent",
  color: tokens.color.textMuted,
  backgroundColor: "transparent",
  font: "inherit",
  fontSize: tokens.fontSize.md,
  fontWeight: 500,
  cursor: "pointer",
  transitionProperty: "background-color, border-color, color",
  transitionDuration: tokens.motion.fast,
  selectors: {
    "&:hover": { color: tokens.color.text, backgroundColor: tokens.color.surfaceHover },
    '&[aria-current="page"]': {
      color: tokens.color.text,
      borderBottomColor: tokens.color.accent,
      fontWeight: 600,
    },
  },
  "@media": {
    "screen and (max-width: 1100px)": {
      paddingInline: tokens.space.sm,
      fontSize: tokens.fontSize.sm,
    },
  },
});

export const topbarSpacer = style({ flex: "1 1 auto" });

export const operations = style({
  display: "flex",
  gap: tokens.space.xs,
  flex: "none",
});

/** The Review Changes counter: figures stay aligned and a dirty save is flagged. */
export const changesCounter = style({
  fontVariantNumeric: "tabular-nums",
  selectors: {
    '&[data-dirty="true"]:not(:disabled)': {
      borderColor: tokens.color.warning,
      color: tokens.color.warning,
    },
  },
});

export const operationText = style({
  "@media": { "screen and (max-width: 1180px)": { display: "none" } },
});

export const operationGlyph = style({
  display: "none",
  fontFamily: tokens.font.mono,
  "@media": { "screen and (max-width: 1180px)": { display: "inline" } },
});

export const sidebar = style({
  gridArea: "sidebar",
  display: "flex",
  minWidth: 0,
  minHeight: 0,
  flexDirection: "column",
  overflow: "hidden",
  borderRight: `1px solid ${tokens.color.border}`,
  backgroundColor: tokens.color.surfaceRaised,
});

export const fileHeader = style({
  flex: "none",
  padding: `${tokens.space.md} ${tokens.space.lg}`,
  borderBottom: `1px solid ${tokens.color.border}`,
});

export const fileLine = style({
  display: "flex",
  alignItems: "center",
  justifyContent: "space-between",
  gap: tokens.space.sm,
  minWidth: 0,
});

export const fileName = style({
  minWidth: 0,
  overflow: "hidden",
  textOverflow: "ellipsis",
  whiteSpace: "nowrap",
  fontSize: tokens.fontSize.md,
  fontWeight: 600,
});

export const sidebarBody = style({
  minWidth: 0,
  minHeight: 0,
  flex: 1,
  overflowY: "auto",
  padding: `${tokens.space.md} ${tokens.space.sm} ${tokens.space.lg}`,
});

export const sidebarEmpty = style({
  margin: 0,
  paddingInline: tokens.space.sm,
  color: tokens.color.textMuted,
  fontSize: tokens.fontSize.sm,
});

export const workspace = style({
  gridArea: "workspace",
  minWidth: 0,
  minHeight: 0,
  overflow: "auto",
  padding: "20px 24px",
  "@media": { "screen and (max-width: 900px)": { padding: tokens.space.md } },
});

export const screen = style({
  display: "flex",
  minHeight: 0,
  flexDirection: "column",
  gap: tokens.space.md,
});

export { subnav as itemSubnav } from "../../ui/patterns/workspace.css";

export const placeholder = style({ minHeight: "180px", justifyContent: "center" });

export const consolePanel = style({
  position: "fixed",
  zIndex: 20,
  left: "50%",
  bottom: consoleHeight,
  transform: "translateX(-50%)",
  display: "flex",
  width: "min(1024px, calc(100vw - 32px))",
  height: "244px",
  minHeight: "150px",
  maxHeight: "60vh",
  resize: "vertical",
  flexDirection: "column",
  overflow: "hidden",
  border: `1px solid ${tokens.color.borderStrong}`,
  borderBottom: 0,
  backgroundColor: tokens.color.surfaceRaised,
  boxShadow: tokens.shadow.lg,
  selectors: { "&[hidden]": { display: "none" } },
});

export const consolePanelHeader = style({
  display: "flex",
  alignItems: "center",
  justifyContent: "space-between",
  gap: tokens.space.sm,
  flex: "none",
  height: "42px",
  padding: `6px ${tokens.space.md}`,
  borderBottom: `1px solid ${tokens.color.border}`,
  backgroundColor: tokens.color.surfaceSunken,
  fontSize: tokens.fontSize.md,
});

export const consolePanelBody = style({
  flex: 1,
  minHeight: 0,
  overflow: "auto",
  padding: `5px ${tokens.space.md}`,
  // The log surface reads as a recessed terminal, one step below the panel.
  backgroundColor: `color-mix(in srgb, ${tokens.color.background} 88%, black)`,
  fontFamily: tokens.font.mono,
  fontSize: tokens.fontSize.xs,
});

export const consoleBar = style({
  gridArea: "console",
  zIndex: 21,
  display: "flex",
  justifyContent: "center",
  minWidth: 0,
  borderTop: `1px solid ${tokens.color.borderStrong}`,
  backgroundColor: tokens.color.surfaceRaised,
});

/** Shows at a glance that the diagnostic stream is live. */
export const consoleIndicator = style({
  width: "7px",
  height: "7px",
  flex: "none",
  borderRadius: "50%",
  backgroundColor: tokens.color.accent,
  boxShadow: `0 0 8px ${tokens.color.accent}`,
});

export const consoleBarButton = style({
  display: "flex",
  alignItems: "center",
  gap: tokens.space.sm,
  width: "min(1024px, calc(100vw - 32px))",
  minWidth: 0,
  paddingInline: tokens.space.md,
  border: 0,
  color: tokens.color.text,
  backgroundColor: "transparent",
  font: "inherit",
  fontSize: tokens.fontSize.sm,
  textAlign: "left",
  cursor: "pointer",
  selectors: { "&:hover": { backgroundColor: tokens.color.surfaceHover } },
});

export const consoleLatest = style({
  flex: 1,
  minWidth: 0,
  overflow: "hidden",
  textOverflow: "ellipsis",
  whiteSpace: "nowrap",
  color: tokens.color.textFaint,
  fontFamily: tokens.font.mono,
  fontSize: tokens.fontSize.xs,
});

/** One console record per row: time, level and the backend's own message. */
export const consoleList = style({
  display: "flex",
  flexDirection: "column",
  margin: 0,
  padding: 0,
  listStyle: "none",
});

export const consoleRow = style({
  display: "grid",
  // The first column holds HH:mm:ssZ, so it is sized for that label rather
  // than for the full RFC 3339 timestamp the record still carries.
  gridTemplateColumns: "76px 52px minmax(0, 1fr)",
  gap: tokens.space.sm,
  alignItems: "baseline",
  minWidth: 0,
  padding: "3px 0",
  borderBottom: `1px solid color-mix(in srgb, ${tokens.color.border} 45%, transparent)`,
});

export const consoleTime = style({
  minWidth: 0,
  overflow: "hidden",
  textOverflow: "ellipsis",
  whiteSpace: "nowrap",
  color: tokens.color.textFaint,
});

/**
 * The severity keeps the backend's own vocabulary; only its colour changes, so
 * a level the backend adds later still renders with the neutral information
 * colour instead of disappearing.
 */
export const consoleLevel = style({
  minWidth: 0,
  overflow: "hidden",
  textOverflow: "ellipsis",
  whiteSpace: "nowrap",
  textTransform: "uppercase",
  fontWeight: 600,
  color: tokens.color.info,
  selectors: {
    '&[data-level="debug"]': { color: tokens.color.textFaint },
    '&[data-level="warning"]': { color: tokens.color.warning },
    '&[data-level="error"]': { color: tokens.color.danger },
  },
});

export const consoleFilter = style({
  display: "flex",
  alignItems: "center",
  gap: tokens.space.sm,
  fontSize: tokens.fontSize.sm,
  color: tokens.color.textMuted,
});

export const consoleEmpty = style({
  padding: "20px",
  color: tokens.color.textFaint,
  textAlign: "center",
});
