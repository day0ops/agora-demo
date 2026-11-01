import { useEffect, useRef, useState } from "react";
import { formatCountdown } from "../format";
import type { Listing } from "../types";

const URGENT_THRESHOLD_MS = 60_000;

export interface CountdownState {
  text: string;
  urgent: boolean;
}

// useCountdown recomputes once a second while the listing is an active
// auction, calling onExpire exactly once the first time the remaining time
// crosses zero. The server is the source of truth for when an auction
// actually closes -- this only prompts a one-time refetch, never a
// replacement for server state. A closed auction (sold/unsold-expired/
// withdrawn) still has type "auction" and a past auctionEndsAt, so the
// status check (not just type) is what stops this rendering a stale
// "ending..." forever.
export function useCountdown(
  listing: Pick<Listing, "type" | "status" | "auctionEndsAt">,
  onExpire?: () => void,
): CountdownState | null {
  const [now, setNow] = useState(() => Date.now());
  const firedRef = useRef(false);
  const isLive =
    listing.type === "auction" && listing.status === "active-auction";

  useEffect(() => {
    firedRef.current = false;
  }, [listing.auctionEndsAt]);

  useEffect(() => {
    if (!isLive) return;
    const id = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(id);
  }, [isLive]);

  const remaining = listing.auctionEndsAt
    ? new Date(listing.auctionEndsAt).getTime() - now
    : Infinity;

  useEffect(() => {
    if (isLive && remaining <= 0 && !firedRef.current && onExpire) {
      firedRef.current = true;
      onExpire();
    }
  });

  if (!isLive) return null;
  return {
    text: formatCountdown(listing.auctionEndsAt, now),
    urgent: remaining < URGENT_THRESHOLD_MS,
  };
}
