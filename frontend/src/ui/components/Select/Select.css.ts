import { style } from "@vanilla-extract/css";
import { tokens } from "../../tokens/contract.css";

export const select = style({
  minWidth: 0,
  height: tokens.controlHeight.md,
  paddingInline: "9px",
  border: `1px solid ${tokens.color.border}`,
  borderRadius: tokens.radius.sm,
  backgroundColor: tokens.color.surfaceRaised,
  color: tokens.color.text,
  font: "inherit",
  cursor: "pointer",
  selectors: {
    "&:hover:not(:disabled)": { borderColor: tokens.color.borderStrong },
    "&:disabled": { cursor: "not-allowed", opacity: 0.55 },
  },
});
