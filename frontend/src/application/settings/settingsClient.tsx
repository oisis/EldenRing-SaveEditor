import { createContext, type ReactNode, useContext } from "react";
import type { SettingsPort } from "./settingsPort";

/**
 * The application layer carries the port but never constructs it: the
 * composition root injects the desktop adapter, and tests inject a stub.
 */
const SettingsPortContext = createContext<SettingsPort | null>(null);

export function SettingsPortProvider({
  port,
  children,
}: {
  port: SettingsPort;
  children: ReactNode;
}) {
  return <SettingsPortContext value={port}>{children}</SettingsPortContext>;
}

export function useSettingsPort(): SettingsPort {
  const port = useContext(SettingsPortContext);
  if (port === null) {
    // A wiring mistake, never a user-facing condition.
    throw new Error("SettingsPortProvider is missing above this component");
  }
  return port;
}
