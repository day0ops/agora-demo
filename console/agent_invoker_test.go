package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

// taskResultJSON builds a kagent-controller-shaped SendMessage result
// (confirmed live, 2026-11-02): the reply lives in the last artifact's
// parts, and contextId sits on the task itself, not nested under status.
func taskResultJSON(contextID, replyText string) json.RawMessage {
	return json.RawMessage(fmt.Sprintf(`{
		"task": {
			"id": "task-1",
			"contextId": %q,
			"artifacts": [{"artifactId": "artifact-1", "parts": [{"kind": "text", "text": %q}]}],
			"status": {"state": "TASK_STATE_COMPLETED"}
		}
	}`, contextID, replyText))
}

func TestAgentInvokerHTTPSendsCorrectRequestAndParsesTaskResult(t *testing.T) {
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
			Result:  taskResultJSON("kagent-context-1", "Here are three listings you might like."),
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(srv.Close)

	inv := newAgentInvokerHTTP(srv.URL + "/agents/kagent-system")
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
	if gotPath != "/agents/kagent-system/shopping-assistant" {
		t.Fatalf("expected path /agents/kagent-system/shopping-assistant, got %q", gotPath)
	}
	if gotAuth != "Bearer test-token" {
		t.Fatalf("expected Authorization: Bearer test-token, got %q", gotAuth)
	}
	if gotBody.Method != "SendMessage" {
		t.Fatalf("expected method SendMessage, got %q", gotBody.Method)
	}
	if gotBody.Params.Message.Role != "ROLE_USER" {
		t.Fatalf("expected role ROLE_USER, got %q", gotBody.Params.Message.Role)
	}
	if gotBody.Params.Message.ContextID != "" {
		t.Fatalf("expected no contextId on a session's first message, got %q", gotBody.Params.Message.ContextID)
	}
	if len(gotBody.Params.Message.Parts) != 1 || gotBody.Params.Message.Parts[0].Text != "what's on sale?" {
		t.Fatalf("unexpected message parts: %+v", gotBody.Params.Message.Parts)
	}
}

func TestAgentInvokerHTTPReusesServerAssignedContextIDOnSecondCall(t *testing.T) {
	var contextIDsSent []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req jsonRPCRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		contextIDsSent = append(contextIDsSent, req.Params.Message.ContextID)
		resp := jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  taskResultJSON("kagent-context-1", "reply"),
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(srv.Close)

	inv := newAgentInvokerHTTP(srv.URL)
	if _, err := inv.Send(context.Background(), "shopping-assistant", "session-1", "first", "test-token"); err != nil {
		t.Fatalf("first call: unexpected error: %v", err)
	}
	if _, err := inv.Send(context.Background(), "shopping-assistant", "session-1", "second", "test-token"); err != nil {
		t.Fatalf("second call: unexpected error: %v", err)
	}

	if len(contextIDsSent) != 2 || contextIDsSent[0] != "" || contextIDsSent[1] != "kagent-context-1" {
		t.Fatalf("expected contextIds [\"\", \"kagent-context-1\"], got %v", contextIDsSent)
	}
}

func TestAgentInvokerHTTPReturnsErrorOnTaskWithNoArtifacts(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req jsonRPCRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		resp := jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  json.RawMessage(`{"task": {"id": "task-1", "contextId": "c1", "artifacts": [], "status": {"state": "TASK_STATE_WORKING"}}}`),
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(srv.Close)

	inv := newAgentInvokerHTTP(srv.URL)
	_, err := inv.Send(context.Background(), "shopping-assistant", "session-1", "hi", "test-token")
	if err == nil {
		t.Fatal("expected an error, got nil")
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
