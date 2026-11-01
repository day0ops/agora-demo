package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAgentInvokerStubReturnsNotConfigured(t *testing.T) {
	var s AgentInvokerStub
	_, err := s.Send(context.Background(), "shopping-assistant", "session-1", "hello", "token-1")
	if !errors.Is(err, ErrAgentInvokerNotConfigured) {
		t.Fatalf("expected ErrAgentInvokerNotConfigured, got %v", err)
	}
}

func TestAgentInvokerHTTPReturnsNotAuthenticatedWithNoToken(t *testing.T) {
	inv := newAgentInvokerHTTP("http://unused")
	_, err := inv.Send(context.Background(), "shopping-assistant", "session-1", "hello", "")
	if !errors.Is(err, ErrNotAuthenticated) {
		t.Fatalf("expected ErrNotAuthenticated, got %v", err)
	}
}

func TestAgentInvokerHTTPSendsCorrectRequestAndParsesMessageResult(t *testing.T) {
	var gotPath, gotAuth, gotMethod string
	var gotBody jsonRPCRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		gotMethod = r.Method
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		resp := jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      gotBody.ID,
			Result: json.RawMessage(`{
				"kind": "message",
				"role": "agent",
				"parts": [{"kind": "text", "text": "Here are three listings you might like."}]
			}`),
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(srv.Close)

	inv := newAgentInvokerHTTP(srv.URL)
	reply, err := inv.Send(context.Background(), "shopping-assistant", "session-1", "what's on sale?", "test-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reply != "Here are three listings you might like." {
		t.Fatalf("unexpected reply: %q", reply)
	}
	if gotMethod != http.MethodPost {
		t.Fatalf("expected POST, got %s", gotMethod)
	}
	if gotPath != "/agents/shopping-assistant" {
		t.Fatalf("expected path /agents/shopping-assistant, got %q", gotPath)
	}
	if gotAuth != "Bearer test-token" {
		t.Fatalf("expected Authorization: Bearer test-token, got %q", gotAuth)
	}
	if gotBody.Method != "message/send" {
		t.Fatalf("expected method message/send, got %q", gotBody.Method)
	}
	if gotBody.Params.Message.ContextID != "session-1" {
		t.Fatalf("expected contextId session-1, got %q", gotBody.Params.Message.ContextID)
	}
	if len(gotBody.Params.Message.Parts) != 1 || gotBody.Params.Message.Parts[0].Text != "what's on sale?" {
		t.Fatalf("unexpected message parts: %+v", gotBody.Params.Message.Parts)
	}
}

func TestAgentInvokerHTTPParsesTaskResult(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req jsonRPCRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		resp := jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: json.RawMessage(`{
				"kind": "task",
				"status": {
					"state": "completed",
					"message": {
						"kind": "message",
						"role": "agent",
						"parts": [{"kind": "text", "text": "Your bid was placed."}]
					}
				}
			}`),
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(srv.Close)

	inv := newAgentInvokerHTTP(srv.URL)
	reply, err := inv.Send(context.Background(), "shopping-assistant", "session-1", "bid $50", "test-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reply != "Your bid was placed." {
		t.Fatalf("unexpected reply: %q", reply)
	}
}

func TestAgentInvokerHTTPReturnsErrorOnJSONRPCError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req jsonRPCRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		resp := jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &jsonRPCError{Code: -32001, Message: "agent not found"},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(srv.Close)

	inv := newAgentInvokerHTTP(srv.URL)
	_, err := inv.Send(context.Background(), "nonexistent-agent", "session-1", "hi", "test-token")
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}

func TestAgentInvokerHTTPReturnsNotAuthenticatedOn401(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(srv.Close)

	inv := newAgentInvokerHTTP(srv.URL)
	_, err := inv.Send(context.Background(), "shopping-assistant", "session-1", "hi", "expired-token")
	if !errors.Is(err, ErrNotAuthenticated) {
		t.Fatalf("expected ErrNotAuthenticated, got %v", err)
	}
}
