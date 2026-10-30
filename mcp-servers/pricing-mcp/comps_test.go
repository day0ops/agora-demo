package main

import (
	"testing"

	"github.com/day0ops/marketplace-demo/fixtures"
)

func TestComputeCompsUsesLiveSoldPricesWhenAvailable(t *testing.T) {
	sold := []fixtures.Listing{
		{Category: "cameras", Status: fixtures.ListingStatusSold, CurrentBidCents: 30000},
		{Category: "cameras", Status: fixtures.ListingStatusSold, CurrentBidCents: 50000},
		{Category: "cameras", Status: fixtures.ListingStatusSold, CurrentBidCents: 40000},
		{Category: "watches", Status: fixtures.ListingStatusSold, CurrentBidCents: 999999},         // different category
		{Category: "cameras", Status: fixtures.ListingStatusActiveAuction, CurrentBidCents: 20000}, // not sold
	}

	got := computeComps(sold, "cameras")
	if got.Source != "live-comps" {
		t.Fatalf("expected live-comps source, got %q", got.Source)
	}
	if got.SampleSize != 3 {
		t.Fatalf("expected sample size 3, got %d", got.SampleSize)
	}
	if got.LowCents != 30000 || got.HighCents != 50000 || got.MedianCents != 40000 {
		t.Fatalf("expected low=30000 median=40000 high=50000, got %+v", got)
	}
}

func TestComputeCompsMedianOfEvenSample(t *testing.T) {
	sold := []fixtures.Listing{
		{Category: "books", Status: fixtures.ListingStatusSold, CurrentBidCents: 1000},
		{Category: "books", Status: fixtures.ListingStatusSold, CurrentBidCents: 2000},
		{Category: "books", Status: fixtures.ListingStatusSold, CurrentBidCents: 3000},
		{Category: "books", Status: fixtures.ListingStatusSold, CurrentBidCents: 4000},
	}

	got := computeComps(sold, "books")
	if got.MedianCents != 2500 {
		t.Fatalf("expected median of {1000,2000,3000,4000} to be 2500, got %d", got.MedianCents)
	}
}

func TestComputeCompsFallsBackToKnownBaselineWithNoSoldData(t *testing.T) {
	got := computeComps(nil, "watches")
	if got.Source != "baseline" {
		t.Fatalf("expected baseline source, got %q", got.Source)
	}
	if got.SampleSize != 0 {
		t.Fatalf("expected sample size 0 for a baseline guess, got %d", got.SampleSize)
	}
	want := staticBaseline["watches"]
	if got.LowCents != want.LowCents || got.MedianCents != want.MedianCents || got.HighCents != want.HighCents {
		t.Fatalf("expected the watches baseline %+v, got %+v", want, got)
	}
}

func TestComputeCompsFallsBackToDefaultForUnknownCategory(t *testing.T) {
	got := computeComps(nil, "antique-furniture")
	if got.Source != "baseline" {
		t.Fatalf("expected baseline source, got %q", got.Source)
	}
	if got.LowCents != defaultBaseline.LowCents ||
		got.MedianCents != defaultBaseline.MedianCents ||
		got.HighCents != defaultBaseline.HighCents {
		t.Fatalf("expected the default baseline range, got %+v", got)
	}
}

func TestComputeCompsNormalizesCategoryCasingAndWhitespace(t *testing.T) {
	sold := []fixtures.Listing{
		{Category: "cameras", Status: fixtures.ListingStatusSold, CurrentBidCents: 40000},
	}

	got := computeComps(sold, "  Cameras ")
	if got.Source != "live-comps" || got.SampleSize != 1 {
		t.Fatalf("expected normalized category to match real sold data, got %+v", got)
	}

	gotBaseline := computeComps(nil, "WATCHES")
	want := staticBaseline["watches"]
	if gotBaseline.MedianCents != want.MedianCents {
		t.Fatalf("expected normalized category to match the watches baseline, got %+v", gotBaseline)
	}
}

func TestComputeCompsIgnoresZeroPriceSoldListings(t *testing.T) {
	// A listing marked Sold with CurrentBidCents still 0 would be a data bug
	// (every real sale has a positive final price) -- must not pollute comps.
	sold := []fixtures.Listing{
		{Category: "home", Status: fixtures.ListingStatusSold, CurrentBidCents: 0},
	}
	got := computeComps(sold, "home")
	if got.Source != "baseline" || got.SampleSize != 0 {
		t.Fatalf("expected a zero-price sold listing to be ignored, got %+v", got)
	}
}
