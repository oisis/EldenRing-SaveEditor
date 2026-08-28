import { type RecipeVariants, recipe } from "@vanilla-extract/recipes";
import { tokens } from "../../tokens/contract.css";

export const badge = recipe({
  base: {
    display: "inline-flex",
    alignItems: "center",
    paddingInline: tokens.space.sm,
    paddingBlock: tokens.space.xs,
    borderRadius: tokens.radius.sm,
    border: `1px solid ${tokens.color.border}`,
    fontSize: tokens.fontSize.sm,
    color: tokens.color.textMuted,
    backgroundColor: tokens.color.surfaceHover,
  },
  variants: {
    tone: {
      neutral: {},
      accent: {
        borderColor: tokens.color.accent,
        color: tokens.color.accentText,
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

export type BadgeVariants = NonNullable<RecipeVariants<typeof badge>>;
