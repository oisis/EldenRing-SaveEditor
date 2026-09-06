import { style } from "@vanilla-extract/css";
import { type RecipeVariants, recipe } from "@vanilla-extract/recipes";
import { tokens } from "../../tokens/contract.css";

export const badge = recipe({
  base: {
    display: "inline-flex",
    alignItems: "center",
    gap: "5px",
    padding: "1px 7px",
    borderRadius: "999px",
    border: `1px solid ${tokens.color.borderStrong}`,
    fontSize: tokens.fontSize.xs,
    fontWeight: 600,
    letterSpacing: "0.03em",
    whiteSpace: "nowrap",
    color: tokens.color.textMuted,
    backgroundColor: tokens.color.surfaceSunken,
  },
  variants: {
    tone: {
      neutral: {},
      accent: {
        borderColor: tokens.color.accent,
        color: tokens.color.accentText,
        backgroundColor: tokens.color.selected,
      },
      warning: {
        borderColor: tokens.color.warning,
        color: tokens.color.warning,
        backgroundColor: tokens.color.warningSurface,
      },
      danger: {
        borderColor: tokens.color.danger,
        color: tokens.color.danger,
        backgroundColor: tokens.color.dangerSurface,
      },
    },
    mono: {
      true: { fontFamily: tokens.font.mono },
      false: {},
    },
  },
  defaultVariants: {
    tone: "neutral",
    mono: false,
  },
});

/** The mockup's status dot; it carries the badge's own colour. */
export const badgeDot = style({
  width: "7px",
  height: "7px",
  flex: "none",
  borderRadius: "50%",
  backgroundColor: "currentColor",
});

export type BadgeVariants = NonNullable<RecipeVariants<typeof badge>>;
