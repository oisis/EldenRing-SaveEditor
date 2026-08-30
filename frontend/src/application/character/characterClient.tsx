import { createContext, type ReactNode, useContext } from "react";
import type { CharacterPort } from "./characterPort";

/**
 * The application layer carries the port but never constructs it: the
 * composition root injects the desktop adapter, and tests inject a stub. That
 * keeps this layer free of any dependency on the layer below it.
 */
const CharacterPortContext = createContext<CharacterPort | null>(null);

export function CharacterPortProvider({
  port,
  children,
}: {
  port: CharacterPort;
  children: ReactNode;
}) {
  return <CharacterPortContext value={port}>{children}</CharacterPortContext>;
}

export function useCharacterPort(): CharacterPort {
  const port = useContext(CharacterPortContext);
  if (port === null) {
    // A wiring mistake, never a user-facing condition: it can only happen if a
    // tree is rendered outside the composition root.
    throw new Error("CharacterPortProvider is missing above this component");
  }
  return port;
}
