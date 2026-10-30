package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/day0ops/marketplace-demo/fixtures"
)

// catalogClient is a thin HTTP client over catalog-api's read-only listings
// endpoint. pricing-mcp holds no sales data of its own.
type catalogClient struct {
	baseURL string
	http    *http.Client
}

func newCatalogClient(baseURL string) *catalogClient {
	return &catalogClient{baseURL: baseURL, http: &http.Client{Timeout: 10 * time.Second}}
}

// soldListings returns every listing catalog-api currently reports with
// status "sold", across all categories -- computeComps filters by category.
func (c *catalogClient) soldListings(ctx context.Context) ([]fixtures.Listing, error) {
	u := c.baseURL + "/listings?status=" + url.QueryEscape(string(fixtures.ListingStatusSold))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("build request for sold listings: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call catalog-api for sold listings: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("catalog-api returned status %d for sold listings", resp.StatusCode)
	}

	var listings []fixtures.Listing
	if err := json.NewDecoder(resp.Body).Decode(&listings); err != nil {
		return nil, fmt.Errorf("decode sold listings response: %w", err)
	}
	return listings, nil
}
