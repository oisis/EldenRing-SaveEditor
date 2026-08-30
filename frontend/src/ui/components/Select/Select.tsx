import { forwardRef, type SelectHTMLAttributes } from "react";
import { select } from "./Select.css";

export type SelectProps = SelectHTMLAttributes<HTMLSelectElement>;

/** The single canonical native select used by SaveForge feature modules. */
export const Select = forwardRef<HTMLSelectElement, SelectProps>(function Select(
  { className, ...rest },
  ref,
) {
  return <select ref={ref} className={className ? `${select} ${className}` : select} {...rest} />;
});
