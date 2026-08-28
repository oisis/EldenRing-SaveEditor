import { createContext, type ReactNode, useContext } from "react";
import type { ApplicationInfoPort } from "./applicationInfoPort";

/**
 * The application layer carries the port but never constructs it: the
 * composition root injects the Wails adapter, and tests inject a stub. That
 * keeps this layer free of any dependency on the infrastructure layer.
 */
const ApplicationInfoPortContext = createContext<ApplicationInfoPort | null>(null);

export function ApplicationInfoPortProvider({
  port,
  children,
}: {
  port: ApplicationInfoPort;
  children: ReactNode;
}) {
  return <ApplicationInfoPortContext value={port}>{children}</ApplicationInfoPortContext>;
}

export function useApplicationInfoPort(): ApplicationInfoPort {
  const port = useContext(ApplicationInfoPortContext);
  if (port === null) {
    // A wiring mistake, never a user-facing condition: it can only happen if a
    // tree is rendered outside the composition root.
    throw new Error("ApplicationInfoPortProvider is missing above this component");
  }
  return port;
}
