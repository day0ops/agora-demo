package fixtures

import (
	"testing"
	"time"
)

func TestSeedReferentialIntegrity(t *testing.T) {
	sellers, buyers, listings, bids, orders := Seed(time.Now())

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

	orderIDs := map[string]bool{}
	for _, o := range orders {
		if orderIDs[o.ID] {
			t.Fatalf("duplicate order ID %q", o.ID)
		}
		orderIDs[o.ID] = true

		if !listingIDs[o.ListingID] {
			t.Errorf("order %q references unknown listing %q", o.ID, o.ListingID)
		}
		if !buyerIDs[o.BuyerID] {
			t.Errorf("order %q references unknown buyer %q", o.ID, o.BuyerID)
		}
		if !sellerIDs[o.SellerID] {
			t.Errorf("order %q references unknown seller %q", o.ID, o.SellerID)
		}
	}
}

// TestSeedListingImageSlugs pins which seeded listings have a committed
// photo. listing-5 and every listing added since intentionally have none -
// no cleanly public-domain/CC0 licensed photo was sourced for them (see
// console/web/public/images/ATTRIBUTIONS.md) - and fall back to their
// category's SVG at render time instead.
func TestSeedListingImageSlugs(t *testing.T) {
	_, _, listings, _, _ := Seed(time.Unix(0, 0).UTC())

	want := map[string]string{
		"listing-1":  "rolex-submariner",
		"listing-2":  "dune-first-edition",
		"listing-3":  "nikon-f3",
		"listing-4":  "art-deco-lamp",
		"listing-5":  "",
		"listing-6":  "",
		"listing-7":  "",
		"listing-8":  "",
		"listing-9":  "",
		"listing-10": "",
		"listing-11": "",
		"listing-12": "",
		"listing-13": "",
		"listing-14": "",
		"listing-15": "",
		"listing-16": "",
		"listing-17": "",
		"listing-18": "",
		"listing-19": "",
		"listing-20": "",
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
	_, _, listings, _, orders := Seed(anchor)

	for _, l := range listings {
		if l.CreatedAt.After(anchor) {
			t.Errorf("listing %q CreatedAt %v is after the seed anchor %v", l.ID, l.CreatedAt, anchor)
		}
	}
	for _, o := range orders {
		if o.CreatedAt.After(anchor) {
			t.Errorf("order %q CreatedAt %v is after the seed anchor %v", o.ID, o.CreatedAt, anchor)
		}
	}
}
