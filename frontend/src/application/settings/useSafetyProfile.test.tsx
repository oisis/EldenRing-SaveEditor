import { renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it } from "vitest";
import { createTestQueryClient, TestProviders } from "../../test/renderWithProviders";
import { queryKeys } from "../queryKeys";
import { useSetSafetyProfile } from "./useSafetyProfile";

describe("useSetSafetyProfile", () => {
  it("invalidates the cached Equipment picker candidates", async () => {
    const queryClient = createTestQueryClient();
    const candidates = queryKeys.equipmentCandidates("session-1", 0, "0", {
      slotType: "right_hand",
      search: "",
      page: 1,
      pageSize: 50,
    });
    // A cached page the profile selected the answer of: fresh before the call.
    // Nothing observes it here, so it must survive the shared client's garbage
    // collection long enough to be inspected after the mutation settled.
    queryClient.setQueryDefaults(candidates, { gcTime: Number.POSITIVE_INFINITY });
    queryClient.setQueryData(candidates, { total: 0 });
    expect(queryClient.getQueryState(candidates)?.isInvalidated).toBe(false);

    const wrapper = ({ children }: { children: ReactNode }) => (
      <TestProviders queryClient={queryClient}>{children}</TestProviders>
    );
    const { result } = renderHook(() => useSetSafetyProfile(), { wrapper });

    result.current.mutate("chaos");

    await waitFor(() => {
      expect(queryClient.getQueryState(candidates)?.isInvalidated).toBe(true);
    });
  });
});
