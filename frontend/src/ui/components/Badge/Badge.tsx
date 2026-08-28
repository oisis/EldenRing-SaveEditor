import type { HTMLAttributes } from "react";
import { type BadgeVariants, badge } from "./Badge.css";

export type BadgeProps = HTMLAttributes<HTMLSpanElement> & BadgeVariants;

/** The single canonical badge used for short, non-interactive labels. */
export function Badge({ tone, mono, className, ...rest }: BadgeProps) {
  const recipeClass = badge({ tone, mono });
  return <span className={className ? `${recipeClass} ${className}` : recipeClass} {...rest} />;
}
