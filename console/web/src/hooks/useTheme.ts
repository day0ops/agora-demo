import { useCallback, useState } from "react";

function currentTheme(): "dark" | "light" {
  return document.documentElement.getAttribute("data-theme") === "light"
    ? "light"
    : "dark";
}

// useTheme reports whatever theme index.html's pre-paint script already
// applied (it does not set the theme itself, so there is no flash between
// that script and this hook mounting), and exposes a toggle that flips it.
export function useTheme(): {
  theme: "dark" | "light";
  toggleTheme: () => void;
} {
  const [theme, setTheme] = useState<"dark" | "light">(currentTheme);

  const toggleTheme = useCallback(() => {
    const next = currentTheme() === "dark" ? "light" : "dark";
    document.documentElement.setAttribute("data-theme", next);
    try {
      localStorage.setItem("mktTheme", next);
    } catch {
      // localStorage may be unavailable (e.g. private browsing) - theme still
      // applies for this page load, it just won't persist across reloads.
    }
    setTheme(next);
  }, []);

  return { theme, toggleTheme };
}
