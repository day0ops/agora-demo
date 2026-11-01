import { initTheme, toggleTheme } from "../theme";
import { icons } from "../icons";
import { escapeHtml } from "../html";
import { canSell, canOperate, type AccountIdentity } from "../identity";
import { getVersion } from "../api";

export const APP_NAME = "Agora";

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

function closeAccountMenu(
  panel: HTMLDivElement,
  trigger: HTMLButtonElement,
): void {
  panel.hidden = true;
  trigger.setAttribute("aria-expanded", "false");
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
    ? `
      <div class="account-menu">
        <button type="button" class="ghost account-menu-trigger" id="account-menu-trigger" aria-haspopup="true" aria-expanded="false">
          <span class="account-avatar">${escapeHtml(identity.username.charAt(0).toUpperCase())}</span>
          <span class="account-name">${escapeHtml(identity.username)}</span>
          ${icons.chevronDown}
        </button>
        <div class="account-menu-panel" id="account-menu-panel" hidden>
          <a class="account-menu-item" href="/account/logout">Log out</a>
        </div>
      </div>
    `
    : `<a class="ghost account-link" href="/account/login">Login</a>`;

  if (identity) {
    const trigger = area.querySelector<HTMLButtonElement>(
      "#account-menu-trigger",
    )!;
    const panel = area.querySelector<HTMLDivElement>("#account-menu-panel")!;
    trigger.addEventListener("click", (e) => {
      e.stopPropagation();
      if (panel.hidden) {
        panel.hidden = false;
        trigger.setAttribute("aria-expanded", "true");
      } else {
        closeAccountMenu(panel, trigger);
      }
    });
    document.addEventListener("click", () => closeAccountMenu(panel, trigger));
    document.addEventListener("keydown", (e) => {
      if (e.key === "Escape") closeAccountMenu(panel, trigger);
    });
  }

  const sellLink = root.querySelector<HTMLAnchorElement>(
    'nav a[data-key="sell"]',
  );
  const opsLink = root.querySelector<HTMLAnchorElement>(
    'nav a[data-key="ops"]',
  );
  if (sellLink) sellLink.style.display = canSell(identity) ? "" : "none";
  if (opsLink) opsLink.style.display = canOperate(identity) ? "" : "none";
}

export function mountFooter(root: HTMLElement): void {
  root.classList.add("site-footer");
  root.innerHTML = `
    <span>${APP_NAME}</span>
    <span id="footer-version"></span>
    <a href="https://github.com/day0ops/agora-demo" target="_blank" rel="noopener noreferrer">GitHub</a>
  `;
  const versionEl = root.querySelector<HTMLSpanElement>("#footer-version")!;
  void getVersion()
    .then((v) => {
      versionEl.textContent = `v${v.version}`;
    })
    .catch(() => {
      // best-effort; the footer just omits the version if unreachable
    });
}
