import { style } from "@vanilla-extract/css";
import { tokens } from "./ui/tokens/contract.css";

/** Colour comes from the document element theme class; this is layout only. */
export const shell = style({
  minHeight: "100vh",
  padding: tokens.space.xl,
  display: "flex",
  flexDirection: "column",
  gap: tokens.space.lg,
});

export const heading = style({
  margin: 0,
  fontSize: tokens.fontSize.xl,
});

export const controls = style({
  display: "flex",
  gap: tokens.space.sm,
  alignItems: "center",
});
