package main

import (
	"sort"
	"strings"

	"github.com/day0ops/agora-demo/fixtures"
)

// priceRangeCents is a suggested pricing range, in cents. SampleSize is the
// number of real sold listings the range was computed from; Source is
// "live-comps" when SampleSize > 0, or "baseline" when there was no real
// sales history yet and a static starting guess was used instead. (A caller
// may further rewrite Source to "baseline-degraded" when the baseline was
// used only because sold-listing data couldn't be fetched at all -- see
// suggestPriceTool.)
type priceRangeCents struct {
	LowCents    int64
	MedianCents int64
	HighCents   int64
	SampleSize  int
	Source      string
}

// staticBaseline is a rough, seed-data-informed starting point for a
// category with no real sold-listing history yet -- the common case early in
// a fresh demo, before enough auctions/sales have happened to form real
// comps. Note: catalog-api does not yet track buy-now purchases as sales
// (only closed auctions reach ListingStatusSold), so buy-now listings never
// contribute to live comps until that lands. Keys are normalized (lowercase,
// trimmed) by normalizeCategory -- look up with that, not a raw string.
var staticBaseline = map[string]priceRangeCents{
	"watches":     {LowCents: 100000, MedianCents: 350000, HighCents: 1200000},
	"books":       {LowCents: 2000, MedianCents: 15000, HighCents: 150000},
	"cameras":     {LowCents: 10000, MedianCents: 45000, HighCents: 250000},
	"home":        {LowCents: 3000, MedianCents: 12000, HighCents: 80000},
	"memorabilia": {LowCents: 5000, MedianCents: 25000, HighCents: 500000},
}

// defaultBaseline is used for a category with no entry in staticBaseline and
// no sold history -- deliberately wide, so a seller can tell this is an
// uninformed guess rather than a real market signal.
var defaultBaseline = priceRangeCents{LowCents: 1000, MedianCents: 10000, HighCents: 100000}

// normalizeCategory makes category matching resilient to the casing/spacing
// variation an LLM agent's free-text input will inevitably produce (e.g.
// "Watches" or " watches "), so it doesn't silently miss real comps or a
// known baseline and fall through to the uninformative default.
func normalizeCategory(category string) string {
	return strings.ToLower(strings.TrimSpace(category))
}

// computeComps suggests a price range for category. sold should be every
// listing catalog-api currently reports with status "sold" (across all
// categories); computeComps filters to category itself.
func computeComps(sold []fixtures.Listing, category string) priceRangeCents {
	category = normalizeCategory(category)

	var prices []int64
	for _, l := range sold {
		if normalizeCategory(l.Category) != category || l.Status != fixtures.ListingStatusSold {
			continue
		}
		if l.CurrentBidCents <= 0 {
			continue
		}
		prices = append(prices, l.CurrentBidCents)
	}

	if len(prices) == 0 {
		r := defaultBaseline
		if baseline, ok := staticBaseline[category]; ok {
			r = baseline
		}
		r.SampleSize = 0
		r.Source = "baseline"
		return r
	}

	sort.Slice(prices, func(i, j int) bool { return prices[i] < prices[j] })
	return priceRangeCents{
		LowCents:    prices[0],
		MedianCents: medianOfSorted(prices),
		HighCents:   prices[len(prices)-1],
		SampleSize:  len(prices),
		Source:      "live-comps",
	}
}

// medianOfSorted returns the median of an already-ascending-sorted,
// non-empty slice. The even-count average is computed as lo+(hi-lo)/2 rather
// than (lo+hi)/2 to avoid a theoretical int64 overflow on the sum.
func medianOfSorted(sorted []int64) int64 {
	n := len(sorted)
	if n%2 == 1 {
		return sorted[n/2]
	}
	lo, hi := sorted[n/2-1], sorted[n/2]
	return lo + (hi-lo)/2
}
