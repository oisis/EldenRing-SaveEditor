import { globalStyle } from "@vanilla-extract/css";
import { tokens } from "./contract.css";

globalStyle("*, *::before, *::after", {
  boxSizing: "border-box",
  /**
   * Reduced motion is honoured globally rather than per component, so a new
   * component cannot forget it.
   */
  "@media": {
    "(prefers-reduced-motion: reduce)": {
      animationDuration: "0.01ms !important",
      animationIterationCount: "1 !important",
      transitionDuration: "0.01ms !important",
    },
  },
});

globalStyle("body", {
  margin: 0,
  fontFamily: tokens.font.body,
  fontSize: tokens.fontSize.md,
  lineHeight: 1.45,
  color: tokens.color.text,
  backgroundColor: tokens.color.background,
});

/**
 * One visible focus ring for the whole application. Components must not remove
 * it; a control that needs a different shape adjusts its own border radius.
 */
globalStyle(":focus-visible", {
  outline: `2px solid ${tokens.color.focus}`,
  outlineOffset: "2px",
  borderRadius: tokens.radius.sm,
});
