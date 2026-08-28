import { screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { ApplicationInfoPanel } from "../../features/application-info/ApplicationInfoPanel";
import {
  failingPort,
  makePort,
  renderApp,
  stubApplicationInfo,
} from "../../test/renderWithProviders";
import { queryKeys } from "../queryKeys";

describe("application info query", () => {
  it("shows the loading state and then the backend result", async () => {
    const getApplicationInfo = vi.fn(makePort().getApplicationInfo);

    await renderApp(<ApplicationInfoPanel />, { port: { getApplicationInfo } });

    expect(screen.getByRole("status")).toHaveTextContent("Loading application information…");
    // Loading offers nothing to retry: the call is still in flight.
    expect(screen.queryByRole("button", { name: "Retry" })).not.toBeInTheDocument();

    expect(await screen.findByText(stubApplicationInfo.version)).toBeInTheDocument();
    expect(screen.getByText("SaveForge 2.0")).toBeInTheDocument();
    expect(screen.getByText("game_catalog 1–16")).toBeInTheDocument();
    expect(screen.getByText("catalog_read")).toBeInTheDocument();
    // Success offers nothing to retry either.
    expect(screen.queryByRole("button", { name: "Retry" })).not.toBeInTheDocument();
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();

    // TanStack Query resolved the view through the injected port, so the query
    // layer wraps the desktop bridge and not an HTTP client.
    expect(getApplicationInfo).toHaveBeenCalledTimes(1);
  });

  it("shows a safe error state and retries through the port", async () => {
    const getApplicationInfo = vi
      .fn(failingPort.getApplicationInfo)
      .mockRejectedValueOnce(new Error("bridge_call_failed"))
      .mockResolvedValueOnce(stubApplicationInfo);

    const { container } = await renderApp(<ApplicationInfoPanel />, {
      port: { getApplicationInfo },
    });

    await waitFor(() => {
      expect(screen.getByRole("alert")).toHaveTextContent(
        "Could not read application information from the backend.",
      );
    });
    // Nothing from the transport reaches the interface.
    expect(container.textContent).not.toContain("bridge_call_failed");
    expect(screen.queryByText(stubApplicationInfo.version)).not.toBeInTheDocument();
    // Retry exists only in the error state.
    expect(screen.getByRole("button", { name: "Retry" })).toBeInTheDocument();

    screen.getByRole("button", { name: "Retry" }).click();

    expect(await screen.findByText(stubApplicationInfo.version)).toBeInTheDocument();
    expect(getApplicationInfo).toHaveBeenCalledTimes(2);
    // A successful retry clears both the message and the button.
    await waitFor(() => {
      expect(screen.queryByRole("button", { name: "Retry" })).not.toBeInTheDocument();
    });
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  });

  it("keeps one source of truth for the query key", () => {
    expect(queryKeys.applicationInfo()).toEqual(["application", "info"]);
  });
});
