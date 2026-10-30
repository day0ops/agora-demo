package main

import (
	"errors"
	"testing"
	"time"

	"github.com/day0ops/marketplace-demo/fixtures"
)

func TestCheckoutBuyNowHappyPath(t *testing.T) {
	s := newStoreWithIDFunc([]fixtures.Listing{
		{ID: "l1", SellerID: "seller-1", Type: fixtures.ListingTypeBuyNow, Status: fixtures.ListingStatusActiveBuyNow, PriceCents: 4500},
	}, nil, nil)

	order, err := s.Checkout("l1", "buyer-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if order.ListingID != "l1" || order.BuyerID != "buyer-1" || order.SellerID != "seller-1" {
		t.Fatalf("unexpected order: %+v", order)
	}
	if order.FinalPriceCents != 4500 {
		t.Fatalf("expected final price 4500, got %d", order.FinalPriceCents)
	}
	if order.Status != fixtures.OrderStatusPending {
		t.Fatalf("expected a new order to start Pending, got %q", order.Status)
	}

	l, err := s.GetListing("l1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if l.Status != fixtures.ListingStatusSold {
		t.Fatalf("expected the buy-now listing to transition to sold, got %q", l.Status)
	}
}

func TestCheckoutBuyNowTwiceRejected(t *testing.T) {
	s := newStoreWithIDFunc([]fixtures.Listing{
		{ID: "l1", SellerID: "seller-1", Type: fixtures.ListingTypeBuyNow, Status: fixtures.ListingStatusActiveBuyNow, PriceCents: 4500},
	}, nil, nil)

	if _, err := s.Checkout("l1", "buyer-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := s.Checkout("l1", "buyer-2"); !errors.Is(err, ErrListingNotForSale) {
		t.Fatalf("expected ErrListingNotForSale on a second checkout (listing is now Sold), got %v", err)
	}
}

func TestCheckoutWonAuctionByWinner(t *testing.T) {
	end := time.Now().Add(time.Minute)
	s := newStoreWithIDFunc(
		[]fixtures.Listing{
			{
				ID: "l1", SellerID: "seller-1", Type: fixtures.ListingTypeAuction, Status: fixtures.ListingStatusActiveAuction,
				PriceCents: 1000, CurrentBidCents: 2500, AuctionEndsAt: &end,
			},
		},
		[]fixtures.Bid{
			{ID: "b1", ListingID: "l1", BuyerID: "buyer-1", AmountCents: 1500},
			{ID: "b2", ListingID: "l1", BuyerID: "buyer-2", AmountCents: 2500},
		},
		nil,
	)
	s.now = func() time.Time { return end.Add(time.Second) } // auction has closed

	order, err := s.Checkout("l1", "buyer-2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if order.BuyerID != "buyer-2" || order.FinalPriceCents != 2500 {
		t.Fatalf("expected the winning bidder's order at 2500, got %+v", order)
	}
}

func TestCheckoutWonAuctionTwiceRejected(t *testing.T) {
	// A won auction's listing status never changes on checkout (unlike
	// buy-now, which flips to Sold) -- hasOrder is the only thing standing
	// between this and the same auction selling twice.
	end := time.Now().Add(time.Minute)
	s := newStoreWithIDFunc(
		[]fixtures.Listing{
			{
				ID: "l1", SellerID: "seller-1", Type: fixtures.ListingTypeAuction, Status: fixtures.ListingStatusActiveAuction,
				PriceCents: 1000, CurrentBidCents: 2500, AuctionEndsAt: &end,
			},
		},
		[]fixtures.Bid{{ID: "b1", ListingID: "l1", BuyerID: "buyer-2", AmountCents: 2500}},
		nil,
	)
	s.now = func() time.Time { return end.Add(time.Second) }

	if _, err := s.Checkout("l1", "buyer-2"); err != nil {
		t.Fatalf("unexpected error on first checkout: %v", err)
	}
	if _, err := s.Checkout("l1", "buyer-2"); !errors.Is(err, ErrListingNotForSale) {
		t.Fatalf("expected ErrListingNotForSale on a second checkout of the same won auction, got %v", err)
	}
}

