import { useEffect, useRef } from "react";
import type { MarketplaceEvent } from "../types";

// useEvents opens the live SSE feed for the component's lifetime. The
// backend sends bare `data: <json>` frames (no `event:` line) with the
// event's own `type` field used for dispatch, so a plain onmessage handler
// covers every event type. The latest onEvent is read through a ref rather
// than recreating the EventSource on every render, so a caller passing a
// fresh inline closure each render doesn't churn the connection.
export function useEvents(onEvent: (e: MarketplaceEvent) => void): void {
  const handlerRef = useRef(onEvent);
  handlerRef.current = onEvent;

  useEffect(() => {
    const es = new EventSource("/api/events");
    es.onmessage = (ev) => {
      try {
        handlerRef.current(JSON.parse(ev.data) as MarketplaceEvent);
      } catch {
        // malformed frame - skip it, the next one will resync the view
      }
    };
    // EventSource retries on its own (browser-native backoff) - this is just
    // visibility into that happening, not a replacement for it.
    es.onerror = () => console.warn("events: SSE connection error, retrying");
    return () => es.close();
  }, []);
}
