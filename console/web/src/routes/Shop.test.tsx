import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { MemoryRouter } from "react-router-dom";
import { Shop } from "./Shop";
import { IdentityProvider } from "../hooks/useIdentity";
import * as api from "../api";
import type { Listing, Bid } from "../types";

const activeAuction: Listing = {
  id: "listing-1",
  sellerId: "seller-1",
  title: "1965 Rolex Submariner",
  description: "Vintage automatic dive watch.",
  category: "watches",
  type: "auction",
  status: "active-auction",
  priceCents: 500000,
  currentBidCents: 520000,
  createdAt: new Date().toISOString(),
};

function renderShop(identityResponse: unknown) {
  vi.spyOn(global, "fetch").mockResolvedValue({
    ok: identityResponse !== null,
    json: () => Promise.resolve(identityResponse ?? {}),
  } as Response);
  return render(
    <MemoryRouter>
      <IdentityProvider>
        <Shop />
      </IdentityProvider>
    </MemoryRouter>,
  );
}

class MockEventSource {
  close() {}
  constructor(public url: string) {}
}

describe("Shop", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    // @ts-expect-error -- test double
    global.EventSource = MockEventSource;
    // jsdom has no matchMedia; ChatPanel (rendered by Shop) reads it on mount
    // to pick its default collapsed state. See ChatPanel.test.tsx for the
    // same stub, needed for the same reason.
    window.matchMedia = vi.fn().mockReturnValue({
      matches: false,
    }) as unknown as typeof window.matchMedia;
  });
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("renders a fetched listing in the grid and shows its detail when selected", async () => {
    vi.spyOn(api, "listListings").mockResolvedValue([activeAuction]);
    vi.spyOn(api, "bidHistory").mockResolvedValue([]);
    const user = userEvent.setup();
    renderShop({
      username: "demo-buyer",
      groups: ["buyers"],
      role: "buyers",
      fixtureId: "buyer-1",
    });

    const card = await screen.findByText("1965 Rolex Submariner");
    await user.click(card);
    expect(
      await screen.findByText("Vintage automatic dive watch."),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Place Bid" }),
    ).toBeInTheDocument();
  });

  it("redirects to /account/login instead of placing a bid when logged out", async () => {
    vi.spyOn(api, "listListings").mockResolvedValue([activeAuction]);
    vi.spyOn(api, "bidHistory").mockResolvedValue([]);
    const placeBidSpy = vi.spyOn(api, "placeBid");
    const user = userEvent.setup();

    const originalLocation = window.location;
    // @ts-expect-error -- replacing window.location for assertion
    delete window.location;
    // @ts-expect-error -- test double
    window.location = { href: "" };

    renderShop(null);
    await user.click(await screen.findByText("1965 Rolex Submariner"));
    await user.type(
      await screen.findByPlaceholderText("Bid amount ($)"),
      "5300",
    );
    await user.click(screen.getByRole("button", { name: "Place Bid" }));

    expect(window.location.href).toBe("/account/login");
    expect(placeBidSpy).not.toHaveBeenCalled();

    // @ts-expect-error -- restoring the real window.location
    window.location = originalLocation;
  });

  it("places a bid when logged in and clears the error on success", async () => {
    vi.spyOn(api, "listListings").mockResolvedValue([activeAuction]);
    vi.spyOn(api, "bidHistory").mockResolvedValue([]);
    const bid: Bid = {
      id: "bid-1",
      listingId: "listing-1",
      buyerId: "buyer-1",
      amountCents: 530000,
      isAutoBid: false,
      placedAt: new Date().toISOString(),
    };
    vi.spyOn(api, "placeBid").mockResolvedValue(bid);
    const user = userEvent.setup();

    renderShop({
      username: "demo-buyer",
      groups: ["buyers"],
      role: "buyers",
      fixtureId: "buyer-1",
    });
    await user.click(await screen.findByText("1965 Rolex Submariner"));
    await user.type(
      await screen.findByPlaceholderText("Bid amount ($)"),
      "5300",
    );
    await user.click(screen.getByRole("button", { name: "Place Bid" }));

    await waitFor(() =>
      expect(api.placeBid).toHaveBeenCalledWith("listing-1", 530000),
    );
  });

  it("shows a login prompt in the orders panel when logged out", async () => {
    vi.spyOn(api, "listListings").mockResolvedValue([]);
    renderShop(null);
    expect(await screen.findByText(/to see your orders/)).toBeInTheDocument();
  });

  it("filters the grid by the search box", async () => {
    const secondListing: Listing = {
      ...activeAuction,
      id: "listing-2",
      title: "Dune hardcover",
      category: "books",
    };
    vi.spyOn(api, "listListings").mockResolvedValue([
      activeAuction,
      secondListing,
    ]);
    const user = userEvent.setup();
    renderShop(null);

    await screen.findByText("1965 Rolex Submariner");
    await user.type(
      screen.getByPlaceholderText("Search by title or category..."),
      "dune",
    );
    expect(screen.queryByText("1965 Rolex Submariner")).not.toBeInTheDocument();
    expect(screen.getByText("Dune hardcover")).toBeInTheDocument();
  });
});
