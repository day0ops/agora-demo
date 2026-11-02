package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
)

// AgentInvoker sends a message to a named concierge agent (e.g.
// "shopping-assistant", "listing-pricing-assistant") on behalf of a user and
// returns the agent's reply. token is the caller's own Keycloak access
// token, forwarded by agentgateway's OIDC gate (oidc-gateway feature) after
// a real browser login -- there is no service-account identity here, every
// call is made as the logged-in user.
type AgentInvoker interface {
	Send(ctx context.Context, agentName, sessionID, message, token string) (reply string, err error)
}

// ErrAgentInvokerNotConfigured is returned by AgentInvokerStub for every call.
var ErrAgentInvokerNotConfigured = errors.New("agent invocation is not wired to a live kagent cluster yet")

// AgentInvokerStub is kept for tests and for running the console with no
// agentgateway configured at all (AGENTGATEWAY_BASE_URL unset).
type AgentInvokerStub struct{}

func (AgentInvokerStub) Send(context.Context, string, string, string, string) (string, error) {
	return "", ErrAgentInvokerNotConfigured
}

// ErrNotAuthenticated means the request carried no usable Keycloak token -
// either oidc-gateway never forwarded one (no session), or the agent's A2A
// route rejected the token as expired/invalid. Both are the same thing from
// the console's point of view: the user isn't validly logged in right now.
var ErrNotAuthenticated = errors.New("not authenticated: please log in")

// AgentInvokerHTTP calls a kagent Agent's A2A endpoint through agentgateway,
// using kagent's A2A JSON-RPC "SendMessage" method (PascalCase, not the A2A
// spec's "message/send" -- confirmed live against this cluster's enterprise
// kagent-controller build) at <baseURL>/agents/<agentName>, authenticated
// with the caller's own forwarded Keycloak access token. See docs/superpowers/specs/
// 2026-11-01-agora-oidc-agentgateway-a2a-design.md in agentic-field-kit for
// why this goes through agentgateway rather than a kagent-controller client
// library: kagent's own AgentInstance gRPC API (the recipe the sibling
// fraud-ops-console demo used) no longer exists at the Agent/Harness/
// AgentTemplate CRD generation this cluster runs.
type AgentInvokerHTTP struct {
	BaseURL string
	Client  *http.Client
}

func newAgentInvokerHTTP(baseURL string) AgentInvokerHTTP {
	return AgentInvokerHTTP{BaseURL: baseURL, Client: &http.Client{Timeout: 30 * time.Second}}
}

type a2aPart struct {
	Kind string `json:"kind"`
	Text string `json:"text"`
}

type a2aMessage struct {
	Kind      string    `json:"kind,omitempty"`
	Role      string    `json:"role"`
	Parts     []a2aPart `json:"parts"`
	MessageID string    `json:"messageId,omitempty"`
	// ContextID carries the console's own sessionID, so repeated calls with
	// the same session continue one A2A conversation instead of each
	// starting a new one.
	ContextID string `json:"contextId,omitempty"`
}

type a2aSendMessageParams struct {
	Message a2aMessage `json:"message"`
}

type jsonRPCRequest struct {
	JSONRPC string               `json:"jsonrpc"`
	ID      string               `json:"id"`
	Method  string               `json:"method"`
	Params  a2aSendMessageParams `json:"params"`
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type a2aTaskStatus struct {
	State   string      `json:"state"`
	Message *a2aMessage `json:"message"`
}

// a2aResult decodes either shape SendMessage's result can take: a direct
// Message (the agent's whole reply, usual case for a short chat turn) or a
// Task (a longer-running unit of work, whose final reply lives at
// status.message). Kind discriminates which one this is.
type a2aResult struct {
	Kind   string         `json:"kind"`
	Parts  []a2aPart      `json:"parts"`
	Status *a2aTaskStatus `json:"status"`
}

type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      string          `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *jsonRPCError   `json:"error"`
}

func replyText(parts []a2aPart) (string, error) {
	for _, p := range parts {
		if p.Text != "" {
			return p.Text, nil
		}
	}
	return "", errors.New("agent reply had no text part")
}

func (a AgentInvokerHTTP) Send(ctx context.Context, agentName, sessionID, message, token string) (string, error) {
	if token == "" {
		return "", ErrNotAuthenticated
	}

	reqBody := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      sessionID,
		Method:  "SendMessage",
		Params: a2aSendMessageParams{
			Message: a2aMessage{
				Role:      "ROLE_USER",
				Parts:     []a2aPart{{Kind: "text", Text: message}},
				MessageID: fmt.Sprintf("msg-%d", time.Now().UnixNano()),
				ContextID: sessionID,
			},
		},
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal A2A request: %w", err)
	}

	reqURL := a.BaseURL + "/agents/" + url.PathEscape(agentName)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("build A2A request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+token)

	resp, err := a.Client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("call agent %q: %w", agentName, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return "", ErrNotAuthenticated
	}
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read agent %q response: %w", agentName, err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("agent %q returned HTTP %d: %s", agentName, resp.StatusCode, respBytes)
	}

	var rpcResp jsonRPCResponse
	if err := json.Unmarshal(respBytes, &rpcResp); err != nil {
		log.Printf("agent %q returned undecodable 200 body (first 500 bytes): %.500s", agentName, respBytes)
		return "", fmt.Errorf("decode agent %q response: %w", agentName, err)
	}
	if rpcResp.Error != nil {
		return "", fmt.Errorf("agent %q returned JSON-RPC error %d: %s", agentName, rpcResp.Error.Code, rpcResp.Error.Message)
	}

	var result a2aResult
	if err := json.Unmarshal(rpcResp.Result, &result); err != nil {
		return "", fmt.Errorf("decode agent %q result: %w", agentName, err)
	}
	switch result.Kind {
	case "message":
		return replyText(result.Parts)
	case "task":
		if result.Status == nil || result.Status.Message == nil {
			return "", fmt.Errorf("agent %q task has no final message (state=%v)", agentName, resultState(result))
		}
		return replyText(result.Status.Message.Parts)
	default:
		return "", fmt.Errorf("agent %q returned unrecognized result kind %q", agentName, result.Kind)
	}
}

func resultState(r a2aResult) string {
	if r.Status == nil {
		return ""
	}
	return r.Status.State
}
