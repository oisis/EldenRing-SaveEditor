import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { Button } from "./Button";
import { button } from "./Button.css";

describe("Button recipe variants", () => {
  it("applies the default variants", () => {
    render(<Button>Default</Button>);

    expect(screen.getByRole("button", { name: "Default" })).toHaveClass(
      ...button({ tone: "neutral", size: "md", pressed: false }).split(" "),
    );
  });

  it("produces a different class list for a different typed variant", () => {
    const neutral = button({ tone: "neutral", size: "md" });
    const accent = button({ tone: "accent", size: "sm" });

    expect(accent).not.toBe(neutral);
    expect(accent.trim()).not.toBe("");
  });

  it("renders the requested variant on the element", () => {
    render(
      <Button tone="accent" size="sm" pressed>
        Accent
      </Button>,
    );

    expect(screen.getByRole("button", { name: "Accent" })).toHaveClass(
      ...button({ tone: "accent", size: "sm", pressed: true }).split(" "),
    );
  });

  it("exposes an explicit pressed toggle state to assistive technology", () => {
    render(<Button pressed>Pressed</Button>);
    render(<Button pressed={false}>Unpressed</Button>);

    expect(screen.getByRole("button", { name: "Pressed" })).toHaveAttribute("aria-pressed", "true");
    expect(screen.getByRole("button", { name: "Unpressed" })).toHaveAttribute(
      "aria-pressed",
      "false",
    );
  });

  it("adds no toggle state when pressed is not used", () => {
    render(<Button>Plain</Button>);

    expect(screen.getByRole("button", { name: "Plain" })).not.toHaveAttribute("aria-pressed");
  });

  it("never submits a surrounding form unless the caller asks for it", () => {
    render(<Button>Action</Button>);
    render(<Button type="submit">Submit</Button>);

    expect(screen.getByRole("button", { name: "Action" })).toHaveAttribute("type", "button");
    expect(screen.getByRole("button", { name: "Submit" })).toHaveAttribute("type", "submit");
  });
});
