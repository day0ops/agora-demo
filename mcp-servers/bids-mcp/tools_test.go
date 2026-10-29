package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/day0ops/marketplace-demo/fixtures"
)

func TestPlaceBidToolSuccess(t *testing.T) {
	var gotPath string
	var gotBody placeBidBody
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(fixtures.Bid{ID: "bid-1", ListingID: "l1", BuyerID: "b1", AmountCents: 1500})
	}))
	t.Cleanup(srv.Close)

	c := newCatalogClient(srv.URL)
	_, out, err := c.placeBidTool(context.Background(), nil, PlaceBidInput{ListingID: "l1", BuyerID: "b1", AmountCents: 1500})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotPath != "/listings/l1/bids" {
		t.Fatalf("expected path /listings/l1/bids, got %s", gotPath)
	}
	if gotBody.BuyerID != "b1" || gotBody.AmountCents != 1500 {
		t.Fatalf("unexpected request body: %+v", gotBody)
	}
	if out.Error != "" || out.Bid.ID != "bid-1" {
		t.Fatalf("unexpected output: %+v", out)
	}
}

func TestPlaceBidToolDomainRejection(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "auction is not open for bidding"})
	}))
	t.Cleanup(srv.Close)

	c := newCatalogClient(srv.URL)
	_, out, err := c.placeBidTool(context.Background(), nil, PlaceBidInput{ListingID: "l1", BuyerID: "b1", AmountCents: 1500})
	if err != nil {
		t.Fatalf("expected a domain rejection surfaced via output, not a tool error, got %v", err)
	}
	if out.Error != "auction is not open for bidding" {
		t.Fatalf("expected the rejection reason in Output.Error, got %+v", out)
	}
}

func TestPlaceBidToolInfraFailureReturnsRealError(t *testing.T) {
	c := newCatalogClient("http://127.0.0.1:1")
	_, _, err := c.placeBidTool(context.Background(), nil, PlaceBidInput{ListingID: "l1", BuyerID: "b1", AmountCents: 1500})
	if err == nil {
		t.Fatal("expected a real tool error for an unreachable catalog-api")
	}
}

func TestGetCurrentPriceTool(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/listings/l1" {
			t.Errorf("expected path /listings/l1, got %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(fixtures.Listing{ID: "l1", CurrentBidCents: 2000})
	}))
	t.Cleanup(srv.Close)

	c := newCatalogClient(srv.URL)
	_, out, err := c.getCurrentPriceTool(context.Background(), nil, GetCurrentPriceInput{ListingID: "l1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Listing.CurrentBidCents != 2000 {
		t.Fatalf("unexpected output: %+v", out)
	}
}

func TestGetBidHistoryTool(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/listings/l1/bids" {
			t.Errorf("expected path /listings/l1/bids, got %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode([]fixtures.Bid{{ID: "b1"}, {ID: "b2"}})
	}))
	t.Cleanup(srv.Close)

	c := newCatalogClient(srv.URL)
	_, out, err := c.getBidHistoryTool(context.Background(), nil, GetBidHistoryInput{ListingID: "l1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Bids) != 2 {
		t.Fatalf("expected 2 bids, got %+v", out.Bids)
	}
}

func TestSetAutoBidMaxTool(t *testing.T) {
	var gotPath string
	var gotBody setAutoBidMaxBody
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_ = json.NewEncoder(w).Encode(fixtures.Listing{ID: "l1", CurrentBidCents: 1000})
	}))
	t.Cleanup(srv.Close)

	c := newCatalogClient(srv.URL)
	_, out, err := c.setAutoBidMaxTool(context.Background(), nil, SetAutoBidMaxInput{ListingID: "l1", BuyerID: "b1", MaxCents: 6000})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotPath != "/listings/l1/autobid" {
		t.Fatalf("expected path /listings/l1/autobid, got %s", gotPath)
	}
	if gotBody.BuyerID != "b1" || gotBody.MaxCents != 6000 {
		t.Fatalf("unexpected request body: %+v", gotBody)
	}
	if out.Listing.CurrentBidCents != 1000 {
		t.Fatalf("unexpected output: %+v", out)
	}
}

func TestWithdrawBidTool(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)

	c := newCatalogClient(srv.URL)
	_, out, err := c.withdrawBidTool(context.Background(), nil, WithdrawBidInput{ListingID: "l1", BidID: "bid-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMethod != http.MethodDelete || gotPath != "/listings/l1/bids/bid-1" {
		t.Fatalf("expected DELETE /listings/l1/bids/bid-1, got %s %s", gotMethod, gotPath)
	}
	if !out.Success {
		t.Fatalf("expected Success=true, got %+v", out)
	}
}

func TestWithdrawBidToolNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "bid not found"})
	}))
	t.Cleanup(srv.Close)

	c := newCatalogClient(srv.URL)
	_, out, err := c.withdrawBidTool(context.Background(), nil, WithdrawBidInput{ListingID: "l1", BidID: "missing"})
	if err != nil {
		t.Fatalf("expected a domain rejection, not a tool error: %v", err)
	}
	if out.Success || out.Error != "bid not found" {
		t.Fatalf("unexpected output: %+v", out)
	}
}

func TestPlaceBidToolRejectionOmitsFakeBidFromJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "bid amount is too low"})
	}))
	t.Cleanup(srv.Close)

	c := newCatalogClient(srv.URL)
	_, out, err := c.placeBidTool(context.Background(), nil, PlaceBidInput{ListingID: "l1", BuyerID: "b1", AmountCents: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	marshaled, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("marshal output: %v", err)
	}
	var asMap map[string]json.RawMessage
	if err := json.Unmarshal(marshaled, &asMap); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if _, present := asMap["bid"]; present {
		t.Fatalf("expected a rejected bid to omit the zero-valued \"bid\" field entirely, got %s", marshaled)
	}
}

func TestListingPathEscapesAgentSuppliedIDs(t *testing.T) {
	got := listingPath("l1/evil", "/bids")
	want := "/listings/l1%2Fevil/bids"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}
