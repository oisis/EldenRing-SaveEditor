import { createContext, type ReactNode, useContext } from "react";
import type { DeploymentPort } from "./deploymentPort";

/**
 * The application layer carries the port but never constructs it: the
 * composition root injects the desktop adapter, and tests inject a stub.
 */
const DeploymentPortContext = createContext<DeploymentPort | null>(null);

export function DeploymentPortProvider({ port, children }: { port: DeploymentPort; children: ReactNode }) {
  return <DeploymentPortContext value={port}>{children}</DeploymentPortContext>;
}

export function useDeploymentPort(): DeploymentPort {
  const port = useContext(DeploymentPortContext);
  if (port === null) {
    // A wiring mistake, never a user-facing condition.
    throw new Error("DeploymentPortProvider is missing above this component");
  }
  return port;
}
