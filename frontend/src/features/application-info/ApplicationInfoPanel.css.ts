import { style } from "@vanilla-extract/css";
import { tokens } from "../../ui/tokens/contract.css";

/** Screen layout only. Base control styling stays in the shared UI library. */
export const definitionList = style({
  display: "grid",
  gridTemplateColumns: "max-content 1fr",
  gap: `${tokens.space.sm} ${tokens.space.lg}`,
  margin: 0,
});

export const term = style({
  color: tokens.color.textMuted,
});

export const description = style({
  margin: 0,
  display: "flex",
  flexWrap: "wrap",
  gap: tokens.space.xs,
});
