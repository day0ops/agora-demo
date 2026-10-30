import { initTheme, toggleTheme } from "./theme";
import { BRAND_ICON } from "./icon";

export const APP_NAME = "Agora";
export const APP_VERSION = "0.1.0";

type View = "index" | "shop" | "sell" | "ops";

const NAV: { href: string; key: View; label: string }[] = [
  { href: "/shop.html", key: "shop", label: "Shop" },
  { href: "/sell.html", key: "sell", label: "Sell" },
  { href: "/ops.html", key: "ops", label: "Operator" },
];

export const BRAND_LINK = `<a class="brand" href="/index.html">${BRAND_ICON}${APP_NAME}</a>`;

export function mountTopbar(root: HTMLElement, active: View): void {
  root.classList.add("topbar");
  root.innerHTML = `
    ${BRAND_LINK}
    <nav>
      ${NAV.map((n) => `<a href="${n.href}" class="${n.key === active ? "active" : ""}">${n.label}</a>`).join("")}
    </nav>
    <button id="theme-toggle" class="ghost" type="button"></button>
  `;
  const toggle = root.querySelector<HTMLButtonElement>("#theme-toggle")!;
  initTheme((label) => (toggle.textContent = label));
  toggle.addEventListener("click", () =>
    toggleTheme((label) => (toggle.textContent = label)),
  );
}

export function mountFooter(root: HTMLElement): void {
  root.classList.add("site-footer");
  root.innerHTML = `
    <span>${APP_NAME}</span>
    <span>v${APP_VERSION}</span>
    <a href="https://github.com/day0ops/marketplace-demo" target="_blank" rel="noopener noreferrer">GitHub</a>
  `;
}
