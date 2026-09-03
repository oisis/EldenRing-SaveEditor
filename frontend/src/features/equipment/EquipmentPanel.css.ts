import { style } from "@vanilla-extract/css";
import { tokens } from "../../ui/tokens/contract.css";

/**
 * Equipment layout only. Shared panel patterns live in ui/patterns.
 *
 * The screen is one composition rather than a row of independent tiles: the
 * groups share a single fluid grid, so they reflow together instead of each
 * keeping a fixed width of its own.
 */

export const board = style({
  display: "grid",
  gridTemplateColumns: "repeat(12, minmax(0, 1fr))",
  gap: tokens.space.md,
  minHeight: 0,
  "@media": {
    "screen and (max-width: 1100px)": { gridTemplateColumns: "repeat(6, minmax(0, 1fr))" },
    "screen and (max-width: 720px)": { gridTemplateColumns: "repeat(2, minmax(0, 1fr))" },
  },
});

/** A group occupying one third, one half or the full width of the board. */
export const groupThird = style({
  gridColumn: "span 4",
  minWidth: 0,
  "@media": {
    "screen and (max-width: 1100px)": { gridColumn: "span 3" },
    "screen and (max-width: 720px)": { gridColumn: "span 2" },
  },
});

export const groupHalf = style({
  gridColumn: "span 6",
  minWidth: 0,
  "@media": {
    "screen and (max-width: 1100px)": { gridColumn: "span 6" },
    "screen and (max-width: 720px)": { gridColumn: "span 2" },
  },
});

export const groupFull = style({ gridColumn: "1 / -1", minWidth: 0 });

export const group = style({
  display: "flex",
  flexDirection: "column",
  gap: tokens.space.sm,
  padding: tokens.space.md,
  border: `1px solid ${tokens.color.border}`,
  borderRadius: tokens.radius.md,
  background: tokens.color.surfaceRaised,
});

export const groupHeader = style({
  display: "flex",
  flexWrap: "wrap",
  alignItems: "center",
  gap: tokens.space.sm,
});

export const groupTitle = style({
  margin: 0,
  fontSize: tokens.fontSize.md,
  fontWeight: 700,
});

/** A run of slots that wraps instead of scrolling. */
export const slotRow = style({
  display: "flex",
  flexWrap: "wrap",
  gap: tokens.space.sm,
});

/** The square field of one slot, used for every group. */
export const slot = style({
  width: "104px",
  height: "104px",
  flexDirection: "column",
  alignItems: "center",
  justifyContent: "center",
  gap: tokens.space.xs,
  padding: tokens.space.xs,
  textAlign: "center",
});

export const slotIcon = style({ width: "36px", height: "36px", objectFit: "contain" });

export const slotIconPlaceholder = style({
  width: "36px",
  height: "36px",
  borderRadius: tokens.radius.sm,
  background: tokens.color.surfaceHover,
});

export const slotName = style({
  display: "-webkit-box",
  overflow: "hidden",
  width: "100%",
  color: tokens.color.text,
  fontSize: tokens.fontSize.sm,
  lineHeight: 1.2,
  WebkitBoxOrient: "vertical",
  WebkitLineClamp: 2,
});

export const slotMeta = style({
  color: tokens.color.textMuted,
  fontSize: tokens.fontSize.sm,
});

/**
 * The six Quick Pouch fields. The confirmed 1.x view lays them out in two
 * columns, filled row by row: up and right, then left and down, then the two
 * ordinary fields.
 */
export const pouchPad = style({
  display: "grid",
  gridTemplateColumns: "repeat(2, minmax(0, 1fr))",
  gap: tokens.space.sm,
  maxWidth: "240px",
});

/**
 * The first ten spell positions, as in the confirmed 1.x view: two columns of
 * five, filled column by column, so position 1 to 5 form the left column.
 */
export const spellPrimaryGrid = style({
  display: "grid",
  gridAutoFlow: "column",
  gridTemplateColumns: "repeat(2, minmax(0, 1fr))",
  gridTemplateRows: "repeat(5, auto)",
  gap: tokens.space.sm,
  maxWidth: "240px",
});

/** Spell positions 11 and 12, in their own two-column row below the first ten. */
export const spellExtraGrid = style({
  display: "grid",
  gridTemplateColumns: "repeat(2, minmax(0, 1fr))",
  gap: tokens.space.sm,
  maxWidth: "240px",
});

/** One picker result row. */
export const candidateList = style({
  display: "flex",
  flexDirection: "column",
  gap: tokens.space.xs,
  margin: 0,
  padding: 0,
  listStyle: "none",
});

export const candidate = style({
  display: "flex",
  alignItems: "center",
  justifyContent: "flex-start",
  gap: tokens.space.sm,
  width: "100%",
  textAlign: "left",
});

export const candidateName = style({
  overflow: "hidden",
  flex: "1 1 auto",
  color: tokens.color.text,
  textOverflow: "ellipsis",
  whiteSpace: "nowrap",
});

export const pickerToolbar = style({
  display: "flex",
  flexWrap: "wrap",
  alignItems: "center",
  gap: tokens.space.sm,
});

export const pickerSearch = style({ flex: "1 1 200px" });

export const pagination = style({
  display: "flex",
  alignItems: "center",
  justifyContent: "center",
  gap: tokens.space.sm,
});
