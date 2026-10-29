package fixtures

import (
	"testing"
	"time"
)

func TestSeedReferentialIntegrity(t *testing.T) {
	sellers, buyers, listings, bids := Seed(time.Now())

	sellerIDs := map[string]bool{}
	for _, s := range sellers {
		if sellerIDs[s.ID] {
			t.Fatalf("duplicate seller ID %q", s.ID)
		}
		sellerIDs[s.ID] = true
	}

	buyerIDs := map[string]bool{}
	for _, b := range buyers {
		if buyerIDs[b.ID] {
			t.Fatalf("duplicate buyer ID %q", b.ID)
		}
		buyerIDs[b.ID] = true
	}

	listingIDs := map[string]bool{}
	for _, l := range listings {
		if listingIDs[l.ID] {
			t.Fatalf("duplicate listing ID %q", l.ID)
		}
		listingIDs[l.ID] = true

		if !sellerIDs[l.SellerID] {
			t.Errorf("listing %q references unknown seller %q", l.ID, l.SellerID)
		}

		switch l.Type {
		case ListingTypeAuction:
			if l.AuctionEndsAt == nil {
				t.Errorf("auction listing %q has no AuctionEndsAt", l.ID)
			}
		case ListingTypeBuyNow:
			if l.AuctionEndsAt != nil {
				t.Errorf("buy-now listing %q unexpectedly has AuctionEndsAt set", l.ID)
			}
		default:
			t.Errorf("listing %q has unknown type %q", l.ID, l.Type)
		}
	}

	bidIDs := map[string]bool{}
	for _, b := range bids {
		if bidIDs[b.ID] {
			t.Fatalf("duplicate bid ID %q", b.ID)
		}
		bidIDs[b.ID] = true

		if !listingIDs[b.ListingID] {
			t.Errorf("bid %q references unknown listing %q", b.ID, b.ListingID)
		}
		if !buyerIDs[b.BuyerID] {
			t.Errorf("bid %q references unknown buyer %q", b.ID, b.BuyerID)
		}
	}
}

func TestSeedIsAnchoredToGivenTime(t *testing.T) {
	anchor := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	_, _, listings, _ := Seed(anchor)

	for _, l := range listings {
		if l.CreatedAt.After(anchor) {
			t.Errorf("listing %q CreatedAt %v is after the seed anchor %v", l.ID, l.CreatedAt, anchor)
		}
	}
}
