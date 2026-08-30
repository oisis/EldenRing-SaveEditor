import { forwardRef, type InputHTMLAttributes } from "react";
import { input } from "./Input.css";

export type InputProps = InputHTMLAttributes<HTMLInputElement>;

/** The single canonical text input used by SaveForge feature modules. */
export const Input = forwardRef<HTMLInputElement, InputProps>(function Input(
  { className, ...rest },
  ref,
) {
  return <input ref={ref} className={className ? `${input} ${className}` : input} {...rest} />;
});
