import { icons } from "../icons";
import { escapeHtml } from "../html";
import { getAgentJourney } from "../adapters/executionAdapter";
import { executionTimeline } from "./executionTimeline";
import { technicalInspector } from "./technicalInspector";

export interface ExecutionTrustCheck {
  score: number;
  reasons: string[];
}

export interface ExecutionSummary {
  title: string;
  body: string;
  trustCheck: ExecutionTrustCheck | "unavailable";
}

export interface OpenExecutionDrawerOptions {
  summary: ExecutionSummary;
  correlationId?: string;
}

let drawerEl: HTMLDivElement | null = null;
let lastFocused: HTMLElement | null = null;

function trustCheckLine(
  trustCheck: ExecutionTrustCheck | "unavailable",
): string {
  if (trustCheck === "unavailable") {
    return "Trust check: not available for this bid.";
  }
  if (trustCheck.reasons.length === 0) {
    return `Trust check: cleared (score ${trustCheck.score.toFixed(2)}).`;
  }
  return `Trust check: flagged (score ${trustCheck.score.toFixed(2)}) - ${trustCheck.reasons.join("; ")}`;
}

function ensureDrawer(): HTMLDivElement {
  if (drawerEl) return drawerEl;
  const el = document.createElement("div");
  el.className = "execution-drawer-root hidden";
  el.innerHTML = `
    <div class="execution-drawer-backdrop" data-close></div>
    <div class="execution-drawer" role="dialog" aria-modal="true" aria-labelledby="execution-drawer-title">
      <div class="execution-drawer-head">
        <h2 id="execution-drawer-title">Behind the scenes</h2>
        <button type="button" class="ghost" data-close aria-label="Close">${icons.close}</button>
      </div>
      <div class="execution-drawer-body">
        <section class="execution-section"><h3>Summary</h3><div id="execution-summary"></div></section>
        <section class="execution-section"><h3>Agent Journey</h3><div id="execution-timeline"></div></section>
        <section class="execution-section"><h3>Technical Inspector</h3><div id="execution-inspector"></div></section>
      </div>
    </div>
  `;
  document.body.appendChild(el);

  el.querySelectorAll<HTMLElement>("[data-close]").forEach((n) =>
    n.addEventListener("click", closeExecutionDrawer),
  );
  el.addEventListener("keydown", (e) => {
    if (e.key === "Escape") {
      closeExecutionDrawer();
      return;
    }
    if (e.key !== "Tab") return;
    const focusable = el.querySelectorAll<HTMLElement>(
      'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])',
    );
    if (focusable.length === 0) return;
    const first = focusable[0];
    const last = focusable[focusable.length - 1];
    if (e.shiftKey && document.activeElement === first) {
      e.preventDefault();
      last.focus();
    } else if (!e.shiftKey && document.activeElement === last) {
      e.preventDefault();
      first.focus();
    }
  });

  drawerEl = el;
  return el;
}

export function closeExecutionDrawer(): void {
  if (!drawerEl || drawerEl.classList.contains("hidden")) return;
  drawerEl.classList.add("hidden");
  lastFocused?.focus();
}

export function openExecutionDrawer(opts: OpenExecutionDrawerOptions): void {
  const el = ensureDrawer();
  lastFocused = document.activeElement as HTMLElement | null;

  el.querySelector<HTMLElement>("#execution-summary")!.innerHTML = `
    <p>${escapeHtml(opts.summary.title)}</p>
    <p>${escapeHtml(opts.summary.body)}</p>
    <p class="trust-check-line">${escapeHtml(trustCheckLine(opts.summary.trustCheck))}</p>
  `;

  const timelineEl = el.querySelector<HTMLElement>("#execution-timeline")!;
  const inspectorEl = el.querySelector<HTMLElement>("#execution-inspector")!;
  timelineEl.innerHTML = "";
  inspectorEl.innerHTML = "";
  void getAgentJourney(opts.correlationId).then((result) => {
    timelineEl.innerHTML = executionTimeline(result);
    inspectorEl.innerHTML = technicalInspector(result);
  });

  el.classList.remove("hidden");
  el.querySelector<HTMLButtonElement>(".execution-drawer-head button")!.focus();
}
