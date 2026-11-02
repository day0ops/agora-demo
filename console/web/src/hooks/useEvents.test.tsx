import { renderHook } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { useEvents } from "./useEvents";

class MockEventSource {
  onmessage: ((ev: MessageEvent) => void) | null = null;
  onerror: (() => void) | null = null;
  closed = false;
  static instances: MockEventSource[] = [];
  constructor(public url: string) {
    MockEventSource.instances.push(this);
  }
  close() {
    this.closed = true;
  }
}

beforeEach(() => {
  MockEventSource.instances = [];
  // @ts-expect-error -- test double, not a full EventSource implementation
  global.EventSource = MockEventSource;
});

describe("useEvents", () => {
  it("subscribes to /api/events and dispatches parsed messages", () => {
    const handler = vi.fn();
    renderHook(() => useEvents(handler));

    const es = MockEventSource.instances[0];
    expect(es.url).toBe("/api/events");
    es.onmessage?.({
      data: JSON.stringify({ type: "listing.created", data: { id: "l1" } }),
    } as MessageEvent);
    expect(handler).toHaveBeenCalledWith({
      type: "listing.created",
      data: { id: "l1" },
    });
  });

  it("closes the connection on unmount", () => {
    const { unmount } = renderHook(() => useEvents(vi.fn()));
    const es = MockEventSource.instances[0];
    unmount();
    expect(es.closed).toBe(true);
  });
});
