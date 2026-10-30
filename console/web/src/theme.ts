export function currentTheme(): "dark" | "light" {
  return document.documentElement.getAttribute("data-theme") === "light"
    ? "light"
    : "dark";
}

// initTheme only reports whatever theme index.html's pre-paint script already
// applied - it does not set the theme itself, so there is no flash between
// the pre-paint script and this running.
export function initTheme(onTheme: (theme: "dark" | "light") => void): void {
  onTheme(currentTheme());
}

export function toggleTheme(onTheme: (theme: "dark" | "light") => void): void {
  const next = currentTheme() === "dark" ? "light" : "dark";
  document.documentElement.setAttribute("data-theme", next);
  try {
    localStorage.setItem("mktTheme", next);
  } catch {
    // localStorage may be unavailable (e.g. private browsing) - theme still
    // applies for this page load, it just won't persist across reloads.
  }
  onTheme(next);
}
