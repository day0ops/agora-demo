package fixtures

import "time"

// Seed returns a fresh, deterministic synthetic dataset anchored at now:
// sellers, buyers, listings spanning both buy-now and auction types across
// every lifecycle status, bids already placed against the open auctions,
// and orders for already-settled transactions. Pass time.Now() for a real
// server; tests should pass a fixed time for reproducibility.
func Seed(now time.Time) ([]Seller, []Buyer, []Listing, []Bid, []Order) {
	sellers := []Seller{
		{ID: "seller-1", Name: "Marina's Vintage Finds", Tier: SellerTierPro},
		{ID: "seller-2", Name: "Dave's Garage Sale", Tier: SellerTierBasic},
		{ID: "seller-3", Name: "Collector's Corner", Tier: SellerTierPro},
		{ID: "seller-4", Name: "Northside Electronics", Tier: SellerTierBasic},
		{ID: "seller-5", Name: "Gallery Nine", Tier: SellerTierPro},
		{ID: "seller-6", Name: "Riverside Music Exchange", Tier: SellerTierBasic},
	}

	buyers := []Buyer{
		{ID: "buyer-1", Name: "Alex Chen"},
		{ID: "buyer-2", Name: "Jordan Patel"},
		{ID: "buyer-3", Name: "Sam Rivera"},
		{ID: "buyer-4", Name: "Priya Nair"},
		{ID: "buyer-5", Name: "Marcus Webb"},
		{ID: "buyer-6", Name: "Elena Ruiz"},
	}

	closingSoon := now.Add(90 * time.Second)
	closingLater := now.Add(10 * time.Minute)
	endedUnsold := now.Add(-13 * 24 * time.Hour)
	endedSold := now.Add(-71 * time.Hour)

	closingSoon2 := now.Add(95 * time.Second)
	closingPainting := now.Add(5 * time.Minute)
	closingGuitar := now.Add(15 * time.Minute)
	closingDracula := now.Add(10 * time.Minute)
	draftAuctionEnd := now.Add(48 * time.Hour)
	withdrawnAuctionEnd := now.Add(-2 * 24 * time.Hour)
	endedSold2 := now.Add(-96 * time.Hour)
	endedUnsold2 := now.Add(-20 * 24 * time.Hour)

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
			Image:           "rolex-submariner",
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
			Image:           "dune-first-edition",
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
			Image:       "nikon-f3",
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
			Image:           "art-deco-lamp",
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
		{
			ID:          "listing-6",
			SellerID:    "seller-4",
			Title:       "Sony WH-1000XM4 Headphones",
			Description: "Noise-cancelling over-ear headphones, barely used.",
			Category:    "electronics",
			Type:        ListingTypeBuyNow,
			Status:      ListingStatusActiveBuyNow,
			PriceCents:  18000,
			CreatedAt:   now.Add(-3 * time.Hour),
		},
		{
			ID:              "listing-7",
			SellerID:        "seller-5",
			Title:           "Untitled abstract oil painting",
			Description:     "Large-format abstract piece, acrylic and oil on canvas.",
			Category:        "art",
			Type:            ListingTypeAuction,
			Status:          ListingStatusActiveAuction,
			PriceCents:      80000,
			CurrentBidCents: 95000,
			AuctionEndsAt:   &closingPainting,
			CreatedAt:       now.Add(-20 * time.Hour),
		},
		{
			ID:              "listing-8",
			SellerID:        "seller-6",
			Title:           "1978 Fender Stratocaster",
			Description:     "Sunburst finish, original pickups, recently set up.",
			Category:        "instruments",
			Type:            ListingTypeAuction,
			Status:          ListingStatusActiveAuction,
			PriceCents:      220000,
			CurrentBidCents: 0,
			AuctionEndsAt:   &closingGuitar,
			CreatedAt:       now.Add(-2 * time.Hour),
		},
		{
			ID:              "listing-9",
			SellerID:        "seller-1",
			Title:           "Omega Speedmaster Professional",
			Description:     "Moonwatch reissue, box and papers included.",
			Category:        "watches",
			Type:            ListingTypeAuction,
			Status:          ListingStatusActiveAuction,
			PriceCents:      450000,
			CurrentBidCents: 470000,
			AuctionEndsAt:   &closingSoon2,
			CreatedAt:       now.Add(-30 * time.Hour),
		},
		{
			ID:          "listing-10",
			SellerID:    "seller-2",
			Title:       "Polaroid SX-70 camera",
			Description: "Folding instant camera, tested and working.",
			Category:    "cameras",
			Type:        ListingTypeBuyNow,
			Status:      ListingStatusActiveBuyNow,
			PriceCents:  22000,
			CreatedAt:   now.Add(-10 * time.Hour),
		},
		{
			ID:              "listing-11",
			SellerID:        "seller-3",
			Title:           "First-edition 'Dracula' by Bram Stoker",
			Description:     "1897 Archibald Constable and Company first edition.",
			Category:        "books",
			Type:            ListingTypeAuction,
			Status:          ListingStatusActiveAuction,
			PriceCents:      300000,
			CurrentBidCents: 0,
			AuctionEndsAt:   &closingDracula,
			CreatedAt:       now.Add(-5 * time.Hour),
		},
		{
			ID:          "listing-12",
			SellerID:    "seller-4",
			Title:       "Vintage Game Boy Color",
			Description: "Kiwi green, tested, includes two cartridges.",
			Category:    "electronics",
			Type:        ListingTypeBuyNow,
			Status:      ListingStatusActiveBuyNow,
			PriceCents:  9000,
			CreatedAt:   now.Add(-1 * time.Hour),
		},
		{
			ID:              "listing-13",
			SellerID:        "seller-5",
			Title:           "Mid-century bronze sculpture",
			Description:     "Abstract figurative bronze, unsigned, estimated 1960s.",
			Category:        "art",
			Type:            ListingTypeAuction,
			Status:          ListingStatusDraft,
			PriceCents:      150000,
			CurrentBidCents: 0,
			AuctionEndsAt:   &draftAuctionEnd,
			CreatedAt:       now.Add(-2 * time.Hour),
		},
		{
			ID:          "listing-14",
			SellerID:    "seller-6",
			Title:       "Yamaha FG800 acoustic guitar",
			Description: "Solid spruce top, includes gig bag.",
			Category:    "instruments",
			Type:        ListingTypeBuyNow,
			Status:      ListingStatusDraft,
			PriceCents:  35000,
			CreatedAt:   now.Add(-1 * time.Hour),
		},
		{
			ID:          "listing-15",
			SellerID:    "seller-1",
			Title:       "Mid-century modern armchair",
			Description: "Reupholstered walnut-frame armchair, 1960s.",
			Category:    "home",
			Type:        ListingTypeBuyNow,
			Status:      ListingStatusWithdrawn,
			PriceCents:  60000,
			CreatedAt:   now.Add(-20 * time.Hour),
		},
		{
			ID:              "listing-16",
			SellerID:        "seller-2",
			Title:           "Nintendo 64 console bundle",
			Description:     "Console, two controllers, four cartridges.",
			Category:        "electronics",
			Type:            ListingTypeAuction,
			Status:          ListingStatusWithdrawn,
			PriceCents:      15000,
			CurrentBidCents: 0,
			AuctionEndsAt:   &withdrawnAuctionEnd,
			CreatedAt:       now.Add(-5 * 24 * time.Hour),
		},
		{
			ID:              "listing-17",
			SellerID:        "seller-3",
			Title:           "Autographed Beatles vinyl record",
			Description:     "Signed 'Abbey Road' LP, certificate of authenticity included.",
			Category:        "memorabilia",
			Type:            ListingTypeAuction,
			Status:          ListingStatusSold,
			PriceCents:      50000,
			CurrentBidCents: 72000,
			AuctionEndsAt:   &endedSold2,
			CreatedAt:       now.Add(-4 * 24 * time.Hour),
		},
		{
			ID:          "listing-18",
			SellerID:    "seller-4",
			Title:       "iPad Pro 2023, 256GB",
			Description: "Excellent condition, includes original box and charger.",
			Category:    "electronics",
			Type:        ListingTypeBuyNow,
			Status:      ListingStatusSold,
			PriceCents:  80000,
			CreatedAt:   now.Add(-6 * 24 * time.Hour),
		},
		{
			ID:          "listing-19",
			SellerID:    "seller-5",
			Title:       "Watercolor landscape set (3 pieces)",
			Description: "Three framed watercolor landscapes, same artist, sold as a set.",
			Category:    "art",
			Type:        ListingTypeBuyNow,
			Status:      ListingStatusActiveBuyNow,
			PriceCents:  25000,
			CreatedAt:   now.Add(-30 * time.Minute),
		},
		{
			ID:              "listing-20",
			SellerID:        "seller-1",
			Title:           "Antique pocket watch collection",
			Description:     "Set of four 19th-century pocket watches, running condition varies.",
			Category:        "watches",
			Type:            ListingTypeAuction,
			Status:          ListingStatusUnsoldExpired,
			PriceCents:      40000,
			CurrentBidCents: 0,
			AuctionEndsAt:   &endedUnsold2,
			CreatedAt:       now.Add(-20 * 24 * time.Hour),
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
		{
			ID: "bid-4", ListingID: "listing-9", BuyerID: "buyer-4",
			AmountCents: 450000, PlacedAt: now.Add(-29 * time.Hour),
		},
		{
			ID: "bid-5", ListingID: "listing-9", BuyerID: "buyer-1",
			AmountCents: 470000, PlacedAt: now.Add(-5 * time.Minute),
		},
		{
			ID: "bid-6", ListingID: "listing-7", BuyerID: "buyer-5",
			AmountCents: 80000, PlacedAt: now.Add(-19 * time.Hour),
		},
		{
			ID: "bid-7", ListingID: "listing-7", BuyerID: "buyer-6",
			AmountCents: 95000, IsAutoBid: true, MaxAutoBidCents: 120000,
			PlacedAt: now.Add(-1 * time.Hour),
		},
		{
			ID: "bid-8", ListingID: "listing-17", BuyerID: "buyer-3",
			AmountCents: 50000, PlacedAt: endedSold2.Add(-2 * time.Hour),
		},
		{
			ID: "bid-9", ListingID: "listing-17", BuyerID: "buyer-1",
			AmountCents: 72000, PlacedAt: endedSold2.Add(-1 * time.Hour),
		},
	}

	orders := []Order{
		{
			ID: "order-1", ListingID: "listing-5", BuyerID: "buyer-3", SellerID: "seller-3",
			FinalPriceCents: 38000, Status: OrderStatusCompleted, CreatedAt: endedSold.Add(2 * time.Hour),
		},
		{
			ID: "order-2", ListingID: "listing-17", BuyerID: "buyer-1", SellerID: "seller-3",
			FinalPriceCents: 72000, Status: OrderStatusShipped, CreatedAt: endedSold2.Add(2 * time.Hour),
		},
		{
			ID: "order-3", ListingID: "listing-18", BuyerID: "buyer-4", SellerID: "seller-4",
			FinalPriceCents: 80000, Status: OrderStatusPaid, CreatedAt: now.Add(-6*24*time.Hour + time.Hour),
		},
	}

	return sellers, buyers, listings, bids, orders
}
