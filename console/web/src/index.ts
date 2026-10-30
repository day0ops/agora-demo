import "./style.css";
import { initTheme, toggleTheme } from "./theme";
import { BRAND_LINK, mountFooter } from "./layout";

const app = document.querySelector<HTMLDivElement>("#app")!;
app.innerHTML = `
  <div class="topbar">
    ${BRAND_LINK}
    <nav></nav>
    <button id="theme-toggle" class="ghost" type="button"></button>
  </div>
  <div class="hero">
    <h1>A live marketplace, run by agents</h1>
    <p>
      Browse live auctions and buy-now listings, place bids, and list items of your
      own - with AI shopping and pricing assistants on hand to help. Pick a view
      below to explore the live catalog.
    </p>
    <div class="hero-cards">
      <a class="hero-card" href="/shop.html">
        <span class="emoji">🛍️</span>
        <b>Shop</b>
        <span>Browse auctions and buy-now listings, bid, and chat with the shopping assistant</span>
      </a>
      <a class="hero-card" href="/sell.html">
        <span class="emoji">🏷️</span>
        <b>Sell</b>
        <span>List items, track your auctions, and chat with the pricing assistant</span>
      </a>
      <a class="hero-card" href="/ops.html">
        <span class="emoji">📊</span>
        <b>Operator</b>
        <span>Watch live marketplace activity across every buyer and seller</span>
      </a>
    </div>
  </div>
  <div id="footer"></div>
`;

const toggle = app.querySelector<HTMLButtonElement>("#theme-toggle")!;
initTheme((label) => (toggle.textContent = label));
toggle.addEventListener("click", () =>
  toggleTheme((label) => (toggle.textContent = label)),
);
mountFooter(app.querySelector("#footer")!);
