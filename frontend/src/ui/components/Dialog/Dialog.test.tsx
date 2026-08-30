import { fireEvent, screen, waitFor } from "@testing-library/react";
import { useRef, useState } from "react";
import { describe, expect, it } from "vitest";
import { renderApp } from "../../../test/renderWithProviders";
import { Button } from "../Button/Button";
import { Dialog } from "./Dialog";

function DialogHarness() {
  const [open, setOpen] = useState(false);
  const triggerRef = useRef<HTMLButtonElement | null>(null);

  return (
    <>
      <Button
        onClick={(event) => {
          triggerRef.current = event.currentTarget;
          setOpen(true);
        }}
      >
        Open details
      </Button>
      <Dialog
        open={open}
        onOpenChange={setOpen}
        title="Item details"
        description="Read-only details"
        closeLabel="Close"
        returnFocusRef={triggerRef}
      >
        <Button>Inside action</Button>
      </Dialog>
    </>
  );
}

describe("Dialog", () => {
  it("closes with Escape and restores focus to the supplied trigger", async () => {
    await renderApp(<DialogHarness />);
    const trigger = screen.getByRole("button", { name: "Open details" });

    fireEvent.click(trigger);
    const dialog = await screen.findByRole("dialog", { name: "Item details" });
    expect(dialog).toHaveTextContent("Read-only details");
    expect(screen.getByRole("button", { name: "Inside action" })).toHaveFocus();

    fireEvent.keyDown(document, { key: "Escape" });

    await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
    expect(trigger).toHaveFocus();
  });
});
