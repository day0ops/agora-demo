package main

import (
	"context"
	"errors"
	"testing"
)

func TestAgentInvokerStubReturnsNotConfigured(t *testing.T) {
	var s AgentInvokerStub
	_, err := s.Send(context.Background(), "shopping-assistant", "session-1", "hello")
	if !errors.Is(err, ErrAgentInvokerNotConfigured) {
		t.Fatalf("expected ErrAgentInvokerNotConfigured, got %v", err)
	}
}
