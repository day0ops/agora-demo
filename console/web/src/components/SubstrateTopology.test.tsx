import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import { SubstrateTopology } from "./SubstrateTopology";

describe("SubstrateTopology", () => {
  it("renders the not-connected banner when status is empty", () => {
    render(
      <SubstrateTopology
        result={{
          status: "empty",
          reason: "Not connected to a live substrate.",
        }}
      />,
    );
    expect(
      screen.getByText("Not connected to a live substrate."),
    ).toBeInTheDocument();
  });

  it("renders one row per actor when status is real", () => {
    render(
      <SubstrateTopology
        result={{
          status: "real",
          data: {
            actors: [
              {
                id: "agora-workers-0",
                state: "running",
                pool: "agora-workers",
                lastActivity: "2s ago",
              },
            ],
          },
        }}
      />,
    );
    expect(screen.getByText("agora-workers-0")).toBeInTheDocument();
  });
});
