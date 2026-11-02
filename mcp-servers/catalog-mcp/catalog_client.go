package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/day0ops/agora-demo/fixtures"
)

// catalogClient is a thin HTTP client over catalog-api's read-only listings
// endpoint. catalog-mcp holds no listing data of its own.
type catalogClient struct {
	baseURL string
	http    *http.Client
}

func newCatalogClient(baseURL string) *catalogClient {
	return &catalogClient{baseURL: baseURL, http: &http.Client{Timeout: 10 * time.Second}}
}

// listingQuery mirrors catalog-api's GET /listings query parameters relevant
// to search_listings. A zero-value field is omitted from the request.
type listingQuery struct {
	Category      string
	MaxPriceCents int64
	MinPriceCents int64
	Keyword       string
}

func (c *catalogClient) listListings(ctx context.Context, q listingQuery) ([]fixtures.Listing, error) {
	params := url.Values{}
	if q.Category != "" {
		params.Set("category", q.Category)
	}
	if q.MaxPriceCents > 0 {
		params.Set("maxPriceCents", strconv.FormatInt(q.MaxPriceCents, 10))
	}
	if q.MinPriceCents > 0 {
		params.Set("minPriceCents", strconv.FormatInt(q.MinPriceCents, 10))
	}
	if q.Keyword != "" {
		params.Set("q", q.Keyword)
	}

	u := c.baseURL + "/listings"
	if encoded := params.Encode(); encoded != "" {
		u += "?" + encoded
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("build request for listings: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call catalog-api for listings: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("catalog-api returned status %d for listings", resp.StatusCode)
	}

	var listings []fixtures.Listing
	if err := json.NewDecoder(resp.Body).Decode(&listings); err != nil {
		return nil, fmt.Errorf("decode listings response: %w", err)
	}
	return listings, nil
}
