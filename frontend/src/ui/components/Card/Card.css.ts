import { style } from "@vanilla-extract/css";
import { tokens } from "../../tokens/contract.css";

export const card = style({
  display: "flex",
  flexDirection: "column",
  gap: tokens.space.md,
  padding: tokens.space.lg,
  border: `1px solid ${tokens.color.border}`,
  borderRadius: tokens.radius.md,
  backgroundColor: tokens.color.surface,
  color: tokens.color.text,
});
