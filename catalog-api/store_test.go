package main

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/day0ops/agora-demo/fixtures"
	"github.com/day0ops/agora-demo/trust-scoring"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	counter := 0
	return newStoreWithIDFunc(nil, nil, func(prefix string) string {
		counter++
		return fmt.Sprintf("%s-test-%d", prefix, counter)
	})
}

func TestListListingsFiltersByStatusAndSeller(t *testing.T) {
	end := time.Now().Add(time.Hour)
	s := newStoreWithIDFunc([]fixtures.Listing{
		{ID: "l1", SellerID: "seller-1", Status: fixtures.ListingStatusActiveAuction, Type: fixtures.ListingTypeAuction, AuctionEndsAt: &end},
		{ID: "l2", SellerID: "seller-1", Status: fixtures.ListingStatusSold, Type: fixtures.ListingTypeAuction, AuctionEndsAt: &end},
		{ID: "l3", SellerID: "seller-2", Status: fixtures.ListingStatusActiveAuction, Type: fixtures.ListingTypeAuction, AuctionEndsAt: &end},
	}, nil, nil)

	all := s.ListListings(ListingFilter{})
	if len(all) != 3 {
		t.Fatalf("expected 3 listings with no filter, got %d", len(all))
	}

	bySeller := s.ListListings(ListingFilter{SellerID: "seller-1"})
	if len(bySeller) != 2 {
		t.Fatalf("expected 2 listings for seller-1, got %d", len(bySeller))
	}

	byStatus := s.ListListings(ListingFilter{Status: fixtures.ListingStatusSold})
	if len(byStatus) != 1 || byStatus[0].ID != "l2" {
		t.Fatalf("expected only l2 for status sold, got %+v", byStatus)
	}
}

func TestListListingsOrderedByCreatedAtThenID(t *testing.T) {
	now := time.Now()
	s := newStoreWithIDFunc([]fixtures.Listing{
		{ID: "z-later", Type: fixtures.ListingTypeBuyNow, CreatedAt: now.Add(time.Minute)},
		{ID: "b-same", Type: fixtures.ListingTypeBuyNow, CreatedAt: now},
		{ID: "a-same", Type: fixtures.ListingTypeBuyNow, CreatedAt: now},
	}, nil, nil)

	result := s.ListListings(ListingFilter{})
	ids := []string{result[0].ID, result[1].ID, result[2].ID}
	want := []string{"a-same", "b-same", "z-later"}
	for i := range want {
		if ids[i] != want[i] {
			t.Fatalf("expected order %v, got %v", want, ids)
		}
	}
}

func TestGetListingNotFound(t *testing.T) {
	s := testStore(t)
	_, err := s.GetListing("missing")
	if !errors.Is(err, ErrListingNotFound) {
		t.Fatalf("expected ErrListingNotFound, got %v", err)
	}
}

