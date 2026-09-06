import { createTheme } from "@vanilla-extract/css";
import { tokens } from "./contract.css";

/**
 * Colour and its shadows are the only dimensions the three themes disagree
 * on. Typography, spacing, control sizes, radii and motion are shared so a
 * theme can never change layout or density.
 */
const shared = {
  font: {
    body: 'ui-sans-serif, system-ui, -apple-system, "Segoe UI", Roboto, Helvetica, Arial, sans-serif',
    mono: "ui-monospace, SFMono-Regular, Menlo, Consolas, monospace",
  },
  fontSize: {
    xs: "11px",
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
    sm: "26px",
    md: "30px",
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
    appBackground: "#f3f4f6",
    surface: "#ffffff",
    surfaceRaised: "#ffffff",
    surfaceSunken: "#f5f6f8",
    surfaceHover: "#f7f7f8",
    border: "#a9b6c6",
    borderStrong: "#6c819d",
    text: "#26241f",
    textMuted: "#554f46",
    textFaint: "#60594e",
    accent: "#2b6f45",
    accentText: "#1f5636",
    accentContrast: "#ffffff",
    selected: "#dde9d9",
    focus: "#1a5fa6",
    info: "#23568f",
    warning: "#7d5507",
    warningSurface: "#f6ecd6",
    danger: "#a52a21",
    dangerSurface: "#f7e2df",
    overlay: "rgba(30, 36, 45, 0.34)",
  },
  ...shared,
  shadow: {
    sm: "0 1px 3px rgba(0, 0, 0, 0.08)",
    lg: "0 8px 30px rgba(0, 0, 0, 0.12)",
  },
});

/** Dark is a neutral dark application theme. */
export const darkTheme = createTheme(tokens, {
  color: {
    background: "#16181c",
    appBackground: "#16181c",
    surface: "#202429",
    surfaceRaised: "#1c1f24",
    surfaceSunken: "#262b31",
    surfaceHover: "#2c3238",
    border: "#343b43",
    borderStrong: "#626e7a",
    text: "#e4e7eb",
    textMuted: "#b3bcc6",
    textFaint: "#9aa4ae",
    accent: "#4f9d69",
    accentText: "#86c99b",
    accentContrast: "#0e1a12",
    selected: "#1f3a2a",
    focus: "#6fb3ff",
    info: "#7aabe4",
    warning: "#e0a63a",
    warningSurface: "#2e2513",
    danger: "#f07d74",
    dangerSurface: "#331c1b",
    overlay: "rgba(8, 9, 11, 0.68)",
  },
  ...shared,
  shadow: {
    sm: "0 1px 2px rgba(0, 0, 0, 0.4)",
    lg: "0 12px 32px rgba(0, 0, 0, 0.55)",
  },
});

/** Elden Ring uses obsidian, aged gold and parchment without copying game UI. */
export const eldenRingTheme = createTheme(tokens, {
  color: {
    background: "#0e0d0b",
    /**
     * The Elden Ring backdrop is layered light, not a flat fill: two gold
     * glows and a faint diagonal weave over the base colour.
     */
    appBackground: [
      "radial-gradient(1100px 620px at 50% -10%, rgba(201, 162, 74, 0.11), transparent 65%)",
      "radial-gradient(800px 500px at 100% 100%, rgba(201, 162, 74, 0.05), transparent 60%)",
      "repeating-linear-gradient(115deg, rgba(201, 162, 74, 0.016) 0 2px, transparent 2px 6px)",
      "#0e0d0b",
    ].join(", "),
    surface: "#191710",
    surfaceRaised: "#16140f",
    surfaceSunken: "#201d15",
    surfaceHover: "#2f2919",
    border: "#40381f",
    borderStrong: "#82714a",
    text: "#f0e7cf",
    textMuted: "#cbbc94",
    textFaint: "#b6a780",
    accent: "#c9a24a",
    accentText: "#e3c574",
    accentContrast: "#17140c",
    selected: "#33290f",
    focus: "#f0d182",
    info: "#a7c0d8",
    warning: "#dcaa41",
    warningSurface: "#2c2210",
    danger: "#e28376",
    dangerSurface: "#2e1613",
    overlay: "rgba(6, 5, 4, 0.74)",
  },
  ...shared,
  shadow: {
    sm: "0 1px 2px rgba(0, 0, 0, 0.5)",
    lg: "0 12px 36px rgba(0, 0, 0, 0.7)",
  },
});

export const themeClassNames = {
  light: lightTheme,
  dark: darkTheme,
  "elden-ring": eldenRingTheme,
} as const;

export type ThemeName = keyof typeof themeClassNames;

export const themeNames = Object.keys(themeClassNames) as ThemeName[];
