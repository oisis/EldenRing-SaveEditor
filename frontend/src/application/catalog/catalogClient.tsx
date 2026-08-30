import { createContext, type ReactNode, useContext } from "react";
import type { CatalogPort } from "./catalogPort";

/**
 * The application layer carries the port but never constructs it: the
 * composition root injects the desktop adapter, and tests inject a stub. That
 * keeps this layer free of any dependency on the layer below it.
 *
 * The provider is global and independent of the save-session provider: the
 * catalog is readable with no save loaded at all.
 */
const CatalogPortContext = createContext<CatalogPort | null>(null);

export function CatalogPortProvider({
  port,
  children,
}: {
  port: CatalogPort;
  children: ReactNode;
}) {
  return <CatalogPortContext value={port}>{children}</CatalogPortContext>;
}

export function useCatalogPort(): CatalogPort {
  const port = useContext(CatalogPortContext);
  if (port === null) {
    // A wiring mistake, never a user-facing condition: it can only happen if a
    // tree is rendered outside the composition root.
    throw new Error("CatalogPortProvider is missing above this component");
  }
  return port;
}
