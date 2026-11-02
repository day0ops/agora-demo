package main

import (
	"context"

	"github.com/day0ops/agora-demo/fixtures"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type SearchListingsInput struct {
	Category      string `json:"category,omitempty" jsonschema:"only return listings in this category, e.g. watches, books, cameras, electronics, art, instruments, home, memorabilia"`
	MaxPriceCents int64  `json:"max_price_cents,omitempty" jsonschema:"only return listings at or below this price in cents (an active auction's current bid counts as its price)"`
	MinPriceCents int64  `json:"min_price_cents,omitempty" jsonschema:"only return listings at or above this price in cents"`
	Keyword       string `json:"keyword,omitempty" jsonschema:"only return listings whose title or description contains this word or phrase"`
}
type SearchListingsOutput struct {
	Listings []fixtures.Listing `json:"listings" jsonschema:"active listings (open for bidding or buy-now purchase right now) matching the search"`
}

// searchListingsTool only ever returns listings a buyer could actually act
// on right now (an open auction or a live buy-now) - sold, withdrawn,
// expired, and draft listings are filtered out here since catalog-api's own
// filter is a single exact status match, not a set. There is no domain-error
// case worth a structured Error field here (unlike bids-mcp's lookups by
// ID) - a search either returns results (possibly zero) or fails at the
// transport level, which is a real Go error, not a declined request.
func (c *catalogClient) searchListingsTool(
	ctx context.Context, _ *mcp.CallToolRequest, in SearchListingsInput,
) (*mcp.CallToolResult, SearchListingsOutput, error) {
	listings, err := c.listListings(ctx, listingQuery(in))
	if err != nil {
		return nil, SearchListingsOutput{}, err
	}

	active := make([]fixtures.Listing, 0, len(listings))
	for _, l := range listings {
		if l.Status == fixtures.ListingStatusActiveAuction || l.Status == fixtures.ListingStatusActiveBuyNow {
			active = append(active, l)
		}
	}
	return nil, SearchListingsOutput{Listings: active}, nil
}
