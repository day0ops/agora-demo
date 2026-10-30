package main

import (
	"errors"
	"fmt"
	"sort"

	"github.com/day0ops/agora-demo/fixtures"
)

// Sentinel errors specific to orders, checkable with errors.Is.
var (
	ErrOrderNotFound       = errors.New("order not found")
	ErrListingNotForSale   = errors.New("listing is not available for purchase")
	ErrNotTheWinningBidder = errors.New("only the winning bidder can check out this auction")
)

// OrderFilter narrows ListOrders. A zero-value field matches any order.
type OrderFilter struct {
	BuyerID  string
	SellerID string
}

// Checkout creates an order for a listing that is purchasable right now:
// either an active buy-now listing (at its PriceCents, which also
// transitions it to Sold), or a closed, won auction (at its
// CurrentBidCents, and only by the buyer who actually holds the winning
// bid). A listing can only ever be checked out once.
func (s *Store) Checkout(listingID, buyerID string) (fixtures.Order, error) {
	if buyerID == "" {
		return fixtures.Order{}, fmt.Errorf("checkout %s: %w", listingID, ErrBuyerIDRequired)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	l, ok := s.listings[listingID]
	if !ok {
		return fixtures.Order{}, fmt.Errorf("checkout %s: %w", listingID, ErrListingNotFound)
	}
	l = s.closeIfExpired(l)

	var finalPriceCents int64
	switch {
	case l.Type == fixtures.ListingTypeBuyNow && l.Status == fixtures.ListingStatusActiveBuyNow:
		finalPriceCents = l.PriceCents
		l.Status = fixtures.ListingStatusSold
		s.listings[listingID] = l

	case l.Type == fixtures.ListingTypeAuction && l.Status == fixtures.ListingStatusSold:
		if s.hasOrder(listingID) {
			return fixtures.Order{}, fmt.Errorf("checkout %s: %w", listingID, ErrListingNotForSale)
		}
		winner := s.winningBidder(listingID)
		if winner == "" || winner != buyerID {
			return fixtures.Order{}, fmt.Errorf("checkout %s: %w", listingID, ErrNotTheWinningBidder)
		}
		finalPriceCents = l.CurrentBidCents

	default:
		return fixtures.Order{}, fmt.Errorf("checkout %s: %w", listingID, ErrListingNotForSale)
	}

	order := fixtures.Order{
		ID:              s.newID("order"),
		ListingID:       listingID,
		BuyerID:         buyerID,
		SellerID:        l.SellerID,
		FinalPriceCents: finalPriceCents,
		Status:          fixtures.OrderStatusPending,
		CreatedAt:       s.now(),
	}
	s.orders[order.ID] = order
	return order, nil
}

// winningBidder returns the buyer with the highest recorded bid on a
// listing, or "" if there are none. Must be called with s.mu held.
func (s *Store) winningBidder(listingID string) string {
	var winner string
	var highest int64
	for _, b := range s.bids[listingID] {
		if b.AmountCents > highest {
			highest = b.AmountCents
			winner = b.BuyerID
		}
	}
	return winner
}

// hasOrder reports whether listingID has already been checked out. Must be
// called with s.mu held. A buy-now listing can't reach this path twice
// because checkout flips its status away from ActiveBuyNow on the first
// success; a won auction's listing status never changes on checkout, so this
// explicit check is what prevents the same auction being checked out twice.
func (s *Store) hasOrder(listingID string) bool {
	for _, o := range s.orders {
		if o.ListingID == listingID {
			return true
		}
	}
	return false
}

// GetOrder returns the order with the given ID.
func (s *Store) GetOrder(id string) (fixtures.Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	o, ok := s.orders[id]
	if !ok {
		return fixtures.Order{}, fmt.Errorf("get order %s: %w", id, ErrOrderNotFound)
	}
	return o, nil
}

// ListOrders returns every order matching filter, ordered by creation time
// then ID for a stable, reproducible response order.
func (s *Store) ListOrders(filter OrderFilter) []fixtures.Order {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := make([]fixtures.Order, 0, len(s.orders))
	for _, o := range s.orders {
		if filter.BuyerID != "" && o.BuyerID != filter.BuyerID {
			continue
		}
		if filter.SellerID != "" && o.SellerID != filter.SellerID {
			continue
		}
		result = append(result, o)
	}

	sort.Slice(result, func(i, j int) bool {
		if !result[i].CreatedAt.Equal(result[j].CreatedAt) {
			return result[i].CreatedAt.Before(result[j].CreatedAt)
		}
		return result[i].ID < result[j].ID
	})
	return result
}
