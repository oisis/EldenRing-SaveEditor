import type { HTMLAttributes, TableHTMLAttributes } from "react";
import { frame, table } from "./Table.css";

export function TableFrame({ className, ...rest }: HTMLAttributes<HTMLDivElement>) {
  return <div className={className ? `${frame} ${className}` : frame} {...rest} />;
}

/** The single canonical semantic table surface. TanStack Table supplies its model. */
export function Table({ className, ...rest }: TableHTMLAttributes<HTMLTableElement>) {
  return <table className={className ? `${table} ${className}` : table} {...rest} />;
}
