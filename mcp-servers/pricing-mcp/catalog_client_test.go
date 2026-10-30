package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/day0ops/agora-demo/fixtures"
)

func TestSoldListingsRequestsTheSoldStatusFilter(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		_ = json.NewEncoder(w).Encode([]fixtures.Listing{{ID: "l1", Status: fixtures.ListingStatusSold}})
	}))
	t.Cleanup(srv.Close)

	c := newCatalogClient(srv.URL)
	listings, err := c.soldListings(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotQuery != "status=sold" {
		t.Fatalf("expected query status=sold, got %q", gotQuery)
	}
	if len(listings) != 1 || listings[0].ID != "l1" {
		t.Fatalf("unexpected listings: %+v", listings)
	}
}

func TestSoldListingsReturnsErrorOnFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	c := newCatalogClient(srv.URL)
	_, err := c.soldListings(context.Background())
	if err == nil {
		t.Fatal("expected an error for a 500 response")
	}
}

func TestSoldListingsReturnsErrorWhenUnreachable(t *testing.T) {
	c := newCatalogClient("http://127.0.0.1:1")
	_, err := c.soldListings(context.Background())
	if err == nil {
		t.Fatal("expected an error for an unreachable server")
	}
}
