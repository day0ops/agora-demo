import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import { StatusPill } from "./StatusPill";

describe("StatusPill", () => {
  it("renders the label and pill class for a known status", () => {
    render(<StatusPill status="active-auction" />);
    const el = screen.getByText("Live auction");
    expect(el.className).toBe("pill warn");
  });

  it("falls back to the raw status and idle class for an unknown status", () => {
    render(<StatusPill status="mystery" />);
    const el = screen.getByText("mystery");
    expect(el.className).toBe("pill idle");
  });
});
