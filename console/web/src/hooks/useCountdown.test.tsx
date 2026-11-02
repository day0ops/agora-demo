import { act, renderHook } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { useCountdown } from "./useCountdown";

beforeEach(() => vi.useFakeTimers());
afterEach(() => vi.useRealTimers());

describe("useCountdown", () => {
  it("returns null for a listing that isn't an active auction", () => {
    const { result } = renderHook(() =>
      useCountdown({
        type: "buy-now",
        status: "active-buy-now",
        auctionEndsAt: undefined,
      }),
    );
    expect(result.current).toBeNull();
  });

  it("returns countdown text and urgency for an active auction", () => {
    const endsAt = new Date(Date.now() + 30_000).toISOString();
    const { result } = renderHook(() =>
      useCountdown({
        type: "auction",
        status: "active-auction",
        auctionEndsAt: endsAt,
      }),
    );
    expect(result.current?.urgent).toBe(true);
    expect(result.current?.text).toMatch(/left/);
  });

  it("calls onExpire exactly once when the remaining time crosses zero", () => {
    const endsAt = new Date(Date.now() + 1000).toISOString();
    const onExpire = vi.fn();
    renderHook(() =>
      useCountdown(
        { type: "auction", status: "active-auction", auctionEndsAt: endsAt },
        onExpire,
      ),
    );
    act(() => vi.advanceTimersByTime(1000));
    act(() => vi.advanceTimersByTime(1000));
    act(() => vi.advanceTimersByTime(1000));
    expect(onExpire).toHaveBeenCalledTimes(1);
  });
});
