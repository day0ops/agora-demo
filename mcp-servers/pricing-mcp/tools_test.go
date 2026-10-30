package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/day0ops/agora-demo/fixtures"
)

func TestSuggestPriceToolUsesLiveComps(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode([]fixtures.Listing{
			{Category: "cameras", Status: fixtures.ListingStatusSold, CurrentBidCents: 40000},
		})
	}))
	t.Cleanup(srv.Close)

	c := newCatalogClient(srv.URL)
	_, out, err := c.suggestPriceTool(context.Background(), nil, SuggestPriceInput{Category: "cameras"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Source != "live-comps" || out.SampleSize != 1 || out.MedianCents != 40000 {
		t.Fatalf("unexpected output: %+v", out)
	}
}

func TestSuggestPriceToolDegradesToBaselineWhenCatalogApiUnreachable(t *testing.T) {
	c := newCatalogClient("http://127.0.0.1:1")
	_, out, err := c.suggestPriceTool(context.Background(), nil, SuggestPriceInput{Category: "watches"})
	if err != nil {
		t.Fatalf("expected graceful degradation, not a tool error: %v", err)
	}
	// "baseline-degraded", not plain "baseline": the caller must be able to
	// tell "no sales yet" apart from "couldn't even check" (e.g. a
	// misconfigured CATALOG_API_URL that would otherwise silently look like
	// a quiet market forever).
	if out.Source != "baseline-degraded" {
		t.Fatalf("expected a degraded baseline when catalog-api is unreachable, got %+v", out)
	}
	if out.MedianCents != staticBaseline["watches"].MedianCents {
		t.Fatalf("expected the watches baseline, got %+v", out)
	}
}

func TestSuggestPriceToolReportsPlainBaselineWhenCatalogApiConfirmsNoSales(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode([]fixtures.Listing{})
	}))
	t.Cleanup(srv.Close)

	c := newCatalogClient(srv.URL)
	_, out, err := c.suggestPriceTool(context.Background(), nil, SuggestPriceInput{Category: "watches"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Source != "baseline" {
		t.Fatalf("expected a plain (non-degraded) baseline when catalog-api confirms zero sales, got %+v", out)
	}
}
