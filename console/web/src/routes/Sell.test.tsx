import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { MemoryRouter } from "react-router-dom";
import { Sell } from "./Sell";
import { IdentityProvider } from "../hooks/useIdentity";
import * as api from "../api";

function renderSell(identityResponse: unknown) {
  vi.spyOn(global, "fetch").mockResolvedValue({
    ok: identityResponse !== null,
    json: () => Promise.resolve(identityResponse ?? {}),
  } as Response);
  return render(
    <MemoryRouter>
      <IdentityProvider>
        <Sell />
      </IdentityProvider>
    </MemoryRouter>,
  );
}

class MockEventSource {
  close() {}
  constructor(public url: string) {}
}

describe("Sell", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    // @ts-expect-error -- test double
    global.EventSource = MockEventSource;
    // jsdom has no matchMedia; ChatPanel (rendered by Sell) reads it on mount
    // to pick its default collapsed state. See ChatPanel.test.tsx for the
    // same stub, needed for the same reason.
    window.matchMedia = vi.fn().mockReturnValue({
      matches: false,
    }) as unknown as typeof window.matchMedia;
  });

  it("shows a denial state for a buyer", async () => {
    vi.spyOn(api, "listListings").mockResolvedValue([]);
    renderSell({
      username: "demo-buyer",
      groups: ["buyers"],
      role: "buyers",
      fixtureId: "buyer-1",
    });
    expect(
      await screen.findByText("You need a seller account"),
    ).toBeInTheDocument();
  });

  it("shows a denial state when logged out", async () => {
    renderSell(null);
    expect(
      await screen.findByText("You need a seller account"),
    ).toBeInTheDocument();
  });

  it("shows the Pro badge and the seller's own listings for a pro seller", async () => {
    vi.spyOn(api, "listListings").mockResolvedValue([
      {
        id: "listing-1",
        sellerId: "seller-1",
        title: "Marina's watch",
        description: "x",
        category: "watches",
        type: "auction",
        status: "active-auction",
        priceCents: 100000,
        currentBidCents: 0,
        createdAt: new Date().toISOString(),
      },
    ]);
    renderSell({
      username: "demo-seller",
      groups: ["sellers"],
      role: "sellers",
      fixtureId: "seller-1",
      sellerTier: "pro",
    });
    expect(await screen.findByText("Marina's watch")).toBeInTheDocument();
    expect(screen.getByText("Pro pricing assistant")).toBeInTheDocument();
  });

  it("creates a listing via the form without a sellerId field in the payload", async () => {
    vi.spyOn(api, "listListings").mockResolvedValue([]);
    const createListingSpy = vi.spyOn(api, "createListing").mockResolvedValue({
      id: "listing-2",
      sellerId: "seller-1",
      title: "New item",
      description: "d",
      category: "home",
      type: "buy-now",
      status: "active-buy-now",
      priceCents: 1000,
      currentBidCents: 0,
      createdAt: new Date().toISOString(),
    });
    const user = userEvent.setup();
    renderSell({
      username: "demo-seller",
      groups: ["sellers"],
      role: "sellers",
      fixtureId: "seller-1",
    });

    await screen.findByText("New listing");
    await user.type(screen.getByLabelText("Title"), "New item");
    await user.type(screen.getByLabelText("Category"), "home");
    await user.selectOptions(screen.getByLabelText("Type"), "buy-now");
    await user.type(screen.getByLabelText("Price ($)"), "10");
    await user.click(screen.getByRole("button", { name: "Create Listing" }));

    expect(createListingSpy).toHaveBeenCalledWith(
      expect.objectContaining({
        title: "New item",
        category: "home",
        type: "buy-now",
        priceCents: 1000,
      }),
    );
    expect(createListingSpy.mock.calls[0][0]).not.toHaveProperty("sellerId");
  });
});
