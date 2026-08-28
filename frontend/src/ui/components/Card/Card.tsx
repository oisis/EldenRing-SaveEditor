import type { HTMLAttributes } from "react";
import { card } from "./Card.css";

export type CardProps = HTMLAttributes<HTMLElement>;

/**
 * The single canonical surface container. It renders a `section`, so callers
 * give it an accessible name with `aria-label` or `aria-labelledby`.
 */
export function Card({ className, ...rest }: CardProps) {
  return <section className={className ? `${card} ${className}` : card} {...rest} />;
}
