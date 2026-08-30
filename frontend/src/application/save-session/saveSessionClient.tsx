import { createContext, type ReactNode, useContext } from "react";
import type { SaveSessionPort } from "./saveSessionPort";

/**
 * The application layer carries the port but never constructs it: the
 * composition root injects the desktop adapter, and tests inject a stub. That
 * keeps this layer free of any dependency on the layer below it.
 */
const SaveSessionPortContext = createContext<SaveSessionPort | null>(null);

export function SaveSessionPortProvider({
  port,
  children,
}: {
  port: SaveSessionPort;
  children: ReactNode;
}) {
  return <SaveSessionPortContext value={port}>{children}</SaveSessionPortContext>;
}

export function useSaveSessionPort(): SaveSessionPort {
  const port = useContext(SaveSessionPortContext);
  if (port === null) {
    // A wiring mistake, never a user-facing condition: it can only happen if a
    // tree is rendered outside the composition root.
    throw new Error("SaveSessionPortProvider is missing above this component");
  }
  return port;
}
