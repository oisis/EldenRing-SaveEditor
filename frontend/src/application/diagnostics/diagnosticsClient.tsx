import { createContext, type ReactNode, useContext } from "react";
import type { DiagnosticsPort } from "./diagnosticsPort";

/**
 * The application layer carries the port but never constructs it: the
 * composition root injects the desktop adapter, and tests inject a stub.
 */
const DiagnosticsPortContext = createContext<DiagnosticsPort | null>(null);

export function DiagnosticsPortProvider({
  port,
  children,
}: {
  port: DiagnosticsPort;
  children: ReactNode;
}) {
  return <DiagnosticsPortContext value={port}>{children}</DiagnosticsPortContext>;
}

export function useDiagnosticsPort(): DiagnosticsPort {
  const port = useContext(DiagnosticsPortContext);
  if (port === null) {
    throw new Error("DiagnosticsPortProvider is missing above this component");
  }
  return port;
}
