import "./style.css";
import {
  mountNavbar,
  mountFooter,
  renderAccountArea,
} from "./components/navbar";
import { mountChatPanel } from "./components/chatPanel";
import { connectEvents } from "./events";
import { listListings, createListing, ApiError } from "./api";
import { currentIdentity, canSell, type AccountIdentity } from "./identity";
import { lotCard } from "./components/lotCard";
import { emptyState } from "./components/emptyState";
import { icons } from "./icons";
import type { Listing, ListingType, MarketplaceEvent } from "./types";

const AUCTION_DURATIONS = [
  { label: "90 seconds (demo)", ms: 90 * 1000 },
  { label: "5 minutes", ms: 5 * 60 * 1000 },
  { label: "15 minutes", ms: 15 * 60 * 1000 },
  { label: "1 hour", ms: 60 * 60 * 1000 },
];

const app = document.querySelector<HTMLDivElement>("#app")!;
app.innerHTML = `
  <header id="topbar"></header>
  <main class="page" id="main"><div class="empty">Loading...</div></main>
  <footer id="footer"></footer>
`;

mountNavbar(document.querySelector("#topbar")!, "sell");
mountFooter(document.querySelector("#footer")!);

const main = document.querySelector<HTMLElement>("#main")!;

function renderDenied(): void {
  main.innerHTML = emptyState({
    title: "You need a seller account",
    body: "Log in as a seller to list items and manage your storefront.",
    icon: icons.sell,
  });
}

function renderSellerPage(identity: AccountIdentity): void {
  main.innerHTML = `
    <div class="layout">
      <div class="panel">
        <h2>My listings</h2>
        <div class="grid" id="listing-grid"><div class="empty">Loading listings...</div></div>
      </div>
      <div class="panel-stack">
        <div class="panel" id="create-panel">
          <h2>New listing</h2>
          <div class="form-step-label">Step 1 &middot; Draft details</div>
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

            <div class="assist-stack">
              <div class="assist-card">
                ${emptyState({
                  title: "Recommended pricing",
                  body: "Price suggestions appear here once the pricing assistant is connected to a live agent cluster.",
                  icon: icons.tag,
                })}
              </div>
              <div class="assist-card">
                ${emptyState({
                  title: "Comparable items",
                  body: "Similar listings and recent sale prices appear here once the pricing assistant is connected to a live agent cluster.",
                  icon: icons.gavel,
                })}
              </div>
            </div>

            <div class="review-step">
              <div class="form-step-label">Step 2 &middot; Review &amp; publish</div>
              <p class="review-step-note">Confirm the details above, then publish your listing.</p>
              <button type="submit" class="create-submit">Create Listing</button>
            </div>
            <div class="note" id="create-error"></div>
          </form>
        </div>
        <div class="panel chat-panel">
          <h2>Pricing Assistant</h2>
          <div id="pro-badge-row"></div>
          <div id="chat-mount" class="chat-mount"></div>
        </div>
      </div>
    </div>
  `;

  mountChatPanel(document.querySelector("#chat-mount")!, {
    agent: "listing-pricing-assistant",
    label: "Pricing Assistant",
  });

  const proBadgeRow = document.querySelector<HTMLDivElement>("#pro-badge-row")!;
  proBadgeRow.innerHTML =
    identity.sellerTier === "pro"
      ? '<span class="pill accent pro-badge">Pro pricing assistant</span>'
      : "";

  const sellerId = identity.fixtureId!;
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

  function renderGrid(): void {
    const mine = [...listings.values()]
      .filter((l) => l.sellerId === sellerId)
      .sort((a, b) => {
        const rankDiff =
          (STATUS_RANK[a.status] ?? 9) - (STATUS_RANK[b.status] ?? 9);
        if (rankDiff !== 0) return rankDiff;
        return a.createdAt < b.createdAt ? 1 : -1;
      });
    grid.innerHTML = mine.length
      ? mine.map((l) => lotCard(l)).join("")
      : emptyState({
          title: "No listings yet",
          body: "Create a listing below to see it appear here.",
          icon: icons.sell,
        });
  }

  async function loadListings(): Promise<void> {
    grid.innerHTML = '<div class="empty">Loading listings...</div>';
    try {
      const mine = await listListings({ sellerId });
      listings.clear();
      mine.forEach((l) => listings.set(l.id, l));
      renderGrid();
    } catch {
      grid.innerHTML = emptyState({
        title: "Catalog unavailable",
        body: "Could not reach catalog-api. Try refreshing in a moment.",
        icon: icons.sell,
      });
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
        title: document.querySelector<HTMLInputElement>("#f-title")!.value,
        description:
          document.querySelector<HTMLTextAreaElement>("#f-description")!.value,
        category:
          document.querySelector<HTMLInputElement>("#f-category")!.value,
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
        if (evt.data.sellerId === sellerId) {
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
        if (evt.data.listing.sellerId === sellerId) {
          listings.set(evt.data.listing.id, evt.data.listing);
          renderGrid();
        }
        break;
    }
  });

  void loadListings();
}

void currentIdentity().then((identity) => {
  renderAccountArea(document.querySelector("#topbar")!, identity);
  if (canSell(identity)) {
    renderSellerPage(identity!);
  } else {
    renderDenied();
  }
});
