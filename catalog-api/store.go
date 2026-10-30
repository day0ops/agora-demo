// Command catalog-api serves the marketplace's listings and bids as a REST
// API. It is the single source of truth for marketplace domain data: the
// purpose-built MCP servers (bids-mcp, pricing-mcp, trust-mcp, orders-mcp)
// call into it rather than holding their own copy, and its REST surface
// doubles as the OpenAPI source agentgateway wraps into listings-mcp with no
// MCP server code of its own.
package main

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/day0ops/agora-demo/fixtures"
)

// Sentinel errors returned by Store, checkable with errors.Is.
var (
	ErrListingNotFound    = errors.New("listing not found")
	ErrBidNotFound        = errors.New("bid not found")
	ErrListingNotAuction  = errors.New("listing is not an auction")
	ErrAuctionClosed      = errors.New("auction is not open for bidding")
	ErrBidTooLow          = errors.New("bid amount is too low")
	ErrBuyerIDRequired    = errors.New("buyerId is required")
	ErrSellerIDRequired   = errors.New("sellerId is required")
	ErrTitleRequired      = errors.New("title is required")
	ErrInvalidPrice       = errors.New("priceCents must be greater than zero")
	ErrAuctionEndRequired = errors.New("auctionEndsAt is required for an auction listing")
	ErrAuctionEndInPast   = errors.New("auctionEndsAt must be in the future")
	ErrInvalidListingType = errors.New("type must be \"auction\" or \"buy-now\"")
)

var idCounter atomic.Int64

// newDefaultID returns a process-wide-unique ID. An atomic counter, not a
// timestamp: a burst of concurrent bids (the auction-closing-surge scenario
// this demo exists to show) can land within the same clock tick, so a
// time-based ID risks collisions that a counter cannot.
func newDefaultID(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, idCounter.Add(1))
}

// seedIDFloor scans seeded IDs for the highest numeric suffix (fixtures.Seed
// hardcodes IDs like "listing-1") so NewStore can raise idCounter above it
// before generating any real ID. Without this, idCounter's zero start value
// made the very first listing/bid/order created after startup collide with
// a seeded ID and silently overwrite it.
func seedIDFloor(listings []fixtures.Listing, bids []fixtures.Bid) int64 {
	var max int64
	consider := func(id string) {
		idx := strings.LastIndexByte(id, '-')
		if idx < 0 {
			return
		}
		if n, err := strconv.ParseInt(id[idx+1:], 10, 64); err == nil && n > max {
			max = n
		}
	}
	for _, l := range listings {
		consider(l.ID)
	}
	for _, b := range bids {
		consider(b.ID)
	}
	return max
}

// bidIncrementCents is the flat amount proxy bidding settles one bid above
// the second-highest standing ceiling. A real marketplace would use a
// price-tiered schedule; a flat increment keeps this demo's resolution
// predictable and easy to reason about.
const bidIncrementCents = 100

// ListingFilter narrows ListListings. A zero-value field matches any listing.
type ListingFilter struct {
	Status   fixtures.ListingStatus
	SellerID string
}

// autoBidCeiling is a buyer's standing willingness to pay for a listing, plus
// the order in which it was first registered -- used to break ties the way a
// real auction would, in favor of whoever registered first.
type autoBidCeiling struct {
	maxCents int64
	seq      int64
}

// Store is an in-memory, thread-safe catalog of listings and bids.
type Store struct {
	mu             sync.Mutex
	listings       map[string]fixtures.Listing
	bids           map[string][]fixtures.Bid            // keyed by listing ID
	autoBids       map[string]map[string]autoBidCeiling // listingID -> buyerID -> ceiling
	orders         map[string]fixtures.Order            // keyed by order ID
	nextCeilingSeq int64
	newID          func(prefix string) string
	now            func() time.Time
}

// NewStore returns a Store seeded with the given listings and bids.
func NewStore(listings []fixtures.Listing, bids []fixtures.Bid) *Store {
	if floor := seedIDFloor(listings, bids); floor > idCounter.Load() {
		idCounter.Store(floor)
	}
	return newStoreWithIDFunc(listings, bids, newDefaultID)
}