func TestCheckoutWonAuctionByNonWinnerRejected(t *testing.T) {
	end := time.Now().Add(time.Minute)
	s := newStoreWithIDFunc(
		[]fixtures.Listing{
			{
				ID: "l1", Type: fixtures.ListingTypeAuction, Status: fixtures.ListingStatusActiveAuction,
				PriceCents: 1000, CurrentBidCents: 2500, AuctionEndsAt: &end,
			},
		},
		[]fixtures.Bid{
			{ID: "b1", ListingID: "l1", BuyerID: "buyer-1", AmountCents: 1500},
			{ID: "b2", ListingID: "l1", BuyerID: "buyer-2", AmountCents: 2500},
		},
		nil,
	)
	s.now = func() time.Time { return end.Add(time.Second) }

	_, err := s.Checkout("l1", "buyer-1")
	if !errors.Is(err, ErrNotTheWinningBidder) {
		t.Fatalf("expected ErrNotTheWinningBidder for the losing bidder, got %v", err)
	}
}

func TestCheckoutActiveAuctionRejected(t *testing.T) {
	end := time.Now().Add(time.Hour)
	s := newStoreWithIDFunc([]fixtures.Listing{
		{ID: "l1", Type: fixtures.ListingTypeAuction, Status: fixtures.ListingStatusActiveAuction, PriceCents: 1000, AuctionEndsAt: &end},
	}, nil, nil)

	_, err := s.Checkout("l1", "buyer-1")
	if !errors.Is(err, ErrListingNotForSale) {
		t.Fatalf("expected ErrListingNotForSale for a still-open auction, got %v", err)
	}
}

func TestCheckoutUnsoldExpiredAuctionRejected(t *testing.T) {
	s := newStoreWithIDFunc([]fixtures.Listing{
		{ID: "l1", Type: fixtures.ListingTypeAuction, Status: fixtures.ListingStatusUnsoldExpired, PriceCents: 1000},
	}, nil, nil)

	_, err := s.Checkout("l1", "buyer-1")
	if !errors.Is(err, ErrListingNotForSale) {
		t.Fatalf("expected ErrListingNotForSale for an unsold-expired auction, got %v", err)
	}
}

func TestCheckoutRequiresBuyerID(t *testing.T) {
	s := newStoreWithIDFunc([]fixtures.Listing{
		{ID: "l1", Type: fixtures.ListingTypeBuyNow, Status: fixtures.ListingStatusActiveBuyNow, PriceCents: 1000},
	}, nil, nil)

	_, err := s.Checkout("l1", "")
	if !errors.Is(err, ErrBuyerIDRequired) {
		t.Fatalf("expected ErrBuyerIDRequired, got %v", err)
	}
}

func TestCheckoutUnknownListingRejected(t *testing.T) {
	s := testStore(t)
	_, err := s.Checkout("missing", "buyer-1")
	if !errors.Is(err, ErrListingNotFound) {
		t.Fatalf("expected ErrListingNotFound, got %v", err)
	}
}

func TestGetOrderNotFound(t *testing.T) {
	s := testStore(t)
	_, err := s.GetOrder("missing")
	if !errors.Is(err, ErrOrderNotFound) {
		t.Fatalf("expected ErrOrderNotFound, got %v", err)
	}
}

func TestListOrdersFiltersAndOrders(t *testing.T) {
	s := newStoreWithIDFunc([]fixtures.Listing{
		{ID: "l1", SellerID: "seller-1", Type: fixtures.ListingTypeBuyNow, Status: fixtures.ListingStatusActiveBuyNow, PriceCents: 1000},
		{ID: "l2", SellerID: "seller-2", Type: fixtures.ListingTypeBuyNow, Status: fixtures.ListingStatusActiveBuyNow, PriceCents: 2000},
	}, nil, nil)

	if _, err := s.Checkout("l1", "buyer-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := s.Checkout("l2", "buyer-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	byBuyer := s.ListOrders(OrderFilter{BuyerID: "buyer-1"})
	if len(byBuyer) != 2 {
		t.Fatalf("expected 2 orders for buyer-1, got %d", len(byBuyer))
	}

	bySeller := s.ListOrders(OrderFilter{SellerID: "seller-2"})
	if len(bySeller) != 1 || bySeller[0].ListingID != "l2" {
		t.Fatalf("expected 1 order for seller-2 (l2), got %+v", bySeller)
	}
}
