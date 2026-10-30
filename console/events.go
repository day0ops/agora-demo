package main

import "sync"

// Event is a single live event broadcast to connected SSE clients -- a new
// bid, a checkout, a listing created. Type tells the frontend how to render
// it; Data is an arbitrary JSON-serializable payload.
type Event struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

// eventHub fans out events to every currently-connected SSE client. Safe for
// concurrent use.
type eventHub struct {
	mu   sync.Mutex
	subs map[chan Event]struct{}
}

func newEventHub() *eventHub {
	return &eventHub{subs: make(map[chan Event]struct{})}
}

// subscribe registers a new subscriber and returns a channel of events plus
// an unsubscribe function the caller must call exactly once when done (e.g.
// on client disconnect).
func (h *eventHub) subscribe() (<-chan Event, func()) {
	ch := make(chan Event, 16)

	h.mu.Lock()
	h.subs[ch] = struct{}{}
	h.mu.Unlock()

	var once sync.Once
	unsubscribe := func() {
		once.Do(func() {
			h.mu.Lock()
			delete(h.subs, ch)
			h.mu.Unlock()
			close(ch)
		})
	}
	return ch, unsubscribe
}

// publish broadcasts an event to every current subscriber. Non-blocking: a
// slow or stuck subscriber's full buffer just drops the event rather than
// blocking the publisher -- the live feed is best-effort, not a reliable
// log (catalog-api remains the source of truth for anything that must not
// be missed).
func (h *eventHub) publish(evt Event) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subs {
		select {
		case ch <- evt:
		default:
			// subscriber's buffer is full; drop for this one rather than block
			// every other subscriber and the publisher.
		}
	}
}

// subscriberCount reports how many clients are currently connected. Used by
// the operator console's live metrics.
func (h *eventHub) subscriberCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.subs)
}
