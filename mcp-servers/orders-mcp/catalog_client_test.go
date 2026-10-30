package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDoSendsMethodPathAndBody(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"ok": "yes"})
	}))
	t.Cleanup(srv.Close)

	c := newCatalogClient(srv.URL)
	var out map[string]string
	err := c.do(context.Background(), http.MethodPost, "/listings/l1/checkout", map[string]any{"buyerId": "b1"}, &out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotMethod != http.MethodPost || gotPath != "/listings/l1/checkout" {
		t.Fatalf("expected POST /listings/l1/checkout, got %s %s", gotMethod, gotPath)
	}
	if gotBody["buyerId"] != "b1" {
		t.Fatalf("expected request body to carry buyerId, got %+v", gotBody)
	}
	if out["ok"] != "yes" {
		t.Fatalf("expected decoded response, got %+v", out)
	}
}

func TestDoReturnsDomainErrorFor403(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "only the winning bidder can check out this auction"})
	}))
	t.Cleanup(srv.Close)

	c := newCatalogClient(srv.URL)
	err := c.do(context.Background(), http.MethodPost, "/listings/l1/checkout", map[string]any{}, nil)
	if err == nil {
		t.Fatal("expected an error")
	}
	msg, ok := domainError(err)
	if !ok {
		t.Fatalf("expected a domain error, got %v", err)
	}
	if msg != "only the winning bidder can check out this auction" {
		t.Fatalf("expected the catalog-api error message, got %q", msg)
	}
}

func TestDoTreats5xxAsNonDomainError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	c := newCatalogClient(srv.URL)
	err := c.do(context.Background(), http.MethodGet, "/orders/o1", nil, nil)
	if err == nil {
		t.Fatal("expected an error")
	}
	if _, ok := domainError(err); ok {
		t.Fatalf("expected a 500 to NOT be classified as a domain error, got ok=true for %v", err)
	}
}

func TestDoReturnsErrorWhenUnreachable(t *testing.T) {
	c := newCatalogClient("http://127.0.0.1:1")
	err := c.do(context.Background(), http.MethodGet, "/orders/o1", nil, nil)
	if err == nil {
		t.Fatal("expected an error for an unreachable server")
	}
	if _, ok := domainError(err); ok {
		t.Fatalf("expected a connection failure to NOT be classified as a domain error, got ok=true for %v", err)
	}
}
