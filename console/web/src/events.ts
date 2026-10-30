import type { MarketplaceEvent } from "./types";

// connectEvents opens the live SSE feed. The backend sends bare `data: <json>`
// frames (no `event:` line) with the event's own `type` field used for
// dispatch, so a plain onmessage handler covers every event type.
export function connectEvents(
  onEvent: (e: MarketplaceEvent) => void,
): EventSource {
  const es = new EventSource("/api/events");
  es.onmessage = (ev) => {
    try {
      onEvent(JSON.parse(ev.data) as MarketplaceEvent);
    } catch {
      // malformed frame - skip it, the next one will resync the view
    }
  };
  // EventSource retries on its own (browser-native backoff) - this is just
  // visibility into that happening, not a replacement for it.
  es.onerror = () => console.warn("events: SSE connection error, retrying");
  return es;
}
