import { escapeHtml } from "../html";
import { formatCents, lotNumber } from "../format";
import { statusPill } from "./statusPill";
import { countdownEl } from "./countdown";
import { listingImageEl } from "./listingImage";
import type { Listing } from "../types";

function priceFor(l: Listing): number {
  return l.type === "auction"
    ? l.currentBidCents || l.priceCents
    : l.priceCents;
}

export function lotCard(
  listing: Listing,
  opts: { selected?: boolean } = {},
): string {
  return `
    <div class="card lot-card ${opts.selected ? "selected" : ""}" data-id="${listing.id}">
      ${listingImageEl(listing)}
      <span class="lot-plate">${lotNumber(listing.id)}</span>
      <div class="title">${escapeHtml(listing.title)}</div>
      <div class="meta">${escapeHtml(listing.category)}</div>
      <div class="price">${formatCents(priceFor(listing))}</div>
      <div class="foot">
        ${statusPill(listing.status)}
        ${countdownEl(listing)}
      </div>
    </div>
  `;
}
