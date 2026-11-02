import { render, screen, fireEvent } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";
import { LotCard } from "./LotCard";
import type { Listing } from "../types";

const listing: Listing = {
  id: "listing-1",
  sellerId: "seller-1",
  title: "1965 Rolex Submariner",
  description: "Vintage automatic dive watch.",
  category: "watches",
  type: "auction",
  status: "active-auction",
  priceCents: 500000,
  currentBidCents: 520000,
  auctionEndsAt: new Date(Date.now() + 90_000).toISOString(),
  createdAt: new Date().toISOString(),
};

describe("LotCard", () => {
  it("renders the listing's title, price, and lot number", () => {
    render(<LotCard listing={listing} />);
    expect(screen.getByText("1965 Rolex Submariner")).toBeInTheDocument();
    expect(screen.getByText("$5,200.00")).toBeInTheDocument();
    expect(screen.getByText("LOT 001")).toBeInTheDocument();
  });

  it("is not a focusable button when interactive is false", () => {
    render(<LotCard listing={listing} />);
    expect(screen.queryByRole("button")).not.toBeInTheDocument();
  });

  it("calls onSelect on click and on Enter when interactive", () => {
    const onSelect = vi.fn();
    render(<LotCard listing={listing} interactive onSelect={onSelect} />);
    const card = screen.getByRole("button");
    fireEvent.click(card);
    fireEvent.keyDown(card, { key: "Enter" });
    expect(onSelect).toHaveBeenCalledTimes(2);
  });
});
