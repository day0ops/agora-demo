import { useCallback, useEffect, useState, type FormEvent } from "react";
import { Navbar } from "../components/Navbar";
import { Footer } from "../components/Footer";
import { ChatPanel } from "../components/ChatPanel";
import { LotCard } from "../components/LotCard";
import { EmptyState } from "../components/EmptyState";
import { IconSell, IconTag, IconGavel } from "../icons";
import {
  useIdentity,
  canSell,
  type AccountIdentity,
} from "../hooks/useIdentity";
import { useEvents } from "../hooks/useEvents";
import { listListings, createListing, ApiError } from "../api";
import type { Listing, ListingType } from "../types";

const AUCTION_DURATIONS = [
  { label: "90 seconds (demo)", ms: 90 * 1000 },
  { label: "5 minutes", ms: 5 * 60 * 1000 },
  { label: "15 minutes", ms: 15 * 60 * 1000 },
  { label: "1 hour", ms: 60 * 60 * 1000 },
];

const STATUS_RANK: Record<string, number> = {
  "active-auction": 0,
  "active-buy-now": 0,
  draft: 1,
  sold: 2,
  "unsold-expired": 3,
  withdrawn: 4,
};

function SellerContent({ identity }: { identity: AccountIdentity }) {
  const sellerId = identity.fixtureId!;
  const [listings, setListings] = useState<Map<string, Listing>>(new Map());
  const [catalogError, setCatalogError] = useState(false);
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [category, setCategory] = useState("");
  const [type, setType] = useState<ListingType>("auction");
  const [price, setPrice] = useState("");
  const [durationIndex, setDurationIndex] = useState(0);
  const [createError, setCreateError] = useState<string | null>(null);

  const loadListings = useCallback(async () => {
    try {
      const mine = await listListings({ sellerId });
      setListings(new Map(mine.map((l) => [l.id, l])));
      setCatalogError(false);
    } catch {
      setCatalogError(true);
    }
  }, [sellerId]);

  useEffect(() => {
    void loadListings();
  }, [loadListings]);

  useEvents((evt) => {
    switch (evt.type) {
      case "listing.created":
        if (evt.data.sellerId === sellerId) {
          setListings((prev) => new Map(prev).set(evt.data.id, evt.data));
        }
        break;
      case "bid.placed":
      case "bid.withdrawn":
      case "order.created":
        void loadListings();
        break;
      case "autobid.set":
        if (evt.data.listing.sellerId === sellerId) {
          setListings((prev) =>
            new Map(prev).set(evt.data.listing.id, evt.data.listing),
          );
        }
        break;
    }
  });

  const mine = [...listings.values()]
    .filter((l) => l.sellerId === sellerId)
    .sort((a, b) => {
      const rankDiff =
        (STATUS_RANK[a.status] ?? 9) - (STATUS_RANK[b.status] ?? 9);
      if (rankDiff !== 0) return rankDiff;
      return a.createdAt < b.createdAt ? 1 : -1;
    });

  async function handleSubmit(e: FormEvent): Promise<void> {
    e.preventDefault();
    setCreateError(null);
    try {
      await createListing({
        title,
        description,
        category,
        type,
        priceCents: Math.round(Number(price) * 100),
        auctionEndsAt:
          type === "auction"
            ? new Date(
                Date.now() + AUCTION_DURATIONS[durationIndex].ms,
              ).toISOString()
            : undefined,
      });
      setTitle("");
      setDescription("");
      setCategory("");
      setType("auction");
      setPrice("");
      setDurationIndex(0);
    } catch (err) {
      setCreateError(
        err instanceof ApiError
          ? err.message
          : "Something went wrong. Please try again.",
      );
    }
  }

  return (
    <div className="layout">
      <div className="panel">
        <h2>My listings</h2>
        <div className="grid">
          {catalogError ? (
            <EmptyState
              title="Catalog unavailable"
              body="Could not reach catalog-api. Try refreshing in a moment."
              icon={<IconSell />}
            />
          ) : mine.length ? (
            mine.map((l) => <LotCard key={l.id} listing={l} />)
          ) : (
            <EmptyState
              title="No listings yet"
              body="Create a listing below to see it appear here."
              icon={<IconSell />}
            />
          )}
        </div>
      </div>
      <div className="panel-stack">
        <div className="panel" id="create-panel">
          <h2>New listing</h2>
          <div className="form-step-label">Step 1 &middot; Draft details</div>
          <form onSubmit={handleSubmit}>
            <label htmlFor="f-title">Title</label>
            <input
              id="f-title"
              required
              value={title}
              onChange={(e) => setTitle(e.target.value)}
            />
            <label htmlFor="f-description">Description</label>
            <textarea
              id="f-description"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
            />
            <label htmlFor="f-category">Category</label>
            <input
              id="f-category"
              required
              value={category}
              onChange={(e) => setCategory(e.target.value)}
            />
            <label htmlFor="f-type">Type</label>
            <select
              id="f-type"
              value={type}
              onChange={(e) => setType(e.target.value as ListingType)}
            >
              <option value="auction">Auction</option>
              <option value="buy-now">Buy now</option>
            </select>
            {type === "auction" && (
              <div id="f-duration-wrap">
                <label htmlFor="f-duration">Auction duration</label>
                <select
                  id="f-duration"
                  value={durationIndex}
                  onChange={(e) => setDurationIndex(Number(e.target.value))}
                >
                  {AUCTION_DURATIONS.map((d, i) => (
                    <option key={d.label} value={i}>
                      {d.label}
                    </option>
                  ))}
                </select>
              </div>
            )}
            <label htmlFor="f-price">Price ($)</label>
            <input
              id="f-price"
              type="number"
              min="0.01"
              step="0.01"
              required
              value={price}
              onChange={(e) => setPrice(e.target.value)}
            />

            <div className="assist-stack">
              <div className="assist-card">
                <EmptyState
                  title="Recommended pricing"
                  body="Price suggestions appear here once the pricing assistant is connected to a live agent cluster."
                  icon={<IconTag />}
                />
              </div>
              <div className="assist-card">
                <EmptyState
                  title="Comparable items"
                  body="Similar listings and recent sale prices appear here once the pricing assistant is connected to a live agent cluster."
                  icon={<IconGavel />}
                />
              </div>
            </div>

            <div className="review-step">
              <div className="form-step-label">
                Step 2 &middot; Review &amp; publish
              </div>
              <p className="review-step-note">
                Confirm the details above, then publish your listing.
              </p>
              <button type="submit" className="create-submit">
                Create Listing
              </button>
            </div>
            {createError && <div className="note error">{createError}</div>}
          </form>
        </div>
        <div className="panel chat-panel">
          <h2>Pricing Assistant</h2>
          {identity.sellerTier === "pro" && (
            <span className="pill accent pro-badge">Pro pricing assistant</span>
          )}
          <ChatPanel
            agent="listing-pricing-assistant"
            label="Pricing Assistant"
          />
        </div>
      </div>
    </div>
  );
}

export function Sell() {
  const { identity, loading } = useIdentity();

  return (
    <>
      <Navbar active="sell" />
      <main className="page">
        {loading ? (
          <div className="empty">Loading...</div>
        ) : identity && canSell(identity) ? (
          <SellerContent identity={identity} />
        ) : (
          <EmptyState
            title="You need a seller account"
            body="Log in as a seller to list items and manage your storefront."
            icon={<IconSell />}
          />
        )}
      </main>
      <Footer />
    </>
  );
}
