import { createContext, type ReactNode, useContext } from "react";
import type { AboutPort } from "./aboutPort";

/**
 * The application layer carries the port but never constructs it: the
 * composition root injects the desktop adapter, and tests inject a stub.
 */
const AboutPortContext = createContext<AboutPort | null>(null);

export function AboutPortProvider({ port, children }: { port: AboutPort; children: ReactNode }) {
  return <AboutPortContext value={port}>{children}</AboutPortContext>;
}

export function useAboutPort(): AboutPort {
  const port = useContext(AboutPortContext);
  if (port === null) {
    // A wiring mistake, never a user-facing condition.
    throw new Error("AboutPortProvider is missing above this component");
  }
  return port;
}
