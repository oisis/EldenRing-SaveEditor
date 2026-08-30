import { createThemeContract } from "@vanilla-extract/css";

/**
 * The semantic design token contract of SaveForge 2.0. Components reference
 * these tokens only; a theme substitutes values for them and never ships its
 * own component implementation.
 *
 * The contract is intentionally small: it holds what the foundation slice
 * actually renders. Tokens for tables, sliders, dialogs and toasts are added
 * with the components that need them.
 */
export const tokens = createThemeContract({
  color: {
    background: null,
    surface: null,
    surfaceRaised: null,
    surfaceHover: null,
    border: null,
    borderStrong: null,
    text: null,
    textMuted: null,
    accent: null,
    accentText: null,
    accentContrast: null,
    focus: null,
    danger: null,
    dangerSurface: null,
    overlay: null,
  },
  font: {
    body: null,
    mono: null,
  },
  fontSize: {
    sm: null,
    md: null,
    lg: null,
    xl: null,
  },
  space: {
    xs: null,
    sm: null,
    md: null,
    lg: null,
    xl: null,
  },
  controlHeight: {
    sm: null,
    md: null,
  },
  radius: {
    sm: null,
    md: null,
  },
  /**
   * Motion durations are tokens so `prefers-reduced-motion` can neutralise
   * every animation in one place instead of per component.
   */
  motion: {
    fast: null,
  },
});
