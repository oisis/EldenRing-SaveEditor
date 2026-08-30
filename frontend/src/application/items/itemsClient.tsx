import { createContext, type ReactNode, useContext } from "react";
import type { ItemsPort } from "./itemsPort";

/**
 * The application layer carries the port but never constructs it: the
 * composition root injects the desktop adapter, and tests inject a stub. That
 * keeps this layer free of any dependency on the layer below it.
 */
const ItemsPortContext = createContext<ItemsPort | null>(null);

export function ItemsPortProvider({ port, children }: { port: ItemsPort; children: ReactNode }) {
  return <ItemsPortContext value={port}>{children}</ItemsPortContext>;
}

export function useItemsPort(): ItemsPort {
  const port = useContext(ItemsPortContext);
  if (port === null) {
    // A wiring mistake, never a user-facing condition: it can only happen if a
    // tree is rendered outside the composition root.
    throw new Error("ItemsPortProvider is missing above this component");
  }
  return port;
}
