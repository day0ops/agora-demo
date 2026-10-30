import { escapeHtml } from "../html";
import type { Listing } from "../types";

const CATEGORY_FALLBACKS = new Set([
  "watches",
  "books",
  "cameras",
  "home",
  "memorabilia",
]);

function normalizeCategory(category: string): string {
  const key = category.trim().toLowerCase();
  return CATEGORY_FALLBACKS.has(key) ? key : "default";
}

// listingImageSrc resolves the real photo for a seeded listing, or the
// listing's category fallback for anything without one (including every
// listing created through the Sell form at runtime).
export function listingImageSrc(
  listing: Pick<Listing, "image" | "category">,
): string {
  if (listing.image) return `/images/listings/${listing.image}.jpg`;
  return `/images/categories/${normalizeCategory(listing.category)}.svg`;
}

// listingImageEl renders an <img> for a listing's photo, with an onerror
// fallback to the category SVG so a missing or unfetchable photo never
// shows as a broken image in the grid.
export function listingImageEl(
  listing: Pick<Listing, "image" | "category" | "title">,
  opts: { eager?: boolean } = {},
): string {
  const fallback = `/images/categories/${normalizeCategory(listing.category)}.svg`;
  const src = listingImageSrc(listing);
  const loading = opts.eager ? "eager" : "lazy";
  return `<img class="listing-image" src="${src}" alt="${escapeHtml(listing.title)}" loading="${loading}" onerror="this.onerror=null;this.src='${fallback}'" />`;
}
