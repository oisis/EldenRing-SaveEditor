import { createContext, type ReactNode, useContext } from "react";
import type { EquipmentPort } from "./equipmentPort";

/**
 * The application layer carries the port but never constructs it: the
 * composition root injects the desktop adapter, and tests inject a stub. That
 * keeps this layer free of any dependency on the layer below it.
 */
const EquipmentPortContext = createContext<EquipmentPort | null>(null);

export function EquipmentPortProvider({
  port,
  children,
}: {
  port: EquipmentPort;
  children: ReactNode;
}) {
  return <EquipmentPortContext value={port}>{children}</EquipmentPortContext>;
}

export function useEquipmentPort(): EquipmentPort {
  const port = useContext(EquipmentPortContext);
  if (port === null) {
    // A wiring mistake, never a user-facing condition: it can only happen if a
    // tree is rendered outside the composition root.
    throw new Error("EquipmentPortProvider is missing above this component");
  }
  return port;
}
