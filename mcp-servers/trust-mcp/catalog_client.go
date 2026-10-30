package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/day0ops/marketplace-demo/fixtures"
)

// catalogClient is a thin, read-only HTTP client over catalog-api's listings
// and bids endpoints. trust-mcp holds no marketplace data of its own.
type catalogClient struct {
	baseURL string
	http    *http.Client
}

func newCatalogClient(baseURL string) *catalogClient {
	return &catalogClient{baseURL: baseURL, http: &http.Client{Timeout: 10 * time.Second}}
}

// apiError represents a non-2xx response from catalog-api. A 4xx means the
// request was understood but rejected (e.g. an unknown listing ID) --
// callers should surface Message to the agent as data, not as a tool
// failure. Anything else (5xx, or an unexpected 3xx) means catalog-api or
// the network itself failed, which callers should propagate as a real tool
// error.
type apiError struct {
	Status  int
	Message string
}

func (e *apiError) Error() string {
	return fmt.Sprintf("catalog-api responded %d: %s", e.Status, e.Message)
}

// domainError extracts a catalog-api rejection message suitable for a tool's
// output field, when err represents a request catalog-api understood but
// declined (4xx). ok is false for anything else (network failure, a 5xx),
// which the caller should instead return as a real Go error.
func domainError(err error) (message string, ok bool) {
	var apiErr *apiError
	if errors.As(err, &apiErr) && apiErr.Status >= 400 && apiErr.Status < 500 {
		return apiErr.Message, true
	}
	return "", false
}

func (c *catalogClient) get(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("build request for %s: %w", path, err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("call catalog-api %s: %w", path, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 300 {
		var errBody struct {
			Error string `json:"error"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		msg := errBody.Error
		if msg == "" {
			msg = fmt.Sprintf("request failed with status %d", resp.StatusCode)
		}
		return &apiError{Status: resp.StatusCode, Message: msg}
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode response from %s: %w", path, err)
	}
	return nil
}

// listing fetches a single listing by ID.
func (c *catalogClient) listing(ctx context.Context, listingID string) (fixtures.Listing, error) {
	var l fixtures.Listing
	err := c.get(ctx, "/listings/"+url.PathEscape(listingID), &l)
	return l, err
}

// bidHistory fetches every bid already placed on a listing, oldest first.
func (c *catalogClient) bidHistory(ctx context.Context, listingID string) ([]fixtures.Bid, error) {
	var bids []fixtures.Bid
	err := c.get(ctx, "/listings/"+url.PathEscape(listingID)+"/bids", &bids)
	return bids, err
}

// listingsBySeller fetches every listing a seller has ever posted.
func (c *catalogClient) listingsBySeller(ctx context.Context, sellerID string) ([]fixtures.Listing, error) {
	var listings []fixtures.Listing
	err := c.get(ctx, "/listings?sellerId="+url.QueryEscape(sellerID), &listings)
	return listings, err
}