// newStoreWithIDFunc is NewStore with an injectable ID generator, so tests
// can assert on generated IDs deterministically. now defaults to time.Now;
// tests needing control over auction-expiry timing set s.now directly
// (store_test.go is part of this package).
func newStoreWithIDFunc(
	listings []fixtures.Listing,
	bids []fixtures.Bid,
	newID func(prefix string) string,
) *Store {
	if newID == nil {
		newID = newDefaultID
	}
	s := &Store{
		listings: make(map[string]fixtures.Listing, len(listings)),
		bids:     make(map[string][]fixtures.Bid),
		autoBids: make(map[string]map[string]autoBidCeiling),
		orders:   make(map[string]fixtures.Order),
		newID:    newID,
		now:      time.Now,
	}
	for _, l := range listings {
		s.listings[l.ID] = copyListing(l)
	}
	for _, b := range bids {
		s.bids[b.ListingID] = append(s.bids[b.ListingID], b)
	}
	return s
}

// copyListing returns l with AuctionEndsAt pointing at a fresh copy of its
// value, so neither the store nor a caller can corrupt the other's state by
// mutating through a shared *time.Time.
func copyListing(l fixtures.Listing) fixtures.Listing {
	if l.AuctionEndsAt != nil {
		t := *l.AuctionEndsAt
		l.AuctionEndsAt = &t
	}
	return l
}

// closeIfExpired transitions an auction listing to Sold (if it received at
// least one bid) or UnsoldExpired (if it did not) once its end time has
// passed, persisting the change. Must be called with s.mu held.
func (s *Store) closeIfExpired(l fixtures.Listing) fixtures.Listing {
	if l.Type != fixtures.ListingTypeAuction || l.Status != fixtures.ListingStatusActiveAuction {
		return l
	}
	if l.AuctionEndsAt == nil || s.now().Before(*l.AuctionEndsAt) {
		return l
	}
	if l.CurrentBidCents > 0 {
		l.Status = fixtures.ListingStatusSold
	} else {
		l.Status = fixtures.ListingStatusUnsoldExpired
	}
	s.listings[l.ID] = l
	return l
}

// ListListings returns every listing matching filter, ordered by creation
// time then ID for a stable, reproducible response order.
func (s *Store) ListListings(filter ListingFilter) []fixtures.Listing {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := make([]fixtures.Listing, 0, len(s.listings))
	for _, l := range s.listings {
		l = s.closeIfExpired(l)
		if filter.Status != "" && l.Status != filter.Status {
			continue
		}
		if filter.SellerID != "" && l.SellerID != filter.SellerID {
			continue
		}
		result = append(result, copyListing(l))
	}

	sort.Slice(result, func(i, j int) bool {
		if !result[i].CreatedAt.Equal(result[j].CreatedAt) {
			return result[i].CreatedAt.Before(result[j].CreatedAt)
		}
		return result[i].ID < result[j].ID
	})
	return result
}

// GetListing returns the listing with the given ID.
func (s *Store) GetListing(id string) (fixtures.Listing, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	l, ok := s.listings[id]
	if !ok {
		return fixtures.Listing{}, fmt.Errorf("get listing %s: %w", id, ErrListingNotFound)
	}
	return copyListing(s.closeIfExpired(l)), nil
}

