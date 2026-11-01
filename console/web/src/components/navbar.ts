import { initTheme, toggleTheme } from "../theme";
import { icons } from "../icons";
import { escapeHtml } from "../html";
import type { AccountIdentity } from "../identity";

export const APP_NAME = "Agora";
export const APP_VERSION = "0.1.0";

type View = "index" | "shop" | "sell" | "ops";

const NAV: { href: string; key: View; label: string }[] = [
  { href: "/shop.html", key: "shop", label: "Shop" },
  { href: "/sell.html", key: "sell", label: "Sell" },
  { href: "/ops.html", key: "ops", label: "Operator" },
];

export const BRAND_LINK = `<a class="brand" href="/index.html">${icons.brand}${APP_NAME}</a>`;

function setThemeToggleIcon(
  button: HTMLButtonElement,
  theme: "dark" | "light",
): void {
  button.innerHTML = theme === "dark" ? icons.moon : icons.sun;
  button.setAttribute(
    "aria-label",
    theme === "dark" ? "Switch to light theme" : "Switch to dark theme",
  );
}

export function mountThemeToggle(toggle: HTMLButtonElement): void {
  initTheme((theme) => setThemeToggleIcon(toggle, theme));
  toggle.addEventListener("click", () =>
    toggleTheme((theme) => setThemeToggleIcon(toggle, theme)),
  );
}

export function mountNavbar(root: HTMLElement, active: View): void {
  root.classList.add("topbar");
  root.innerHTML = `
    ${BRAND_LINK}
    <nav>
      ${NAV.map((n) => `<a href="${n.href}" data-key="${n.key}" class="${n.key === active ? "active" : ""}">${n.label}</a>`).join("")}
    </nav>
    <div class="account-area" id="account-area"></div>
    <button id="theme-toggle" class="ghost" type="button"></button>
  `;
  mountThemeToggle(root.querySelector<HTMLButtonElement>("#theme-toggle")!);
}

// renderAccountArea fills in the login/account UI and hides nav links the
// visitor's role doesn't grant, once GET /account/me has resolved. Split
// from mountNavbar so every page can render its static chrome immediately
// and patch in identity-dependent UI as soon as it's known, rather than
// blocking the whole page on that fetch.
export function renderAccountArea(
  root: HTMLElement,
  identity: AccountIdentity | null,
): void {
  const area = root.querySelector<HTMLDivElement>("#account-area")!;
  area.innerHTML = identity
    ? `<span class="account-name">${escapeHtml(identity.username)}</span><a class="account-link" href="/account/logout">Log out</a>`
    : `<a class="account-link" href="/account/login">Login</a>`;

  const canSell =
    identity?.role === "sellers" || identity?.role === "operators";
  const canOperate = identity?.role === "operators";
  const sellLink = root.querySelector<HTMLAnchorElement>(
    'nav a[data-key="sell"]',
  );
  const opsLink = root.querySelector<HTMLAnchorElement>(
    'nav a[data-key="ops"]',
  );
  if (sellLink) sellLink.style.display = canSell ? "" : "none";
  if (opsLink) opsLink.style.display = canOperate ? "" : "none";
}

export function mountFooter(root: HTMLElement): void {
  root.classList.add("site-footer");
  root.innerHTML = `
    <span>${APP_NAME}</span>
    <span>v${APP_VERSION}</span>
    <a href="https://github.com/day0ops/agora-demo" target="_blank" rel="noopener noreferrer">GitHub</a>
  `;
}
