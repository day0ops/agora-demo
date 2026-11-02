package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/day0ops/agora-demo/fixtures"
)

func newTestServer(catalogURL string) *consoleServer {
	return newConsoleServer(newCatalogClient(catalogURL), newEventHub(), AgentInvokerStub{})
}

func TestHandleVersionReturnsBuildVersion(t *testing.T) {
	srv := newTestServer("http://unused")
	rec := httptest.NewRecorder()
	srv.routes().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/version", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var resp versionResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Version != version {
		t.Fatalf("expected version %q, got %q", version, resp.Version)
	}
}

func TestHandleListListingsRelaysCatalogData(t *testing.T) {
	catalog := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]fixtures.Listing{{ID: "l1"}})
	}))
	t.Cleanup(catalog.Close)

	srv := newTestServer(catalog.URL)
	rec := httptest.NewRecorder()
	srv.routes().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/listings", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var listings []fixtures.Listing
	if err := json.Unmarshal(rec.Body.Bytes(), &listings); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(listings) != 1 || listings[0].ID != "l1" {
		t.Fatalf("unexpected listings: %+v", listings)
	}
}

func TestHandleGetListingRelaysCatalogErrorStatusAndMessage(t *testing.T) {
	catalog := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "listing not found"})
	}))
	t.Cleanup(catalog.Close)

	srv := newTestServer(catalog.URL)
	rec := httptest.NewRecorder()
	srv.routes().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/listings/missing", nil))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected the console to relay catalog-api's 404 verbatim, got %d", rec.Code)
	}
	var body errorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Error != "listing not found" {
		t.Fatalf("expected catalog-api's message to be relayed, got %q", body.Error)
	}
}

func TestHandleGetListingReports502WhenCatalogUnreachable(t *testing.T) {
	// A closed server guarantees connection failure, not any particular HTTP
	// status: this exercises the non-apiError branch of writeCatalogError.
	catalog := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	catalog.Close()

	srv := newTestServer(catalog.URL)
	rec := httptest.NewRecorder()
	srv.routes().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/listings/l1", nil))

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("expected 502 when catalog-api is unreachable, got %d", rec.Code)
	}
}

type fakeInvoker struct {
	reply    string
	err      error
	gotToken string
}

func (f *fakeInvoker) Send(_ context.Context, _, _, _, token string) (string, error) {
	f.gotToken = token
	if f.err != nil {
		return "", f.err
	}
	return f.reply, nil
}

func TestHandleChatReturnsReplyOnSuccess(t *testing.T) {
	inv := &fakeInvoker{reply: "Here are three listings you might like."}
	srv := newConsoleServer(newCatalogClient("http://unused"), newEventHub(), inv)

	body, _ := json.Marshal(chatRequest{Agent: "shopping-assistant", SessionID: "s1", Message: "hi"})
	req := httptest.NewRequest(http.MethodPost, "/account/chat", bytes.NewReader(body))
	req.Header.Set("X-Agora-User-Token", "user-token-123")
	rec := httptest.NewRecorder()
	srv.routes().ServeHTTP(rec, req)

	var resp chatResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Reply != "Here are three listings you might like." || resp.Error != "" {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if inv.gotToken != "user-token-123" {
		t.Fatalf("expected the forwarded token to reach the invoker, got %q", inv.gotToken)
	}
}

func TestHandleChatReturnsNotAuthenticatedErrorWithNoTokenHeader(t *testing.T) {
	inv := &fakeInvoker{reply: "should not be used"}
	srv := newConsoleServer(newCatalogClient("http://unused"), newEventHub(), inv)

	body, _ := json.Marshal(chatRequest{Agent: "shopping-assistant", SessionID: "s1", Message: "hi"})
	rec := httptest.NewRecorder()
	srv.routes().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/account/chat", bytes.NewReader(body)))

	var resp chatResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Reply != "" {
		t.Fatalf("expected no reply when unauthenticated, got %q", resp.Reply)
	}
	if resp.Error != ErrNotAuthenticated.Error() {
		t.Fatalf("expected the not-authenticated error, got %q", resp.Error)
	}
	if inv.gotToken != "" {
		t.Fatal("expected the invoker never to be called with no token")
	}
}

func TestHandleChatReturns200WithErrorWhenInvokerFails(t *testing.T) {
	srv := newConsoleServer(newCatalogClient("http://unused"), newEventHub(), AgentInvokerStub{})

	body, _ := json.Marshal(chatRequest{Agent: "shopping-assistant", SessionID: "s1", Message: "hi"})
	req := httptest.NewRequest(http.MethodPost, "/account/chat", bytes.NewReader(body))
	req.Header.Set("X-Agora-User-Token", "user-token-123")
	rec := httptest.NewRecorder()
	srv.routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 (agent unavailability is a normal chat response, not a transport error), got %d", rec.Code)
	}
	var resp chatResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Error == "" || resp.Reply != "" {
		t.Fatalf("expected an Error and no Reply, got %+v", resp)
	}
}

func TestHandleEventsStreamsPublishedEvents(t *testing.T) {
	srv := newTestServer("http://unused")
	httpServer := httptest.NewServer(srv.routes())
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, httpServer.URL+"/api/events", nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("connect to event stream: %v", err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })

	// Give the handler a moment to register its subscription before
	// publishing, since subscription happens after headers are already
	// flushed to the client.
	deadline := time.Now().Add(time.Second)
	for srv.hub.subscriberCount() == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if srv.hub.subscriberCount() != 1 {
		t.Fatalf("expected the SSE client to be subscribed, got %d subscribers", srv.hub.subscriberCount())
	}

	srv.hub.publish(Event{Type: "bid.placed", Data: "hello"})

	reader := bufio.NewReader(resp.Body)
	dataLine, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("read data line: %v", err)
	}
	trimmed := strings.TrimSpace(dataLine)
	if !strings.HasPrefix(trimmed, "data: ") {
		t.Fatalf("unexpected SSE line: %q", dataLine)
	}
	var evt Event
	if err := json.Unmarshal([]byte(strings.TrimPrefix(trimmed, "data: ")), &evt); err != nil {
		t.Fatalf("decode event payload: %v", err)
	}
	if evt.Type != "bid.placed" || evt.Data != "hello" {
		t.Fatalf("unexpected event: %+v", evt)
	}

	// The SSE spec terminates a frame with a blank line - without it, a real
	// browser EventSource buffers forever and never dispatches the event.
	terminator, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("read frame terminator: %v", err)
	}
	if strings.TrimRight(terminator, "\n") != "" {
		t.Fatalf("expected a blank line terminating the SSE frame, got %q", terminator)
	}
}
