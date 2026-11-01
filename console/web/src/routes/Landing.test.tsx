import { render, screen } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { MemoryRouter } from "react-router-dom";
import { Landing } from "./Landing";
import { IdentityProvider } from "../hooks/useIdentity";
import * as api from "../api";
import type { Listing } from "../types";

function renderLanding() {
  return render(
    <MemoryRouter>
      <IdentityProvider>
        <Landing />
      </IdentityProvider>
    </MemoryRouter>,
  );
}

const sampleListing: Listing = {
  id: "listing-1",
  sellerId: "seller-1",
  title: "Rolex Submariner",
  description: "x",
  category: "watches",
  type: "auction",
  status: "active-auction",
  priceCents: 500000,
  currentBidCents: 520000,
  createdAt: new Date().toISOString(),
};

describe("Landing", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    vi.spyOn(global, "fetch").mockResolvedValue({
      ok: false,
      json: () => Promise.resolve({}),
    } as Response);
  });

  it("shows the Login CTA when logged out", async () => {
    vi.spyOn(api, "listListings").mockResolvedValue([]);
    renderLanding();
    // Navbar's account-area link and the hero's secondary CTA both render
    // "Login" when logged out, so this is 2 matches rather than 1.
    expect(await screen.findAllByText("Login")).toHaveLength(2);
  });

  it("hides the Login CTA once already logged in", async () => {
    vi.spyOn(global, "fetch").mockResolvedValue({
      ok: true,
      json: () =>
        Promise.resolve({
          username: "demo-buyer",
          groups: ["buyers"],
          role: "buyers",
        }),
    } as Response);
    vi.spyOn(api, "listListings").mockResolvedValue([]);
    renderLanding();
    await screen.findByText("demo-buyer");
    expect(screen.queryByText("Login")).not.toBeInTheDocument();
  });

  it("renders a loaded listing in the Featured rail", async () => {
    vi.spyOn(api, "listListings").mockResolvedValue([sampleListing]);
    renderLanding();
    // A single active-auction listing qualifies for all three rails
    // (Featured, Closing soon, Just listed), so it renders 3 times.
    expect(await screen.findAllByText("Rolex Submariner")).toHaveLength(3);
  });

  it("shows a catalog-unavailable state in every rail when the fetch fails", async () => {
    vi.spyOn(api, "listListings").mockRejectedValue(new Error("network error"));
    renderLanding();
    expect(await screen.findAllByText("Catalog unavailable")).toHaveLength(3);
  });
});
