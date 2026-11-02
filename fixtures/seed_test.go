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

// TestSeedListingImageSlugs pins which photo slug each seeded listing
// resolves to (see console/web/public/images/ATTRIBUTIONS.md for sourcing).
// Listings created through the Sell form at runtime have no slug and fall
// back to their category's SVG instead.
func TestSeedListingImageSlugs(t *testing.T) {
	_, _, listings, _, _ := Seed(time.Unix(0, 0).UTC())

	want := map[string]string{
		"listing-1":  "rolex-submariner",
		"listing-2":  "dune-first-edition",
		"listing-3":  "nikon-f3",
		"listing-4":  "art-deco-lamp",
		"listing-5":  "baseball-signed",
		"listing-6":  "sony-headphones",
		"listing-7":  "abstract-oil-painting",
		"listing-8":  "fender-stratocaster",
		"listing-9":  "omega-speedmaster",
		"listing-10": "polaroid-sx70",
		"listing-11": "dracula-first-edition",
		"listing-12": "gameboy-color",
		"listing-13": "bronze-sculpture",
		"listing-14": "yamaha-acoustic-guitar",
		"listing-15": "mid-century-armchair",
		"listing-16": "nintendo-64",
		"listing-17": "vinyl-record",
		"listing-18": "ipad-pro",
		"listing-19": "watercolor-landscape",
		"listing-20": "antique-pocket-watch",
		"listing-21": "diamond-engagement-ring",
		"listing-22": "leica-iiia-rangefinder",
		"listing-23": "morgan-silver-dollar-1893s",
		"listing-24": "penny-black-stamp-1840",
		"listing-25": "automaton-grandfather-clock-1790",
		"listing-26": "hartnell-wedding-gown-1951",
		"listing-27": "great-gatsby-first-edition-1925",
		"listing-28": "stainer-style-violin",
		"listing-29": "vintage-pearl-necklace",
		"listing-30": "chateau-cormeil-figeac-bordeaux",
		"listing-31": "wilson-baseball-glove",
		"listing-32": "wilson-tennis-racquet",
		"listing-33": "red-kitchenaid-mixer",
		"listing-34": "griswold-cast-iron-skillet",
		"listing-35": "sony-tr730-transistor-radio",
		"listing-36": "sony-wm-fx421-walkman",
		"listing-37": "classic-student-violin",
		"listing-38": "springbar-canvas-tent",
		"listing-39": "vintage-monopoly-small-box",
		"listing-40": "weston-master-light-meter",
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
