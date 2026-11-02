import { render, screen } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { Footer } from "./Footer";

describe("Footer", () => {
  beforeEach(() => vi.restoreAllMocks());

  it("renders the app name, GitHub link, and fetched version", async () => {
    vi.spyOn(global, "fetch").mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ version: "v0.3.0" }),
    } as Response);
    render(<Footer />);
    expect(screen.getByText("Agora")).toBeInTheDocument();
    expect(await screen.findByText("v0.3.0")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "GitHub" })).toHaveAttribute(
      "href",
      "https://github.com/day0ops/agora-demo",
    );
  });

  it("omits the version text if the fetch fails", async () => {
    vi.spyOn(global, "fetch").mockRejectedValue(new Error("network error"));
    render(<Footer />);
    expect(screen.getByText("Agora")).toBeInTheDocument();
    await new Promise((r) => setTimeout(r, 0));
    expect(screen.queryByText(/^v\d/)).toBeNull();
  });
});
