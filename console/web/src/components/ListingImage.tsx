import type { Listing } from "../types";

const CATEGORY_FALLBACKS = new Set([
  "watches",
  "books",
  "cameras",
  "home",
  "memorabilia",
  "electronics",
  "art",
  "instruments",
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

// ListingImage falls back to the category SVG on a load error (missing or
// unfetchable photo) so it never shows as a broken image in the grid.
export function ListingImage({
  listing,
  eager = false,
}: {
  listing: Pick<Listing, "image" | "category" | "title">;
  eager?: boolean;
}) {
  const fallback = `/images/categories/${normalizeCategory(listing.category)}.svg`;
  return (
    <img
      className="listing-image"
      src={listingImageSrc(listing)}
      alt={listing.title}
      loading={eager ? "eager" : "lazy"}
      onError={(e) => {
        e.currentTarget.onerror = null;
        e.currentTarget.src = fallback;
      }}
    />
  );
}
