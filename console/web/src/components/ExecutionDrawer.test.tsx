import {
  render,
  screen,
  fireEvent,
  waitFor,
  act,
} from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";
import {
  useExecutionDrawer,
  type OpenExecutionDrawerOptions,
} from "./ExecutionDrawer";
import * as adapter from "../adapters/executionAdapter";

function TestHost({
  onOpenReady,
}: {
  onOpenReady: (open: (opts: OpenExecutionDrawerOptions) => void) => void;
}) {
  const { open, drawer } = useExecutionDrawer();
  onOpenReady(open);
  return <>{drawer}</>;
}

describe("useExecutionDrawer", () => {
  it("renders nothing until opened, shows the summary, then closes on the close button", async () => {
    vi.spyOn(adapter, "getAgentJourney").mockResolvedValue({
      status: "empty",
      reason: "No correlation id recorded for this activity.",
    });
    let openFn: (opts: OpenExecutionDrawerOptions) => void = () => {};
    render(<TestHost onOpenReady={(fn) => (openFn = fn)} />);
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();

    act(() =>
      openFn({
        summary: {
          title: "Sold!",
          body: "Final price $38.00.",
          trustCheck: "unavailable",
        },
      }),
    );
    expect(screen.getByRole("dialog")).toBeInTheDocument();
    expect(screen.getByText("Sold!")).toBeInTheDocument();
    expect(
      screen.getByText("Trust check: not available for this bid."),
    ).toBeInTheDocument();

    // Both ExecutionTimeline and TechnicalInspector render the shared
    // NotConnectedBanner for the same empty journey, so this text appears
    // twice; getAllByText mirrors the getAllByRole pattern already used for
    // TechnicalInspector's column headers.
    await waitFor(() =>
      expect(
        screen.getAllByText("No correlation id recorded for this activity."),
      ).toHaveLength(2),
    );

    fireEvent.click(screen.getByLabelText("Close"));
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  it("closes on Escape", () => {
    let openFn: (opts: OpenExecutionDrawerOptions) => void = () => {};
    render(<TestHost onOpenReady={(fn) => (openFn = fn)} />);
    act(() =>
      openFn({ summary: { title: "t", body: "b", trustCheck: "unavailable" } }),
    );
    fireEvent.keyDown(screen.getByRole("dialog"), { key: "Escape" });
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });
});
