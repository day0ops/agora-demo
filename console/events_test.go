package main

import (
	"testing"
	"time"
)

func TestEventHubDeliversToSubscriber(t *testing.T) {
	h := newEventHub()
	ch, unsubscribe := h.subscribe()
	t.Cleanup(unsubscribe)

	h.publish(Event{Type: "bid.placed", Data: "hello"})

	select {
	case evt := <-ch:
		if evt.Type != "bid.placed" || evt.Data != "hello" {
			t.Fatalf("unexpected event: %+v", evt)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestEventHubDeliversToAllSubscribers(t *testing.T) {
	h := newEventHub()
	ch1, unsub1 := h.subscribe()
	ch2, unsub2 := h.subscribe()
	t.Cleanup(unsub1)
	t.Cleanup(unsub2)

	h.publish(Event{Type: "test"})

	for i, ch := range []<-chan Event{ch1, ch2} {
		select {
		case <-ch:
		case <-time.After(time.Second):
			t.Fatalf("subscriber %d did not receive the event", i)
		}
	}
}

func TestEventHubUnsubscribeStopsDelivery(t *testing.T) {
	h := newEventHub()
	ch, unsubscribe := h.subscribe()
	unsubscribe()

	h.publish(Event{Type: "test"})

	// The channel should be closed, so a receive returns the zero value
	// immediately rather than blocking.
	select {
	case evt, open := <-ch:
		if open {
			t.Fatalf("expected the channel to be closed after unsubscribe, got event %+v", evt)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out: channel was not closed after unsubscribe")
	}
}

func TestEventHubUnsubscribeIsIdempotent(t *testing.T) {
	h := newEventHub()
	_, unsubscribe := h.subscribe()

	unsubscribe()
	unsubscribe() // must not panic (double close)
}

func TestEventHubPublishDoesNotBlockOnFullSubscriberBuffer(t *testing.T) {
	h := newEventHub()
	_, unsubscribe := h.subscribe() // never drained
	t.Cleanup(unsubscribe)

	done := make(chan struct{})
	go func() {
		for i := 0; i < 100; i++ { // far more than the channel's buffer size
			h.publish(Event{Type: "test"})
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("publish blocked on a full subscriber buffer")
	}
}

func TestEventHubSubscriberCount(t *testing.T) {
	h := newEventHub()
	if h.subscriberCount() != 0 {
		t.Fatalf("expected 0 subscribers, got %d", h.subscriberCount())
	}

	_, unsub1 := h.subscribe()
	_, unsub2 := h.subscribe()
	if h.subscriberCount() != 2 {
		t.Fatalf("expected 2 subscribers, got %d", h.subscriberCount())
	}

	unsub1()
	if h.subscriberCount() != 1 {
		t.Fatalf("expected 1 subscriber after unsubscribe, got %d", h.subscriberCount())
	}
	unsub2()
}
