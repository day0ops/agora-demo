export function currentTheme(): "dark" | "light" {
  return document.documentElement.getAttribute("data-theme") === "light"
    ? "light"
    : "dark";
}

// initTheme only syncs the toggle button's label to whatever index.html's
// pre-paint script already applied - it does not set the theme itself, so
// there is no flash between the pre-paint script and this running.
export function initTheme(onLabel: (label: string) => void): void {
  onLabel(currentTheme() === "dark" ? "\u{1F319}" : "☀️");
}

export function toggleTheme(onLabel: (label: string) => void): void {
  const next = currentTheme() === "dark" ? "light" : "dark";
  document.documentElement.setAttribute("data-theme", next);
  try {
    localStorage.setItem("mktTheme", next);
  } catch {
    // localStorage may be unavailable (e.g. private browsing) - theme still
    // applies for this page load, it just won't persist across reloads.
  }
  onLabel(next === "dark" ? "\u{1F319}" : "☀️");
}
