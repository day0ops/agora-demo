import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import { ExecutionTimeline } from "./ExecutionTimeline";

describe("ExecutionTimeline", () => {
  it("renders the not-connected banner when status is empty", () => {
    render(
      <ExecutionTimeline
        result={{ status: "empty", reason: "No live substrate." }}
      />,
    );
    expect(screen.getByText("No live substrate.")).toBeInTheDocument();
  });

  it("renders each step when status is real", () => {
    render(
      <ExecutionTimeline
        result={{
          status: "real",
          data: {
            steps: [
              {
                actor: "auto-bidder",
                state: "x",
                pool: "p",
                mcpCall: "c",
                policy: "pol",
              },
            ],
          },
        }}
      />,
    );
    expect(screen.getByText("auto-bidder")).toBeInTheDocument();
  });
});
