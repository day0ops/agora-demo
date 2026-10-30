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

// TestSeedListingImageSlugs pins which seeded listings have a committed
// photo. listing-5 intentionally has none - no cleanly public-domain/CC0
// licensed photo of a signed 1998 World Series baseball was found (see
// console/web/public/images/ATTRIBUTIONS.md) - and falls back to its
// category's SVG at render time instead.
func TestSeedListingImageSlugs(t *testing.T) {
	_, _, listings, _ := Seed(time.Unix(0, 0).UTC())

	want := map[string]string{
		"listing-1": "rolex-submariner",
		"listing-2": "dune-first-edition",
		"listing-3": "nikon-f3",
		"listing-4": "art-deco-lamp",
		"listing-5": "",
	}
	for _, l := range listings {
		if got, ok := want[l.ID]; !ok {
			t.Errorf("listing %q not covered by this test's expectations", l.ID)
		} else if l.Image != got {
			t.Errorf("listing %q: Image = %q, want %q", l.ID, l.Image, got)
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
