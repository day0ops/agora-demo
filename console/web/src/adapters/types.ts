// AdapterResult is the contract every Phase 2/3 data adapter returns.
// Components switch on `status` - never infer "real" from a merely non-null
// value - so an adapter can go from always-empty to real with no caller
// changes. See docs/superpowers/specs/2026-10-30-agora-ui-redesign-design.md
// section 17.2.
export type AdapterResult<T> =
  { status: "real"; data: T } | { status: "empty"; reason: string };
