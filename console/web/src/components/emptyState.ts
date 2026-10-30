export function emptyState(opts: {
  title: string;
  body: string;
  icon?: string;
}): string {
  return `
    <div class="empty-state">
      ${opts.icon ? `<div class="empty-state-icon">${opts.icon}</div>` : ""}
      <div class="empty-state-title">${opts.title}</div>
      <div class="empty-state-body">${opts.body}</div>
    </div>
  `;
}
