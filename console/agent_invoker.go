package main

import (
	"context"
	"errors"
)

// AgentInvoker sends a message to a named concierge agent (e.g.
// "shopping-assistant", "listing-pricing-assistant") on behalf of a user and
// returns the agent's reply.
//
// The real implementation talks to kagent Agent Substrate via
// CreateAgentInstance + A2A SendMessage -- a recipe already proven working
// for the sibling fraud-ops-console demo. It is deliberately NOT wired in
// here: that recipe was built and verified against an older kagent
// architecture generation. kagent v1.0.0-alpha12 (the version this project
// now targets, per agentic-field-kit's foundation compatibility work)
// replaced the Harness/AgentTemplate label-selector admission model with an
// explicit Agent CRD pairing, and changed how kagent's own native
// token-propagation plugin is configured. Reusing the old recipe without
// re-verifying it against a live alpha12 cluster would mean shipping
// integration code nobody has confirmed still works against the version
// this actually targets. AgentInvokerStub below is the interim
// implementation this server actually runs, so the console can be built,
// tested, and demoed end to end (browsing, bidding, checkout, live events)
// without a live cluster, while making it unmistakable that no real agent
// response is being produced yet.
type AgentInvoker interface {
	// Send returns the agent's reply text, or an error if the agent could
	// not be reached or failed to respond.
	Send(ctx context.Context, agentName, sessionID, message string) (reply string, err error)
}

// ErrAgentInvokerNotConfigured is returned by AgentInvokerStub for every
// call.
var ErrAgentInvokerNotConfigured = errors.New("agent invocation is not wired to a live kagent cluster yet")

// AgentInvokerStub is the AgentInvoker this server runs until a real,
// version-verified kagent-backed implementation replaces it.
type AgentInvokerStub struct{}

func (AgentInvokerStub) Send(context.Context, string, string, string) (string, error) {
	return "", ErrAgentInvokerNotConfigured
}
