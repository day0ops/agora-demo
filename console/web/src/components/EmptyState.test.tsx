import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import { EmptyState } from "./EmptyState";

describe("EmptyState", () => {
  it("renders title and body, with no icon wrapper when icon is omitted", () => {
    const { container } = render(
      <EmptyState title="No listings yet" body="Check back soon." />,
    );
    expect(screen.getByText("No listings yet")).toBeInTheDocument();
    expect(screen.getByText("Check back soon.")).toBeInTheDocument();
    expect(container.querySelector(".empty-state-icon")).toBeNull();
  });

  it("renders the icon wrapper when an icon is given", () => {
    const { container } = render(
      <EmptyState title="t" body="b" icon={<svg data-testid="icon" />} />,
    );
    expect(container.querySelector(".empty-state-icon")).not.toBeNull();
    expect(screen.getByTestId("icon")).toBeInTheDocument();
  });
});
