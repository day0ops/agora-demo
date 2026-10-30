package main

import (
	"context"
	"log"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type SuggestPriceInput struct {
	Category string `json:"category" jsonschema:"the listing category to suggest a price for, e.g. watches, books, cameras"`
}
type SuggestPriceOutput struct {
	LowCents    int64  `json:"low_cents" jsonschema:"suggested low end of the price range, in cents"`
	MedianCents int64  `json:"median_cents" jsonschema:"suggested price, in cents"`
	HighCents   int64  `json:"high_cents" jsonschema:"suggested high end of the price range, in cents"`
	SampleSize  int    `json:"sample_size" jsonschema:"number of real recent sales this suggestion is based on, 0 if none"`
	Source      string `json:"source" jsonschema:"\"live-comps\" when based on real sales, \"baseline\" when there is confirmed to be no sales history yet, \"baseline-degraded\" when the sales-history service could not be reached at all and this is a fallback guess, not a confirmed absence of data"`
}

// suggestPriceTool degrades gracefully: if catalog-api can't be reached for
// real sold-listing data, it falls back to the static baseline rather than
// failing the whole tool call -- a price suggestion is still more useful to
// a seller than a hard error. The failure is still surfaced on the wire via
// Source="baseline-degraded" (distinct from a confirmed, healthy "no sales
// yet" baseline), so a persistently misconfigured CATALOG_API_URL doesn't
// look identical to a quiet market forever.
func (c *catalogClient) suggestPriceTool(
	ctx context.Context, _ *mcp.CallToolRequest, in SuggestPriceInput,
) (*mcp.CallToolResult, SuggestPriceOutput, error) {
	sold, err := c.soldListings(ctx)
	degraded := err != nil
	if degraded {
		log.Printf("suggest_price: could not fetch sold listings, falling back to baseline: %v", err)
	}

	r := computeComps(sold, in.Category)
	out := SuggestPriceOutput(r)
	if degraded && out.Source == "baseline" {
		out.Source = "baseline-degraded"
	}
	return nil, out, nil
}
