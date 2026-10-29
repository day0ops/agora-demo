package fixtures

import "time"

// Seed returns a fresh, deterministic synthetic dataset anchored at now:
// sellers, buyers, listings spanning both buy-now and auction types across
// every lifecycle status, and a few bids already placed against the open
// auctions. Pass time.Now() for a real server; tests should pass a fixed
// time for reproducibility.
func Seed(now time.Time) ([]Seller, []Buyer, []Listing, []Bid) {
	sellers := []Seller{
		{ID: "seller-1", Name: "Marina's Vintage Finds", Tier: SellerTierPro},
		{ID: "seller-2", Name: "Dave's Garage Sale", Tier: SellerTierBasic},
		{ID: "seller-3", Name: "Collector's Corner", Tier: SellerTierPro},
	}

	buyers := []Buyer{
		{ID: "buyer-1", Name: "Alex Chen"},
		{ID: "buyer-2", Name: "Jordan Patel"},
		{ID: "buyer-3", Name: "Sam Rivera"},
	}

	closingSoon := now.Add(90 * time.Second)
	closingLater := now.Add(10 * time.Minute)
	endedUnsold := now.Add(-13 * 24 * time.Hour)
	endedSold := now.Add(-71 * time.Hour)

	listings := []Listing{
		{
			ID:              "listing-1",
			SellerID:        "seller-1",
			Title:           "1965 Rolex Submariner",
			Description:     "Vintage automatic dive watch, original papers included.",
			Category:        "watches",
			Type:            ListingTypeAuction,
			Status:          ListingStatusActiveAuction,
			PriceCents:      500000,
			CurrentBidCents: 520000,
			AuctionEndsAt:   &closingSoon,
			CreatedAt:       now.Add(-48 * time.Hour),
		},
		{
			ID:              "listing-2",
			SellerID:        "seller-3",
			Title:           "First-edition 'Dune' hardcover",
			Description:     "1965 Chilton Books first edition, very good condition.",
			Category:        "books",
			Type:            ListingTypeAuction,
			Status:          ListingStatusActiveAuction,
			PriceCents:      150000,
			CurrentBidCents: 0,
			AuctionEndsAt:   &closingLater,
			CreatedAt:       now.Add(-24 * time.Hour),
		},
		{
			ID:          "listing-3",
			SellerID:    "seller-2",
			Title:       "Vintage Nikon F3 camera body",
			Description: "Mint condition, recently serviced.",
			Category:    "cameras",
			Type:        ListingTypeBuyNow,
			Status:      ListingStatusActiveBuyNow,
			PriceCents:  45000,
			CreatedAt:   now.Add(-6 * time.Hour),
		},
		{
			ID:              "listing-4",
			SellerID:        "seller-1",
			Title:           "Art Deco table lamp",
			Description:     "1930s brass and glass table lamp, rewired for modern use.",
			Category:        "home",
			Type:            ListingTypeAuction,
			Status:          ListingStatusUnsoldExpired,
			PriceCents:      12000,
			CurrentBidCents: 0,
			AuctionEndsAt:   &endedUnsold,
			CreatedAt:       now.Add(-14 * 24 * time.Hour),
		},
		{
			ID:              "listing-5",
			SellerID:        "seller-3",
			Title:           "Signed baseball, 1998 World Series",
			Description:     "Certificate of authenticity included.",
			Category:        "memorabilia",
			Type:            ListingTypeAuction,
			Status:          ListingStatusSold,
			PriceCents:      20000,
			CurrentBidCents: 38000,
			AuctionEndsAt:   &endedSold,
			CreatedAt:       now.Add(-72 * time.Hour),
		},
	}

	bids := []Bid{
		{
			ID: "bid-1", ListingID: "listing-1", BuyerID: "buyer-2",
			AmountCents: 500000, PlacedAt: now.Add(-47 * time.Hour),
		},
		{
			ID: "bid-2", ListingID: "listing-1", BuyerID: "buyer-1",
			AmountCents: 520000, IsAutoBid: true, MaxAutoBidCents: 600000,
			PlacedAt: now.Add(-10 * time.Minute),
		},
		{
			ID: "bid-3", ListingID: "listing-5", BuyerID: "buyer-3",
			AmountCents: 38000, PlacedAt: endedSold.Add(-1 * time.Hour),
		},
	}

	return sellers, buyers, listings, bids
}