// CreateListing adds a new listing with a freshly generated ID. Auction
// listings require a future auctionEndsAt; buy-now listings ignore it.
func (s *Store) CreateListing(
	sellerID, title, description, category string,
	listingType fixtures.ListingType,
	priceCents int64,
	auctionEndsAt *time.Time,
) (fixtures.Listing, error) {
	if sellerID == "" {
		return fixtures.Listing{}, fmt.Errorf("create listing: %w", ErrSellerIDRequired)
	}
	if title == "" {
		return fixtures.Listing{}, fmt.Errorf("create listing: %w", ErrTitleRequired)
	}
	if priceCents <= 0 {
		return fixtures.Listing{}, fmt.Errorf("create listing: %w", ErrInvalidPrice)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	var status fixtures.ListingStatus
	var endsAt *time.Time
	switch listingType {
	case fixtures.ListingTypeAuction:
		if auctionEndsAt == nil {
			return fixtures.Listing{}, fmt.Errorf("create listing: %w", ErrAuctionEndRequired)
		}
		if !auctionEndsAt.After(s.now()) {
			return fixtures.Listing{}, fmt.Errorf("create listing: %w", ErrAuctionEndInPast)
		}
		status = fixtures.ListingStatusActiveAuction
		t := *auctionEndsAt
		endsAt = &t
	case fixtures.ListingTypeBuyNow:
		status = fixtures.ListingStatusActiveBuyNow
	default:
		return fixtures.Listing{}, fmt.Errorf("create listing: %w", ErrInvalidListingType)
	}

	l := fixtures.Listing{
		ID:            s.newID("listing"),
		SellerID:      sellerID,
		Title:         title,
		Description:   description,
		Category:      category,
		Type:          listingType,
		Status:        status,
		PriceCents:    priceCents,
		AuctionEndsAt: endsAt,
		CreatedAt:     s.now(),
	}
	s.listings[l.ID] = l
	return copyListing(l), nil
}

// PlaceBid places a manual bid against an open auction listing. The first
// bid on a listing must meet or exceed its starting price; every bid after
// that must strictly exceed the current price. On success, the listing's
// CurrentBidCents is updated to amountCents.
func (s *Store) PlaceBid(listingID, buyerID string, amountCents int64) (fixtures.Bid, error) {
	if buyerID == "" {
		return fixtures.Bid{}, fmt.Errorf("place bid on %s: %w", listingID, ErrBuyerIDRequired)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	l, ok := s.listings[listingID]
	if !ok {
		return fixtures.Bid{}, fmt.Errorf("place bid on %s: %w", listingID, ErrListingNotFound)
	}
	l = s.closeIfExpired(l)
	if l.Type != fixtures.ListingTypeAuction {
		return fixtures.Bid{}, fmt.Errorf("place bid on %s: %w", listingID, ErrListingNotAuction)
	}
	if l.Status != fixtures.ListingStatusActiveAuction {
		return fixtures.Bid{}, fmt.Errorf("place bid on %s: %w", listingID, ErrAuctionClosed)
	}

	minBid := l.PriceCents
	if l.CurrentBidCents > 0 {
		minBid = l.CurrentBidCents + 1
	}
	if amountCents < minBid {
		return fixtures.Bid{}, fmt.Errorf("place bid on %s: %w", listingID, ErrBidTooLow)
	}

	bid := fixtures.Bid{
		ID:          s.newID("bid"),
		ListingID:   listingID,
		BuyerID:     buyerID,
		AmountCents: amountCents,
		PlacedAt:    s.now(),
	}
	s.bids[listingID] = append(s.bids[listingID], bid)
	l.CurrentBidCents = amountCents

	// A manual bid is also a ceiling of exactly its own amount, so a
	// stronger standing auto-bid ceiling from another buyer immediately
	// outbids it via resolveProxyBidding below.
	s.registerCeiling(listingID, buyerID, amountCents)
	s.listings[listingID] = s.resolveProxyBidding(l)

	return bid, nil
}

// registerCeiling records buyerID's willingness to pay up to maxCents for
// listingID, never lowering a previously registered ceiling. A buyer's first
// registration fixes their tie-break priority (registerCeiling does not
// reset it on a later raise). Must be called with s.mu held.
func (s *Store) registerCeiling(listingID, buyerID string, maxCents int64) {
	if s.autoBids[listingID] == nil {
		s.autoBids[listingID] = map[string]autoBidCeiling{}
	}
	existing, ok := s.autoBids[listingID][buyerID]
	if ok && maxCents <= existing.maxCents {
		return
	}
	seq := existing.seq
	if !ok {
		s.nextCeilingSeq++
		seq = s.nextCeilingSeq
	}
	s.autoBids[listingID][buyerID] = autoBidCeiling{maxCents: maxCents, seq: seq}
}

// resolveProxyBidding recomputes a listing's current price from every
// registered ceiling (a standing auto-bid max from SetAutoBidMax, or a
// buyer's own manual bid amount, which registerCeiling also records as that
// buyer's ceiling). Classic proxy bidding: the price settles at one
// bidIncrementCents above the second-highest ceiling, capped at the top
// ceiling, or at the listing's starting price when only one ceiling is
// registered (nobody is pushing the leader yet); ties on the top ceiling go
// to whichever buyer registered first -- but only when that tie actually
// raises the price. A manual bid that merely matches an existing equal
// ceiling (both at, say, 5000) leaves the current price unchanged (the
// "never lowers the current price" rule below applies to a no-op tie too),
// so the earlier registrant does NOT reclaim the lead in that specific case;
// the bid history and CurrentBidCents stay mutually consistent regardless,
// since both reflect whoever's bid is actually on record. No ceiling below
// the starting price can win the auction at all. When the price changes, an
// auto-generated Bid is recorded for the new leader. Must be called with
// s.mu held.
func (s *Store) resolveProxyBidding(l fixtures.Listing) fixtures.Listing {
	raw := s.autoBids[l.ID]
	if len(raw) == 0 {
		return l
	}

	type bidder struct {
		buyerID  string
		maxCents int64
		seq      int64
	}
	entrants := make([]bidder, 0, len(raw))
	for buyerID, c := range raw {
		entrants = append(entrants, bidder{buyerID: buyerID, maxCents: c.maxCents, seq: c.seq})
	}
	sort.Slice(entrants, func(i, j int) bool {
		if entrants[i].maxCents != entrants[j].maxCents {
			return entrants[i].maxCents > entrants[j].maxCents
		}
		return entrants[i].seq < entrants[j].seq
	})

	top := entrants[0]
	if top.maxCents < l.PriceCents {
		// No registered ceiling can even meet the starting price yet.
		return l
	}

	var secondCeiling int64
	if len(entrants) > 1 {
		secondCeiling = entrants[1].maxCents
	}

	price := l.PriceCents
	if floor := secondCeiling + bidIncrementCents; floor > price {
		price = floor
	}
	if price > top.maxCents {
		price = top.maxCents
	}
	if price <= l.CurrentBidCents {
		return l
	}

	l.CurrentBidCents = price
	s.bids[l.ID] = append(s.bids[l.ID], fixtures.Bid{
		ID:              s.newID("bid"),
		ListingID:       l.ID,
		BuyerID:         top.buyerID,
		AmountCents:     price,
		IsAutoBid:       true,
		MaxAutoBidCents: top.maxCents,
		PlacedAt:        s.now(),
	})
	s.listings[l.ID] = l
	return l
}

// SetAutoBidMax registers or raises a buyer's standing proxy-bid ceiling on
// an open auction, then immediately resolves bidding against every other
// registered ceiling. See resolveProxyBidding.
func (s *Store) SetAutoBidMax(listingID, buyerID string, maxCents int64) (fixtures.Listing, error) {
	if buyerID == "" {
		return fixtures.Listing{}, fmt.Errorf("set auto-bid max on %s: %w", listingID, ErrBuyerIDRequired)
	}
	if maxCents <= 0 {
		return fixtures.Listing{}, fmt.Errorf("set auto-bid max on %s: %w", listingID, ErrInvalidPrice)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	l, ok := s.listings[listingID]
	if !ok {
		return fixtures.Listing{}, fmt.Errorf("set auto-bid max on %s: %w", listingID, ErrListingNotFound)
	}
	l = s.closeIfExpired(l)
	if l.Type != fixtures.ListingTypeAuction {
		return fixtures.Listing{}, fmt.Errorf("set auto-bid max on %s: %w", listingID, ErrListingNotAuction)
	}
	if l.Status != fixtures.ListingStatusActiveAuction {
		return fixtures.Listing{}, fmt.Errorf("set auto-bid max on %s: %w", listingID, ErrAuctionClosed)
	}

	s.registerCeiling(listingID, buyerID, maxCents)
	l = s.resolveProxyBidding(l)
	s.listings[listingID] = l
	return copyListing(l), nil
}

// WithdrawBid removes a specific bid from a listing by its ID. The listing's
// current price reverts to the next-highest remaining bid, or to zero if
// none remain. Only open auctions can have a bid withdrawn.
func (s *Store) WithdrawBid(listingID, bidID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	l, ok := s.listings[listingID]
	if !ok {
		return fmt.Errorf("withdraw bid on %s: %w", listingID, ErrListingNotFound)
	}
	l = s.closeIfExpired(l)
	if l.Status != fixtures.ListingStatusActiveAuction {
		return fmt.Errorf("withdraw bid on %s: %w", listingID, ErrAuctionClosed)
	}

	bids := s.bids[listingID]
	removeIdx := -1
	for i, b := range bids {
		if b.ID == bidID {
			removeIdx = i
			break
		}
	}
	if removeIdx == -1 {
		return fmt.Errorf("withdraw bid %s on %s: %w", bidID, listingID, ErrBidNotFound)
	}
	withdrawnBuyer := bids[removeIdx].BuyerID

	remaining := append(bids[:removeIdx:removeIdx], bids[removeIdx+1:]...)
	s.bids[listingID] = remaining

	// Withdrawing takes the buyer out of this auction entirely. Without this,
	// a stale standing ceiling could silently resurrect them as leader on the
	// very next bid from someone else, with no new action on their part.
	delete(s.autoBids[listingID], withdrawnBuyer)

	var highest int64
	for _, b := range remaining {
		if b.AmountCents > highest {
			highest = b.AmountCents
		}
	}
	l.CurrentBidCents = highest
	s.listings[listingID] = s.resolveProxyBidding(l)

	return nil
}

// BidHistory returns every bid placed on a listing, oldest first.
func (s *Store) BidHistory(listingID string) ([]fixtures.Bid, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.listings[listingID]; !ok {
		return nil, fmt.Errorf("bid history for %s: %w", listingID, ErrListingNotFound)
	}

	history := make([]fixtures.Bid, len(s.bids[listingID]))
	copy(history, s.bids[listingID])
	return history, nil
}
