import "./style.css";
import { mountNavbar, mountFooter } from "./components/navbar";
import { mountChat } from "./chat";
import { connectEvents } from "./events";
import { listListings, createListing, ApiError } from "./api";
import { SELLERS } from "./identities";
import { escapeHtml } from "./html";
import { getStored, setStored } from "./storage";
import { formatCents } from "./format";
import { statusPill } from "./components/statusPill";
import type { Listing, ListingType, MarketplaceEvent } from "./types";

const AUCTION_DURATIONS = [
  { label: "90 seconds (demo)", ms: 90 * 1000 },
  { label: "5 minutes", ms: 5 * 60 * 1000 },
  { label: "15 minutes", ms: 15 * 60 * 1000 },
  { label: "1 hour", ms: 60 * 60 * 1000 },
];

const app = document.querySelector<HTMLDivElement>("#app")!;
app.innerHTML = `
  <div id="topbar"></div>
  <div class="page">
    <div class="identity-picker">
      Selling as
      <select id="seller-select"></select>
    </div>
    <div class="layout">
      <div class="panel">
        <h2>My listings</h2>
        <div class="grid" id="listing-grid"><div class="empty">Loading listings...</div></div>
      </div>
      <div class="panel-stack">
        <div class="panel" id="create-panel">
          <h2>New listing</h2>
          <form id="create-form">
            <label for="f-title">Title</label>
            <input id="f-title" required />
            <label for="f-description">Description</label>
            <textarea id="f-description"></textarea>
            <label for="f-category">Category</label>
            <input id="f-category" required />
            <label for="f-type">Type</label>
            <select id="f-type">
              <option value="auction">Auction</option>
              <option value="buy-now">Buy now</option>
            </select>
            <div id="f-duration-wrap">
              <label for="f-duration">Auction duration</label>
              <select id="f-duration">
                ${AUCTION_DURATIONS.map((d, i) => `<option value="${i}">${d.label}</option>`).join("")}
              </select>
            </div>
            <label for="f-price">Price ($)</label>
            <input id="f-price" type="number" min="0.01" step="0.01" required />
            <button type="submit" class="create-submit">Create Listing</button>
            <div class="note" id="create-error"></div>
          </form>
        </div>
        <div class="panel chat-panel">
          <h2>Pricing Assistant</h2>
          <div id="chat-mount" class="chat-mount"></div>
        </div>
      </div>
    </div>
  </div>
  <div id="footer"></div>
`;

mountNavbar(document.querySelector("#topbar")!, "sell");
mountFooter(document.querySelector("#footer")!);
mountChat(
  document.querySelector("#chat-mount")!,
  "listing-pricing-assistant",
  "Pricing Assistant",
);

const sellerSelect =
  document.querySelector<HTMLSelectElement>("#seller-select")!;
sellerSelect.innerHTML = SELLERS.map(
  (s) => `<option value="${s.id}">${escapeHtml(s.name)}</option>`,
).join("");
sellerSelect.value = getStored("mktSellerId") || SELLERS[0].id;
sellerSelect.addEventListener("change", () => {
  setStored("mktSellerId", sellerSelect.value);
  void loadListings();
});
function currentSellerId(): string {
  return sellerSelect.value;
}

const listings = new Map<string, Listing>();
const grid = document.querySelector<HTMLDivElement>("#listing-grid")!;

const STATUS_RANK: Record<string, number> = {
  "active-auction": 0,
  "active-buy-now": 0,
  draft: 1,
  sold: 2,
  "unsold-expired": 3,
  withdrawn: 4,
};

function priceFor(l: Listing): number {
  return l.type === "auction"
    ? l.currentBidCents || l.priceCents
    : l.priceCents;
}

function renderCard(l: Listing): string {
  return `
    <div class="card">
      <div class="title">${escapeHtml(l.title)}</div>
      <div class="meta">${escapeHtml(l.category)} &middot; ${l.type === "auction" ? "Auction" : "Buy now"}</div>
      <div class="price">${formatCents(priceFor(l))}</div>
      <div class="foot">
        ${statusPill(l.status)}
      </div>
    </div>
  `;
}

function renderGrid(): void {
  const mine = [...listings.values()]
    .filter((l) => l.sellerId === currentSellerId())
    .sort((a, b) => {
      const rankDiff =
        (STATUS_RANK[a.status] ?? 9) - (STATUS_RANK[b.status] ?? 9);
      if (rankDiff !== 0) return rankDiff;
      return a.createdAt < b.createdAt ? 1 : -1;
    });
  grid.innerHTML = mine.length
    ? mine.map(renderCard).join("")
    : '<div class="empty">No listings yet - create one.</div>';
}

async function loadListings(): Promise<void> {
  grid.innerHTML = '<div class="empty">Loading listings...</div>';
  try {
    const mine = await listListings({ sellerId: currentSellerId() });
    listings.clear();
    mine.forEach((l) => listings.set(l.id, l));
    renderGrid();
  } catch {
    grid.innerHTML = '<div class="empty">Could not reach catalog-api.</div>';
  }
}

const typeSelect = document.querySelector<HTMLSelectElement>("#f-type")!;
const durationWrap =
  document.querySelector<HTMLDivElement>("#f-duration-wrap")!;
function syncDurationVisibility(): void {
  durationWrap.style.display =
    typeSelect.value === "auction" ? "block" : "none";
}
typeSelect.addEventListener("change", syncDurationVisibility);
syncDurationVisibility();

const createForm = document.querySelector<HTMLFormElement>("#create-form")!;
const createError = document.querySelector<HTMLDivElement>("#create-error")!;
createError.style.display = "none";

createForm.addEventListener("submit", async (e) => {
  e.preventDefault();
  createError.style.display = "none";

  const type = typeSelect.value as ListingType;
  const priceDollars = Number(
    document.querySelector<HTMLInputElement>("#f-price")!.value,
  );
  const durationIndex = Number(
    document.querySelector<HTMLSelectElement>("#f-duration")!.value,
  );

  try {
    await createListing({
      sellerId: currentSellerId(),
      title: document.querySelector<HTMLInputElement>("#f-title")!.value,
      description:
        document.querySelector<HTMLTextAreaElement>("#f-description")!.value,
      category: document.querySelector<HTMLInputElement>("#f-category")!.value,
      type,
      priceCents: Math.round(priceDollars * 100),
      auctionEndsAt:
        type === "auction"
          ? new Date(
              Date.now() + AUCTION_DURATIONS[durationIndex].ms,
            ).toISOString()
          : undefined,
    });
    createForm.reset();
    syncDurationVisibility();
  } catch (err) {
    createError.textContent =
      err instanceof ApiError
        ? err.message
        : "Something went wrong. Please try again.";
    createError.classList.add("error");
    createError.style.display = "block";
  }
});

connectEvents((evt: MarketplaceEvent) => {
  switch (evt.type) {
    case "listing.created":
      if (evt.data.sellerId === currentSellerId()) {
        listings.set(evt.data.id, evt.data);
        renderGrid();
      }
      break;
    case "bid.placed":
    case "bid.withdrawn":
    case "order.created":
      void loadListings();
      break;
    case "autobid.set":
      if (evt.data.listing.sellerId === currentSellerId()) {
        listings.set(evt.data.listing.id, evt.data.listing);
        renderGrid();
      }
      break;
  }
});

void loadListings();
