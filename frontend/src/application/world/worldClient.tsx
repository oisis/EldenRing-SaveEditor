import { createContext, type ReactNode, useContext } from "react";
import type { WorldPort } from "./worldPort";

/**
 * The application layer carries the port but never constructs it: the
 * composition root injects the desktop adapter, and tests inject a stub.
 */
const WorldPortContext = createContext<WorldPort | null>(null);

export function WorldPortProvider({ port, children }: { port: WorldPort; children: ReactNode }) {
  return <WorldPortContext value={port}>{children}</WorldPortContext>;
}

export function useWorldPort(): WorldPort {
  const port = useContext(WorldPortContext);
  if (port === null) {
    // A wiring mistake, never a user-facing condition: it can only happen if a
    // tree is rendered outside the composition root.
    throw new Error("WorldPortProvider is missing above this component");
  }
  return port;
}
