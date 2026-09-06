import type { HTMLAttributes } from "react";
import { type BadgeVariants, badge, badgeDot } from "./Badge.css";

export type BadgeProps = HTMLAttributes<HTMLSpanElement> &
  BadgeVariants & {
    /** Prefixes the label with the mockup's status dot; decorative only. */
    dot?: boolean;
  };

/** The single canonical badge used for short, non-interactive labels. */
export function Badge({ tone, mono, dot, className, children, ...rest }: BadgeProps) {
  const recipeClass = badge({ tone, mono });
  return (
    <span className={className ? `${recipeClass} ${className}` : recipeClass} {...rest}>
      {dot === true && <span className={badgeDot} aria-hidden="true" />}
      {children}
    </span>
  );
}
