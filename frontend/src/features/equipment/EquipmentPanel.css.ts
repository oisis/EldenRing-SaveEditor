import { globalStyle, style } from "@vanilla-extract/css";
import { tokens } from "../../ui/tokens/contract.css";

/** One fluid composition: equipment at left, pouch/physick and spells at right. */
export const board = style({
  display: "grid",
  gridTemplateColumns: "repeat(9, minmax(0, 1fr))",
  gridTemplateAreas:
    '"right right right ammo ammo pouch pouch spells spells" "left left left ammo ammo pouch pouch spells spells" "armor armor armor armor . pouch pouch spells spells" "tal tal tal tal . physick physick spells spells" "quick quick quick quick quick physick physick spells spells"',
  gap: tokens.space.md,
  alignItems: "start",
  minWidth: 0,
  "@media": {
    "screen and (max-width: 950px)": {
      gridTemplateColumns: "repeat(4, minmax(0, 1fr))",
      gridTemplateAreas:
        '"right right left left" "ammo ammo armor armor" "tal tal quick quick" "pouch pouch spells spells" "physick physick spells spells"',
    },
  },
});

export const rightGroup = style({ gridArea: "right" });
export const leftGroup = style({ gridArea: "left" });
export const ammoGroup = style({ gridArea: "ammo" });
export const armorGroup = style({ gridArea: "armor" });
export const talismanGroup = style({ gridArea: "tal" });
export const quickGroup = style({ gridArea: "quick" });
export const pouchGroup = style({ gridArea: "pouch" });
export const physickGroup = style({ gridArea: "physick" });
export const spellGroup = style({ gridArea: "spells" });

export const group = style({
  display: "flex",
  flexDirection: "column",
  gap: tokens.space.sm,
  minWidth: 0,
});
export const groupHeader = style({
  display: "flex",
  flexWrap: "wrap",
  alignItems: "center",
  gap: tokens.space.xs,
});
export const groupTitle = style({
  margin: 0,
  fontSize: tokens.fontSize.sm,
  fontWeight: 600,
  color: tokens.color.textMuted,
  textTransform: "uppercase",
  letterSpacing: "0.06em",
});
export const slotRow = style({
  display: "grid",
  gridTemplateColumns: "repeat(3, minmax(0, 1fr))",
  gap: tokens.space.xs,
});
export const slot = style({
  width: "100%",
  height: "auto",
  aspectRatio: "1",
  minWidth: 0,
  minHeight: 0,
  flexDirection: "column",
  alignItems: "center",
  justifyContent: "center",
  gap: "2px",
  padding: tokens.space.xs,
  textAlign: "center",
  overflow: "hidden",
});
export const slotIcon = style({ width: "40%", height: "40%", objectFit: "contain", flexShrink: 1 });
export const slotIconPlaceholder = style({
  width: "36%",
  aspectRatio: "1",
  borderRadius: tokens.radius.sm,
  background: tokens.color.surfaceHover,
});
export const slotName = style({
  display: "-webkit-box",
  overflow: "hidden",
  width: "100%",
  color: tokens.color.text,
  fontSize: tokens.fontSize.sm,
  lineHeight: 1.1,
  WebkitBoxOrient: "vertical",
  WebkitLineClamp: 1,
});
export const slotMeta = style({
  color: tokens.color.textMuted,
  fontSize: "10px",
  overflow: "hidden",
  maxWidth: "100%",
  textOverflow: "ellipsis",
  whiteSpace: "nowrap",
});
export const pouchPad = style({
  display: "grid",
  gridTemplateColumns: "repeat(2, minmax(0, 1fr))",
  gap: tokens.space.xs,
});
/** Keep the confirmed column-major spell order and the separate final pair. */
export const spellPrimaryGrid = style({
  display: "grid",
  gridAutoFlow: "column",
  gridTemplateColumns: "repeat(2, minmax(0, 1fr))",
  gridTemplateRows: "repeat(5, auto)",
  gap: tokens.space.xs,
});
export const spellExtraGrid = style({
  display: "grid",
  gridTemplateColumns: "repeat(2, minmax(0, 1fr))",
  gap: tokens.space.xs,
});
globalStyle(`.${armorGroup} .${slotRow}, .${talismanGroup} .${slotRow}`, {
  gridTemplateColumns: "repeat(4, minmax(0, 1fr))",
});
globalStyle(`.${quickGroup} .${slotRow}`, { gridTemplateColumns: "repeat(5, minmax(0, 1fr))" });
globalStyle(`.${ammoGroup} .${slotRow}, .${physickGroup} .${slotRow}`, {
  gridTemplateColumns: "repeat(2, minmax(0, 1fr))",
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