func TestGetListingReturnsIndependentAuctionEndsAtPointer(t *testing.T) {
	end := time.Now().Add(time.Hour)
	s := newStoreWithIDFunc([]fixtures.Listing{
		{ID: "l1", Type: fixtures.ListingTypeAuction, Status: fixtures.ListingStatusActiveAuction, AuctionEndsAt: &end},
	}, nil, nil)

	first, err := s.GetListing("l1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	*first.AuctionEndsAt = time.Now().Add(-time.Hour) // attempt to corrupt the store's copy

	second, err := s.GetListing("l1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !second.AuctionEndsAt.Equal(end) {
		t.Fatalf("GetListing leaked a mutable pointer: store's AuctionEndsAt changed to %v", second.AuctionEndsAt)
	}
}

func TestAuctionClosesToSoldWhenPastEndWithBids(t *testing.T) {
	end := time.Now().Add(time.Minute)
	s := newStoreWithIDFunc([]fixtures.Listing{
		{
			ID: "l1", Type: fixtures.ListingTypeAuction, Status: fixtures.ListingStatusActiveAuction,
			PriceCents: 1000, CurrentBidCents: 1500, AuctionEndsAt: &end,
		},
	}, nil, nil)
	s.now = func() time.Time { return end.Add(time.Second) }

	l, err := s.GetListing("l1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if l.Status != fixtures.ListingStatusSold {
		t.Fatalf("expected status sold once past end time with a bid, got %q", l.Status)
	}
}

func TestAuctionClosesToUnsoldExpiredWhenPastEndWithNoBids(t *testing.T) {
	end := time.Now().Add(time.Minute)
	s := newStoreWithIDFunc([]fixtures.Listing{
		{
			ID: "l1", Type: fixtures.ListingTypeAuction, Status: fixtures.ListingStatusActiveAuction,
			PriceCents: 1000, CurrentBidCents: 0, AuctionEndsAt: &end,
		},
	}, nil, nil)
	s.now = func() time.Time { return end.Add(time.Second) }

	l, err := s.GetListing("l1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if l.Status != fixtures.ListingStatusUnsoldExpired {
		t.Fatalf("expected status unsold-expired once past end time with no bids, got %q", l.Status)
	}
}

func TestPlaceBidRejectsOnceAuctionHasExpired(t *testing.T) {
	end := time.Now().Add(time.Minute)
	s := newStoreWithIDFunc([]fixtures.Listing{
		{
			ID: "l1", Type: fixtures.ListingTypeAuction, Status: fixtures.ListingStatusActiveAuction,
			PriceCents: 1000, AuctionEndsAt: &end,
		},
	}, nil, nil)
	s.now = func() time.Time { return end.Add(time.Second) }

	_, err := s.PlaceBid("l1", "buyer-1", 2000)
	if !errors.Is(err, ErrAuctionClosed) {
		t.Fatalf("expected ErrAuctionClosed for a bid after the auction's end time, got %v", err)
	}
}

func TestCreateListingValidation(t *testing.T) {
	futureEnd := time.Now().Add(time.Hour)
	pastEnd := time.Now().Add(-time.Hour)
	cases := []struct {
		name          string
		sellerID      string
		title         string
		priceCents    int64
		listingType   fixtures.ListingType
		auctionEndsAt *time.Time
		wantErr       error
	}{
		{"missing seller", "", "t", 100, fixtures.ListingTypeBuyNow, nil, ErrSellerIDRequired},
		{"missing title", "s1", "", 100, fixtures.ListingTypeBuyNow, nil, ErrTitleRequired},
		{"invalid price", "s1", "t", 0, fixtures.ListingTypeBuyNow, nil, ErrInvalidPrice},
		{"auction missing end", "s1", "t", 100, fixtures.ListingTypeAuction, nil, ErrAuctionEndRequired},
		{"auction end in the past", "s1", "t", 100, fixtures.ListingTypeAuction, &pastEnd, ErrAuctionEndInPast},
		{"invalid type", "s1", "t", 100, fixtures.ListingType("garage-sale"), nil, ErrInvalidListingType},
		{"valid buy-now", "s1", "t", 100, fixtures.ListingTypeBuyNow, nil, nil},
		{"valid auction", "s1", "t", 100, fixtures.ListingTypeAuction, &futureEnd, nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := testStore(t)
			l, err := s.CreateListing(tc.sellerID, tc.title, "desc", "cat", tc.listingType, tc.priceCents, tc.auctionEndsAt)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("expected %v, got %v", tc.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if l.ID == "" {
				t.Fatal("expected a generated ID")
			}
			wantStatus := fixtures.ListingStatusActiveBuyNow
			if tc.listingType == fixtures.ListingTypeAuction {
				wantStatus = fixtures.ListingStatusActiveAuction
			}
			if l.Status != wantStatus {
				t.Fatalf("expected status %q, got %q", wantStatus, l.Status)
			}
		})
	}
}

func TestCreateListingIgnoresAuctionEndsAtForBuyNow(t *testing.T) {
	end := time.Now().Add(time.Hour)
	s := testStore(t)

	l, err := s.CreateListing("s1", "t", "desc", "cat", fixtures.ListingTypeBuyNow, 100, &end)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if l.AuctionEndsAt != nil {
		t.Fatalf("expected a buy-now listing to have no AuctionEndsAt, got %v", l.AuctionEndsAt)
	}
}

func TestPlaceBidHappyPath(t *testing.T) {
	end := time.Now().Add(time.Hour)
	s := newStoreWithIDFunc([]fixtures.Listing{
		{
			ID: "l1", SellerID: "seller-1", Type: fixtures.ListingTypeAuction,
			Status: fixtures.ListingStatusActiveAuction, PriceCents: 1000, CurrentBidCents: 1000,
			AuctionEndsAt: &end,
		},
	}, nil, func(prefix string) string { return prefix + "-1" })

	bid, err := s.PlaceBid("l1", "buyer-1", 1500)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bid.AmountCents != 1500 {
		t.Fatalf("expected bid amount 1500, got %d", bid.AmountCents)
	}

	l, err := s.GetListing("l1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if l.CurrentBidCents != 1500 {
		t.Fatalf("expected listing current bid 1500, got %d", l.CurrentBidCents)
	}
}

func TestPlaceBidFirstBidMayEqualStartingPrice(t *testing.T) {
	end := time.Now().Add(time.Hour)
	s := newStoreWithIDFunc([]fixtures.Listing{
		{
			ID: "l1", Type: fixtures.ListingTypeAuction, Status: fixtures.ListingStatusActiveAuction,
			PriceCents: 1000, CurrentBidCents: 0, AuctionEndsAt: &end,
		},
	}, nil, nil)

	bid, err := s.PlaceBid("l1", "buyer-1", 1000)
	if err != nil {
		t.Fatalf("expected a bid exactly at the starting price to be accepted, got error: %v", err)
	}
	if bid.AmountCents != 1000 {
		t.Fatalf("expected bid amount 1000, got %d", bid.AmountCents)
	}
}

func TestPlaceBidFirstBidBelowStartingPriceRejected(t *testing.T) {
	end := time.Now().Add(time.Hour)
	s := newStoreWithIDFunc([]fixtures.Listing{
		{
			ID: "l1", Type: fixtures.ListingTypeAuction, Status: fixtures.ListingStatusActiveAuction,
			PriceCents: 150000, CurrentBidCents: 0, AuctionEndsAt: &end,
		},
	}, nil, nil)

	_, err := s.PlaceBid("l1", "buyer-1", 1)
	if !errors.Is(err, ErrBidTooLow) {
		t.Fatalf("expected ErrBidTooLow for a bid far below the starting price, got %v", err)
	}
}

func TestPlaceBidRequiresBuyerID(t *testing.T) {
	end := time.Now().Add(time.Hour)
	s := newStoreWithIDFunc([]fixtures.Listing{
		{ID: "l1", Type: fixtures.ListingTypeAuction, Status: fixtures.ListingStatusActiveAuction, PriceCents: 100, AuctionEndsAt: &end},
	}, nil, nil)

	_, err := s.PlaceBid("l1", "", 200)
	if !errors.Is(err, ErrBuyerIDRequired) {
		t.Fatalf("expected ErrBuyerIDRequired, got %v", err)
	}
}

func TestPlaceBidRejectsNonAuction(t *testing.T) {
	s := newStoreWithIDFunc([]fixtures.Listing{
		{ID: "l1", Type: fixtures.ListingTypeBuyNow, Status: fixtures.ListingStatusActiveBuyNow, PriceCents: 1000},
	}, nil, nil)

	_, err := s.PlaceBid("l1", "buyer-1", 1500)
	if !errors.Is(err, ErrListingNotAuction) {
		t.Fatalf("expected ErrListingNotAuction, got %v", err)
	}
}

func TestPlaceBidRejectsClosedAuction(t *testing.T) {
	s := newStoreWithIDFunc([]fixtures.Listing{
		{ID: "l1", Type: fixtures.ListingTypeAuction, Status: fixtures.ListingStatusSold, PriceCents: 1000, CurrentBidCents: 2000},
	}, nil, nil)

	_, err := s.PlaceBid("l1", "buyer-1", 2500)
	if !errors.Is(err, ErrAuctionClosed) {
		t.Fatalf("expected ErrAuctionClosed, got %v", err)
	}
}

func TestPlaceBidRejectsBidEqualToCurrentPrice(t *testing.T) {
	end := time.Now().Add(time.Hour)
	s := newStoreWithIDFunc([]fixtures.Listing{
		{
			ID: "l1", Type: fixtures.ListingTypeAuction, Status: fixtures.ListingStatusActiveAuction,
			PriceCents: 1000, CurrentBidCents: 1500, AuctionEndsAt: &end,
		},
	}, nil, nil)

	_, err := s.PlaceBid("l1", "buyer-1", 1500)
	if !errors.Is(err, ErrBidTooLow) {
		t.Fatalf("expected ErrBidTooLow for a bid merely equal to the current price, got %v", err)
	}
}

func TestWithdrawBidByIDRevertsToNextHighest(t *testing.T) {
	end := time.Now().Add(time.Hour)
	s := newStoreWithIDFunc(
		[]fixtures.Listing{
			{
				ID: "l1", Type: fixtures.ListingTypeAuction, Status: fixtures.ListingStatusActiveAuction,
				PriceCents: 1000, CurrentBidCents: 2000, AuctionEndsAt: &end,
			},
		},
		[]fixtures.Bid{
			{ID: "b1", ListingID: "l1", BuyerID: "buyer-1", AmountCents: 1500},
			{ID: "b2", ListingID: "l1", BuyerID: "buyer-2", AmountCents: 2000},
		},
		nil,
	)

	if err := s.WithdrawBid("l1", "b2"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	l, err := s.GetListing("l1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if l.CurrentBidCents != 1500 {
		t.Fatalf("expected current bid to revert to 1500, got %d", l.CurrentBidCents)
	}

	history, err := s.BidHistory("l1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(history) != 1 || history[0].ID != "b1" {
		t.Fatalf("expected only b1 to remain, got %+v", history)
	}
}

func TestWithdrawBidToZeroWhenNoneRemain(t *testing.T) {
	end := time.Now().Add(time.Hour)
	s := newStoreWithIDFunc(
		[]fixtures.Listing{
			{
				ID: "l1", Type: fixtures.ListingTypeAuction, Status: fixtures.ListingStatusActiveAuction,
				PriceCents: 1000, CurrentBidCents: 1500, AuctionEndsAt: &end,
			},
		},
		[]fixtures.Bid{{ID: "b1", ListingID: "l1", BuyerID: "buyer-1", AmountCents: 1500}},
		nil,
	)

	if err := s.WithdrawBid("l1", "b1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	l, err := s.GetListing("l1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if l.CurrentBidCents != 0 {
		t.Fatalf("expected current bid to revert to 0, got %d", l.CurrentBidCents)
	}
}

func TestWithdrawBidNotFound(t *testing.T) {
	end := time.Now().Add(time.Hour)
	s := newStoreWithIDFunc([]fixtures.Listing{
		{ID: "l1", Type: fixtures.ListingTypeAuction, Status: fixtures.ListingStatusActiveAuction, AuctionEndsAt: &end},
	}, nil, nil)

	err := s.WithdrawBid("l1", "no-such-bid")
	if !errors.Is(err, ErrBidNotFound) {
		t.Fatalf("expected ErrBidNotFound, got %v", err)
	}
}

func TestWithdrawBidRejectsOnClosedAuction(t *testing.T) {
	s := newStoreWithIDFunc(
		[]fixtures.Listing{{ID: "l1", Type: fixtures.ListingTypeAuction, Status: fixtures.ListingStatusSold}},
		[]fixtures.Bid{{ID: "b1", ListingID: "l1", BuyerID: "buyer-1", AmountCents: 1000}},
		nil,
	)

	err := s.WithdrawBid("l1", "b1")
	if !errors.Is(err, ErrAuctionClosed) {
		t.Fatalf("expected ErrAuctionClosed for a withdrawal on a closed auction, got %v", err)
	}
}

func TestBidHistoryOrderAndIsolation(t *testing.T) {
	s := newStoreWithIDFunc(
		[]fixtures.Listing{{ID: "l1", Type: fixtures.ListingTypeAuction}},
		[]fixtures.Bid{
			{ID: "b1", ListingID: "l1", BuyerID: "buyer-1", AmountCents: 1000},
			{ID: "b2", ListingID: "l1", BuyerID: "buyer-2", AmountCents: 1500},
		},
		nil,
	)

	history, err := s.BidHistory("l1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(history) != 2 || history[0].ID != "b1" || history[1].ID != "b2" {
		t.Fatalf("expected bids in insertion order, got %+v", history)
	}

	// Mutating the returned slice must not affect the store's own copy.
	history[0].AmountCents = 999999
	again, err := s.BidHistory("l1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if again[0].AmountCents != 1000 {
		t.Fatalf("BidHistory leaked a mutable reference to internal state: got %d", again[0].AmountCents)
	}
}

func TestNewDefaultIDNeverCollidesUnderConcurrency(t *testing.T) {
	const n = 1000
	ids := make(chan string, n)
	for i := 0; i < n; i++ {
		go func() { ids <- newDefaultID("bid") }()
	}

	seen := make(map[string]bool, n)
	for i := 0; i < n; i++ {
		id := <-ids
		if seen[id] {
			t.Fatalf("duplicate ID generated under concurrency: %s", id)
		}
		seen[id] = true
	}
}

func TestSetAutoBidMaxSingleBuyerLeadsAtStartingPrice(t *testing.T) {
	end := time.Now().Add(time.Hour)
	s := newStoreWithIDFunc([]fixtures.Listing{
		{ID: "l1", Type: fixtures.ListingTypeAuction, Status: fixtures.ListingStatusActiveAuction, PriceCents: 1000, AuctionEndsAt: &end},
	}, nil, nil)

	l, err := s.SetAutoBidMax("l1", "buyer-1", 6000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if l.CurrentBidCents != 1000 {
		t.Fatalf("expected a lone ceiling to only meet the starting price, got %d", l.CurrentBidCents)
	}

	history, err := s.BidHistory("l1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(history) != 1 || history[0].BuyerID != "buyer-1" || !history[0].IsAutoBid {
		t.Fatalf("expected one auto-generated bid for buyer-1, got %+v", history)
	}
}

func TestSetAutoBidMaxTwoBuyersResolveAboveSecondCeiling(t *testing.T) {
	end := time.Now().Add(time.Hour)
	s := newStoreWithIDFunc([]fixtures.Listing{
		{ID: "l1", Type: fixtures.ListingTypeAuction, Status: fixtures.ListingStatusActiveAuction, PriceCents: 1000, AuctionEndsAt: &end},
	}, nil, nil)

	if _, err := s.SetAutoBidMax("l1", "buyer-1", 6000); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	l, err := s.SetAutoBidMax("l1", "buyer-2", 5000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wantPrice := int64(5000 + bidIncrementCents)
	if l.CurrentBidCents != wantPrice {
		t.Fatalf("expected price %d (second ceiling + increment), got %d", wantPrice, l.CurrentBidCents)
	}

	history, err := s.BidHistory("l1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	last := history[len(history)-1]
	if last.BuyerID != "buyer-1" || last.AmountCents != wantPrice || last.MaxAutoBidCents != 6000 {
		t.Fatalf("expected buyer-1 to hold the lead at %d with max 6000, got %+v", wantPrice, last)
	}
}

func TestSetAutoBidMaxCappedAtOwnCeiling(t *testing.T) {
	end := time.Now().Add(time.Hour)
	s := newStoreWithIDFunc([]fixtures.Listing{
		{ID: "l1", Type: fixtures.ListingTypeAuction, Status: fixtures.ListingStatusActiveAuction, PriceCents: 1000, AuctionEndsAt: &end},
	}, nil, nil)

	if _, err := s.SetAutoBidMax("l1", "buyer-1", 1200); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	l, err := s.SetAutoBidMax("l1", "buyer-2", 5000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// buyer-1's ceiling (1200) is below what buyer-2's ceiling would otherwise
	// push the price to (1200+increment would exceed 1200), so buyer-2 wins
	// outright at one increment above buyer-1's ceiling, capped by nothing
	// since buyer-2's own ceiling (5000) is far higher.
	if l.CurrentBidCents <= 1200 {
		t.Fatalf("expected buyer-2 to take the lead above buyer-1's 1200 ceiling, got %d", l.CurrentBidCents)
	}
}

func TestSetAutoBidMaxDoesNotLowerAStrongerExistingPrice(t *testing.T) {
	end := time.Now().Add(time.Hour)
	s := newStoreWithIDFunc([]fixtures.Listing{
		{ID: "l1", Type: fixtures.ListingTypeAuction, Status: fixtures.ListingStatusActiveAuction, PriceCents: 1000, AuctionEndsAt: &end},
	}, nil, nil)

	if _, err := s.PlaceBid("l1", "buyer-3", 8000); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	l, err := s.SetAutoBidMax("l1", "buyer-1", 6000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if l.CurrentBidCents != 8000 {
		t.Fatalf("expected buyer-3's manual 8000 bid to stand against a weaker 6000 ceiling, got %d", l.CurrentBidCents)
	}
}

func TestPlaceBidGetsImmediatelyOutbidByAStrongerStandingCeiling(t *testing.T) {
	end := time.Now().Add(time.Hour)
	s := newStoreWithIDFunc([]fixtures.Listing{
		{ID: "l1", Type: fixtures.ListingTypeAuction, Status: fixtures.ListingStatusActiveAuction, PriceCents: 1000, AuctionEndsAt: &end},
	}, nil, nil)

	if _, err := s.SetAutoBidMax("l1", "buyer-1", 6000); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	bid, err := s.PlaceBid("l1", "buyer-2", 2000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bid.AmountCents != 2000 {
		t.Fatalf("expected the recorded manual bid to be exactly 2000, got %d", bid.AmountCents)
	}

	l, err := s.GetListing("l1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wantPrice := int64(2000 + bidIncrementCents)
	if l.CurrentBidCents != wantPrice {
		t.Fatalf("expected buyer-1's standing ceiling to immediately outbid to %d, got %d", wantPrice, l.CurrentBidCents)
	}

	history, err := s.BidHistory("l1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	last := history[len(history)-1]
	if last.BuyerID != "buyer-1" || !last.IsAutoBid {
		t.Fatalf("expected the final bid to be buyer-1's auto-bid, got %+v", last)
	}
}

func TestSetAutoBidMaxRejectsNonPositiveMax(t *testing.T) {
	s := testStore(t)
	_, err := s.SetAutoBidMax("l1", "buyer-1", 0)
	if !errors.Is(err, ErrInvalidPrice) {
		t.Fatalf("expected ErrInvalidPrice, got %v", err)
	}
}

func TestSetAutoBidMaxRequiresBuyerID(t *testing.T) {
	s := testStore(t)
	_, err := s.SetAutoBidMax("l1", "", 1000)
	if !errors.Is(err, ErrBuyerIDRequired) {
		t.Fatalf("expected ErrBuyerIDRequired, got %v", err)
	}
}

func TestSetAutoBidMaxRejectsOnClosedAuction(t *testing.T) {
	s := newStoreWithIDFunc([]fixtures.Listing{
		{ID: "l1", Type: fixtures.ListingTypeAuction, Status: fixtures.ListingStatusSold},
	}, nil, nil)

	_, err := s.SetAutoBidMax("l1", "buyer-1", 1000)
	if !errors.Is(err, ErrAuctionClosed) {
		t.Fatalf("expected ErrAuctionClosed, got %v", err)
	}
}

func TestSetAutoBidMaxRejectsNonAuction(t *testing.T) {
	s := newStoreWithIDFunc([]fixtures.Listing{
		{ID: "l1", Type: fixtures.ListingTypeBuyNow, Status: fixtures.ListingStatusActiveBuyNow},
	}, nil, nil)

	_, err := s.SetAutoBidMax("l1", "buyer-1", 1000)
	if !errors.Is(err, ErrListingNotAuction) {
		t.Fatalf("expected ErrListingNotAuction, got %v", err)
	}
}

func TestResolveProxyBiddingNeverSettlesBelowStartingPrice(t *testing.T) {
	end := time.Now().Add(time.Hour)
	s := newStoreWithIDFunc([]fixtures.Listing{
		{ID: "l1", Type: fixtures.ListingTypeAuction, Status: fixtures.ListingStatusActiveAuction, PriceCents: 1000, AuctionEndsAt: &end},
	}, nil, nil)

	// A lone ceiling below the starting price must not win the auction at
	// all -- it must not drag the price down to meet it.
	l, err := s.SetAutoBidMax("l1", "buyer-1", 500)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if l.CurrentBidCents != 0 {
		t.Fatalf("expected a sub-starting-price ceiling to leave the listing unbid, got CurrentBidCents=%d", l.CurrentBidCents)
	}

	history, err := s.BidHistory("l1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(history) != 0 {
		t.Fatalf("expected no bid recorded for a ceiling below the starting price, got %+v", history)
	}
}

func TestResolveProxyBiddingNeverSettlesBelowStartingPriceWithTwoLowCeilings(t *testing.T) {
	end := time.Now().Add(time.Hour)
	s := newStoreWithIDFunc([]fixtures.Listing{
		{ID: "l1", Type: fixtures.ListingTypeAuction, Status: fixtures.ListingStatusActiveAuction, PriceCents: 1000, AuctionEndsAt: &end},
	}, nil, nil)

	if _, err := s.SetAutoBidMax("l1", "buyer-2", 400); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	l, err := s.SetAutoBidMax("l1", "buyer-1", 600)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if l.CurrentBidCents != 0 {
		t.Fatalf("expected two ceilings both below the starting price to leave the listing unbid, got %d", l.CurrentBidCents)
	}
}

func TestWithdrawBidClearsTheBuyersStandingCeiling(t *testing.T) {
	end := time.Now().Add(time.Hour)
	s := newStoreWithIDFunc([]fixtures.Listing{
		{ID: "l1", Type: fixtures.ListingTypeAuction, Status: fixtures.ListingStatusActiveAuction, PriceCents: 1000, AuctionEndsAt: &end},
	}, nil, nil)

	if _, err := s.SetAutoBidMax("l1", "buyer-1", 6000); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	bid, err := s.PlaceBid("l1", "buyer-2", 2000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// This resolves to an auto-bid for buyer-1 at 2100; find and withdraw it.
	history, err := s.BidHistory("l1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var buyer1AutoBidID string
	for _, h := range history {
		if h.BuyerID == "buyer-1" && h.IsAutoBid {
			buyer1AutoBidID = h.ID
		}
	}
	if buyer1AutoBidID == "" {
		t.Fatal("expected to find buyer-1's auto-generated bid")
	}
	if err := s.WithdrawBid("l1", buyer1AutoBidID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	l, err := s.GetListing("l1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if l.CurrentBidCents != bid.AmountCents {
		t.Fatalf("expected the price to revert to buyer-2's manual bid %d, got %d", bid.AmountCents, l.CurrentBidCents)
	}

	// A fresh bid from a third party must not resurrect buyer-1's withdrawn
	// ceiling -- they withdrew, they are out until they act again.
	if _, err := s.PlaceBid("l1", "buyer-3", 2500); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	l, err = s.GetListing("l1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if l.CurrentBidCents != 2500 {
		t.Fatalf("expected buyer-3's manual bid of 2500 to stand (buyer-1's ceiling must stay cleared), got %d", l.CurrentBidCents)
	}
}

// TestNewStoreGeneratedIDsDoNotCollideWithSeedIDs guards against the real ID
// generator (newDefaultID, only exercised through the real NewStore - every
// other test in this file injects a deterministic test ID func) starting
// its counter at 0 and immediately colliding with fixtures.Seed()'s
// hardcoded "listing-1" etc, which silently overwrote the seeded listing.
func TestNewStoreGeneratedIDsDoNotCollideWithSeedIDs(t *testing.T) {
	_, _, listings, bids := fixtures.Seed(time.Now())
	s := NewStore(listings, bids)

	before, err := s.GetListing("listing-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	created, err := s.CreateListing("seller-1", "New item", "desc", "misc", fixtures.ListingTypeBuyNow, 1000, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, l := range listings {
		if created.ID == l.ID {
			t.Fatalf("newly created listing ID %q collides with a seeded listing ID", created.ID)
		}
	}

	after, err := s.GetListing("listing-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Compare fields rather than the whole struct: AuctionEndsAt is a
	// *time.Time that GetListing clones on every call (copyListing), so two
	// otherwise-identical reads always differ by pointer identity.
	if after.Title != before.Title || after.PriceCents != before.PriceCents ||
		after.CurrentBidCents != before.CurrentBidCents || after.Status != before.Status {
		t.Fatalf("seeded listing-1 was mutated by creating an unrelated listing: before=%+v after=%+v", before, after)
	}
}

func TestPlaceBidRecordsTrustCheck(t *testing.T) {
	end := time.Now().Add(time.Hour)
	s := newStoreWithIDFunc([]fixtures.Listing{
		{ID: "l1", Type: fixtures.ListingTypeAuction, Status: fixtures.ListingStatusActiveAuction, PriceCents: 1000, AuctionEndsAt: &end},
	}, nil, nil)

	bid, err := s.PlaceBid("l1", "buyer-1", 1000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bid.TrustCheck == nil {
		t.Fatalf("expected a trust check result, got nil")
	}
	if bid.TrustCheck.ScoredBy != "heuristic" {
		t.Fatalf("expected ScoredBy=heuristic, got %q", bid.TrustCheck.ScoredBy)
	}
	if bid.TrustCheck.Score != 0 {
		t.Fatalf("expected zero risk for a first bid with no history, got %+v", bid.TrustCheck)
	}
}

func TestPlaceBidTrustCheckFlagsSelfBidOverLead(t *testing.T) {
	end := time.Now().Add(time.Hour)
	s := newStoreWithIDFunc([]fixtures.Listing{
		{ID: "l1", Type: fixtures.ListingTypeAuction, Status: fixtures.ListingStatusActiveAuction, PriceCents: 1000, AuctionEndsAt: &end},
	}, nil, nil)

	if _, err := s.PlaceBid("l1", "buyer-1", 1000); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	bid, err := s.PlaceBid("l1", "buyer-1", 2000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bid.TrustCheck == nil || bid.TrustCheck.Score < 0.5 {
		t.Fatalf("expected a meaningful risk score for self-bidding over own lead, got %+v", bid.TrustCheck)
	}
}

type failingScorer struct{}

func (failingScorer) ScoreBidRisk(context.Context, trustscoring.BidRiskInput) (trustscoring.RiskScore, error) {
	return trustscoring.RiskScore{}, errors.New("scorer unavailable")
}

func TestPlaceBidTrustCheckNilWhenScorerFails(t *testing.T) {
	end := time.Now().Add(time.Hour)
	s := newStoreWithIDFunc([]fixtures.Listing{
		{ID: "l1", Type: fixtures.ListingTypeAuction, Status: fixtures.ListingStatusActiveAuction, PriceCents: 1000, AuctionEndsAt: &end},
	}, nil, nil)
	s.scorer = failingScorer{}

	bid, err := s.PlaceBid("l1", "buyer-1", 1000)
	if err != nil {
		t.Fatalf("expected bid to succeed even when the scorer fails, got error: %v", err)
	}
	if bid.TrustCheck != nil {
		t.Fatalf("expected nil TrustCheck when the scorer fails, got %+v", bid.TrustCheck)
	}
}

type panickingScorer struct{}

func (panickingScorer) ScoreBidRisk(context.Context, trustscoring.BidRiskInput) (trustscoring.RiskScore, error) {
	panic("scorer programming error")
}

func TestPlaceBidTrustCheckNilWhenScorerPanics(t *testing.T) {
	end := time.Now().Add(time.Hour)
	s := newStoreWithIDFunc([]fixtures.Listing{
		{ID: "l1", Type: fixtures.ListingTypeAuction, Status: fixtures.ListingStatusActiveAuction, PriceCents: 1000, AuctionEndsAt: &end},
	}, nil, nil)
	s.scorer = panickingScorer{}

	bid, err := s.PlaceBid("l1", "buyer-1", 1000)
	if err != nil {
		t.Fatalf("expected bid to succeed even when the scorer panics, got error: %v", err)
	}
	if bid.TrustCheck != nil {
		t.Fatalf("expected nil TrustCheck when the scorer panics, got %+v", bid.TrustCheck)
	}
}

func TestSetAutoBidMaxTieBreaksByRegistrationOrder(t *testing.T) {
	end := time.Now().Add(time.Hour)
	s := newStoreWithIDFunc([]fixtures.Listing{
		{ID: "l1", Type: fixtures.ListingTypeAuction, Status: fixtures.ListingStatusActiveAuction, PriceCents: 1000, AuctionEndsAt: &end},
	}, nil, nil)

	if _, err := s.SetAutoBidMax("l1", "buyer-1", 5000); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := s.SetAutoBidMax("l1", "buyer-2", 5000); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	history, err := s.BidHistory("l1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	last := history[len(history)-1]
	if last.BuyerID != "buyer-1" {
		t.Fatalf("expected the earlier-registered equal ceiling (buyer-1) to win the tie, got %s", last.BuyerID)
	}
}
