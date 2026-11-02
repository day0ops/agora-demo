package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/day0ops/agora-demo/fixtures"
)

func TestSearchListingsToolFiltersOutInactiveStatuses(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode([]fixtures.Listing{
			{ID: "cam1", Category: "cameras", Status: fixtures.ListingStatusActiveBuyNow, PriceCents: 45000},
			{ID: "cam2", Category: "cameras", Status: fixtures.ListingStatusSold, PriceCents: 22000},
			{ID: "watch1", Category: "watches", Status: fixtures.ListingStatusActiveAuction, PriceCents: 500000},
			{ID: "watch2", Category: "watches", Status: fixtures.ListingStatusDraft, PriceCents: 10000},
		})
	}))
	t.Cleanup(srv.Close)

	c := newCatalogClient(srv.URL)
	_, out, err := c.searchListingsTool(context.Background(), nil, SearchListingsInput{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Listings) != 2 {
		t.Fatalf("expected only the 2 active listings, got %+v", out.Listings)
	}
	for _, l := range out.Listings {
		if l.ID == "cam2" || l.ID == "watch2" {
			t.Fatalf("expected sold/draft listings excluded, got %+v", l)
		}
	}
}

func TestSearchListingsToolPassesFiltersAsQueryParams(t *testing.T) {
	var gotQuery url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		_ = json.NewEncoder(w).Encode([]fixtures.Listing{})
	}))
	t.Cleanup(srv.Close)

	c := newCatalogClient(srv.URL)
	_, _, err := c.searchListingsTool(context.Background(), nil, SearchListingsInput{
		Category:      "cameras",
		MaxPriceCents: 50000,
		MinPriceCents: 10000,
		Keyword:       "nikon",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotQuery.Get("category") != "cameras" {
		t.Fatalf("expected category=cameras, got %q", gotQuery.Get("category"))
	}
	if gotQuery.Get("maxPriceCents") != "50000" {
		t.Fatalf("expected maxPriceCents=50000, got %q", gotQuery.Get("maxPriceCents"))
	}
	if gotQuery.Get("minPriceCents") != "10000" {
		t.Fatalf("expected minPriceCents=10000, got %q", gotQuery.Get("minPriceCents"))
	}
	if gotQuery.Get("q") != "nikon" {
		t.Fatalf("expected q=nikon, got %q", gotQuery.Get("q"))
	}
}

func TestSearchListingsToolReturnsErrorWhenCatalogApiUnreachable(t *testing.T) {
	c := newCatalogClient("http://127.0.0.1:1")
	_, _, err := c.searchListingsTool(context.Background(), nil, SearchListingsInput{})
	if err == nil {
		t.Fatal("expected an error when catalog-api is unreachable")
	}
}
