import { style } from "@vanilla-extract/css";
import { tokens } from "../../../ui/tokens/contract.css";

export const panel = style({ minHeight: 0 });

export const toolbar = style({
  display: "flex",
  flexWrap: "wrap",
  alignItems: "center",
  gap: tokens.space.sm,
});

export const search = style({ flex: "1 1 240px", maxWidth: "480px" });
export const family = style({ flex: "0 1 180px" });
export const spacer = style({ flex: "1 1 auto" });
export const viewSwitch = style({
  display: "flex",
  gap: tokens.space.xs,
  margin: 0,
  padding: 0,
  border: 0,
});

export const message = style({ margin: 0, color: tokens.color.textMuted });
export const alert = style({ margin: 0, color: tokens.color.danger });

export const grid = style({
  display: "grid",
  gridTemplateColumns: "repeat(5, minmax(0, 1fr))",
  gap: tokens.space.md,
  minHeight: 0,
  "@media": {
    "screen and (max-width: 980px)": { gridTemplateColumns: "repeat(4, minmax(0, 1fr))" },
    "screen and (max-width: 720px)": { gridTemplateColumns: "repeat(2, minmax(0, 1fr))" },
  },
});

export const tile = style({
  minWidth: 0,
  height: "112px",
  flexDirection: "column",
  alignItems: "stretch",
  justifyContent: "space-between",
  padding: tokens.space.md,
  textAlign: "left",
});

export const tileName = style({
  overflow: "hidden",
  color: tokens.color.text,
  fontWeight: 700,
  textOverflow: "ellipsis",
  whiteSpace: "nowrap",
});

export const tileMeta = style({
  overflow: "hidden",
  color: tokens.color.textMuted,
  fontSize: tokens.fontSize.sm,
  textOverflow: "ellipsis",
  whiteSpace: "nowrap",
});

export const tableFrame = style({ height: "min(520px, calc(100vh - 260px))", minHeight: "280px" });
export const actionCell = style({ width: "1%", whiteSpace: "nowrap" });

export const pagination = style({
  display: "flex",
  alignItems: "center",
  justifyContent: "center",
  gap: tokens.space.sm,
});

export const detailHeading = style({ margin: 0, fontSize: tokens.fontSize.lg });
export const detailText = style({ margin: 0, color: tokens.color.textMuted });

export const facts = style({
  display: "grid",
  gridTemplateColumns: "repeat(3, minmax(0, 1fr))",
  gap: tokens.space.sm,
  margin: 0,
  "@media": { "screen and (max-width: 560px)": { gridTemplateColumns: "1fr" } },
});

export const fact = style({
  display: "flex",
  minWidth: 0,
  flexDirection: "column",
  gap: tokens.space.xs,
  padding: tokens.space.sm,
  border: `1px solid ${tokens.color.border}`,
  borderRadius: tokens.radius.sm,
  backgroundColor: tokens.color.surfaceRaised,
});

export const factLabel = style({ color: tokens.color.textMuted, fontSize: tokens.fontSize.sm });
export const factValue = style({ overflowWrap: "anywhere", fontWeight: 700 });

export const variantList = style({
  display: "grid",
  gridTemplateColumns: "repeat(2, minmax(0, 1fr))",
  gap: tokens.space.sm,
  margin: 0,
  padding: 0,
  listStyle: "none",
  "@media": { "screen and (max-width: 560px)": { gridTemplateColumns: "1fr" } },
});

export const variant = style({
  padding: tokens.space.sm,
  border: `1px solid ${tokens.color.border}`,
  borderRadius: tokens.radius.sm,
  color: tokens.color.textMuted,
});

export const visuallyHidden = style({
  position: "absolute",
  width: "1px",
  height: "1px",
  padding: 0,
  margin: "-1px",
  overflow: "hidden",
  clip: "rect(0, 0, 0, 0)",
  whiteSpace: "nowrap",
  border: 0,
});
