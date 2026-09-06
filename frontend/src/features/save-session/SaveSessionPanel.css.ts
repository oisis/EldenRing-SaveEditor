import { style } from "@vanilla-extract/css";
import { tokens } from "../../ui/tokens/contract.css";

export const layout = style({
  display: "grid",
  gridTemplateColumns: "minmax(0, 1fr) minmax(0, 260px)",
  gap: tokens.space.lg,
  alignItems: "start",
  "@media": { "screen and (max-width: 720px)": { gridTemplateColumns: "1fr" } },
});

export const layoutSingle = style({ minWidth: 0 });

export const homeGrid = style({
  display: "grid",
  gridTemplateColumns: "repeat(2, minmax(0, 1fr))",
  gap: tokens.space.lg,
  alignItems: "start",
  minWidth: 0,
  "@media": { "screen and (max-width: 1000px)": { gridTemplateColumns: "minmax(0, 1fr)" } },
});

export const homeCard = style({ minWidth: 0, padding: 0, gap: 0, overflow: "hidden" });
export const cardHeader = style({
  display: "flex",
  alignItems: "center",
  justifyContent: "space-between",
  flexWrap: "wrap",
  gap: tokens.space.sm,
  padding: `${tokens.space.md} ${tokens.space.lg}`,
  borderBottom: `1px solid ${tokens.color.border}`,
});
export const cardTitle = style({ margin: 0, fontSize: tokens.fontSize.md, fontWeight: 600 });
export const cardBody = style({ padding: tokens.space.lg, margin: 0 });
/**
 * The mockup's `.kv` list: one grid for the whole list, so every label column
 * lines up on its own longest label instead of on a fixed width.
 */
export const facts = style({
  display: "grid",
  gridTemplateColumns: "auto minmax(0, 1fr)",
  rowGap: "3px",
  columnGap: tokens.space.lg,
  alignItems: "baseline",
  margin: 0,
});
/** The row wrapper exists for readability; the grid above owns the columns. */
export const fact = style({ display: "contents" });
export const factValue = style({
  margin: 0,
  overflowWrap: "anywhere",
  fontFamily: tokens.font.mono,
  fontSize: tokens.fontSize.sm,
});
export const summaryTiles = style({
  display: "grid",
  gridTemplateColumns: "repeat(2, minmax(0, 1fr))",
  gap: tokens.space.md,
});
export const summaryTile = style({
  display: "flex",
  flexDirection: "column",
  gap: tokens.space.xs,
  padding: tokens.space.md,
  border: `1px solid ${tokens.color.border}`,
  borderRadius: tokens.radius.sm,
  backgroundColor: tokens.color.surfaceSunken,
});
export const summaryLabel = style({
  color: tokens.color.textMuted,
  fontSize: tokens.fontSize.xs,
  textTransform: "uppercase",
  letterSpacing: "0.06em",
});
export const summaryValue = style({
  fontSize: tokens.fontSize.xl,
  fontWeight: 650,
  fontVariantNumeric: "tabular-nums",
});
export const recentList = style({ listStyle: "none", margin: 0, padding: 0 });
export const recentRow = style({
  display: "flex",
  alignItems: "center",
  gap: tokens.space.md,
  paddingRight: tokens.space.md,
  borderBottom: `1px solid ${tokens.color.border}`,
  selectors: { "&:last-child": { borderBottom: 0 } },
});
export const recentOpen = style({
  display: "flex",
  alignItems: "baseline",
  flexWrap: "wrap",
  gap: tokens.space.md,
  flex: 1,
  minWidth: 0,
  padding: `7px ${tokens.space.lg}`,
  border: 0,
  background: "transparent",
  color: tokens.color.text,
  font: "inherit",
  textAlign: "left",
  cursor: "pointer",
  selectors: {
    "&:hover:not(:disabled)": { backgroundColor: tokens.color.surfaceHover },
    "&:disabled": { opacity: 0.5, cursor: "not-allowed" },
  },
});
export const recentName = style({ fontWeight: 500, overflowWrap: "anywhere" });
export const recentPath = style({
  flex: "1 1 160px",
  minWidth: 0,
  overflow: "hidden",
  textOverflow: "ellipsis",
  whiteSpace: "nowrap",
  color: tokens.color.textFaint,
  fontFamily: tokens.font.mono,
  fontSize: tokens.fontSize.xs,
});
export const recentTime = style({
  marginLeft: "auto",
  color: tokens.color.textFaint,
  fontSize: tokens.fontSize.xs,
});

export const stack = style({
  display: "flex",
  minWidth: 0,
  flexDirection: "column",
  gap: tokens.space.md,
});

export const actions = style({
  display: "flex",
  flexWrap: "wrap",
  gap: tokens.space.sm,
});

/**
 * The standing warnings banner. It is deliberately not the danger panel: a save
 * with warnings is editable, and only a blocked one gets the danger treatment.
 */
export const warningBanner = style({
  display: "flex",
  flexWrap: "wrap",
  alignItems: "center",
  gap: tokens.space.sm,
  margin: 0,
  padding: tokens.space.md,
  borderRadius: tokens.radius.sm,
  border: `1px solid ${tokens.color.accent}`,
  backgroundColor: tokens.color.surfaceRaised,
  color: tokens.color.text,
});

export const reportList = style({
  display: "flex",
  flexDirection: "column",
  gap: tokens.space.xs,
  margin: 0,
  paddingLeft: tokens.space.lg,
});

export const reportItem = style({ overflowWrap: "anywhere" });

export const reportScope = style({
  color: tokens.color.textMuted,
  fontFamily: tokens.font.mono,
  fontSize: tokens.fontSize.sm,
});
