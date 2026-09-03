import { createContext, type ReactNode, useContext } from "react";
import type { FavoritesPort } from "./favoritesPort";

const FavoritesPortContext = createContext<FavoritesPort | null>(null);

export function FavoritesPortProvider({
  port,
  children,
}: {
  port: FavoritesPort;
  children: ReactNode;
}) {
  return <FavoritesPortContext value={port}>{children}</FavoritesPortContext>;
}

export function useFavoritesPort(): FavoritesPort {
  const port = useContext(FavoritesPortContext);
  if (port === null) {
    throw new Error("FavoritesPortProvider is missing above this component");
  }
  return port;
}
