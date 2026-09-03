import { createContext, type ReactNode, useContext } from "react";
import type { NetworkPort } from "./networkPort";

const NetworkPortContext = createContext<NetworkPort | null>(null);

export function NetworkPortProvider({
  port,
  children,
}: {
  port: NetworkPort;
  children: ReactNode;
}) {
  return <NetworkPortContext value={port}>{children}</NetworkPortContext>;
}

export function useNetworkPort(): NetworkPort {
  const port = useContext(NetworkPortContext);
  if (port === null) {
    throw new Error("NetworkPortProvider is missing above this component");
  }
  return port;
}
