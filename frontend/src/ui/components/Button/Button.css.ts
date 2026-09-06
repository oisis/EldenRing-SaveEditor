import { type RecipeVariants, recipe } from "@vanilla-extract/recipes";
import { tokens } from "../../tokens/contract.css";

export const button = recipe({
  base: {
    display: "inline-flex",
    alignItems: "center",
    justifyContent: "center",
    gap: tokens.space.xs,
    border: `1px solid ${tokens.color.border}`,
    borderRadius: tokens.radius.sm,
    cursor: "pointer",
    font: "inherit",
    fontSize: tokens.fontSize.md,
    color: tokens.color.text,
    backgroundColor: tokens.color.surfaceSunken,
    transitionProperty: "background-color, border-color, color",
    transitionDuration: tokens.motion.fast,
    selectors: {
      "&:hover:not(:disabled)": {
        backgroundColor: tokens.color.surfaceHover,
        borderColor: tokens.color.borderStrong,
      },
      "&:disabled": {
        cursor: "not-allowed",
        opacity: 0.55,
      },
    },
  },
  variants: {
    tone: {
      neutral: {},
      accent: {
        backgroundColor: tokens.color.accent,
        borderColor: tokens.color.accent,
        color: tokens.color.accentContrast,
        fontWeight: 600,
      },
    },
    size: {
      sm: {
        height: tokens.controlHeight.sm,
        paddingInline: tokens.space.sm,
        fontSize: tokens.fontSize.sm,
      },
      md: {
        height: tokens.controlHeight.md,
        paddingInline: tokens.space.md,
      },
    },
    pressed: {
      true: {
        borderColor: tokens.color.accent,
        backgroundColor: tokens.color.selected,
      },
      false: {},
    },
  },
  defaultVariants: {
    tone: "neutral",
    size: "md",
    pressed: false,
  },
});

export type ButtonVariants = NonNullable<RecipeVariants<typeof button>>;
