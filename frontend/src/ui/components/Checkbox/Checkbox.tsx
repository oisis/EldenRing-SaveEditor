import { forwardRef, type InputHTMLAttributes } from "react";
import { checkbox } from "./Checkbox.css";

export type CheckboxProps = Omit<InputHTMLAttributes<HTMLInputElement>, "type">;

/** The canonical native checkbox used by SaveForge feature modules. */
export const Checkbox = forwardRef<HTMLInputElement, CheckboxProps>(function Checkbox(
  { className, ...rest },
  ref,
) {
  return (
    <input
      ref={ref}
      type="checkbox"
      className={className ? `${checkbox} ${className}` : checkbox}
      {...rest}
    />
  );
});
