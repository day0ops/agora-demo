import { initTheme, toggleTheme } from "../theme";
import { icons } from "../icons";

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
      ${NAV.map((n) => `<a href="${n.href}" class="${n.key === active ? "active" : ""}">${n.label}</a>`).join("")}
    </nav>
    <button id="theme-toggle" class="ghost" type="button"></button>
  `;
  mountThemeToggle(root.querySelector<HTMLButtonElement>("#theme-toggle")!);
}

export function mountFooter(root: HTMLElement): void {
  root.classList.add("site-footer");
  root.innerHTML = `
    <span>${APP_NAME}</span>
    <span>v${APP_VERSION}</span>
    <a href="https://github.com/day0ops/agora-demo" target="_blank" rel="noopener noreferrer">GitHub</a>
  `;
}
