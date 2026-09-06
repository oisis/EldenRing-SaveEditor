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
  background: tokens.color.appBackground,
  backgroundAttachment: "fixed",
});

globalStyle("::selection", { background: tokens.color.selected });

globalStyle("h1, h2, h3, h4, p", { margin: 0 });
globalStyle("h2", { fontSize: tokens.fontSize.md, fontWeight: 600 });
globalStyle("h3, h4", { fontSize: tokens.fontSize.sm, fontWeight: 600 });
globalStyle("*", {
  scrollbarWidth: "thin",
  scrollbarColor: `${tokens.color.borderStrong} transparent`,
});
/** Firefox honours the two properties above; WebKit needs the pseudo-elements. */
globalStyle("*::-webkit-scrollbar", { width: "10px", height: "10px" });
globalStyle("*::-webkit-scrollbar-track", { background: "transparent" });
globalStyle("*::-webkit-scrollbar-thumb", {
  background: tokens.color.borderStrong,
  borderRadius: "6px",
  border: "2px solid transparent",
  backgroundClip: "content-box",
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
