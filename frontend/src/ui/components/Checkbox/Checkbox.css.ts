import { style } from "@vanilla-extract/css";
import { tokens } from "../../tokens/contract.css";

export const checkbox = style({
  width: "1rem",
  height: "1rem",
  margin: 0,
  accentColor: tokens.color.accent,
  cursor: "pointer",
  selectors: {
    "&:disabled": { cursor: "not-allowed", opacity: 0.55 },
  },
});
