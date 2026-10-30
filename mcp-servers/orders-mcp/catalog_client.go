package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// catalogClient is a thin HTTP client over catalog-api's checkout/orders
// endpoints. orders-mcp holds no marketplace data of its own -- catalog-api
// is the single source of truth, so every tool call is a live round trip.
type catalogClient struct {
	baseURL string
	http    *http.Client
}

func newCatalogClient(baseURL string) *catalogClient {
	return &catalogClient{baseURL: baseURL, http: &http.Client{Timeout: 10 * time.Second}}
}

// apiError represents a non-2xx response from catalog-api. A 4xx means the
// request was understood but rejected (bad input, a business-rule conflict
// like checking out someone else's won auction) -- callers should surface
// Message to the agent as data, not as a tool failure. Anything else (5xx,
// or an unexpected 3xx this client never deliberately produces) means
// catalog-api or the network itself failed, which callers should propagate
// as a real tool error.
type apiError struct {
	Status  int
	Message string
}

func (e *apiError) Error() string {
	return fmt.Sprintf("catalog-api responded %d: %s", e.Status, e.Message)
}

// domainError extracts a catalog-api rejection message suitable for a tool's
// output field, when err represents a request catalog-api understood but
// declined (4xx, including 403 "not the winning bidder"). ok is false for
// anything else (network failure, a 5xx), which the caller should instead
// return as a real Go error.
func domainError(err error) (message string, ok bool) {
	var apiErr *apiError
	if errors.As(err, &apiErr) && apiErr.Status >= 400 && apiErr.Status < 500 {
		return apiErr.Message, true
	}
	return "", false
}

// do sends a JSON request to catalog-api and decodes a JSON response into
// out (ignored if nil, or if the response has no body). body is marshaled as
// the request body when non-nil.
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
