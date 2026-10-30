export const icons: Record<string, string> = {
  // A Greek-temple/colonnade mark (pediment, architrave, columns) for the
  // Agora brand - the ancient agora was lined with colonnaded stoas.
  brand: `
    <svg viewBox="0 0 24 24" fill="currentColor" width="20" height="20" aria-hidden="true">
      <path d="M12 2 22 9.5H2L12 2Z"/>
      <rect x="3" y="9.5" width="18" height="2.2"/>
      <rect x="5" y="12.6" width="2.1" height="6.6"/>
      <rect x="10.95" y="12.6" width="2.1" height="6.6"/>
      <rect x="16.9" y="12.6" width="2.1" height="6.6"/>
      <rect x="2" y="19.6" width="20" height="2.2"/>
    </svg>
  `,
  shop: `<svg viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><path d="M8.3 8V6a3.7 3.7 0 0 1 7.4 0v2h-1.6V6a2.1 2.1 0 0 0-4.2 0v2H8.3Z"/><path d="M4 8h16l-1.5 12H5.5L4 8Z"/></svg>`,
  sell: `<svg viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><path fill-rule="evenodd" d="M9 4H21V20H9L2 12ZM7 10.4a1.6 1.6 0 1 0 0 3.2a1.6 1.6 0 1 0 0-3.2Z"/></svg>`,
  ops: `<svg viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><path d="M4 20V10h4v10H4Zm6 0V4h4v16h-4Zm6 0v-7h4v7h-4Z"/></svg>`,
  sun: `<svg viewBox="0 0 24 24" fill="currentColor" width="18" height="18" aria-hidden="true"><path d="M12 7a5 5 0 1 0 0 10a5 5 0 1 0 0-10Z"/><rect x="11" y="1" width="2" height="4" rx="1"/><rect x="11" y="19" width="2" height="4" rx="1"/><rect x="1" y="11" width="4" height="2" rx="1"/><rect x="19" y="11" width="4" height="2" rx="1"/></svg>`,
  moon: `<svg viewBox="0 0 24 24" fill="currentColor" width="18" height="18" aria-hidden="true"><path d="M20.5 14.5A8.5 8.5 0 1 1 9.9 3.9a6.5 6.5 0 0 0 10.6 10.6Z"/></svg>`,
  search: `<svg viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><path fill-rule="evenodd" d="M10 3a7 7 0 1 0 0 14a7 7 0 1 0 0-14ZM10 4.8a5.2 5.2 0 1 0 0 10.4a5.2 5.2 0 1 0 0-10.4Z"/><rect x="14" y="14.5" width="9" height="2.4" rx="1.2" transform="rotate(45 14 15.7)"/></svg>`,
  gavel: `<svg viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><rect x="4" y="4" width="13" height="5" rx="1.5"/><rect x="12.5" y="8.5" width="3" height="10" rx="1.5"/><rect x="3" y="20" width="9" height="2.2" rx="1"/></svg>`,
  tag: `<svg viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><path fill-rule="evenodd" d="M3 4H13L21 12L13 20H3V4ZM6.3 6.8a1.3 1.3 0 1 0 0 2.6a1.3 1.3 0 1 0 0-2.6Z"/></svg>`,
  watches: `<svg viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><path fill-rule="evenodd" d="M12 2.2a9.8 9.8 0 1 0 0 19.6a9.8 9.8 0 1 0 0-19.6ZM12 4.2a7.8 7.8 0 1 0 0 15.6a7.8 7.8 0 1 0 0-15.6Z"/><rect x="11.2" y="6.5" width="1.6" height="6" rx="0.8"/><rect x="12" y="11.3" width="5" height="1.6" rx="0.8" transform="rotate(35 12 12)"/></svg>`,
  books: `<svg viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><rect x="3" y="8" width="4" height="12" rx="0.8"/><rect x="8.5" y="5" width="4" height="15" rx="0.8"/><rect x="14" y="10" width="4" height="10" rx="0.8"/><rect x="19" y="7" width="2.5" height="13" rx="0.8"/></svg>`,
  cameras: `<svg viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><path d="M9 4H15L16.5 7H22V19H2V7H7.5L9 4Z"/><path fill-rule="evenodd" d="M12 8.5a4.5 4.5 0 1 0 0 9a4.5 4.5 0 1 0 0-9ZM12 10.7a2.3 2.3 0 1 0 0 4.6a2.3 2.3 0 1 0 0-4.6Z"/></svg>`,
  home: `<svg viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><path d="M12 3 2 12h3v8h5v-6h4v6h5v-8h3L12 3Z"/></svg>`,
  memorabilia: `<svg viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><path d="M12 2 14.4 8.8 21.5 8.9 15.8 13.2 17.9 20.1 12 16 6.1 20.1 8.2 13.2 2.5 8.9 9.7 8.8Z"/></svg>`,
  default: `<svg viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><path fill-rule="evenodd" d="M4 4H20V20H4ZM8 8H16V16H8Z"/></svg>`,
};

export function categoryIcon(category: string): string {
  return icons[category.trim().toLowerCase()] ?? icons.default;
}
