# Issue Update: EDM-3407

## Root Cause

Device status showed "While Rebooting: **unknown** failed: required resource not found" because the error chain from the Rebooting phase (e.g. BeforeRebooting hook or CreateRollback) was a single-wrap: the device layer wraps with `ErrPhaseActivatingConfig` and the underlying error (e.g. `ErrReadingHookActionsFrom` when a hook file cannot be read) was not exposed as a second-level "component" to `FormatError`. The formatting logic only extracts a component when the error implements `Unwrap() []error`; single-wrap chains left component nil, so the UI displayed "unknown".

## Fix

We infer a user-facing component by walking the error chain and matching known sentinels when `FormatError` cannot extract a component via `splitWrapped`. Added `InferComponentDisplay(err)` in the errors package to map `ErrReadingHookActionsFrom` and `ErrMissingRenderedSpec` to display strings ("reading hook actions from", "missing rendered spec"). When component is nil after the usual extraction, we call this helper and set component to a display-only error so the message shows a meaningful component instead of "unknown". No changes to device, hook, or spec manager; fix is limited to status message formatting.

## Testing

- [X] Unit tests added: `TestMessage/Rebooting_phase_with_inferred_component_(EDM-3407)`, `TestInferComponentDisplay`
- [X] Regression test fails without the fix (message contains "unknown"), passes with the fix
- [X] Full unit suite passes (3608 tests)
- [X] Lint passes
- [ ] Integration tests not run (change is formatting-only)
- [ ] E2E not run (existing structured-error E2E remains valid)

## Files Changed

- `internal/agent/device/errors/errors.go` — Added `InferComponentDisplay(err error) string`
- `internal/agent/device/errors/structured.go` — Added `componentDisplayError`; in `FormatError`, when component is nil, set from `InferComponentDisplay(rest)`
- `internal/agent/device/errors/structured_test.go` — New TestMessage case for EDM-3407
- `internal/agent/device/errors/errors_test.go` — New `TestInferComponentDisplay`

## Breaking Changes / Migrations

None. Message format unchanged; only the value of the component field improves from "unknown" to a concrete string when the chain contains a known sentinel.

Fixed in branch: `bugfix/EDM-3407-rollback-rebooting-unknown-error` (PR to follow).
