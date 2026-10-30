import { formatCountdown } from "../format";
import type { Listing } from "../types";

const URGENT_THRESHOLD_MS = 60_000;

export function countdownEl(
  listing: Listing,
  now: number = Date.now(),
): string {
  // A closed auction (sold/unsold-expired/withdrawn) still has type
  // "auction" and a past auctionEndsAt, so without the status check this
  // rendered a stale "ending..." forever instead of nothing.
  if (listing.type !== "auction" || listing.status !== "active-auction") {
    return "";
  }
  const remaining = listing.auctionEndsAt
    ? new Date(listing.auctionEndsAt).getTime() - now
    : Infinity;
  const urgent = remaining < URGENT_THRESHOLD_MS ? " urgent" : "";
  return `<span class="countdown${urgent}" id="cd-${listing.id}">${formatCountdown(listing.auctionEndsAt, now)}</span>`;
}

// Updates every rendered countdownEl span in place and returns the ids of
// listings whose auction has just reached zero, so a caller can trigger a
// one-time refetch (the server is the source of truth for when an auction
// actually closes).
export function refreshCountdowns(
  listings: Iterable<Listing>,
  now: number,
): string[] {
  const expired: string[] = [];
  for (const l of listings) {
    if (l.type !== "auction" || l.status !== "active-auction") continue;
    const remaining = l.auctionEndsAt
      ? new Date(l.auctionEndsAt).getTime() - now
      : Infinity;
    const el = document.getElementById(`cd-${l.id}`);
    if (el) {
      el.textContent = formatCountdown(l.auctionEndsAt, now);
      el.classList.toggle("urgent", remaining < URGENT_THRESHOLD_MS);
    }
    if (remaining <= 0) expired.push(l.id);
  }
  return expired;
}
