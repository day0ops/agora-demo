// localStorage may be unavailable (e.g. private browsing, or a storage-
// blocked embed) - these wrap every access so a thrown SecurityError can't
// blank a whole page over a non-essential identity/theme preference.
export function getStored(key: string): string | null {
  try {
    return localStorage.getItem(key);
  } catch {
    return null;
  }
}

export function setStored(key: string, value: string): void {
  try {
    localStorage.setItem(key, value);
  } catch {
    // the selection still applies for this page load; it just won't persist.
  }
}
