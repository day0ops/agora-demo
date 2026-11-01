import { StatusPill } from "./StatusPill";
import { ListingImage } from "./ListingImage";
import { useCountdown } from "../hooks/useCountdown";
import { formatCents, lotNumber } from "../format";
import type { Listing } from "../types";

function priceFor(l: Listing): number {
  return l.type === "auction"
    ? l.currentBidCents || l.priceCents
    : l.priceCents;
}

// Only Shop's grid wires onSelect/interactive; Sell's grid is a read-only
// status view, so it skips the button semantics a true click target needs.
export function LotCard({
  listing,
  selected = false,
  interactive = false,
  onSelect,
  onExpire,
}: {
  listing: Listing;
  selected?: boolean;
  interactive?: boolean;
  onSelect?: () => void;
  onExpire?: () => void;
}) {
  const countdown = useCountdown(listing, onExpire);

  return (
    <div
      className={`card lot-card ${selected ? "selected" : ""}`}
      data-id={listing.id}
      tabIndex={interactive ? 0 : undefined}
      role={interactive ? "button" : undefined}
      aria-current={interactive ? (selected ? "true" : "false") : undefined}
      onClick={interactive ? onSelect : undefined}
      onKeyDown={
        interactive
          ? (e) => {
              if (e.key === "Enter" || e.key === " ") {
                e.preventDefault();
                onSelect?.();
              }
            }
          : undefined
      }
    >
      <ListingImage listing={listing} />
      <span className="lot-plate">{lotNumber(listing.id)}</span>
      <div className="title">{listing.title}</div>
      <div className="meta">{listing.category}</div>
      <div className="price">{formatCents(priceFor(listing))}</div>
      <div className="foot">
        <StatusPill status={listing.status} />
        {countdown && (
          <span className={`countdown${countdown.urgent ? " urgent" : ""}`}>
            {countdown.text}
          </span>
        )}
      </div>
    </div>
  );
}
