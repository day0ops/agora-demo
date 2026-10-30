import "./style.css";
import { mountNavbar, mountFooter } from "./components/navbar";
import { listListings } from "./api";
import { listingImageEl } from "./components/listingImage";
import { statusPill } from "./components/statusPill";
import { emptyState } from "./components/emptyState";
import { escapeHtml } from "./html";
import { formatCents } from "./format";
import { icons } from "./icons";
import type { Listing } from "./types";

const app = document.querySelector<HTMLDivElement>("#app")!;
app.innerHTML = `
  <div id="topbar"></div>
  <div class="hero">
    <h1>A marketplace where agents do the work.</h1>
    <p>
      Discover, negotiate, buy, and sell with intelligent agents working
      together on your behalf.
    </p>
    <div class="hero-ctas">
      <a class="primary" href="/shop.html">Explore marketplace</a>
      <a class="secondary" href="/ops.html">See agents in action</a>
    </div>
  </div>
  <div class="rails">
    <section class="rail">
      <h2>Featured</h2>
      <div class="rail-track" id="rail-featured"><div class="empty">Loading...</div></div>
    </section>
    <section class="rail">
      <h2>Closing soon</h2>
      <div class="rail-track" id="rail-closing-soon"><div class="empty">Loading...</div></div>
    </section>
    <section class="rail">
      <h2>Just listed</h2>
      <div class="rail-track" id="rail-just-listed"><div class="empty">Loading...</div></div>
    </section>
  </div>
  <div id="footer"></div>
`;

mountNavbar(document.querySelector("#topbar")!, "index");
mountFooter(document.querySelector("#footer")!);

const featuredTrack = document.querySelector<HTMLDivElement>("#rail-featured")!;
const closingSoonTrack =
  document.querySelector<HTMLDivElement>("#rail-closing-soon")!;
const justListedTrack =
  document.querySelector<HTMLDivElement>("#rail-just-listed")!;

function priceFor(l: Listing): number {
  return l.type === "auction"
    ? l.currentBidCents || l.priceCents
    : l.priceCents;
}

function renderRailCard(l: Listing): string {
  return `
    <a class="rail-card" href="/shop.html">
      ${listingImageEl(l)}
      <div class="rail-card-title">${escapeHtml(l.title)}</div>
      <div class="rail-card-foot">
        <span class="price">${formatCents(priceFor(l))}</span>
        ${statusPill(l.status)}
      </div>
    </a>
  `;
}

function renderRail(
  track: HTMLDivElement,
  listings: Listing[],
  empty: { title: string; body: string },
): void {
  track.innerHTML = listings.length
    ? listings.map(renderRailCard).join("")
    : emptyState({ ...empty, icon: icons.shop });
}

function renderUnreachable(track: HTMLDivElement): void {
  track.innerHTML = emptyState({
    title: "Catalog unavailable",
    body: "Could not reach catalog-api. Try refreshing in a moment.",
    icon: icons.shop,
  });
}

async function loadRails(): Promise<void> {
  let listings: Listing[];
  try {
    listings = await listListings();
  } catch {
    renderUnreachable(featuredTrack);
    renderUnreachable(closingSoonTrack);
    renderUnreachable(justListedTrack);
    return;
  }

  const active = listings.filter(
    (l) => l.status === "active-auction" || l.status === "active-buy-now",
  );

  const featured = [...active]
    .sort((a, b) => priceFor(b) - priceFor(a))
    .slice(0, 4);
  renderRail(featuredTrack, featured, {
    title: "No listings yet",
    body: "Check back soon as sellers list items.",
  });

  const closingSoon = active
    .filter((l) => l.status === "active-auction")
    .sort(
      (a, b) =>
        new Date(a.auctionEndsAt ?? 0).getTime() -
        new Date(b.auctionEndsAt ?? 0).getTime(),
    )
    .slice(0, 4);
  renderRail(closingSoonTrack, closingSoon, {
    title: "No auctions closing soon",
    body: "Live auctions will appear here as they near their end.",
  });

  const justListed = [...active]
    .sort(
      (a, b) =>
        new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime(),
    )
    .slice(0, 4);
  renderRail(justListedTrack, justListed, {
    title: "No listings yet",
    body: "New listings will appear here as sellers add them.",
  });
}

void loadRails();
