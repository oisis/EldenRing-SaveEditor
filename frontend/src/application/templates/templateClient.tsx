import { createContext, type ReactNode, useContext } from "react";
import type { TemplatePort } from "./templatePort";

/**
 * The application layer carries the port but never constructs it: the
 * composition root injects the desktop adapter, and tests inject a stub.
 */
const TemplatePortContext = createContext<TemplatePort | null>(null);

export function TemplatePortProvider({ port, children }: { port: TemplatePort; children: ReactNode }) {
  return <TemplatePortContext value={port}>{children}</TemplatePortContext>;
}

export function useTemplatePort(): TemplatePort {
  const port = useContext(TemplatePortContext);
  if (port === null) {
    // A wiring mistake, never a user-facing condition.
    throw new Error("TemplatePortProvider is missing above this component");
  }
  return port;
}
