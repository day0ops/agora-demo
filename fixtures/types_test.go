package fixtures

import "testing"

func TestBidZeroValueHasNoTrustCheckOrCorrelationID(t *testing.T) {
	var b Bid
	if b.TrustCheck != nil {
		t.Fatalf("expected a zero-value Bid to have no trust check, got %+v", b.TrustCheck)
	}
	if b.CorrelationID != "" {
		t.Fatalf("expected a zero-value Bid to have no correlation id, got %q", b.CorrelationID)
	}
}

func TestOrderZeroValueHasNoCorrelationID(t *testing.T) {
	var o Order
	if o.CorrelationID != "" {
		t.Fatalf("expected a zero-value Order to have no correlation id, got %q", o.CorrelationID)
	}
}
