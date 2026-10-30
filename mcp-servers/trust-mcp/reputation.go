package main

import "github.com/day0ops/marketplace-demo/fixtures"

// SellerReputation summarizes a seller's listing activity. It is a plain
// aggregation of real catalog-api data (not a fabricated trust score) --
// honest about being a listing-activity summary, not a sophisticated
// reputation model this dataset has no way to actually support.
type SellerReputation struct {
	SellerID           string  `json:"seller_id"`
	TotalListings      int     `json:"total_listings"`
	SoldCount          int     `json:"sold_count"`
	UnsoldExpiredCount int     `json:"unsold_expired_count"`
	WithdrawnCount     int     `json:"withdrawn_count"`
	ActiveCount        int     `json:"active_count"`
	SoldRate           float64 `json:"sold_rate"` // sold / (sold + unsold-expired); 0 if nothing has closed yet
	Tenure             string  `json:"tenure"`    // "new" (fewer than 3 listings ever) or "established"
}

// computeReputation summarizes listings, which should be every listing the
// given seller has ever posted.
func computeReputation(sellerID string, listings []fixtures.Listing) SellerReputation {
	rep := SellerReputation{SellerID: sellerID, TotalListings: len(listings)}

	for _, l := range listings {
		switch l.Status {
		case fixtures.ListingStatusSold:
			rep.SoldCount++
		case fixtures.ListingStatusUnsoldExpired:
			rep.UnsoldExpiredCount++
		case fixtures.ListingStatusWithdrawn:
			rep.WithdrawnCount++
		case fixtures.ListingStatusActiveAuction, fixtures.ListingStatusActiveBuyNow, fixtures.ListingStatusDraft:
			rep.ActiveCount++
		}
	}

	if closed := rep.SoldCount + rep.UnsoldExpiredCount; closed > 0 {
		rep.SoldRate = float64(rep.SoldCount) / float64(closed)
	}

	if rep.TotalListings < 3 {
		rep.Tenure = "new"
	} else {
		rep.Tenure = "established"
	}

	return rep
}
