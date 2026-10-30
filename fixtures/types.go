// Package fixtures provides the shared synthetic marketplace dataset:
// sellers, buyers, listings, bids, and orders used across the catalog API,
// MCP servers, and console. All data is synthetic.
package fixtures

import "time"

// SellerTier identifies a seller's subscription plan, which determines the
// LLM model tier used for their AI pricing assistant.
type SellerTier string

const (
	SellerTierBasic SellerTier = "basic"
	SellerTierPro   SellerTier = "pro"
)

// Seller is a marketplace seller account.
type Seller struct {
	ID   string     `json:"id"`
	Name string     `json:"name"`
	Tier SellerTier `json:"tier"`
}

// Buyer is a marketplace buyer account.
type Buyer struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ListingType distinguishes a fixed-price listing from a live auction.
type ListingType string

const (
	ListingTypeBuyNow  ListingType = "buy-now"
	ListingTypeAuction ListingType = "auction"
)

// ListingStatus tracks a listing through its lifecycle.
type ListingStatus string

const (
	ListingStatusDraft         ListingStatus = "draft"
	ListingStatusActiveAuction ListingStatus = "active-auction"
	ListingStatusActiveBuyNow  ListingStatus = "active-buy-now"
	ListingStatusSold          ListingStatus = "sold"
	ListingStatusUnsoldExpired ListingStatus = "unsold-expired"
	ListingStatusWithdrawn     ListingStatus = "withdrawn"
)

// Listing is a single item for sale, either at a fixed price or at auction.
// Monetary fields are in cents to avoid floating-point rounding.
type Listing struct {
	ID              string        `json:"id"`
	SellerID        string        `json:"sellerId"`
	Title           string        `json:"title"`
	Description     string        `json:"description"`
	Category        string        `json:"category"`
	Type            ListingType   `json:"type"`
	Status          ListingStatus `json:"status"`
	PriceCents      int64         `json:"priceCents"`      // buy-now price, or an auction's starting price
	CurrentBidCents int64         `json:"currentBidCents"` // 0 until the first bid lands (auction only)
	AuctionEndsAt   *time.Time    `json:"auctionEndsAt,omitempty"`
	CreatedAt       time.Time     `json:"createdAt"`
	// Image is a slug into the console's committed image set
	// (console/web/public/images/listings/<Image>.jpg). Empty for
	// user-created listings, which fall back to a category SVG.
	Image string `json:"image,omitempty"`
}

// TrustCheckResult is a fraud/shill-bid risk assessment recorded against a
// bid at the moment it was placed, produced by the trust-scoring package.
// Nil when no trust check could be completed - never a fabricated or
// default-zero result standing in for a real one.
type TrustCheckResult struct {
	Score     float64   `json:"score"`
	Reasons   []string  `json:"reasons"`
	ScoredBy  string    `json:"scoredBy"`
	CheckedAt time.Time `json:"checkedAt"`
}

// Bid is a single bid placed against an auction listing. IsAutoBid and
// MaxAutoBidCents describe whether a bid was placed automatically by the
// proxy-bidding engine on a buyer's behalf, and the ceiling it was bidding
// up to; they are historical/display metadata only, not inputs to placing a
// manual bid.
type Bid struct {
	ID              string    `json:"id"`
	ListingID       string    `json:"listingId"`
	BuyerID         string    `json:"buyerId"`
	AmountCents     int64     `json:"amountCents"`
	IsAutoBid       bool      `json:"isAutoBid"`
	MaxAutoBidCents int64     `json:"maxAutoBidCents,omitempty"`
	PlacedAt        time.Time `json:"placedAt"`
	// TrustCheck is the fraud/shill-bid risk assessment for this bid, nil if
	// none could be completed. See TrustCheckResult.
	TrustCheck *TrustCheckResult `json:"trustCheck,omitempty"`
	// CorrelationID links this bid to the agent execution that placed it, if
	// any. Empty until an agent is wired to set it (not in this phase).
	CorrelationID string `json:"correlationId,omitempty"`
}

// OrderStatus tracks a completed transaction through fulfillment.
type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "pending"
	OrderStatusPaid      OrderStatus = "paid"
	OrderStatusShipped   OrderStatus = "shipped"
	OrderStatusCompleted OrderStatus = "completed"
	OrderStatusDisputed  OrderStatus = "disputed"
)

// Order is a completed transaction between a buyer and seller for a listing.
type Order struct {
	ID              string      `json:"id"`
	ListingID       string      `json:"listingId"`
	BuyerID         string      `json:"buyerId"`
	SellerID        string      `json:"sellerId"`
	FinalPriceCents int64       `json:"finalPriceCents"`
	Status          OrderStatus `json:"status"`
	CreatedAt       time.Time   `json:"createdAt"`
	// CorrelationID links this order to the agent execution that produced
	// it, if any. Empty until an agent is wired to set it (not in this phase).
	CorrelationID string `json:"correlationId,omitempty"`
}
