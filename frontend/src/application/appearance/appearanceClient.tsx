import { createContext, type ReactNode, useContext } from "react";
import type { AppearancePort } from "./appearancePort";

const AppearancePortContext = createContext<AppearancePort | null>(null);

export function AppearancePortProvider({
  port,
  children,
}: {
  port: AppearancePort;
  children: ReactNode;
}) {
  return <AppearancePortContext value={port}>{children}</AppearancePortContext>;
}

export function useAppearancePort(): AppearancePort {
  const port = useContext(AppearancePortContext);
  if (port === null) {
    throw new Error("AppearancePortProvider is missing above this component");
  }
  return port;
}
