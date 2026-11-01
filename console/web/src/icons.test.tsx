import { render } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import { IconBrand, IconChevronDown } from "./icons";

describe("icons", () => {
  it("renders the brand icon as an svg", () => {
    const { container } = render(<IconBrand />);
    expect(container.querySelector("svg")).toBeInTheDocument();
  });

  it("renders the chevron-down icon as an svg", () => {
    const { container } = render(<IconChevronDown />);
    expect(container.querySelector("svg")).toBeInTheDocument();
  });
});
