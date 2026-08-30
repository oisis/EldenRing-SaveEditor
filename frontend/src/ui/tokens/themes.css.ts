import { createTheme } from "@vanilla-extract/css";
import { tokens } from "./contract.css";

/**
 * Colour is the only dimension the three themes disagree on. Typography,
 * spacing, control sizes, radii and motion are shared so a theme can never
 * change layout or density.
 */
const shared = {
  font: {
    body: 'ui-sans-serif, system-ui, -apple-system, "Segoe UI", Roboto, Helvetica, Arial, sans-serif',
    mono: "ui-monospace, SFMono-Regular, Menlo, Consolas, monospace",
  },
  fontSize: {
    sm: "12px",
    md: "13px",
    lg: "15px",
    xl: "19px",
  },
  space: {
    xs: "4px",
    sm: "8px",
    md: "12px",
    lg: "16px",
    xl: "24px",
  },
  controlHeight: {
    sm: "28px",
    md: "34px",
  },
  radius: {
    sm: "6px",
    md: "10px",
  },
  motion: {
    fast: "120ms",
  },
};

/** Light follows the SaveForge 1.6.13 colour direction on cool neutral greys. */
export const lightTheme = createTheme(tokens, {
  color: {
    background: "#f3f4f6",
    surface: "#ffffff",
    surfaceRaised: "#ffffff",
    surfaceHover: "#f5f6f8",
    border: "#a9b6c6",
    borderStrong: "#6c819d",
    text: "#26241f",
    textMuted: "#554f46",
    accent: "#2b6f45",
    accentText: "#1f5636",
    accentContrast: "#ffffff",
    focus: "#1a5fa6",
    danger: "#a52a21",
    dangerSurface: "#f7e2df",
    overlay: "rgba(20, 27, 35, 0.52)",
  },
  ...shared,
});

/** Dark is a neutral dark application theme. */
export const darkTheme = createTheme(tokens, {
  color: {
    background: "#16181c",
    surface: "#202429",
    surfaceRaised: "#1c1f24",
    surfaceHover: "#2c3238",
    border: "#343b43",
    borderStrong: "#626e7a",
    text: "#e4e7eb",
    textMuted: "#b3bcc6",
    accent: "#4f9d69",
    accentText: "#86c99b",
    accentContrast: "#0e1a12",
    focus: "#6fb3ff",
    danger: "#f07d74",
    dangerSurface: "#331c1b",
    overlay: "rgba(0, 0, 0, 0.72)",
  },
  ...shared,
});

/** Elden Ring uses obsidian, aged gold and parchment without copying game UI. */
export const eldenRingTheme = createTheme(tokens, {
  color: {
    background: "#0e0d0b",
    surface: "#191710",
    surfaceRaised: "#16140f",
    surfaceHover: "#2f2919",
    border: "#40381f",
    borderStrong: "#82714a",
    text: "#f0e7cf",
    textMuted: "#cbbc94",
    accent: "#c9a24a",
    accentText: "#e3c574",
    accentContrast: "#17140c",
    focus: "#f0d182",
    danger: "#e28376",
    dangerSurface: "#2e1613",
    overlay: "rgba(0, 0, 0, 0.78)",
  },
  ...shared,
});

export const themeClassNames = {
  light: lightTheme,
  dark: darkTheme,
  "elden-ring": eldenRingTheme,
} as const;

export type ThemeName = keyof typeof themeClassNames;

export const themeNames = Object.keys(themeClassNames) as ThemeName[];
