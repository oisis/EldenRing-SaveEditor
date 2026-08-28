import type { ButtonHTMLAttributes } from "react";
import { type ButtonVariants, button } from "./Button.css";

export type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & ButtonVariants;

/**
 * The single canonical button of SaveForge 2.0. Feature modules compose it and
 * must not redefine its base styling or ship a parallel button.
 *
 * `pressed` is a toggle state, not only a style: it is exposed to assistive
 * technology as `aria-pressed`. A button that omits `pressed` stays a plain
 * button and gains no toggle semantics.
 */
export function Button({ tone, size, pressed, className, type, ...rest }: ButtonProps) {
  const recipeClass = button({ tone, size, pressed });
  return (
    <button
      // A button inside a form defaults to submitting it; SaveForge buttons are
      // actions unless a caller explicitly asks for a submit button.
      type={type ?? "button"}
      className={className ? `${recipeClass} ${className}` : recipeClass}
      aria-pressed={pressed}
      {...rest}
    />
  );
}
