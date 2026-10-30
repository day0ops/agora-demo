export type ListingType = "buy-now" | "auction";

export type ListingStatus =
  | "draft"
  | "active-auction"
  | "active-buy-now"
  | "sold"
  | "unsold-expired"
  | "withdrawn";

// Mirrors fixtures.Listing (fixtures/types.go). Monetary fields are cents.
export interface Listing {
  id: string;
  sellerId: string;
  title: string;
  description: string;
  category: string;
  type: ListingType;
  status: ListingStatus;
  priceCents: number;
  currentBidCents: number;
  auctionEndsAt?: string;
  createdAt: string;
  image?: string;
}

// Mirrors fixtures.Bid.
export interface Bid {
  id: string;
  listingId: string;
  buyerId: string;
  amountCents: number;
  isAutoBid: boolean;
  maxAutoBidCents?: number;
  placedAt: string;
  trustCheck?: TrustCheckResult;
  correlationId?: string;
}

// Mirrors fixtures.TrustCheckResult.
export interface TrustCheckResult {
  score: number;
  reasons: string[];
  scoredBy: string;
  checkedAt: string;
}

export type OrderStatus =
  "pending" | "paid" | "shipped" | "completed" | "disputed";

// Mirrors fixtures.Order.
export interface Order {
  id: string;
  listingId: string;
  buyerId: string;
  sellerId: string;
  finalPriceCents: number;
  status: OrderStatus;
  createdAt: string;
  correlationId?: string;
}

// Mirrors the console backend's Event envelope (console/events.go) and the
// payloads each handler publishes (console/handlers.go).
export type MarketplaceEvent =
  | { type: "listing.created"; data: Listing }
  | { type: "bid.placed"; data: { listingId: string; bid: Bid } }
  | { type: "bid.withdrawn"; data: { listingId: string; bidId: string } }
  | { type: "autobid.set"; data: { listingId: string; listing: Listing } }
  | { type: "order.created"; data: Order };
