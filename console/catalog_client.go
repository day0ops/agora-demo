package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/day0ops/agora-demo/fixtures"
)

// catalogClient is the console's client over catalog-api -- the single
// source of truth for every view. The console holds no marketplace data of
// its own.
type catalogClient struct {
	baseURL string
	http    *http.Client
}

func newCatalogClient(baseURL string) *catalogClient {
	return &catalogClient{baseURL: baseURL, http: &http.Client{Timeout: 10 * time.Second}}
}

// apiError represents a non-2xx response from catalog-api. A 4xx means the
// request was understood but rejected; callers may surface Message to the
// end user as a normal validation/conflict message. Anything else (5xx, or
// an unexpected 3xx) means catalog-api or the network itself failed.
type apiError struct {
	Status  int
	Message string
}

func (e *apiError) Error() string {
	return fmt.Sprintf("catalog-api responded %d: %s", e.Status, e.Message)
}

// domainError extracts a catalog-api rejection message, when err represents
// a request catalog-api understood but declined (4xx). ok is false for
// anything else (network failure, a 5xx).
func domainError(err error) (message string, ok bool) {
	var apiErr *apiError
	if errors.As(err, &apiErr) && apiErr.Status >= 400 && apiErr.Status < 500 {
		return apiErr.Message, true
	}
	return "", false
}

func (c *catalogClient) do(ctx context.Context, method, path string, body, out any) error {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request for %s %s: %w", method, path, err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	if err != nil {
		return fmt.Errorf("build request for %s %s: %w", method, path, err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("call catalog-api %s %s: %w", method, path, err)
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

	if out == nil || resp.StatusCode == http.StatusNoContent {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode response from %s %s: %w", method, path, err)
	}
	return nil
}

// -- Listings -----------------------------------------------------------

func (c *catalogClient) listListings(ctx context.Context, status, sellerID string) ([]fixtures.Listing, error) {
	q := url.Values{}
	if status != "" {
		q.Set("status", status)
	}
	if sellerID != "" {
		q.Set("sellerId", sellerID)
	}
	path := "/listings"
	if encoded := q.Encode(); encoded != "" {
		path += "?" + encoded
	}
	var listings []fixtures.Listing
	err := c.do(ctx, http.MethodGet, path, nil, &listings)
	return listings, err
}

func (c *catalogClient) getListing(ctx context.Context, listingID string) (fixtures.Listing, error) {
	var l fixtures.Listing
	err := c.do(ctx, http.MethodGet, "/listings/"+url.PathEscape(listingID), nil, &l)
	return l, err
}

type createListingRequest struct {
	SellerID      string               `json:"sellerId"`
	Title         string               `json:"title"`
	Description   string               `json:"description"`
	Category      string               `json:"category"`
	Type          fixtures.ListingType `json:"type"`
	PriceCents    int64                `json:"priceCents"`
	AuctionEndsAt *time.Time           `json:"auctionEndsAt,omitempty"`
}

func (c *catalogClient) createListing(ctx context.Context, req createListingRequest) (fixtures.Listing, error) {
	var l fixtures.Listing
	err := c.do(ctx, http.MethodPost, "/listings", req, &l)
	return l, err
}

// -- Bids -----------------------------------------------------------------

func (c *catalogClient) bidHistory(ctx context.Context, listingID string) ([]fixtures.Bid, error) {
	var bids []fixtures.Bid
	err := c.do(ctx, http.MethodGet, "/listings/"+url.PathEscape(listingID)+"/bids", nil, &bids)
	return bids, err
}

func (c *catalogClient) placeBid(ctx context.Context, listingID, buyerID string, amountCents int64) (fixtures.Bid, error) {
	var b fixtures.Bid
	body := struct {
		BuyerID     string `json:"buyerId"`
		AmountCents int64  `json:"amountCents"`
	}{buyerID, amountCents}
	err := c.do(ctx, http.MethodPost, "/listings/"+url.PathEscape(listingID)+"/bids", body, &b)
	return b, err
}

func (c *catalogClient) setAutoBidMax(ctx context.Context, listingID, buyerID string, maxCents int64) (fixtures.Listing, error) {
	var l fixtures.Listing
	body := struct {
		BuyerID  string `json:"buyerId"`
		MaxCents int64  `json:"maxCents"`
	}{buyerID, maxCents}
	err := c.do(ctx, http.MethodPost, "/listings/"+url.PathEscape(listingID)+"/autobid", body, &l)
	return l, err
}

func (c *catalogClient) withdrawBid(ctx context.Context, listingID, bidID string) error {
	path := "/listings/" + url.PathEscape(listingID) + "/bids/" + url.PathEscape(bidID)
	return c.do(ctx, http.MethodDelete, path, nil, nil)
}

// -- Orders -----------------------------------------------------------------

func (c *catalogClient) checkout(ctx context.Context, listingID, buyerID string) (fixtures.Order, error) {
	var o fixtures.Order
	body := struct {
		BuyerID string `json:"buyerId"`
	}{buyerID}
	err := c.do(ctx, http.MethodPost, "/listings/"+url.PathEscape(listingID)+"/checkout", body, &o)
	return o, err
}

func (c *catalogClient) listOrders(ctx context.Context, buyerID, sellerID string) ([]fixtures.Order, error) {
	q := url.Values{}
	if buyerID != "" {
		q.Set("buyerId", buyerID)
	}
	if sellerID != "" {
		q.Set("sellerId", sellerID)
	}
	path := "/orders"
	if encoded := q.Encode(); encoded != "" {
		path += "?" + encoded
	}
	var orders []fixtures.Order
	err := c.do(ctx, http.MethodGet, path, nil, &orders)
	return orders, err
}
