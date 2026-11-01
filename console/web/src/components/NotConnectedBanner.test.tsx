import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import { NotConnectedBanner } from "./NotConnectedBanner";

describe("NotConnectedBanner", () => {
  it("renders the given reason", () => {
    render(<NotConnectedBanner reason="Not connected to a live substrate." />);
    expect(
      screen.getByText("Not connected to a live substrate."),
    ).toBeInTheDocument();
  });
});
