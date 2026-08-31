import { style } from "@vanilla-extract/css";
import { tokens } from "../../ui/tokens/contract.css";

const sidebarWidth = "258px";
const topbarHeight = "52px";
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
  backgroundColor: tokens.color.background,
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
  paddingInline: tokens.space.md,
  borderBottom: `1px solid ${tokens.color.border}`,
  backgroundColor: tokens.color.surfaceRaised,
  "@media": {
    "screen and (max-width: 720px)": {
      minHeight: topbarHeight,
      overflowX: "auto",
    },
  },
});

export const moduleNav = style({
  display: "flex",
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
  fontWeight: 500,
  cursor: "pointer",
  transitionProperty: "background-color, border-color, color",
  transitionDuration: tokens.motion.fast,
  selectors: {
    "&:hover": { color: tokens.color.text, backgroundColor: tokens.color.surfaceHover },
    '&[aria-current="page"]': {
      color: tokens.color.text,
      borderBottomColor: tokens.color.accent,
      fontWeight: 700,
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
  padding: tokens.space.md,
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
  fontWeight: 700,
});

export const sidebarBody = style({
  minWidth: 0,
  minHeight: 0,
  flex: 1,
  overflowY: "auto",
  padding: tokens.space.md,
});

export const sidebarEmpty = style({ margin: 0, color: tokens.color.textMuted });

export const workspace = style({
  gridArea: "workspace",
  minWidth: 0,
  minHeight: 0,
  overflow: "auto",
  padding: tokens.space.lg,
  "@media": { "screen and (max-width: 900px)": { padding: tokens.space.md } },
});

export const screen = style({
  display: "flex",
  minHeight: 0,
  flexDirection: "column",
  gap: tokens.space.md,
});

export const itemSubnav = style({ display: "flex", gap: tokens.space.xs });

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
  maxHeight: "60vh",
  flexDirection: "column",
  overflow: "hidden",
  border: `1px solid ${tokens.color.borderStrong}`,
  backgroundColor: tokens.color.surfaceRaised,
  selectors: { "&[hidden]": { display: "none" } },
});

export const consolePanelHeader = style({
  display: "flex",
  alignItems: "center",
  justifyContent: "space-between",
  gap: tokens.space.sm,
  padding: tokens.space.sm,
  borderBottom: `1px solid ${tokens.color.border}`,
});

export const consolePanelBody = style({
  flex: 1,
  minHeight: 0,
  overflow: "auto",
  padding: tokens.space.md,
});

export const consoleBar = style({
  gridArea: "console",
  zIndex: 21,
  display: "flex",
  justifyContent: "center",
  minWidth: 0,
  borderTop: `1px solid ${tokens.color.border}`,
  backgroundColor: tokens.color.surfaceRaised,
});

export const consoleBarButton = style({
  display: "flex",
  alignItems: "center",
  gap: tokens.space.md,
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
  minWidth: 0,
  overflow: "hidden",
  textOverflow: "ellipsis",
  whiteSpace: "nowrap",
  color: tokens.color.textMuted,
});
