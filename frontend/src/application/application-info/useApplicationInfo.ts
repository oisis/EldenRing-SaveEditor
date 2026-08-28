import { useQuery } from "@tanstack/react-query";
import { queryKeys } from "../queryKeys";
import { useApplicationInfoPort } from "./applicationInfoClient";
import type { ApplicationInfo } from "./applicationInfoPort";

/**
 * The public application hook for backend application info. Feature modules use
 * this and never the generated Wails bindings.
 *
 * The query is not retried: a failing desktop bridge call does not become
 * healthy by repeating it, and the screen offers an explicit Retry instead.
 */
export function useApplicationInfo() {
  const port = useApplicationInfoPort();
  return useQuery<ApplicationInfo>({
    queryKey: queryKeys.applicationInfo(),
    queryFn: () => port.getApplicationInfo(),
    retry: false,
  });
}
