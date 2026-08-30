import { style } from "@vanilla-extract/css";
import { tokens } from "../../tokens/contract.css";

export const input = style({
  width: "100%",
  minWidth: 0,
  height: tokens.controlHeight.md,
  paddingInline: tokens.space.sm,
  border: `1px solid ${tokens.color.border}`,
  borderRadius: tokens.radius.sm,
  backgroundColor: tokens.color.surfaceRaised,
  color: tokens.color.text,
  font: "inherit",
  selectors: {
    "&::placeholder": { color: tokens.color.textMuted },
    "&:hover:not(:disabled)": { borderColor: tokens.color.borderStrong },
    "&:disabled": { cursor: "not-allowed", opacity: 0.55 },
  },
});
