package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/day0ops/marketplace-demo/fixtures"
)

// stubScorer lets tests control ScoreBidRisk's result without exercising the
// real heuristic, so tool-wiring tests stay independent of scoring logic.
type stubScorer struct {
	result RiskScore
	err    error
}

func (s stubScorer) ScoreBidRisk(context.Context, BidRiskInput) (RiskScore, error) {
	return s.result, s.err
}

func TestScoreBidRiskToolHappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/listings/l1":
			_ = json.NewEncoder(w).Encode(fixtures.Listing{ID: "l1", PriceCents: 1000})
		case "/listings/l1/bids":
			_ = json.NewEncoder(w).Encode([]fixtures.Bid{{BuyerID: "buyer-2", AmountCents: 1200}})
		}
	}))
	t.Cleanup(srv.Close)

	s := &server{
		catalog: newCatalogClient(srv.URL),
		scorer:  stubScorer{result: RiskScore{Score: 0.5, Reasons: []string{"test reason"}, ScoredBy: "heuristic"}},
	}
	_, out, err := s.scoreBidRiskTool(context.Background(), nil, ScoreBidRiskInput{ListingID: "l1", BuyerID: "buyer-1", AmountCents: 1500})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Risk.Score != 0.5 || len(out.Risk.Reasons) != 1 || out.Risk.ScoredBy != "heuristic" {
		t.Fatalf("unexpected output: %+v", out)
	}
}

func TestScoreBidRiskToolListingNotFoundIsDomainError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "listing not found"})
	}))
	t.Cleanup(srv.Close)

	s := &server{catalog: newCatalogClient(srv.URL), scorer: stubScorer{}}
	_, out, err := s.scoreBidRiskTool(context.Background(), nil, ScoreBidRiskInput{ListingID: "missing", BuyerID: "buyer-1", AmountCents: 1000})
	if err != nil {
		t.Fatalf("expected a domain rejection, not a tool error: %v", err)
	}
	if out.Error != "listing not found" {
		t.Fatalf("unexpected output: %+v", out)
	}
}

func TestScoreBidRiskToolRejectionOmitsFakeRiskFromJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "listing not found"})
	}))
	t.Cleanup(srv.Close)

	s := &server{catalog: newCatalogClient(srv.URL), scorer: stubScorer{}}
	_, out, err := s.scoreBidRiskTool(context.Background(), nil, ScoreBidRiskInput{ListingID: "missing", BuyerID: "buyer-1", AmountCents: 1000})
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
	if _, present := asMap["risk"]; present {
		t.Fatalf("expected a rejected lookup to omit the zero-valued \"risk\" field entirely (it would otherwise read as a confident zero-risk score), got %s", marshaled)
	}
}

func TestScoreBidRiskToolInfraFailureReturnsRealError(t *testing.T) {
	s := &server{catalog: newCatalogClient("http://127.0.0.1:1"), scorer: stubScorer{}}
	_, _, err := s.scoreBidRiskTool(context.Background(), nil, ScoreBidRiskInput{ListingID: "l1", BuyerID: "buyer-1", AmountCents: 1000})
	if err == nil {
		t.Fatal("expected a real tool error for an unreachable catalog-api")
	}
}

func TestScoreBidRiskToolScorerFailureReturnsRealError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/listings/l1":
			_ = json.NewEncoder(w).Encode(fixtures.Listing{ID: "l1"})
		case "/listings/l1/bids":
			_ = json.NewEncoder(w).Encode([]fixtures.Bid{})
		}
	}))
	t.Cleanup(srv.Close)

	s := &server{catalog: newCatalogClient(srv.URL), scorer: stubScorer{err: errors.New("scorer backend unavailable")}}
	_, _, err := s.scoreBidRiskTool(context.Background(), nil, ScoreBidRiskInput{ListingID: "l1", BuyerID: "buyer-1", AmountCents: 1000})
	if err == nil {
		t.Fatal("expected a scorer failure to propagate as a real tool error")
	}
}

func TestCheckSellerReputationToolHappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode([]fixtures.Listing{
			{Status: fixtures.ListingStatusSold},
			{Status: fixtures.ListingStatusActiveAuction},
		})
	}))
	t.Cleanup(srv.Close)

	s := &server{catalog: newCatalogClient(srv.URL), scorer: stubScorer{}}
	_, out, err := s.checkSellerReputationTool(context.Background(), nil, CheckSellerReputationInput{SellerID: "seller-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Reputation.TotalListings != 2 || out.Reputation.SoldCount != 1 {
		t.Fatalf("unexpected output: %+v", out)
	}
}

func TestCheckSellerReputationToolRequiresSellerID(t *testing.T) {
	// GET /listings?sellerId= on the real catalog-api is a filter, not a
	// lookup -- an empty value matches every listing in the marketplace, not
	// "no seller". This must be caught locally, not sent to catalog-api at
	// all (it would happily return 200 with the whole catalog).
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		_ = json.NewEncoder(w).Encode([]fixtures.Listing{})
	}))
	t.Cleanup(srv.Close)

	s := &server{catalog: newCatalogClient(srv.URL), scorer: stubScorer{}}
	_, out, err := s.checkSellerReputationTool(context.Background(), nil, CheckSellerReputationInput{})
	if err != nil {
		t.Fatalf("expected a domain rejection, not a tool error: %v", err)
	}
	if out.Error != "seller_id is required" {
		t.Fatalf("unexpected output: %+v", out)
	}
	if called {
		t.Fatal("expected the empty-seller_id guard to short-circuit before calling catalog-api")
	}
}

func TestCheckSellerReputationToolRejectionOmitsFakeReputationFromJSON(t *testing.T) {
	s := &server{catalog: newCatalogClient("unused"), scorer: stubScorer{}}
	_, out, err := s.checkSellerReputationTool(context.Background(), nil, CheckSellerReputationInput{})
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
	if _, present := asMap["reputation"]; present {
		t.Fatalf("expected a rejected lookup to omit the zero-valued \"reputation\" field entirely, got %s", marshaled)
	}
}
