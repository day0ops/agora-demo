import { useCallback, useEffect, useState } from "react";
import { Navbar } from "../components/Navbar";
import { Footer } from "../components/Footer";
import { LotCard } from "../components/LotCard";
import { ChatPanel } from "../components/ChatPanel";
import { EmptyState } from "../components/EmptyState";
import { ListingImage } from "../components/ListingImage";
import { StatusPill } from "../components/StatusPill";
import {
  useExecutionDrawer,
  type OpenExecutionDrawerOptions,
} from "../components/ExecutionDrawer";
import { IconShop, IconScan } from "../icons";
import { useIdentity } from "../hooks/useIdentity";
import { useEvents } from "../hooks/useEvents";
import {
  listListings,
  getListing,
  bidHistory,
  placeBid,
  setAutoBidMax,
  withdrawBid,
  checkout,
  listOrders,
  ApiError,
} from "../api";
import { formatCents, lotNumber } from "../format";
import type { Listing, Bid, Order } from "../types";

function highestBid(bids: Bid[]): Bid | undefined {
  return bids.reduce<Bid | undefined>(
    (max, b) => (!max || b.amountCents > max.amountCents ? b : max),
    undefined,
  );
}

function DetailPanel({
  listing,
  buyerId,
  requireLogin,
  onOrdersChanged,
  onOpenExecutionDrawer,
}: {
  listing: Listing | null;
  buyerId: string | null;
  requireLogin: () => boolean;
  onOrdersChanged: () => void;
  onOpenExecutionDrawer: (opts: OpenExecutionDrawerOptions) => void;
}) {
  const [bids, setBids] = useState<Bid[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [bidAmount, setBidAmount] = useState("");
  const [autobidAmount, setAutobidAmount] = useState("");

  useEffect(() => {
    setError(null);
    setBidAmount("");
    setAutobidAmount("");
    if (listing?.type === "auction") {
      bidHistory(listing.id)
        .then(setBids)
        .catch(() => setBids([]));
    } else {
      setBids([]);
    }
  }, [listing?.id, listing?.type]);

  if (!listing) {
    return (
      <div className="panel detail">
        <h2>Listing</h2>
        <div className="empty">
          Select a listing to view details and place a bid.
        </div>
      </div>
    );
  }

  const price =
    listing.type === "auction"
      ? listing.currentBidCents || listing.priceCents
      : listing.priceCents;
  const isAuction = listing.type === "auction";
  const isOpen =
    listing.status === "active-auction" || listing.status === "active-buy-now";
  const isSold = listing.status === "sold";
  const isWon = isAuction && isSold;
  const winningBid = isSold ? highestBid(bids) : undefined;
  const isPassed =
    listing.status === "unsold-expired" || listing.status === "withdrawn";

  async function withAuth(action: () => Promise<void>): Promise<void> {
    if (!requireLogin()) return;
    setError(null);
    try {
      await action();
    } catch (err) {
      setError(
        err instanceof ApiError
          ? err.message
          : "Something went wrong. Please try again.",
      );
    }
  }

  return (
    <div className="panel detail">
      <h2>Listing</h2>
      <div className="detail">
        <ListingImage listing={listing} eager />
        {isSold && <span className="sold-stamp">Sold</span>}
        {isSold && (
          <button
            type="button"
            className="link behind-the-scenes-btn"
            onClick={() =>
              onOpenExecutionDrawer({
                summary: {
                  title: winningBid
                    ? `${winningBid.buyerId} won "${listing.title}"`
                    : `"${listing.title}" sold`,
                  body: `Final price ${formatCents(price)}.`,
                  trustCheck: winningBid?.trustCheck
                    ? {
                        score: winningBid.trustCheck.score,
                        reasons: winningBid.trustCheck.reasons,
                      }
                    : "unavailable",
                },
                correlationId: winningBid?.correlationId,
              })
            }
          >
            <IconScan /> Behind the scenes
          </button>
        )}
        <span className="lot-plate">{lotNumber(listing.id)}</span>
        <h3>{listing.title}</h3>
        <p className="desc">{listing.description}</p>
        <div className="price-row">
          <span className="price">{formatCents(price)}</span>
          <StatusPill status={listing.status} />
        </div>
        {isAuction && (
          <ul className="bid-history">
            {bids.length ? (
              [...bids].reverse().map((b) => (
                <li key={b.id}>
                  <span className="who">
                    {b.buyerId}
                    {b.isAutoBid ? " (auto)" : ""}
                  </span>
                  <span className="amt">{formatCents(b.amountCents)}</span>
                  {b.buyerId === buyerId && isOpen && (
                    <button
                      type="button"
                      className="link withdraw-bid-btn"
                      onClick={() =>
                        withAuth(() => withdrawBid(listing.id, b.id))
                      }
                    >
                      Withdraw
                    </button>
                  )}
                </li>
              ))
            ) : (
              <li>
                <span className="empty">No bids yet</span>
              </li>
            )}
          </ul>
        )}
        {isWon ? (
          <button
            className="buy-now-btn"
            onClick={() =>
              withAuth(async () => {
                await checkout(listing.id);
                onOrdersChanged();
              })
            }
          >
            Claim Your Win - {formatCents(price)}
          </button>
        ) : isPassed ? (
          <div className="note">This lot passed with no winning bid.</div>
        ) : isSold ? (
          <div className="note">This listing has sold.</div>
        ) : isAuction ? (
          <>
            <form
              className="inline"
              onSubmit={(e) => {
                e.preventDefault();
                void withAuth(async () => {
                  await placeBid(
                    listing.id,
                    Math.round(Number(bidAmount) * 100),
                  );
                });
              }}
            >
              <label className="sr-only" htmlFor="bid-amount">
                Bid amount ($)
              </label>
              <input
                type="number"
                id="bid-amount"
                min="0.01"
                step="0.01"
                placeholder="Bid amount ($)"
                required
                value={bidAmount}
                onChange={(e) => setBidAmount(e.target.value)}
              />
              <button type="submit">Place Bid</button>
            </form>
            <form
              className="inline"
              onSubmit={(e) => {
                e.preventDefault();
                void withAuth(async () => {
                  await setAutoBidMax(
                    listing.id,
                    Math.round(Number(autobidAmount) * 100),
                  );
                });
              }}
            >
              <label className="sr-only" htmlFor="autobid-amount">
                Auto-bid max ($)
              </label>
              <input
                type="number"
                id="autobid-amount"
                min="0.01"
                step="0.01"
                placeholder="Auto-bid max ($)"
                required
                value={autobidAmount}
                onChange={(e) => setAutobidAmount(e.target.value)}
              />
              <button type="submit" className="secondary">
                Set Auto-Bid
              </button>
            </form>
          </>
        ) : (
          <button
            className="buy-now-btn"
            onClick={() =>
              withAuth(async () => {
                await checkout(listing.id);
                onOrdersChanged();
              })
            }
          >
            Buy Now - {formatCents(price)}
          </button>
        )}
        {error && <div className="note error">{error}</div>}
      </div>
    </div>
  );
}

function OrdersPanel({
  orders,
  listings,
  buyerId,
  onOpenExecutionDrawer,
}: {
  orders: Order[];
  listings: Map<string, Listing>;
  buyerId: string | null;
  onOpenExecutionDrawer: (opts: OpenExecutionDrawerOptions) => void;
}) {
  async function showOrder(o: Order): Promise<void> {
    const title = listings.get(o.listingId)?.title ?? o.listingId;
    let winningBid: Bid | undefined;
    try {
      winningBid = highestBid(await bidHistory(o.listingId));
    } catch {
      // best-effort; the drawer still opens with an order-only summary
    }
    onOpenExecutionDrawer({
      summary: {
        title: winningBid
          ? `${o.buyerId} won "${title}"`
          : `${o.buyerId} purchased "${title}"`,
        body: `Final price ${formatCents(o.finalPriceCents)}.`,
        trustCheck: winningBid?.trustCheck
          ? {
              score: winningBid.trustCheck.score,
              reasons: winningBid.trustCheck.reasons,
            }
          : "unavailable",
      },
      correlationId: winningBid?.correlationId || o.correlationId,
    });
  }

  return (
    <div className="panel" id="orders-panel">
      <h2>Your Orders</h2>
      <ul className="orders-ledger">
        {!buyerId ? (
          <li className="empty">
            <a href="/account/login">Log in</a> to see your orders.
          </li>
        ) : orders.length ? (
          [...orders].reverse().map((o) => (
            <li key={o.id}>
              <span className="item">
                {listings.get(o.listingId)?.title ?? o.listingId}
              </span>
              <span className="amt">{formatCents(o.finalPriceCents)}</span>
              <button
                type="button"
                className="link behind-the-scenes-btn"
                onClick={() => void showOrder(o)}
              >
                <IconScan /> Behind the scenes
              </button>
            </li>
          ))
        ) : (
          <li className="empty">No orders yet.</li>
        )}
      </ul>
    </div>
  );
}

export function Shop() {
  const { identity } = useIdentity();
  const [listings, setListings] = useState<Map<string, Listing>>(new Map());
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState("");
  const [categoryFilter, setCategoryFilter] = useState("");
  const [statusFilter, setStatusFilter] = useState<"active" | "sold">("active");
  const [orders, setOrders] = useState<Order[]>([]);
  const [catalogError, setCatalogError] = useState(false);
  const { open: openExecutionDrawer, drawer } = useExecutionDrawer();

  const buyerId = identity?.fixtureId ?? null;

  function requireLogin(): boolean {
    if (buyerId) return true;
    window.location.href = "/account/login";
    return false;
  }

  const loadListings = useCallback(async () => {
    try {
      const fetched =
        statusFilter === "active"
          ? await Promise.all([
              listListings({ status: "active-auction" }),
              listListings({ status: "active-buy-now" }),
            ]).then(([a, b]) => [...a, ...b])
          : await listListings({ status: "sold" });
      setListings(new Map(fetched.map((l) => [l.id, l])));
      setCatalogError(false);
    } catch {
      setCatalogError(true);
    }
  }, [statusFilter]);

  useEffect(() => {
    void loadListings();
  }, [loadListings]);

  const loadOrders = useCallback(async () => {
    if (!buyerId) return;
    try {
      setOrders(await listOrders());
    } catch {
      // best-effort; the ledger just keeps its last known state
    }
  }, [buyerId]);

  useEffect(() => {
    void loadOrders();
  }, [loadOrders]);

  function refreshListing(id: string): void {
    getListing(id)
      .then((l) => setListings((prev) => new Map(prev).set(l.id, l)))
      .catch(() => {
        // best-effort refresh; the next SSE event or poll will retry
      });
  }

  useEvents((evt) => {
    switch (evt.type) {
      case "listing.created":
        setListings((prev) => new Map(prev).set(evt.data.id, evt.data));
        break;
      case "bid.placed":
      case "bid.withdrawn":
        refreshListing(evt.data.listingId);
        break;
      case "autobid.set":
        setListings((prev) =>
          new Map(prev).set(evt.data.listing.id, evt.data.listing),
        );
        break;
      case "order.created":
        refreshListing(evt.data.listingId);
        if (evt.data.buyerId === buyerId) void loadOrders();
        break;
    }
  });

  const matchingStatus = [...listings.values()]
    .filter((l) =>
      statusFilter === "active"
        ? l.status === "active-auction" || l.status === "active-buy-now"
        : l.status === "sold",
    )
    .sort((a, b) => (a.createdAt < b.createdAt ? 1 : -1));

  const categories = [...new Set(matchingStatus.map((l) => l.category))].sort();
  const effectiveCategoryFilter = categories.includes(categoryFilter)
    ? categoryFilter
    : "";

  const visible = matchingStatus.filter((l) => {
    const q = searchQuery.trim().toLowerCase();
    const matchesSearch =
      !q ||
      l.title.toLowerCase().includes(q) ||
      l.category.toLowerCase().includes(q);
    const matchesCategory =
      !effectiveCategoryFilter || l.category === effectiveCategoryFilter;
    return matchesSearch && matchesCategory;
  });

  const selected = selectedId ? (listings.get(selectedId) ?? null) : null;

  return (
    <>
      <Navbar active="shop" />
      <main className="page">
        <div className="layout">
          <div className="panel">
            <h2>Live auctions &amp; buy-now listings</h2>
            <div className="shop-filters">
              <label className="sr-only" htmlFor="search-input">
                Search listings
              </label>
              <input
                type="search"
                id="search-input"
                placeholder="Search by title or category..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
              />
              <label className="sr-only" htmlFor="category-select">
                Filter by category
              </label>
              <select
                id="category-select"
                value={effectiveCategoryFilter}
                onChange={(e) => setCategoryFilter(e.target.value)}
              >
                <option value="">All categories</option>
                {categories.map((c) => (
                  <option key={c} value={c}>
                    {c}
                  </option>
                ))}
              </select>
              <label className="sr-only" htmlFor="status-filter">
                Filter by status
              </label>
              <select
                id="status-filter"
                value={statusFilter}
                onChange={(e) => {
                  setStatusFilter(e.target.value as "active" | "sold");
                  setSelectedId(null);
                }}
              >
                <option value="active">Active listings</option>
                <option value="sold">Sold</option>
              </select>
            </div>
            <div className="grid">
              {catalogError ? (
                <EmptyState
                  title="Catalog unavailable"
                  body="Could not reach catalog-api. Try refreshing in a moment."
                  icon={<IconShop />}
                />
              ) : visible.length ? (
                visible.map((l) => (
                  <LotCard
                    key={l.id}
                    listing={l}
                    selected={l.id === selectedId}
                    interactive
                    onSelect={() => setSelectedId(l.id)}
                    onExpire={() => refreshListing(l.id)}
                  />
                ))
              ) : (
                <EmptyState
                  title={
                    matchingStatus.length
                      ? "No matching listings"
                      : statusFilter === "active"
                        ? "No active listings"
                        : "No sold listings yet"
                  }
                  body={
                    matchingStatus.length
                      ? "Try a different search or category."
                      : statusFilter === "active"
                        ? "Check back soon as sellers list items."
                        : "Sold listings will appear here once something sells."
                  }
                  icon={<IconShop />}
                />
              )}
            </div>
          </div>
          <div className="panel-stack">
            <DetailPanel
              listing={selected}
              buyerId={buyerId}
              requireLogin={requireLogin}
              onOrdersChanged={() => void loadOrders()}
              onOpenExecutionDrawer={openExecutionDrawer}
            />
            <div className="panel chat-panel">
              <h2>Shopping Assistant</h2>
              <ChatPanel
                agent="shopping-assistant"
                label="Shopping Assistant"
              />
            </div>
          </div>
        </div>
        <OrdersPanel
          orders={orders}
          listings={listings}
          buyerId={buyerId}
          onOpenExecutionDrawer={openExecutionDrawer}
        />
      </main>
      <Footer />
      {drawer}
    </>
  );
}
